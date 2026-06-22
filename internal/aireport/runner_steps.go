package aireport

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/provider/goprovider"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/report"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// crapStepResult holds the outputs of runCRAPStep.
type crapStepResult struct {
	JSON                json.RawMessage
	CRAPload            int
	GazeCRAPload        *int
	TotalFunctions      int
	SSADegradedPackages []string
}

// runCRAPStep runs the CRAP analysis pipeline and returns the JSON output
// alongside the typed CRAPload and GazeCRAPload values for threshold
// evaluation (avoiding a second JSON unmarshal in EvaluateThresholds).
//
// coverProfile is the path to a pre-generated Go coverage profile. When
// non-empty, it is forwarded to crap.Options.CoverProfile so that crap.Analyze
// reads the supplied file directly instead of spawning go test internally
// (FR-001, FR-002). An empty string uses the default internal generation path.
//
// ccProvider is an optional ContractCoverageProvider for GazeCRAP scoring.
// When non-nil, it is set on crap.Options.ContractCoverageProvider, enabling
// GazeCRAP scores, quadrant classification, and GazeCRAPload computation.
// When nil, only line-coverage-based CRAP scores are produced (spec 022).
func runCRAPStep(patterns []string, moduleDir string, coverProfile string, stderr io.Writer, ccProvider crap.ContractCoverageProvider) (*crapStepResult, error) {
	opts := crap.DefaultOptions()
	opts.CoverProfile = coverProfile
	opts.Stderr = stderr
	opts.ComplexityProvider = goprovider.NewComplexityProvider()
	opts.LineCoverageProvider = goprovider.NewLineCoverageProvider(stderr)
	if ccProvider != nil {
		opts.ContractCoverageProvider = ccProvider
	}

	rpt, err := crap.Analyze(patterns, moduleDir, opts)
	if err != nil {
		return nil, fmt.Errorf("CRAP analysis: %w", err)
	}

	raw, err := captureJSON(func(w io.Writer) error {
		return crap.WriteJSON(w, rpt)
	})
	if err != nil {
		return nil, err
	}

	res := &crapStepResult{
		JSON:                raw,
		CRAPload:            rpt.Summary.CRAPload,
		GazeCRAPload:        rpt.Summary.GazeCRAPload,
		TotalFunctions:      rpt.Summary.TotalFunctions,
		SSADegradedPackages: rpt.Summary.SSADegradedPackages,
	}
	return res, nil
}

// qualityStepResult holds the outputs of runQualityStep.
type qualityStepResult struct {
	JSON                json.RawMessage
	AvgContractCoverage int
	SSADegraded         bool
	SSADegradedPackages []string
}

// runQualityStep runs the quality pipeline across all matched packages and
// returns the aggregated JSON output alongside the typed AvgContractCoverage
// value for threshold evaluation.
func runQualityStep(patterns []string, moduleDir string, stderr io.Writer) (*qualityStepResult, error) {
	pkgPaths, err := loader.ResolvePackagePaths(patterns, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("resolving packages for quality: %w", err)
	}
	if len(pkgPaths) == 0 {
		return nil, fmt.Errorf("no packages matched patterns %v", patterns)
	}

	gazeConfig := loadGazeConfigBestEffort(moduleDir)

	// Hoist LoadModule out of the per-package loop — O(1) instead of O(n).
	modPkgs := resolveModulePackages(moduleDir)

	var allReports []taxonomy.QualityReport
	var degradedPkgs []string
	for _, pkgPath := range pkgPaths {
		reports, degradedPkg := runQualityForPackage(pkgPath, gazeConfig, modPkgs, stderr)
		if degradedPkg != "" {
			degradedPkgs = append(degradedPkgs, degradedPkg)
		}
		allReports = append(allReports, reports...)
	}

	summary := quality.BuildPackageSummary(allReports)
	if len(degradedPkgs) > 0 {
		summary.SSADegraded = true
		summary.SSADegradedPackages = degradedPkgs
	}
	raw, err := captureJSON(func(w io.Writer) error {
		return quality.WriteJSON(w, allReports, summary)
	})
	if err != nil {
		return nil, err
	}

	avgCov := 0
	if summary != nil {
		avgCov = int(summary.AverageContractCoverage)
	}
	return &qualityStepResult{
		JSON:                raw,
		AvgContractCoverage: avgCov,
		SSADegraded:         len(degradedPkgs) > 0,
		SSADegradedPackages: degradedPkgs,
	}, nil
}

// runQualityForPackage runs the quality pipeline on a single package.
// modPkgs should be pre-resolved by the caller (hoist LoadModule out of loops).
// Returns (nil, "") if the package has no tests or analysis fails.
// The second return value is the degraded package path (empty string
// if not degraded, package path if SSA construction failed).
func runQualityForPackage(
	pkgPath string,
	gazeConfig *config.GazeConfig,
	modPkgs []*packages.Package,
	stderr io.Writer,
) ([]taxonomy.QualityReport, string) {
	includeUnexported := loader.IsMainPkg(pkgPath)
	if includeUnexported {
		_, _ = fmt.Fprintf(stderr, "package main detected for %s, including unexported functions\n", pkgPath)
	}
	analysisOpts := analysis.Options{IncludeUnexported: includeUnexported}
	results, err := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
	if err != nil || len(results) == 0 {
		return nil, ""
	}

	cfg := gazeConfig
	classified, err := runClassifyResults(results, pkgPath, cfg, modPkgs)
	if err != nil || len(classified) == 0 {
		return nil, ""
	}

	testPkg, err := loadTestPackageForQuality(pkgPath)
	if err != nil {
		return nil, ""
	}

	qualOpts := quality.Options{Stderr: stderr}
	reports, summary, err := quality.Assess(classified, testPkg, qualOpts)
	if err != nil {
		return nil, ""
	}
	if summary != nil && summary.SSADegraded {
		return reports, pkgPath
	}
	return reports, ""
}

// classifyStepResult holds the outputs of runClassifyStep.
type classifyStepResult struct {
	JSON        json.RawMessage
	Contractual int
	Ambiguous   int
	Incidental  int
}

// runClassifyStep runs classification on all matched packages and returns the JSON output
// alongside typed classification label counts.
func runClassifyStep(patterns []string, moduleDir string) (*classifyStepResult, error) {
	// Use the first resolved package path for analysis + classify.
	pkgPaths, err := loader.ResolvePackagePaths(patterns, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("resolving packages for classification: %w", err)
	}
	if len(pkgPaths) == 0 {
		return nil, fmt.Errorf("no packages matched patterns %v", patterns)
	}

	// Hoist LoadModule out of the per-package loop — O(1) instead of O(n).
	modPkgs := resolveModulePackages(moduleDir)

	gazeConfig := loadGazeConfigBestEffort(moduleDir)
	var allResults []taxonomy.AnalysisResult

	for _, pkgPath := range pkgPaths {
		analysisOpts := analysis.Options{IncludeUnexported: loader.IsMainPkg(pkgPath)}
		results, err := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
		if err != nil || len(results) == 0 {
			continue
		}
		classified, err := runClassifyResults(results, pkgPath, gazeConfig, modPkgs)
		if err != nil {
			continue
		}
		allResults = append(allResults, classified...)
	}

	raw, err := captureJSON(func(w io.Writer) error {
		return report.WriteJSON(w, allResults, "")
	})
	if err != nil {
		return nil, err
	}

	contractual, ambiguous, incidental := classify.CountLabels(allResults)
	return &classifyStepResult{
		JSON:        raw,
		Contractual: contractual,
		Ambiguous:   ambiguous,
		Incidental:  incidental,
	}, nil
}

// runDocscanStep runs the documentation scanner and returns the JSON output.
func runDocscanStep(moduleDir string) (json.RawMessage, error) {
	cfg := loadGazeConfigBestEffort(moduleDir)
	scanOpts := docscan.ScanOptions{Config: cfg}

	docs, err := docscan.Scan(moduleDir, scanOpts)
	if err != nil {
		return nil, fmt.Errorf("docscan: %w", err)
	}
	return captureJSON(func(w io.Writer) error {
		enc := json.NewEncoder(w)
		return enc.Encode(docs)
	})
}

// runClassifyResults runs the mechanical classification pipeline.
// modPkgs must be pre-resolved by the caller via resolveModulePackages to
// avoid calling loader.LoadModule inside a per-package loop (O(n) → O(1)).
func runClassifyResults(
	results []taxonomy.AnalysisResult,
	pkgPath string,
	cfg *config.GazeConfig,
	modPkgs []*packages.Package,
) ([]taxonomy.AnalysisResult, error) {
	targetResult, err := loader.Load(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("loading target package for classification: %w", err)
	}

	clOpts := classify.Options{
		Config:         cfg,
		ModulePackages: modPkgs,
		TargetPkg:      targetResult.Pkg,
	}
	return classify.Classify(results, clOpts), nil
}

// resolveModulePackages loads all module packages from moduleDir for use in
// classification. Returns nil (not an error) if loading fails, so callers can
// degrade gracefully.
func resolveModulePackages(moduleDir string) []*packages.Package {
	if moduleDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil
		}
		root, findErr := loader.FindModuleRoot(cwd)
		if findErr != nil {
			return nil
		}
		moduleDir = root
	}
	modResult, err := loader.LoadModule(moduleDir)
	if err != nil {
		return nil
	}
	return modResult.Packages
}

// loadGazeConfigBestEffort loads the GazeConfig from the module root,
// falling back to the default config on any error.
func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig {
	cfgPath := filepath.Join(moduleDir, ".gaze.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}

// loadTestPackageForQuality loads a Go package with test files included.
func loadTestPackageForQuality(pkgPath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("loading test package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %q", pkgPath)
	}
	for _, pkg := range pkgs {
		if quality.HasTestSyntax(pkg) {
			return pkg, nil
		}
	}
	return nil, fmt.Errorf("no test package found for %q", pkgPath)
}
