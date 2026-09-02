package aireport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/cliutil"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/docscan/apidoc"
	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/protocol"
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
	loadConfig          func(string, io.Writer) *config.GazeConfig
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
	CRAPload            *int
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
		CRAPload:            intPtr(rpt.Summary.CRAPload),
		GazeCRAPload:        rpt.Summary.GazeCRAPload,
		TotalFunctions:      rpt.Summary.TotalFunctions,
		SSADegradedPackages: rpt.Summary.SSADegradedPackages,
	}
	return res, nil
}

// qualityStepResult holds the outputs of runQualityStep.
type qualityStepResult struct {
	JSON                json.RawMessage
	AvgContractCoverage *int
	SSADegraded         bool
	SSADegradedPackages []string
	SkippedTests        int
	SkippedTestNames    []string
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

	gazeConfig := d.loadConfig(moduleDir, stderr)

	// Hoist LoadModule out of the per-package loop — O(1) instead of O(n).
	modPkgs := d.resolveModulePkgs(moduleDir)

	var allReports []taxonomy.QualityReport
	var degradedPkgs []string
	totalSkipped := 0
	var allSkippedNames []string
	for _, pkgPath := range pkgPaths {
		reports, degradedPkg, skipped, skippedNames := runQualityForPackage(pkgPath, gazeConfig, modPkgs, stderr, deps...)
		if degradedPkg != "" {
			degradedPkgs = append(degradedPkgs, degradedPkg)
		}
		allReports = append(allReports, reports...)
		totalSkipped += skipped
		allSkippedNames = append(allSkippedNames, skippedNames...)
	}

	// BuildPackageSummary aggregates report-level data only; skipped test
	// data and SSA degradation must be set post-hoc since skipped tests
	// don't produce QualityReport entries.
	summary := quality.BuildPackageSummary(allReports)
	summary.SkippedTests = totalSkipped
	summary.SkippedTestNames = allSkippedNames
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
		AvgContractCoverage: intPtr(avgCov),
		SSADegraded:         len(degradedPkgs) > 0,
		SSADegradedPackages: degradedPkgs,
		SkippedTests:        totalSkipped,
		SkippedTestNames:    allSkippedNames,
	}, nil
}

// runQualityForPackage runs the quality pipeline on a single package.
// modPkgs should be pre-resolved by the caller (hoist LoadModule out of loops).
// Returns (nil, "", 0, nil) if the package has no tests or analysis fails.
// The second return value is the degraded package path (empty string
// if not degraded, package path if SSA construction failed).
// The third return value is the number of skipped test functions
// (tests where no target function could be resolved).
// The fourth return value is the names of skipped test functions.
func runQualityForPackage(
	pkgPath string,
	gazeConfig *config.GazeConfig,
	modPkgs []*packages.Package,
	stderr io.Writer,
	deps ...qualityPipelineDeps,
) ([]taxonomy.QualityReport, string, int, []string) {
	d := resolveQualityDeps(deps)

	includeUnexported := loader.IsMainPkg(pkgPath)
	if includeUnexported {
		_, _ = fmt.Fprintf(stderr, "package main detected for %s, including unexported functions\n", pkgPath)
	}
	analysisOpts := analysis.Options{IncludeUnexported: includeUnexported}
	results, err := d.loadAndAnalyze(pkgPath, analysisOpts)
	if err != nil || len(results) == 0 {
		return nil, "", 0, nil
	}

	cfg := gazeConfig
	classified, err := d.classifyResults(results, pkgPath, cfg, modPkgs)
	if err != nil || len(classified) == 0 {
		return nil, "", 0, nil
	}

	testPkg, err := d.loadTestPkg(pkgPath)
	if err != nil {
		return nil, "", 0, nil
	}

	qualOpts := quality.Options{Stderr: stderr}
	reports, summary, err := d.assess(classified, testPkg, qualOpts)
	if err != nil {
		return nil, "", 0, nil
	}

	skipped := 0
	var skippedNames []string
	if summary != nil {
		skipped = summary.SkippedTests
		skippedNames = summary.SkippedTestNames
	}
	if summary != nil && summary.SSADegraded {
		return reports, pkgPath, skipped, skippedNames
	}
	return reports, "", skipped, skippedNames
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
func runClassifyStep(patterns []string, moduleDir string, stderr io.Writer, deps ...qualityPipelineDeps) (*classifyStepResult, error) {
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

	gazeConfig := d.loadConfig(moduleDir, stderr)
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

// docscanEnvelope wraps the docscan output in a structured envelope
// with optional API coverage data. This type is local to the report
// pipeline — the CLI-layer DocscanOutput in cmd/gaze/main.go cannot
// be imported from internal packages.
type docscanEnvelope struct {
	Documents   []docscan.DocumentFile    `json:"documents"`
	APICoverage *apidoc.APICoverageReport `json:"api_coverage"`
}

// runDocscanStep runs the documentation scanner and returns the JSON output.
// When sess is non-nil and initialized, it uses the external analyzer for
// language-aware documentation coverage analysis. When sess is nil, only
// the heuristic docscan is performed.
func runDocscanStep(moduleDir string, sess *adapter.Session, stderr io.Writer) (json.RawMessage, error) {
	cfg := config.LoadFromDir(moduleDir, stderr)
	scanOpts := docscan.ScanOptions{Config: cfg}

	docs, err := docscan.Scan(moduleDir, scanOpts)
	if err != nil {
		return nil, fmt.Errorf("docscan: %w", err)
	}

	var apiCoverage *apidoc.APICoverageReport
	if sess != nil {
		apiCoverage = runDocscanAnalyzer(moduleDir, sess, docs, stderr)
	}

	envelope := docscanEnvelope{
		Documents:   docs,
		APICoverage: apiCoverage,
	}
	return cliutil.CaptureJSON(func(w io.Writer) error {
		enc := json.NewEncoder(w)
		return enc.Encode(envelope)
	})
}

// runDocscanAnalyzer calls the external analyzer for doc_coverage and
// analyze data, then runs apidoc.Analyze. Returns nil on any failure
// (graceful degradation with warning to stderr).
func runDocscanAnalyzer(moduleDir string, sess *adapter.Session, docs []docscan.DocumentFile, stderr io.Writer) *apidoc.APICoverageReport {
	ctx := context.Background()

	// Get doc_coverage data (optional capability — returns nil if unsupported).
	docCovResult, err := sess.DocCoverage(ctx, protocol.DocCoverageParams{
		RootPath: moduleDir,
		Patterns: []string{"./..."},
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "warning: doc_coverage call failed, falling back to heuristic: %v\n", err)
		docCovResult = nil
	}

	// Get analyze results for function list.
	analyzeResult, err := sess.Analyze(ctx, protocol.AnalyzeParams{
		RootPath: moduleDir,
		Patterns: []string{"./..."},
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "warning: analyze call failed for docscan, skipping API coverage: %v\n", err)
		return nil
	}

	var functions []protocol.AnalyzedFunction
	if analyzeResult != nil {
		functions = analyzeResult.Functions
	}

	data := &apidoc.AnalyzerData{
		Functions:   functions,
		DocCoverage: docCovResult,
		Language:    sess.Language(),
	}

	report, err := apidoc.Analyze(docs, data)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "warning: apidoc.Analyze failed: %v\n", err)
		return nil
	}
	return report
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
