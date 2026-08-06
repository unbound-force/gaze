package aireport

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/cliutil"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/provider/goprovider"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/report"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// qualityPipelineDeps holds injectable function dependencies for
// runQualityStep, runQualityForPackage, and runClassifyStep. When a
// field is nil, the real implementation is used. This enables unit
// testing of the quality/classify orchestration logic without running
// real package loading or SSA analysis.
//
// Design decision: follows the pipelineStepFuncs pattern
// (runner.go:243) — variadic parameter with nil-means-default
// resolution. Chosen over interface-based DI per SOLID Interface
// Segregation: these are internal, co-located functions, not a
// public contract.
type qualityPipelineDeps struct {
	resolvePackagePaths func([]string, string) ([]string, error)
	loadAndAnalyze      func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)
	classifyResults     func([]taxonomy.AnalysisResult, string, *config.GazeConfig, []*packages.Package) ([]taxonomy.AnalysisResult, error)
	loadTestPkg         func(string) (*packages.Package, error)
	assess              func([]taxonomy.AnalysisResult, *packages.Package, quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error)
	resolveModulePkgs   func(string) []*packages.Package
	loadConfig          func(string) *config.GazeConfig
}

// resolveQualityDeps resolves nil fields to their production defaults.
// Accepts a variadic slice for ergonomic call-site usage: callers pass
// zero or one qualityPipelineDeps value.
func resolveQualityDeps(deps []qualityPipelineDeps) qualityPipelineDeps {
	var d qualityPipelineDeps
	if len(deps) > 0 {
		d = deps[0]
	}
	if d.resolvePackagePaths == nil {
		// Wrap loader.ResolvePackagePaths to match the DI function
		// type (2-param). Warnings are discarded (nil stderr) to
		// avoid cascading the io.Writer parameter through the DI
		// type and all test fakes. Direct callers in cmd/gaze and
		// goprovider pass stderr for user-facing warning output.
		d.resolvePackagePaths = func(patterns []string, moduleDir string) ([]string, error) {
			return loader.ResolvePackagePaths(patterns, moduleDir, nil)
		}
	}
	if d.loadAndAnalyze == nil {
		d.loadAndAnalyze = analysis.LoadAndAnalyze
	}
	if d.classifyResults == nil {
		d.classifyResults = runClassifyResults
	}
	if d.loadTestPkg == nil {
		d.loadTestPkg = goprovider.LoadTestPackage
	}
	if d.assess == nil {
		d.assess = quality.Assess
	}
	if d.resolveModulePkgs == nil {
		d.resolveModulePkgs = resolveModulePackages
	}
	if d.loadConfig == nil {
		d.loadConfig = config.LoadFromDir
	}
	return d
}

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
func runCRAPStep(patterns []string, moduleDir string, coverProfile string, stderr io.Writer, ccProvider crap.ContractCoverageProvider, short bool) (*crapStepResult, error) {
	opts := crap.DefaultOptions()
	opts.CoverProfile = coverProfile
	opts.Stderr = stderr
	opts.ComplexityProvider = goprovider.NewComplexityProvider()
	opts.LineCoverageProvider = &goprovider.GoLineCoverageProvider{Stderr: stderr, Short: short}
	if ccProvider != nil {
		opts.ContractCoverageProvider = ccProvider
	}

	rpt, err := crap.Analyze(patterns, moduleDir, opts)
	if err != nil {
		return nil, fmt.Errorf("CRAP analysis: %w", err)
	}

	raw, err := cliutil.CaptureJSON(func(w io.Writer) error {
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
func runQualityStep(patterns []string, moduleDir string, stderr io.Writer, deps ...qualityPipelineDeps) (*qualityStepResult, error) {
	d := resolveQualityDeps(deps)

	pkgPaths, err := d.resolvePackagePaths(patterns, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("resolving packages for quality: %w", err)
	}
	if len(pkgPaths) == 0 {
		return nil, fmt.Errorf("no packages matched patterns %v", patterns)
	}

	gazeConfig := d.loadConfig(moduleDir)

	// Hoist LoadModule out of the per-package loop — O(1) instead of O(n).
	modPkgs := d.resolveModulePkgs(moduleDir)

	var allReports []taxonomy.QualityReport
	var degradedPkgs []string
	for _, pkgPath := range pkgPaths {
		reports, degradedPkg := runQualityForPackage(pkgPath, gazeConfig, modPkgs, stderr, deps...)
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
	raw, err := cliutil.CaptureJSON(func(w io.Writer) error {
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
	deps ...qualityPipelineDeps,
) ([]taxonomy.QualityReport, string) {
	d := resolveQualityDeps(deps)

	includeUnexported := loader.IsMainPkg(pkgPath)
	if includeUnexported {
		_, _ = fmt.Fprintf(stderr, "package main detected for %s, including unexported functions\n", pkgPath)
	}
	analysisOpts := analysis.Options{IncludeUnexported: includeUnexported}
	results, err := d.loadAndAnalyze(pkgPath, analysisOpts)
	if err != nil || len(results) == 0 {
		return nil, ""
	}

	cfg := gazeConfig
	classified, err := d.classifyResults(results, pkgPath, cfg, modPkgs)
	if err != nil || len(classified) == 0 {
		return nil, ""
	}

	testPkg, err := d.loadTestPkg(pkgPath)
	if err != nil {
		return nil, ""
	}

	qualOpts := quality.Options{Stderr: stderr}
	reports, summary, err := d.assess(classified, testPkg, qualOpts)
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
func runClassifyStep(patterns []string, moduleDir string, deps ...qualityPipelineDeps) (*classifyStepResult, error) {
	d := resolveQualityDeps(deps)

	// Use the first resolved package path for analysis + classify.
	pkgPaths, err := d.resolvePackagePaths(patterns, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("resolving packages for classification: %w", err)
	}
	if len(pkgPaths) == 0 {
		return nil, fmt.Errorf("no packages matched patterns %v", patterns)
	}

	// Hoist LoadModule out of the per-package loop — O(1) instead of O(n).
	modPkgs := d.resolveModulePkgs(moduleDir)

	gazeConfig := d.loadConfig(moduleDir)
	var allResults []taxonomy.AnalysisResult

	for _, pkgPath := range pkgPaths {
		analysisOpts := analysis.Options{IncludeUnexported: loader.IsMainPkg(pkgPath)}
		results, err := d.loadAndAnalyze(pkgPath, analysisOpts)
		if err != nil || len(results) == 0 {
			continue
		}
		classified, err := d.classifyResults(results, pkgPath, gazeConfig, modPkgs)
		if err != nil {
			continue
		}
		allResults = append(allResults, classified...)
	}

	raw, err := cliutil.CaptureJSON(func(w io.Writer) error {
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
	cfg := config.LoadFromDir(moduleDir)
	scanOpts := docscan.ScanOptions{Config: cfg}

	docs, err := docscan.Scan(moduleDir, scanOpts)
	if err != nil {
		return nil, fmt.Errorf("docscan: %w", err)
	}
	return cliutil.CaptureJSON(func(w io.Writer) error {
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
