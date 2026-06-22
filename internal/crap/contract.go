// contract.go contains the contract coverage callback builder,
// extracted from cmd/gaze/main.go to enable import by internal/aireport.

package crap

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// BuildContractCoverageFunc runs the quality pipeline across the
// given package patterns and returns a ContractCoverageFunc callback
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
) (func(pkg, function string) (ContractCoverageInfo, bool), []string) {
	pkgPaths, err := loader.ResolvePackagePaths(patterns, moduleDir)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "quality pipeline: failed to resolve packages: %v\n", err)
		return nil, nil
	}

	if len(pkgPaths) == 0 {
		return nil, nil
	}

	// Load config once for all packages.
	gazeConfig := loadGazeConfigBestEffort(moduleDir)

	// Build coverage map: "shortPkg:qualifiedName" -> coverage info.
	coverageMap := make(map[string]ContractCoverageInfo)
	// effectsSet tracks functions that have >0 detected side effects,
	// regardless of whether they have test coverage. Used to
	// distinguish "no_test_coverage" from "no_effects_detected" when
	// a function is absent from the coverage map.
	effectsSet := make(map[string]bool)
	var degradedPkgs []string

	for _, pkgPath := range pkgPaths {
		// Build the effects set from analysis results before the
		// quality pipeline runs. This captures functions with
		// effects even when loadTestPackage fails (no tests).
		analysisOpts := analysis.Options{
			IncludeUnexported: loader.IsMainPkg(pkgPath),
		}
		analysisResults, analysisErr := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
		if analysisErr == nil {
			for _, result := range analysisResults {
				if len(result.SideEffects) > 0 {
					shortPkg := extractShortPkgName(result.Target.Package)
					key := shortPkg + ":" + result.Target.QualifiedName()
					effectsSet[key] = true
				}
			}
		}

		reports, degradedPkg := analyzePackageCoverage(pkgPath, moduleDir, gazeConfig, stderr, aiMapperFn...)
		if degradedPkg != "" {
			degradedPkgs = append(degradedPkgs, degradedPkg)
		}
		for _, report := range reports {
			// Skip degraded reports — they have zero-valued
			// TargetFunction and would create phantom entries
			// with empty-string keys in the coverage map.
			if report.TargetFunction.Function == "" {
				continue
			}
			shortPkg := extractShortPkgName(report.TargetFunction.Package)
			key := shortPkg + ":" + report.TargetFunction.QualifiedName()

			info := ContractCoverageInfo{
				Percentage: report.ContractCoverage.Percentage,
			}

			// Compute coverage reason from classification data.
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

			if existing, ok := coverageMap[key]; !ok || info.Percentage > existing.Percentage {
				coverageMap[key] = info
			}
		}
	}

	if len(coverageMap) == 0 && len(effectsSet) == 0 {
		return nil, degradedPkgs
	}

	_, _ = fmt.Fprintf(stderr, "quality pipeline complete: %d functions with coverage\n", len(coverageMap))

	return func(pkg, function string) (ContractCoverageInfo, bool) {
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
			return ContractCoverageInfo{Reason: "no_test_coverage"}, false
		}
		return ContractCoverageInfo{Reason: "no_effects_detected"}, false
	}, degradedPkgs
}

// analyzePackageCoverage runs the 4-step quality pipeline on a single
// package (analysis -> classify -> test-load -> quality assess) and
// returns the quality reports. The second return value is the degraded
// package path (empty if SSA succeeded). Returns nil if any step fails.
//
// The optional aiMapperFn parameter enables AI-assisted assertion
// mapping when non-nil. It is passed through to quality.Options.AIMapperFunc.
func analyzePackageCoverage(
	pkgPath string,
	moduleDir string,
	gazeConfig *config.GazeConfig,
	stderr io.Writer,
	aiMapperFn ...quality.AIMapperFunc,
) ([]taxonomy.QualityReport, string) {
	analysisOpts := analysis.Options{
		IncludeUnexported: loader.IsMainPkg(pkgPath),
	}

	// Step 1: Analyze (Spec 001).
	results, err := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
	if err != nil {
		return nil, ""
	}
	if len(results) == 0 {
		return nil, ""
	}

	// Step 2: Classify (Spec 002).
	classified := classifyResults(results, pkgPath, moduleDir, gazeConfig)
	if classified == nil {
		return nil, ""
	}

	// Step 3: Load test package.
	testPkg, err := loadTestPackage(pkgPath)
	if err != nil {
		return nil, ""
	}

	// Step 4: Assess quality (Spec 003).
	qualOpts := quality.Options{
		Stderr: stderr,
	}
	if len(aiMapperFn) > 0 && aiMapperFn[0] != nil {
		qualOpts.AIMapperFunc = aiMapperFn[0]
	}
	reports, summary, err := quality.Assess(classified, testPkg, qualOpts)
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

// loadTestPackage loads a Go package with test files for quality
// assessment.
func loadTestPackage(pkgPath string) (*packages.Package, error) {
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

// loadGazeConfigBestEffort loads the GazeConfig from the module root,
// falling back to the default config on any error.
//
// NOTE: keep in sync with internal/aireport/runner_steps.go:loadGazeConfigBestEffort.
// Consolidation deferred — see specs/022-report-gazecrap-pipeline/tasks.md.
func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig {
	cfgPath := filepath.Join(moduleDir, ".gaze.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}

// extractShortPkgName returns the short package name from a full
// import path. For "github.com/unbound-force/gaze/internal/crap", it
// returns "crap".
func extractShortPkgName(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		return importPath[idx+1:]
	}
	return importPath
}
