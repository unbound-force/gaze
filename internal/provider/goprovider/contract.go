// contract.go contains the contract coverage provider implementation,
// including the quality pipeline orchestration functions that build
// contract coverage data for GazeCRAP scoring.

package goprovider

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/loader"
	"github.com/unbound-force/gaze/v2/internal/quality"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// GoContractCoverageProvider implements crap.ContractCoverageProvider
// by orchestrating the quality pipeline (side effect analysis,
// classification, test loading, assertion mapping, contract coverage
// computation) and returning a simple lookup function.
type GoContractCoverageProvider struct {
	// Stderr receives diagnostic output from the quality pipeline.
	// If nil, diagnostics are suppressed.
	Stderr io.Writer

	// AIMapperFunc is an optional AI-assisted assertion mapping
	// callback. When non-nil, it is passed through to the quality
	// pipeline for unmapped assertion fallback.
	AIMapperFunc quality.AIMapperFunc
}

// Ensure GoContractCoverageProvider satisfies
// crap.ContractCoverageProvider.
var _ crap.ContractCoverageProvider = (*GoContractCoverageProvider)(nil)

// NewContractCoverageProvider creates a GoContractCoverageProvider
// with the given stderr writer for diagnostic output and an optional
// AI mapper function for assertion mapping fallback.
func NewContractCoverageProvider(
	stderr io.Writer,
	aiMapperFn ...quality.AIMapperFunc,
) *GoContractCoverageProvider {
	p := &GoContractCoverageProvider{
		Stderr: stderr,
	}
	if len(aiMapperFn) > 0 && aiMapperFn[0] != nil {
		p.AIMapperFunc = aiMapperFn[0]
	}
	return p
}

// Build runs the quality pipeline for the given package patterns and
// returns a contract coverage lookup function, a list of degraded
// package paths, and an error. The lookup function returns
// ContractCoverageInfo for a given (pkg, function) pair, or
// (zero, false) if no quality data exists.
//
// The returned error is always nil because the pipeline handles
// errors internally via best-effort semantics.
func (p *GoContractCoverageProvider) Build(
	patterns []string,
	rootDir string,
) (func(pkg, function string) (crap.ContractCoverageInfo, bool), []string, error) {
	stderr := p.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	ccFunc, degradedPkgs := BuildContractCoverageFunc(
		patterns, rootDir, stderr, p.AIMapperFunc,
	)

	return ccFunc, degradedPkgs, nil
}

// buildContractCoverageFuncDeps holds injectable dependencies for
// buildContractCoverageFuncImpl, enabling unit testing with synthetic
// implementations instead of calling real package resolution, config
// loading, and side effect analysis.
type buildContractCoverageFuncDeps struct {
	resolvePackagePaths func([]string, string, io.Writer) ([]string, error)
	loadConfig          func(string, io.Writer) *config.GazeConfig
	buildEffectsSetFn   func([]string, func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)) map[string]bool
	buildCoverageMapFn  func([]string, string, *config.GazeConfig, io.Writer, ...contractCoverageDeps) (map[string]crap.ContractCoverageInfo, []string)
}

// BuildContractCoverageFunc runs the quality pipeline across the
// given package patterns and returns a contract coverage callback
// for GazeCRAP scoring. This is best-effort: if the quality pipeline
// fails for any package (no tests, config errors, etc.), those
// packages are silently skipped. Returns nil if no coverage data
// could be collected.
//
// The returned degradedPkgs list contains package paths where SSA
// construction failed during quality analysis.
//
// The optional aiMapperFn parameter enables AI-assisted assertion
// mapping when non-nil. It is propagated to each per-package
// quality.Assess call via analyzePackageCoverage.
func BuildContractCoverageFunc(
	patterns []string,
	moduleDir string,
	stderr io.Writer,
	aiMapperFn ...quality.AIMapperFunc,
) (func(pkg, function string) (crap.ContractCoverageInfo, bool), []string) {
	return buildContractCoverageFuncImpl(patterns, moduleDir, stderr, buildContractCoverageFuncDeps{}, aiMapperFn...)
}

// buildContractCoverageFuncImpl is the injectable implementation of
// BuildContractCoverageFunc. When deps fields are nil, production
// implementations are used.
func buildContractCoverageFuncImpl(
	patterns []string,
	moduleDir string,
	stderr io.Writer,
	deps buildContractCoverageFuncDeps,
	aiMapperFn ...quality.AIMapperFunc,
) (func(pkg, function string) (crap.ContractCoverageInfo, bool), []string) {
	resolvePaths := deps.resolvePackagePaths
	if resolvePaths == nil {
		resolvePaths = loader.ResolvePackagePaths
	}
	loadCfg := deps.loadConfig
	if loadCfg == nil {
		loadCfg = config.LoadFromDir
	}
	buildEffects := deps.buildEffectsSetFn
	if buildEffects == nil {
		buildEffects = buildEffectsSet
	}
	buildCovMap := deps.buildCoverageMapFn
	if buildCovMap == nil {
		buildCovMap = buildCoverageMap
	}

	pkgPaths, err := resolvePaths(patterns, moduleDir, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "quality pipeline: failed to resolve packages: %v\n", err)
		return nil, nil
	}

	if len(pkgPaths) == 0 {
		return nil, nil
	}

	// Load config once for all packages.
	gazeConfig := loadCfg(moduleDir, stderr)

	// Build effects set: functions with >0 detected side effects,
	// regardless of whether they have test coverage. Used to
	// distinguish "no_test_coverage" from "no_effects_detected"
	// when a function is absent from the coverage map.
	effectsSet := buildEffects(pkgPaths, nil)

	// Build coverage map via quality pipeline.
	ccDeps := contractCoverageDeps{
		aiMapperFn: firstAIMapper(aiMapperFn),
	}
	coverageMap, degradedPkgs := buildCovMap(pkgPaths, moduleDir, gazeConfig, stderr, ccDeps)

	if len(coverageMap) == 0 && len(effectsSet) == 0 {
		return nil, degradedPkgs
	}

	_, _ = fmt.Fprintf(stderr, "quality pipeline complete: %d functions with coverage\n", len(coverageMap))

	return func(pkg, function string) (crap.ContractCoverageInfo, bool) {
		key := pkg + ":" + function
		info, ok := coverageMap[key]
		if ok {
			return info, true
		}
		// Function not in coverage map — distinguish between
		// "has effects but no test" and "no effects detected".
		// Return ok=false so the CRAP pipeline excludes these from
		// GazeCRAP calculations (no test = no coverage data, not
		// 0% coverage). The Reason is informational for display.
		if effectsSet[key] {
			return crap.ContractCoverageInfo{Reason: "no_test_coverage"}, false
		}
		return crap.ContractCoverageInfo{Reason: "no_effects_detected"}, false
	}, degradedPkgs
}

// buildEffectsSet runs side effect analysis on each package path and
// returns a set of "shortPkg:qualifiedName" keys for functions that
// have at least one detected side effect. Analysis errors for
// individual packages are silently skipped (best-effort).
func buildEffectsSet(
	pkgPaths []string,
	loadAndAnalyzeFn func(string, analysis.Options) ([]taxonomy.AnalysisResult, error),
) map[string]bool {
	if loadAndAnalyzeFn == nil {
		loadAndAnalyzeFn = analysis.LoadAndAnalyze
	}
	effectsSet := make(map[string]bool)
	for _, pkgPath := range pkgPaths {
		analysisOpts := analysis.Options{
			IncludeUnexported: loader.IsMainPkg(pkgPath),
		}
		results, err := loadAndAnalyzeFn(pkgPath, analysisOpts)
		if err != nil {
			continue
		}
		for _, result := range results {
			if len(result.SideEffects) > 0 {
				shortPkg := extractShortPkgName(result.Target.Package)
				key := shortPkg + ":" + result.Target.QualifiedName()
				effectsSet[key] = true
			}
		}
	}
	return effectsSet
}

// computeCoverageReason constructs a ContractCoverageInfo from a
// quality report. It sets the Percentage from contract coverage data
// and computes the Reason field when no contractual effects exist.
func computeCoverageReason(report taxonomy.QualityReport) crap.ContractCoverageInfo {
	info := crap.ContractCoverageInfo{
		Percentage: report.ContractCoverage.Percentage,
	}
	if report.ContractCoverage.TotalContractual == 0 {
		minConf, maxConf := 100, 0
		effectCount := 0
		for _, e := range report.AmbiguousEffects {
			if e.Classification != nil {
				effectCount++
				if e.Classification.Confidence < minConf {
					minConf = e.Classification.Confidence
				}
				if e.Classification.Confidence > maxConf {
					maxConf = e.Classification.Confidence
				}
			}
		}
		if effectCount > 0 {
			info.Reason = "all_effects_ambiguous"
			info.MinConfidence = minConf
			info.MaxConfidence = maxConf
		} else {
			info.Reason = "no_effects_detected"
		}
	}
	return info
}

// buildCoverageMap runs the quality pipeline on each package path and
// returns a coverage map and list of SSA-degraded package paths. The
// map keys are "shortPkg:qualifiedName" and values are
// ContractCoverageInfo computed via computeCoverageReason. When
// multiple reports exist for the same function, the entry with the
// higher Percentage wins. Reports with empty TargetFunction.Function
// (degraded) are skipped.
func buildCoverageMap(
	pkgPaths []string,
	moduleDir string,
	gazeConfig *config.GazeConfig,
	stderr io.Writer,
	deps ...contractCoverageDeps,
) (map[string]crap.ContractCoverageInfo, []string) {
	coverageMap := make(map[string]crap.ContractCoverageInfo)
	var degradedPkgs []string

	for _, pkgPath := range pkgPaths {
		var ccDeps contractCoverageDeps
		if len(deps) > 0 {
			ccDeps = deps[0]
		}
		reports, degradedPkg := analyzePackageCoverage(pkgPath, moduleDir, gazeConfig, stderr, ccDeps)
		if degradedPkg != "" {
			degradedPkgs = append(degradedPkgs, degradedPkg)
		}
		for _, report := range reports {
			if report.TargetFunction.Function == "" {
				continue
			}
			shortPkg := extractShortPkgName(report.TargetFunction.Package)
			key := shortPkg + ":" + report.TargetFunction.QualifiedName()

			info := computeCoverageReason(report)
			if existing, ok := coverageMap[key]; !ok || info.Percentage > existing.Percentage {
				coverageMap[key] = info
			}
		}
	}
	return coverageMap, degradedPkgs
}

// contractCoverageDeps holds injectable dependencies for
// analyzePackageCoverage, enabling unit testing with synthetic
// implementations instead of loading real Go packages.
type contractCoverageDeps struct {
	loadAndAnalyze  func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)
	classifyResults func([]taxonomy.AnalysisResult, string, string, *config.GazeConfig) []taxonomy.AnalysisResult
	loadTestPkg     func(string) (*packages.Package, error)
	assess          func([]taxonomy.AnalysisResult, *packages.Package, quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error)
	aiMapperFn      quality.AIMapperFunc
}

// analyzePackageCoverage runs the 4-step quality pipeline on a single
// package (analysis -> classify -> test-load -> quality assess) and
// returns the quality reports. The second return value is the degraded
// package path (empty if SSA succeeded). Returns nil if any step fails.
//
// The optional deps parameter enables dependency injection for
// testing. When omitted (or when individual fields are nil),
// production implementations are used as defaults.
func analyzePackageCoverage(
	pkgPath string,
	moduleDir string,
	gazeConfig *config.GazeConfig,
	stderr io.Writer,
	deps ...contractCoverageDeps,
) ([]taxonomy.QualityReport, string) {
	// Resolve deps with nil-means-default pattern.
	var d contractCoverageDeps
	if len(deps) > 0 {
		d = deps[0]
	}
	if d.loadAndAnalyze == nil {
		d.loadAndAnalyze = analysis.LoadAndAnalyze
	}
	if d.classifyResults == nil {
		d.classifyResults = classifyResults
	}
	if d.loadTestPkg == nil {
		d.loadTestPkg = LoadTestPackage
	}
	if d.assess == nil {
		d.assess = quality.Assess
	}

	analysisOpts := analysis.Options{
		IncludeUnexported: loader.IsMainPkg(pkgPath),
	}

	// Step 1: Analyze (Spec 001).
	results, err := d.loadAndAnalyze(pkgPath, analysisOpts)
	if err != nil {
		return nil, ""
	}
	if len(results) == 0 {
		return nil, ""
	}

	// Step 2: Classify (Spec 002).
	classified := d.classifyResults(results, pkgPath, moduleDir, gazeConfig)
	if classified == nil {
		return nil, ""
	}

	// Step 3: Load test package.
	testPkg, err := d.loadTestPkg(pkgPath)
	if err != nil {
		return nil, ""
	}

	// Step 4: Assess quality (Spec 003).
	qualOpts := quality.Options{
		Stderr: stderr,
	}
	if d.aiMapperFn != nil {
		qualOpts.AIMapperFunc = d.aiMapperFn
	}
	reports, summary, err := d.assess(classified, testPkg, qualOpts)
	if err != nil {
		return nil, ""
	}
	if summary != nil && summary.SSADegraded {
		_, _ = fmt.Fprintf(stderr, "warning: SSA degraded for %s, contract coverage unavailable\n", pkgPath)
		return reports, pkgPath
	}
	return reports, ""
}

// classifyResults runs classification on analysis results for a single
// package. This is a simplified version of the cmd/gaze runClassify
// that doesn't require the package-main logger or verbose mode.
func classifyResults(
	results []taxonomy.AnalysisResult,
	pkgPath string,
	moduleDir string,
	cfg *config.GazeConfig,
) []taxonomy.AnalysisResult {
	// Load the target package for AST access.
	targetResult, err := loader.Load(pkgPath)
	if err != nil {
		return nil
	}

	// Load the module for caller/interface analysis.
	modResult, modErr := loader.LoadModule(moduleDir)
	var modPkgs []*packages.Package
	if modErr == nil {
		modPkgs = modResult.Packages
	}

	clOpts := classify.Options{
		Config:         cfg,
		ModulePackages: modPkgs,
		TargetPkg:      targetResult.Pkg,
	}

	return classify.Classify(results, clOpts)
}

// LoadTestPackage loads a Go package with test files for quality
// assessment.
func LoadTestPackage(pkgPath string) (*packages.Package, error) {
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

	// Check for package load errors.
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			msgs := make([]string, len(pkg.Errors))
			for i, e := range pkg.Errors {
				msgs[i] = e.Error()
			}
			return nil, fmt.Errorf("package %s has errors: %s",
				pkg.PkgPath, strings.Join(msgs, "; "))
		}
	}

	// When Tests=true, packages.Load returns multiple packages:
	// the base package, the internal test package (same name, with
	// test files merged), and possibly an external test package
	// (with _test suffix). Prefer the package that contains test
	// function declarations in its syntax.
	for _, pkg := range pkgs {
		if quality.HasTestSyntax(pkg) {
			return pkg, nil
		}
	}

	// No package has test syntax — return an error rather than
	// silently skipping the package.
	return nil, fmt.Errorf("no test files found for %q", pkgPath)
}

// firstAIMapper extracts the first non-nil AIMapperFunc from a
// variadic slice, returning nil if none is provided.
func firstAIMapper(fns []quality.AIMapperFunc) quality.AIMapperFunc {
	if len(fns) > 0 && fns[0] != nil {
		return fns[0]
	}
	return nil
}

// extractShortPkgName returns the short package name from a full
// import path. For "github.com/unbound-force/gaze/v2/internal/crap", it
// returns "crap".
func extractShortPkgName(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		return importPath[idx+1:]
	}
	return importPath
}
