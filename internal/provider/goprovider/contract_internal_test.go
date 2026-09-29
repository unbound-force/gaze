package goprovider

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/quality"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// extractShortPkgName tests
// ---------------------------------------------------------------------------

func TestExtractShortPkgName_WithSlash(t *testing.T) {
	got := extractShortPkgName("github.com/unbound-force/gaze/v2/internal/crap")
	if got != "crap" {
		t.Errorf("extractShortPkgName(...crap) = %q, want %q", got, "crap")
	}
}

func TestExtractShortPkgName_NoSlash(t *testing.T) {
	got := extractShortPkgName("main")
	if got != "main" {
		t.Errorf("extractShortPkgName(main) = %q, want %q", got, "main")
	}
}

func TestExtractShortPkgName_TrailingSlash(t *testing.T) {
	got := extractShortPkgName("github.com/user/repo/")
	if got != "" {
		t.Errorf("extractShortPkgName(.../repo/) = %q, want %q (empty)", got, "")
	}
}

func TestExtractShortPkgName_Empty(t *testing.T) {
	got := extractShortPkgName("")
	if got != "" {
		t.Errorf("extractShortPkgName('') = %q, want %q", got, "")
	}
}

// ---------------------------------------------------------------------------
// analyzePackageCoverage tests
// ---------------------------------------------------------------------------

func TestAnalyzePackageCoverage_ValidPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: loads real packages via analysis pipeline")
	}
	gazeConfig := config.DefaultConfig()
	var stderr bytes.Buffer
	reports, _ := analyzePackageCoverage(
		"github.com/unbound-force/gaze/v2/internal/quality/testdata/src/welltested",
		".",
		gazeConfig,
		&stderr,
	)
	if len(reports) == 0 {
		t.Error("expected non-nil quality reports for well-tested package")
	}
}

func TestAnalyzePackageCoverage_InvalidPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: invokes go/packages.Load via analysis.LoadAndAnalyze")
	}
	gazeConfig := config.DefaultConfig()
	var stderr bytes.Buffer
	reports, _ := analyzePackageCoverage(
		"github.com/nonexistent/does/not/exist",
		".",
		gazeConfig,
		&stderr,
	)
	if reports != nil {
		t.Error("expected nil reports for non-existent package")
	}
}

// ---------------------------------------------------------------------------
// BuildContractCoverageFunc tests
// ---------------------------------------------------------------------------

func TestBuildContractCoverageFunc_InvalidPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: invokes go/packages.Load via resolvePackagePaths")
	}
	var buf bytes.Buffer
	fn, _ := BuildContractCoverageFunc(
		[]string{"github.com/nonexistent/package/does/not/exist"},
		t.TempDir(),
		&buf,
	)
	if fn != nil {
		_, ok := fn("nonexistent", "Foo")
		if ok {
			t.Error("expected ok=false for unknown pkg:func key")
		}
	}
}

func TestBuildContractCoverageFunc_WelltestedPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs quality pipeline (package loading)")
	}

	pattern := "github.com/unbound-force/gaze/v2/internal/quality/testdata/src/welltested"

	var buf bytes.Buffer
	fn, _ := BuildContractCoverageFunc([]string{pattern}, ".", &buf)

	if fn == nil {
		t.Fatal("BuildContractCoverageFunc returned nil; expected non-nil closure for well-tested package")
	}

	info, ok := fn("welltested", "Add")
	t.Logf("welltested:Add contract coverage: %.1f%% (found=%v, reason=%q)", info.Percentage, ok, info.Reason)
	if !ok {
		t.Fatal("expected ok=true for welltested:Add, got ok=false")
	}
	if info.Percentage <= 0 {
		t.Errorf("expected pct > 0 for welltested:Add (well-tested fixture should have non-zero coverage), got %.1f", info.Percentage)
	}
}

// ---------------------------------------------------------------------------
// analyzePackageCoverage DI tests (Task 1.2)
// ---------------------------------------------------------------------------

// syntheticResult returns a minimal AnalysisResult for DI tests.
func syntheticResult() taxonomy.AnalysisResult {
	return taxonomy.AnalysisResult{
		Target: taxonomy.FunctionTarget{
			Package:  "example.com/pkg",
			Function: "DoWork",
		},
		SideEffects: []taxonomy.SideEffect{
			{
				ID:   "se-abc12345",
				Type: taxonomy.ReturnValue,
				Tier: taxonomy.TierP0,
			},
		},
	}
}

// successDeps returns a contractCoverageDeps that simulates a
// successful 4-step pipeline with synthetic data.
func successDeps() contractCoverageDeps {
	return contractCoverageDeps{
		loadAndAnalyze: func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
			return []taxonomy.AnalysisResult{syntheticResult()}, nil
		},
		classifyResults: func(results []taxonomy.AnalysisResult, _ string, _ string, _ *config.GazeConfig) []taxonomy.AnalysisResult {
			return results
		},
		loadTestPkg: func(_ string) (*packages.Package, error) {
			return &packages.Package{Name: "test"}, nil
		},
		assess: func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
			return []taxonomy.QualityReport{
				{
					TestFunction: "TestDoWork",
					TargetFunction: taxonomy.FunctionTarget{
						Package:  "example.com/pkg",
						Function: "DoWork",
					},
					ContractCoverage: taxonomy.ContractCoverage{
						Percentage: 75.0,
					},
				},
			}, &taxonomy.PackageSummary{TotalTests: 1}, nil
		},
	}
}

func TestAnalyzePackageCoverage_DI_Success(t *testing.T) {
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr,
		successDeps(),
	)
	if len(reports) == 0 {
		t.Fatal("expected non-empty reports for successful pipeline")
	}
	if reports[0].ContractCoverage.Percentage != 75.0 {
		t.Errorf("expected 75.0%% coverage, got %.1f%%", reports[0].ContractCoverage.Percentage)
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_AnalysisError(t *testing.T) {
	deps := successDeps()
	deps.loadAndAnalyze = func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return nil, errors.New("analysis failed")
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if reports != nil {
		t.Errorf("expected nil reports on analysis error, got %d reports", len(reports))
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_EmptyResults(t *testing.T) {
	deps := successDeps()
	deps.loadAndAnalyze = func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return []taxonomy.AnalysisResult{}, nil
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if reports != nil {
		t.Errorf("expected nil reports on empty results, got %d reports", len(reports))
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_ClassifyNil(t *testing.T) {
	deps := successDeps()
	deps.classifyResults = func(_ []taxonomy.AnalysisResult, _ string, _ string, _ *config.GazeConfig) []taxonomy.AnalysisResult {
		return nil
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if reports != nil {
		t.Errorf("expected nil reports when classify returns nil, got %d reports", len(reports))
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_LoadTestError(t *testing.T) {
	deps := successDeps()
	deps.loadTestPkg = func(_ string) (*packages.Package, error) {
		return nil, errors.New("no test files")
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if reports != nil {
		t.Errorf("expected nil reports on loadTestPkg error, got %d reports", len(reports))
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_AssessError(t *testing.T) {
	deps := successDeps()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return nil, nil, errors.New("assess failed")
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if reports != nil {
		t.Errorf("expected nil reports on assess error, got %d reports", len(reports))
	}
	if degradedPkg != "" {
		t.Errorf("expected empty degradedPkg, got %q", degradedPkg)
	}
}

func TestAnalyzePackageCoverage_DI_SSADegraded(t *testing.T) {
	deps := successDeps()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		reports := []taxonomy.QualityReport{
			{
				TestFunction: "TestDoWork",
				TargetFunction: taxonomy.FunctionTarget{
					Package:  "example.com/pkg",
					Function: "DoWork",
				},
			},
		}
		summary := &taxonomy.PackageSummary{
			TotalTests:  1,
			SSADegraded: true,
		}
		return reports, summary, nil
	}
	var stderr bytes.Buffer
	reports, degradedPkg := analyzePackageCoverage(
		"example.com/pkg", ".", config.DefaultConfig(), &stderr, deps,
	)
	if len(reports) == 0 {
		t.Fatal("expected non-empty reports when SSA is degraded")
	}
	if degradedPkg != "example.com/pkg" {
		t.Errorf("expected degradedPkg = %q, got %q", "example.com/pkg", degradedPkg)
	}
	// Verify the warning was emitted to stderr.
	if !strings.Contains(stderr.String(), "SSA degraded") {
		t.Errorf("expected SSA degradation warning in stderr, got %q", stderr.String())
	}
}

// ---------------------------------------------------------------------------
// computeCoverageReason tests (Task 2.1)
// ---------------------------------------------------------------------------

func TestComputeCoverageReason_WithContractualEffects(t *testing.T) {
	report := taxonomy.QualityReport{
		ContractCoverage: taxonomy.ContractCoverage{
			Percentage:       75.0,
			TotalContractual: 3,
		},
	}
	info := computeCoverageReason(report)
	if info.Percentage != 75.0 {
		t.Errorf("Percentage = %.1f, want 75.0", info.Percentage)
	}
	if info.Reason != "" {
		t.Errorf("Reason = %q, want empty string", info.Reason)
	}
}

func TestComputeCoverageReason_AllAmbiguous(t *testing.T) {
	report := taxonomy.QualityReport{
		ContractCoverage: taxonomy.ContractCoverage{
			TotalContractual: 0,
		},
		AmbiguousEffects: []taxonomy.SideEffect{
			{Classification: &taxonomy.Classification{Confidence: 78}},
			{Classification: &taxonomy.Classification{Confidence: 79}},
		},
	}
	info := computeCoverageReason(report)
	if info.Reason != "all_effects_ambiguous" {
		t.Errorf("Reason = %q, want %q", info.Reason, "all_effects_ambiguous")
	}
	if info.MinConfidence != 78 {
		t.Errorf("MinConfidence = %d, want 78", info.MinConfidence)
	}
	if info.MaxConfidence != 79 {
		t.Errorf("MaxConfidence = %d, want 79", info.MaxConfidence)
	}
}

func TestComputeCoverageReason_NoEffects(t *testing.T) {
	report := taxonomy.QualityReport{
		ContractCoverage: taxonomy.ContractCoverage{
			TotalContractual: 0,
		},
		AmbiguousEffects: []taxonomy.SideEffect{},
	}
	info := computeCoverageReason(report)
	if info.Reason != "no_effects_detected" {
		t.Errorf("Reason = %q, want %q", info.Reason, "no_effects_detected")
	}
}

func TestComputeCoverageReason_NilClassification(t *testing.T) {
	report := taxonomy.QualityReport{
		ContractCoverage: taxonomy.ContractCoverage{
			TotalContractual: 0,
		},
		AmbiguousEffects: []taxonomy.SideEffect{
			{Classification: nil},
			{Classification: nil},
		},
	}
	info := computeCoverageReason(report)
	// Nil classifications are skipped; with no valid entries,
	// effectCount == 0, so reason should be "no_effects_detected".
	if info.Reason != "no_effects_detected" {
		t.Errorf("Reason = %q, want %q", info.Reason, "no_effects_detected")
	}
	if info.MinConfidence != 0 {
		t.Errorf("MinConfidence = %d, want 0", info.MinConfidence)
	}
	if info.MaxConfidence != 0 {
		t.Errorf("MaxConfidence = %d, want 0", info.MaxConfidence)
	}
}

// ---------------------------------------------------------------------------
// buildEffectsSet tests (Task 2.2)
// ---------------------------------------------------------------------------

func TestBuildEffectsSet_WithEffects(t *testing.T) {
	fakeFn := func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return []taxonomy.AnalysisResult{syntheticResult()}, nil
	}
	got := buildEffectsSet([]string{"example.com/pkg"}, fakeFn)
	if !got["pkg:DoWork"] {
		t.Errorf("expected key %q in effects set, got %v", "pkg:DoWork", got)
	}
}

func TestBuildEffectsSet_NoEffects(t *testing.T) {
	fakeFn := func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{
					Package:  "example.com/pkg",
					Function: "NoOp",
				},
				SideEffects: []taxonomy.SideEffect{}, // no effects
			},
		}, nil
	}
	got := buildEffectsSet([]string{"example.com/pkg"}, fakeFn)
	if len(got) != 0 {
		t.Errorf("expected empty effects set, got %v", got)
	}
}

func TestBuildEffectsSet_AnalysisError(t *testing.T) {
	fakeFn := func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return nil, errors.New("analysis failed")
	}
	got := buildEffectsSet([]string{"example.com/broken"}, fakeFn)
	if len(got) != 0 {
		t.Errorf("expected empty effects set on error, got %v", got)
	}
}

func TestBuildEffectsSet_EmptyPaths(t *testing.T) {
	fakeFn := func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		t.Fatal("loadAndAnalyze should not be called with empty paths")
		return nil, nil
	}
	got := buildEffectsSet([]string{}, fakeFn)
	if len(got) != 0 {
		t.Errorf("expected empty effects set for empty paths, got %v", got)
	}
}

// ---------------------------------------------------------------------------
// buildCoverageMap tests (Task 2.3)
// ---------------------------------------------------------------------------

func TestBuildCoverageMap_Success(t *testing.T) {
	var stderr bytes.Buffer
	deps := successDeps()
	coverageMap, degradedPkgs := buildCoverageMap(
		[]string{"example.com/pkg"}, ".", config.DefaultConfig(), &stderr, deps,
	)
	if len(coverageMap) != 1 {
		t.Fatalf("expected 1 entry in coverage map, got %d", len(coverageMap))
	}
	info, ok := coverageMap["pkg:DoWork"]
	if !ok {
		t.Fatal("expected key \"pkg:DoWork\" in coverage map")
	}
	if info.Percentage != 75.0 {
		t.Errorf("Percentage = %.1f, want 75.0", info.Percentage)
	}
	if len(degradedPkgs) != 0 {
		t.Errorf("expected no degraded packages, got %v", degradedPkgs)
	}
}

func TestBuildCoverageMap_DegradedReport(t *testing.T) {
	deps := successDeps()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return []taxonomy.QualityReport{
			{
				TestFunction: "TestDoWork",
				TargetFunction: taxonomy.FunctionTarget{
					Package:  "example.com/pkg",
					Function: "", // degraded — empty function name
				},
				ContractCoverage: taxonomy.ContractCoverage{
					Percentage: 50.0,
				},
			},
		}, &taxonomy.PackageSummary{TotalTests: 1}, nil
	}
	var stderr bytes.Buffer
	coverageMap, _ := buildCoverageMap(
		[]string{"example.com/pkg"}, ".", config.DefaultConfig(), &stderr, deps,
	)
	if len(coverageMap) != 0 {
		t.Errorf("expected empty coverage map for degraded report, got %d entries", len(coverageMap))
	}
}

func TestBuildCoverageMap_SSADegradation(t *testing.T) {
	deps := successDeps()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		reports := []taxonomy.QualityReport{
			{
				TestFunction: "TestDoWork",
				TargetFunction: taxonomy.FunctionTarget{
					Package:  "example.com/pkg",
					Function: "DoWork",
				},
			},
		}
		summary := &taxonomy.PackageSummary{
			TotalTests:  1,
			SSADegraded: true,
		}
		return reports, summary, nil
	}
	var stderr bytes.Buffer
	_, degradedPkgs := buildCoverageMap(
		[]string{"example.com/pkg"}, ".", config.DefaultConfig(), &stderr, deps,
	)
	if len(degradedPkgs) != 1 || degradedPkgs[0] != "example.com/pkg" {
		t.Errorf("expected degraded packages [\"example.com/pkg\"], got %v", degradedPkgs)
	}
}

func TestBuildCoverageMap_HigherCoverageWins(t *testing.T) {
	// Helper that builds deps returning two reports for the same function
	// in the given percentage order.
	makeDeps := func(first, second float64) contractCoverageDeps {
		deps := successDeps()
		deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
			return []taxonomy.QualityReport{
				{
					TestFunction: "TestDoWork_A",
					TargetFunction: taxonomy.FunctionTarget{
						Package:  "example.com/pkg",
						Function: "DoWork",
					},
					ContractCoverage: taxonomy.ContractCoverage{
						Percentage:       first,
						TotalContractual: 2,
					},
				},
				{
					TestFunction: "TestDoWork_B",
					TargetFunction: taxonomy.FunctionTarget{
						Package:  "example.com/pkg",
						Function: "DoWork",
					},
					ContractCoverage: taxonomy.ContractCoverage{
						Percentage:       second,
						TotalContractual: 2,
					},
				},
			}, &taxonomy.PackageSummary{TotalTests: 2}, nil
		}
		return deps
	}

	// Order 1: 50 then 80 → 80 wins.
	var stderr bytes.Buffer
	coverageMap, _ := buildCoverageMap(
		[]string{"example.com/pkg"}, ".", config.DefaultConfig(), &stderr,
		makeDeps(50.0, 80.0),
	)
	info, ok := coverageMap["pkg:DoWork"]
	if !ok {
		t.Fatal("expected key \"pkg:DoWork\" in coverage map")
	}
	if info.Percentage != 80.0 {
		t.Errorf("order 50→80: Percentage = %.1f, want 80.0", info.Percentage)
	}

	// Order 2: 80 then 50 → 80 still wins.
	stderr.Reset()
	coverageMap, _ = buildCoverageMap(
		[]string{"example.com/pkg"}, ".", config.DefaultConfig(), &stderr,
		makeDeps(80.0, 50.0),
	)
	info, ok = coverageMap["pkg:DoWork"]
	if !ok {
		t.Fatal("expected key \"pkg:DoWork\" in coverage map")
	}
	if info.Percentage != 80.0 {
		t.Errorf("order 80→50: Percentage = %.1f, want 80.0", info.Percentage)
	}
}

func TestBuildCoverageMap_EmptyPaths(t *testing.T) {
	var stderr bytes.Buffer
	coverageMap, degradedPkgs := buildCoverageMap(
		[]string{}, ".", config.DefaultConfig(), &stderr,
	)
	if len(coverageMap) != 0 {
		t.Errorf("expected empty coverage map for empty paths, got %d entries", len(coverageMap))
	}
	if len(degradedPkgs) != 0 {
		t.Errorf("expected no degraded packages, got %v", degradedPkgs)
	}
}

// ---------------------------------------------------------------------------
// LoadTestPackage tests (Task 1.3)
// ---------------------------------------------------------------------------

func TestLoadTestPackage_WithTests(t *testing.T) {
	pkg, err := LoadTestPackage("github.com/unbound-force/gaze/v2/internal/quality/testdata/src/welltested")
	if err != nil {
		t.Fatalf("expected no error for package with tests, got: %v", err)
	}
	if pkg == nil {
		t.Fatal("expected non-nil package")
	}
}

func TestLoadTestPackage_WithoutTests(t *testing.T) {
	_, err := LoadTestPackage("github.com/unbound-force/gaze/v2/internal/analysis/testdata/src/returns")
	if err == nil {
		t.Fatal("expected error for package without test files")
	}
	if !strings.Contains(err.Error(), "no test files found") {
		t.Errorf("expected error containing %q, got %q", "no test files found", err.Error())
	}
}

func TestLoadTestPackage_NonExistent(t *testing.T) {
	_, err := LoadTestPackage("github.com/nonexistent/does-not-exist")
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
}

// ---------------------------------------------------------------------------
// buildContractCoverageFuncImpl DI tests (Task 2.2 / 2.3)
// ---------------------------------------------------------------------------

// successBCCFDeps returns a buildContractCoverageFuncDeps with synthetic
// implementations that simulate a successful pipeline. The effects set
// contains "pkg:DoWork" and the coverage map contains "pkg:DoWork" at 75%.
func successBCCFDeps() buildContractCoverageFuncDeps {
	return buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return []string{"example.com/pkg"}, nil
		},
		loadConfig: func(_ string, _ io.Writer) *config.GazeConfig {
			return config.DefaultConfig()
		},
		buildEffectsSetFn: func(_ []string, _ func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)) map[string]bool {
			return map[string]bool{"pkg:DoWork": true}
		},
		buildCoverageMapFn: func(_ []string, _ string, _ *config.GazeConfig, _ io.Writer, _ ...contractCoverageDeps) (map[string]crap.ContractCoverageInfo, []string) {
			return map[string]crap.ContractCoverageInfo{
				"pkg:DoWork": {Percentage: 75.0},
			}, nil
		},
	}
}

func TestBuildContractCoverageFuncImpl_ResolveError(t *testing.T) {
	deps := buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return nil, errors.New("resolve failed")
		},
	}
	var stderr bytes.Buffer
	fn, degraded := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn != nil {
		t.Fatal("expected nil function when resolve fails")
	}
	if degraded != nil {
		t.Errorf("expected nil degradedPkgs, got %v", degraded)
	}
	if !strings.Contains(stderr.String(), "failed to resolve packages") {
		t.Errorf("expected stderr to contain resolve error, got %q", stderr.String())
	}
}

func TestBuildContractCoverageFuncImpl_EmptyPkgPaths(t *testing.T) {
	deps := buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return []string{}, nil
		},
	}
	var stderr bytes.Buffer
	fn, degraded := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn != nil {
		t.Fatal("expected nil function for empty package paths")
	}
	if degraded != nil {
		t.Errorf("expected nil degradedPkgs, got %v", degraded)
	}
}

func TestBuildContractCoverageFuncImpl_BothMapsEmpty(t *testing.T) {
	deps := buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return []string{"example.com/pkg"}, nil
		},
		loadConfig: func(_ string, _ io.Writer) *config.GazeConfig {
			return config.DefaultConfig()
		},
		buildEffectsSetFn: func(_ []string, _ func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)) map[string]bool {
			return map[string]bool{}
		},
		buildCoverageMapFn: func(_ []string, _ string, _ *config.GazeConfig, _ io.Writer, _ ...contractCoverageDeps) (map[string]crap.ContractCoverageInfo, []string) {
			return map[string]crap.ContractCoverageInfo{}, []string{"example.com/pkg"}
		},
	}
	var stderr bytes.Buffer
	fn, degraded := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn != nil {
		t.Fatal("expected nil function when both maps are empty")
	}
	if len(degraded) != 1 || degraded[0] != "example.com/pkg" {
		t.Errorf("expected degradedPkgs=[example.com/pkg], got %v", degraded)
	}
}

func TestBuildContractCoverageFuncImpl_ClosureFound(t *testing.T) {
	deps := successBCCFDeps()
	var stderr bytes.Buffer
	fn, _ := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn == nil {
		t.Fatal("expected non-nil closure")
	}
	info, ok := fn("pkg", "DoWork")
	if !ok {
		t.Fatal("expected ok=true for known function")
	}
	if info.Percentage != 75.0 {
		t.Errorf("expected 75.0%% coverage, got %.1f%%", info.Percentage)
	}
}

func TestBuildContractCoverageFuncImpl_ClosureNoTestCoverage(t *testing.T) {
	deps := buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return []string{"example.com/pkg"}, nil
		},
		loadConfig: func(_ string, _ io.Writer) *config.GazeConfig {
			return config.DefaultConfig()
		},
		buildEffectsSetFn: func(_ []string, _ func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)) map[string]bool {
			return map[string]bool{"pkg:Untested": true}
		},
		buildCoverageMapFn: func(_ []string, _ string, _ *config.GazeConfig, _ io.Writer, _ ...contractCoverageDeps) (map[string]crap.ContractCoverageInfo, []string) {
			// Coverage map has at least one entry so
			// len(coverageMap)==0 && len(effectsSet)==0 is false.
			return map[string]crap.ContractCoverageInfo{
				"pkg:Other": {Percentage: 50.0},
			}, nil
		},
	}
	var stderr bytes.Buffer
	fn, _ := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn == nil {
		t.Fatal("expected non-nil closure")
	}
	info, ok := fn("pkg", "Untested")
	if ok {
		t.Fatal("expected ok=false for function with effects but no coverage")
	}
	if info.Reason != "no_test_coverage" {
		t.Errorf("expected Reason=%q, got %q", "no_test_coverage", info.Reason)
	}
}

func TestBuildContractCoverageFuncImpl_ClosureNoEffects(t *testing.T) {
	deps := successBCCFDeps()
	var stderr bytes.Buffer
	fn, _ := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn == nil {
		t.Fatal("expected non-nil closure")
	}
	info, ok := fn("pkg", "Unknown")
	if ok {
		t.Fatal("expected ok=false for unknown function")
	}
	if info.Reason != "no_effects_detected" {
		t.Errorf("expected Reason=%q, got %q", "no_effects_detected", info.Reason)
	}
}

func TestBuildContractCoverageFuncImpl_HappyPath(t *testing.T) {
	deps := buildContractCoverageFuncDeps{
		resolvePackagePaths: func(_ []string, _ string, _ io.Writer) ([]string, error) {
			return []string{"example.com/pkg"}, nil
		},
		loadConfig: func(_ string, _ io.Writer) *config.GazeConfig {
			return config.DefaultConfig()
		},
		buildEffectsSetFn: func(_ []string, _ func(string, analysis.Options) ([]taxonomy.AnalysisResult, error)) map[string]bool {
			return map[string]bool{"pkg:DoWork": true, "pkg:Helper": true}
		},
		buildCoverageMapFn: func(_ []string, _ string, _ *config.GazeConfig, _ io.Writer, _ ...contractCoverageDeps) (map[string]crap.ContractCoverageInfo, []string) {
			return map[string]crap.ContractCoverageInfo{
				"pkg:DoWork": {Percentage: 80.0},
			}, []string{"example.com/degraded"}
		},
	}
	var stderr bytes.Buffer
	fn, degraded := buildContractCoverageFuncImpl(
		[]string{"./..."}, ".", &stderr, deps,
	)
	if fn == nil {
		t.Fatal("expected non-nil closure")
	}

	// Verify degraded packages are passed through.
	if len(degraded) != 1 || degraded[0] != "example.com/degraded" {
		t.Errorf("expected degradedPkgs=[example.com/degraded], got %v", degraded)
	}

	// Verify stderr contains completion message.
	if !strings.Contains(stderr.String(), "quality pipeline complete") {
		t.Errorf("expected completion message in stderr, got %q", stderr.String())
	}

	// Verify closure: known function returns coverage.
	info, ok := fn("pkg", "DoWork")
	if !ok {
		t.Fatal("expected ok=true for DoWork")
	}
	if info.Percentage != 80.0 {
		t.Errorf("expected 80.0%%, got %.1f%%", info.Percentage)
	}

	// Verify closure: function with effects but no coverage.
	info, ok = fn("pkg", "Helper")
	if ok {
		t.Fatal("expected ok=false for Helper (has effects, no coverage)")
	}
	if info.Reason != "no_test_coverage" {
		t.Errorf("expected Reason=%q, got %q", "no_test_coverage", info.Reason)
	}

	// Verify closure: unknown function returns no_effects_detected.
	info, ok = fn("pkg", "Missing")
	if ok {
		t.Fatal("expected ok=false for Missing")
	}
	if info.Reason != "no_effects_detected" {
		t.Errorf("expected Reason=%q, got %q", "no_effects_detected", info.Reason)
	}
}
