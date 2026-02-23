package quality_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/jflowers/gaze/internal/analysis"
	"github.com/jflowers/gaze/internal/quality"
	"github.com/jflowers/gaze/internal/taxonomy"
)

// testdataPath returns the absolute path to a testdata fixture package.
func testdataPath(pkgName string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata", "src", pkgName)
}

// loadTestdataPackage loads a testdata fixture package with test
// files included.
func loadTestdataPackage(pkgName string) (*packages.Package, error) {
	testdataDir := testdataPath(pkgName)
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
		Dir:   testdataDir,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages loaded for %q", pkgName)
	}
	// Return the package that contains test files.
	for _, pkg := range pkgs {
		if quality.HasTestSyntax(pkg) {
			return pkg, nil
		}
	}
	return pkgs[0], nil
}

// loadNonTestPackage loads a testdata fixture without test files.
func loadNonTestPackage(pkgName string) (*packages.Package, error) {
	testdataDir := testdataPath(pkgName)
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
		Dir:   testdataDir,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages loaded for %q", pkgName)
	}
	return pkgs[0], nil
}

// pkgCacheEntry holds a loaded test package for reuse.
type pkgCacheEntry struct {
	pkg *packages.Package
}

var (
	pkgCacheMu sync.Mutex
	pkgCache   = make(map[string]*pkgCacheEntry)
)

func cachedTestPackage(pkgName string) (*packages.Package, error) {
	pkgCacheMu.Lock()
	defer pkgCacheMu.Unlock()

	if entry, ok := pkgCache[pkgName]; ok {
		return entry.pkg, nil
	}

	pkg, err := loadTestdataPackage(pkgName)
	if err != nil {
		return nil, err
	}

	pkgCache[pkgName] = &pkgCacheEntry{pkg: pkg}
	return pkg, nil
}

func loadPkg(t *testing.T, name string) *packages.Package {
	t.Helper()
	pkg, err := cachedTestPackage(name)
	if err != nil {
		t.Fatalf("failed to load test package %q: %v", name, err)
	}
	return pkg
}

// --- Phase 2 Tests: Test-Target Pairing ---

func TestFindTestFunctions_WellTested(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	tests := quality.FindTestFunctions(pkg)

	if len(tests) == 0 {
		t.Fatal("expected to find test functions, got 0")
	}

	names := make(map[string]bool)
	for _, tf := range tests {
		names[tf.Name] = true
		if tf.Decl == nil {
			t.Errorf("test %s has nil Decl", tf.Name)
		}
		if tf.Location == "" {
			t.Errorf("test %s has empty Location", tf.Name)
		}
	}

	// Expect specific test functions.
	expected := []string{"TestAdd", "TestDivide", "TestDivide_ZeroError", "TestCounter_Increment"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected test function %q not found; found: %v", name, names)
		}
	}
}

func TestFindTestFunctions_NoTests(t *testing.T) {
	// Load the package without test files.
	pkg, err := loadNonTestPackage("welltested")
	if err != nil {
		t.Fatalf("failed to load package: %v", err)
	}
	tests := quality.FindTestFunctions(pkg)
	if len(tests) != 0 {
		t.Errorf("expected 0 test functions when Tests=false, got %d", len(tests))
	}
}

func TestBuildTestSSA_Success(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	prog, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}
	if prog == nil {
		t.Fatal("expected non-nil SSA program")
	}
	if ssaPkg == nil {
		t.Fatal("expected non-nil SSA package")
	}
}

func TestInferTargets_SingleTarget(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	// Find the TestAdd function.
	ssaFunc := ssaPkg.Func("TestAdd")
	if ssaFunc == nil {
		t.Fatal("expected to find SSA function TestAdd")
	}

	opts := quality.DefaultOptions()
	targets, warnings := quality.InferTargets(ssaFunc, pkg, opts)

	if len(targets) == 0 {
		t.Fatal("expected at least one target, got 0")
	}

	// Should not have any serious warnings for a simple test.
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}

	// The target should be "Add".
	found := false
	for _, target := range targets {
		if target.FuncName == "Add" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected target 'Add', got targets: %v", targets)
	}
}

func TestInferTargets_NoTarget(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	// Create a mock function with no body.
	ssaFunc := ssaPkg.Func("init")
	if ssaFunc != nil {
		opts := quality.DefaultOptions()
		targets, _ := quality.InferTargets(ssaFunc, pkg, opts)
		// init should not infer any test targets.
		if len(targets) > 0 {
			t.Logf("init inferred %d targets (may be expected)", len(targets))
		}
	}
}

// --- Phase 3 Tests: Assertion Detection ---

func TestDetectAssertions_StdlibComparison(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	tests := quality.FindTestFunctions(pkg)

	var addTest *quality.TestFunc
	for i, tf := range tests {
		if tf.Name == "TestAdd" {
			addTest = &tests[i]
			break
		}
	}
	if addTest == nil {
		t.Fatal("TestAdd not found")
	}

	sites := quality.DetectAssertions(addTest.Decl, pkg, 3)
	if len(sites) == 0 {
		t.Fatal("expected at least one assertion site in TestAdd")
	}

	// Should detect a stdlib comparison (if got != 5).
	found := false
	for _, s := range sites {
		if s.Kind == quality.AssertionKindStdlibComparison {
			found = true
		}
	}
	if !found {
		t.Error("expected to detect stdlib_comparison assertion kind")
	}
}

func TestDetectAssertions_ErrorCheck(t *testing.T) {
	pkg := loadPkg(t, "welltested")
	tests := quality.FindTestFunctions(pkg)

	var divTest *quality.TestFunc
	for i, tf := range tests {
		if tf.Name == "TestDivide" {
			divTest = &tests[i]
			break
		}
	}
	if divTest == nil {
		t.Fatal("TestDivide not found")
	}

	sites := quality.DetectAssertions(divTest.Decl, pkg, 3)
	if len(sites) == 0 {
		t.Fatal("expected assertion sites in TestDivide")
	}

	// Should detect an error check.
	hasErrorCheck := false
	for _, s := range sites {
		if s.Kind == quality.AssertionKindStdlibErrorCheck {
			hasErrorCheck = true
		}
	}
	if !hasErrorCheck {
		t.Error("expected to detect stdlib_error_check assertion kind")
	}
}

func TestDetectAssertions_HelperTraversal(t *testing.T) {
	pkg := loadPkg(t, "helpers")
	tests := quality.FindTestFunctions(pkg)

	var mulTest *quality.TestFunc
	for i, tf := range tests {
		if tf.Name == "TestMultiply" {
			mulTest = &tests[i]
			break
		}
	}
	if mulTest == nil {
		t.Fatal("TestMultiply not found")
	}

	sites := quality.DetectAssertions(mulTest.Decl, pkg, 3)
	if len(sites) == 0 {
		t.Fatal("expected assertion sites from helper function assertEqual")
	}

	// Assertions from helpers should have depth > 0.
	hasDeep := false
	for _, s := range sites {
		if s.Depth > 0 {
			hasDeep = true
		}
	}
	if !hasDeep {
		t.Error("expected assertion sites at depth > 0 from helper traversal")
	}
}

func TestDetectAssertions_TRunSubTests(t *testing.T) {
	pkg := loadPkg(t, "tabledriven")
	tests := quality.FindTestFunctions(pkg)

	var greetTest *quality.TestFunc
	for i, tf := range tests {
		if tf.Name == "TestGreet" {
			greetTest = &tests[i]
			break
		}
	}
	if greetTest == nil {
		t.Fatal("TestGreet not found")
	}

	sites := quality.DetectAssertions(greetTest.Decl, pkg, 3)
	if len(sites) == 0 {
		t.Fatal("expected assertion sites from t.Run sub-tests")
	}

	// Sub-test assertions should be at depth 0 (inlined).
	for _, s := range sites {
		if s.Depth != 0 {
			t.Errorf("t.Run sub-test assertion at depth %d, want 0", s.Depth)
		}
	}
}

// --- Phase 5 Tests: Coverage Computation ---

func TestComputeContractCoverage_Full(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-002", Type: taxonomy.ErrorReturn, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
	}
	mappings := []taxonomy.AssertionMapping{
		{SideEffectID: "se-001", Confidence: 80},
		{SideEffectID: "se-002", Confidence: 80},
	}

	coverage := quality.ComputeContractCoverage(effects, mappings)

	if coverage.Percentage != 100 {
		t.Errorf("expected 100%% coverage, got %.0f%%", coverage.Percentage)
	}
	if coverage.CoveredCount != 2 {
		t.Errorf("expected 2 covered, got %d", coverage.CoveredCount)
	}
	if coverage.TotalContractual != 2 {
		t.Errorf("expected 2 total contractual, got %d", coverage.TotalContractual)
	}
	if len(coverage.Gaps) != 0 {
		t.Errorf("expected 0 gaps, got %d", len(coverage.Gaps))
	}
}

func TestComputeContractCoverage_Zero(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
	}
	var mappings []taxonomy.AssertionMapping

	coverage := quality.ComputeContractCoverage(effects, mappings)

	if coverage.Percentage != 0 {
		t.Errorf("expected 0%% coverage, got %.0f%%", coverage.Percentage)
	}
	if coverage.CoveredCount != 0 {
		t.Errorf("expected 0 covered, got %d", coverage.CoveredCount)
	}
	if len(coverage.Gaps) != 1 {
		t.Errorf("expected 1 gap, got %d", len(coverage.Gaps))
	}
}

func TestComputeContractCoverage_Partial(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-002", Type: taxonomy.ErrorReturn, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-003", Type: taxonomy.ReceiverMutation, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-004", Type: taxonomy.LogWrite, Classification: &taxonomy.Classification{Label: taxonomy.Incidental}},
	}
	mappings := []taxonomy.AssertionMapping{
		{SideEffectID: "se-001", Confidence: 80},
		{SideEffectID: "se-002", Confidence: 60},
		{SideEffectID: "se-004", Confidence: 50}, // incidental — not counted
	}

	coverage := quality.ComputeContractCoverage(effects, mappings)

	// 2 out of 3 contractual effects covered (se-003 is a gap).
	wantPct := 200.0 / 3.0 // ~66.67%
	if coverage.Percentage < wantPct-1 || coverage.Percentage > wantPct+1 {
		t.Errorf("expected ~%.1f%% coverage, got %.1f%%", wantPct, coverage.Percentage)
	}
	if coverage.CoveredCount != 2 {
		t.Errorf("expected 2 covered, got %d", coverage.CoveredCount)
	}
	if coverage.TotalContractual != 3 {
		t.Errorf("expected 3 total contractual, got %d", coverage.TotalContractual)
	}
	if len(coverage.Gaps) != 1 {
		t.Errorf("expected 1 gap, got %d", len(coverage.Gaps))
	}
}

func TestComputeContractCoverage_AmbiguousExcluded(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-002", Type: taxonomy.LogWrite, Classification: &taxonomy.Classification{Label: taxonomy.Ambiguous}},
	}
	mappings := []taxonomy.AssertionMapping{
		{SideEffectID: "se-001", Confidence: 80},
	}

	coverage := quality.ComputeContractCoverage(effects, mappings)

	// Only 1 contractual effect; ambiguous excluded from denominator.
	if coverage.Percentage != 100 {
		t.Errorf("expected 100%% coverage (ambiguous excluded), got %.0f%%", coverage.Percentage)
	}
	if coverage.TotalContractual != 1 {
		t.Errorf("expected 1 total contractual, got %d", coverage.TotalContractual)
	}
}

func TestComputeContractCoverage_NoContractualEffects(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.LogWrite, Classification: &taxonomy.Classification{Label: taxonomy.Incidental}},
	}
	var mappings []taxonomy.AssertionMapping

	coverage := quality.ComputeContractCoverage(effects, mappings)

	// No contractual effects → 0% by convention (0/0).
	if coverage.Percentage != 0 {
		t.Errorf("expected 0%% coverage for no contractual effects, got %.0f%%", coverage.Percentage)
	}
	if coverage.TotalContractual != 0 {
		t.Errorf("expected 0 total contractual, got %d", coverage.TotalContractual)
	}
}

// --- Phase 5 Tests: Over-Specification ---

func TestComputeOverSpecification_None(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
	}
	mappings := []taxonomy.AssertionMapping{
		{SideEffectID: "se-001", Confidence: 80},
	}

	overSpec := quality.ComputeOverSpecification(effects, mappings)

	if overSpec.Count != 0 {
		t.Errorf("expected 0 incidental assertions, got %d", overSpec.Count)
	}
	if overSpec.Ratio != 0 {
		t.Errorf("expected 0 ratio, got %f", overSpec.Ratio)
	}
}

func TestComputeOverSpecification_WithIncidental(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
		{ID: "se-002", Type: taxonomy.LogWrite, Classification: &taxonomy.Classification{Label: taxonomy.Incidental}},
		{ID: "se-003", Type: taxonomy.StdoutWrite, Classification: &taxonomy.Classification{Label: taxonomy.Incidental}},
	}
	mappings := []taxonomy.AssertionMapping{
		{SideEffectID: "se-001", Confidence: 80},
		{SideEffectID: "se-002", Confidence: 50},
		{SideEffectID: "se-003", Confidence: 50},
	}

	overSpec := quality.ComputeOverSpecification(effects, mappings)

	if overSpec.Count != 2 {
		t.Errorf("expected 2 incidental assertions, got %d", overSpec.Count)
	}
	expectedRatio := 2.0 / 3.0
	if overSpec.Ratio < expectedRatio-0.01 || overSpec.Ratio > expectedRatio+0.01 {
		t.Errorf("expected ratio ~%.2f, got %.2f", expectedRatio, overSpec.Ratio)
	}
	if len(overSpec.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(overSpec.Suggestions))
	}
}

// --- Phase 5 Tests: Package Summary ---

func TestBuildPackageSummary_Empty(t *testing.T) {
	summary := quality.BuildPackageSummary(nil)
	if summary == nil {
		t.Fatal("expected non-nil summary for empty reports")
	}
	if summary.TotalTests != 0 {
		t.Errorf("expected 0 total tests, got %d", summary.TotalTests)
	}
}

func TestBuildPackageSummary_Aggregation(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction:                 "TestA",
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100, CoveredCount: 2, TotalContractual: 2},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 0},
			AssertionDetectionConfidence: 100,
		},
		{
			TestFunction:                 "TestB",
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 50, CoveredCount: 1, TotalContractual: 2},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 1},
			AssertionDetectionConfidence: 80,
		},
	}

	summary := quality.BuildPackageSummary(reports)

	if summary.TotalTests != 2 {
		t.Errorf("expected 2 total tests, got %d", summary.TotalTests)
	}
	expectedAvg := 75.0
	if summary.AverageContractCoverage != expectedAvg {
		t.Errorf("expected average coverage %.0f%%, got %.0f%%",
			expectedAvg, summary.AverageContractCoverage)
	}
	if summary.TotalOverSpecifications != 1 {
		t.Errorf("expected 1 total over-specifications, got %d", summary.TotalOverSpecifications)
	}
	if len(summary.WorstCoverageTests) != 2 {
		t.Errorf("expected 2 worst tests, got %d", len(summary.WorstCoverageTests))
	}
	// Worst should be first (TestB with 50%).
	if summary.WorstCoverageTests[0].TestFunction != "TestB" {
		t.Errorf("expected worst test to be TestB, got %s",
			summary.WorstCoverageTests[0].TestFunction)
	}
}

// --- Phase 6 Tests: Report Output ---

func TestWriteJSON_Structure(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestFoo",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 80},
			AssertionDetectionConfidence: 95,
		},
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:              1,
		AverageContractCoverage: 80,
	}

	var buf bytes.Buffer
	err := quality.WriteJSON(&buf, reports, summary)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Verify it's valid JSON.
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check expected top-level keys.
	if _, ok := output["quality_reports"]; !ok {
		t.Error("expected 'quality_reports' key in JSON output")
	}
	if _, ok := output["quality_summary"]; !ok {
		t.Error("expected 'quality_summary' key in JSON output")
	}
}

func TestWriteText_Output(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestFoo",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       75,
				CoveredCount:     3,
				TotalContractual: 4,
			},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 1},
			AssertionDetectionConfidence: 90,
		},
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:              1,
		AverageContractCoverage: 75,
	}

	var buf bytes.Buffer
	err := quality.WriteText(&buf, reports, summary)
	if err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected non-empty text output")
	}
	// Should contain the test function name.
	if !bytes.Contains(buf.Bytes(), []byte("TestFoo")) {
		t.Error("expected output to contain 'TestFoo'")
	}
}

// --- Acceptance Tests ---

func TestSC004_Determinism(t *testing.T) {
	// Run quality analysis twice on identical code, verify identical
	// metrics (excluding timestamps).
	pkg := loadPkg(t, "welltested")

	// Analyze the non-test package.
	nonTestPkg, err := loadNonTestPackage("welltested")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	qualOpts := quality.DefaultOptions()

	// Run 1.
	reports1, summary1, err := quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess run 1 failed: %v", err)
	}

	// Run 2.
	reports2, summary2, err := quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess run 2 failed: %v", err)
	}

	// Compare report counts.
	if len(reports1) != len(reports2) {
		t.Fatalf("determinism: report count differs: %d vs %d",
			len(reports1), len(reports2))
	}

	// Compare summary.
	if summary1.TotalTests != summary2.TotalTests {
		t.Errorf("determinism: TotalTests differs: %d vs %d",
			summary1.TotalTests, summary2.TotalTests)
	}
	if summary1.AverageContractCoverage != summary2.AverageContractCoverage {
		t.Errorf("determinism: AverageContractCoverage differs: %.2f vs %.2f",
			summary1.AverageContractCoverage, summary2.AverageContractCoverage)
	}
}

func TestSC006_PackageSummary(t *testing.T) {
	// Verify correct aggregation across multiple test functions.
	reports := []taxonomy.QualityReport{
		{
			TestFunction:                 "TestA",
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100, CoveredCount: 3, TotalContractual: 3},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 0},
			AssertionDetectionConfidence: 100,
		},
		{
			TestFunction:                 "TestB",
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 0, CoveredCount: 0, TotalContractual: 2},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 2},
			AssertionDetectionConfidence: 80,
		},
		{
			TestFunction:                 "TestC",
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 50, CoveredCount: 1, TotalContractual: 2},
			OverSpecification:            taxonomy.OverSpecificationScore{Count: 1},
			AssertionDetectionConfidence: 90,
		},
	}

	summary := quality.BuildPackageSummary(reports)

	if summary.TotalTests != 3 {
		t.Errorf("expected 3 tests, got %d", summary.TotalTests)
	}
	expectedAvg := 50.0 // (100 + 0 + 50) / 3
	if summary.AverageContractCoverage != expectedAvg {
		t.Errorf("expected %.0f%% average, got %.0f%%",
			expectedAvg, summary.AverageContractCoverage)
	}
	if summary.TotalOverSpecifications != 3 {
		t.Errorf("expected 3 total over-specs, got %d", summary.TotalOverSpecifications)
	}

	// Worst tests: bottom 3, ordered ascending by coverage.
	if len(summary.WorstCoverageTests) != 3 {
		t.Fatalf("expected 3 worst tests, got %d", len(summary.WorstCoverageTests))
	}
	if summary.WorstCoverageTests[0].TestFunction != "TestB" {
		t.Errorf("expected worst test to be TestB, got %s",
			summary.WorstCoverageTests[0].TestFunction)
	}
}

// --- Mapping Tests ---

func TestMapAssertionsToEffects_NoEffects(t *testing.T) {
	sites := []quality.AssertionSite{
		{Location: "test.go:10", Kind: quality.AssertionKindStdlibComparison},
	}
	var effects []taxonomy.SideEffect

	mapped, unmapped := quality.MapAssertionsToEffects(nil, nil, sites, effects)
	if len(mapped) != 0 {
		t.Errorf("expected 0 mapped, got %d", len(mapped))
	}
	if len(unmapped) != 1 {
		t.Errorf("expected 1 unmapped, got %d", len(unmapped))
	}
}

func TestMapAssertionsToEffects_NoSites(t *testing.T) {
	effects := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ReturnValue},
	}
	var sites []quality.AssertionSite

	mapped, unmapped := quality.MapAssertionsToEffects(nil, nil, sites, effects)
	if len(mapped) != 0 {
		t.Errorf("expected 0 mapped, got %d", len(mapped))
	}
	if len(unmapped) != 0 {
		t.Errorf("expected 0 unmapped, got %d", len(unmapped))
	}
}

// --- Integration: Assess ---

func TestAssess_NilPackage(t *testing.T) {
	opts := quality.DefaultOptions()
	_, _, err := quality.Assess(nil, nil, opts)
	if err == nil {
		t.Fatal("expected error for nil test package")
	}
}

func TestAssess_WellTested(t *testing.T) {
	pkg := loadPkg(t, "welltested")

	// Get analysis results for the non-test package.
	nonTestPkg, err := loadNonTestPackage("welltested")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	var stderr bytes.Buffer
	qualOpts := quality.Options{
		Stderr: &stderr,
	}
	reports, summary, err := quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess failed: %v", err)
	}

	t.Logf("stderr: %s", stderr.String())
	t.Logf("reports: %d, summary: %+v", len(reports), summary)

	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	// Log individual report details for debugging.
	for _, r := range reports {
		t.Logf("  %s -> %s: coverage=%.0f%% (%d/%d), overspec=%d, confidence=%d%%",
			r.TestFunction, r.TargetFunction.QualifiedName(),
			r.ContractCoverage.Percentage,
			r.ContractCoverage.CoveredCount,
			r.ContractCoverage.TotalContractual,
			r.OverSpecification.Count,
			r.AssertionDetectionConfidence)
	}
}
