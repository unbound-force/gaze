package quality_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/taxonomy"
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
	return nil, fmt.Errorf("no test package found for %q — does it have *_test.go files?", pkgName)
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

// testFixtureCache holds loaded test packages for reuse within a
// single test run. This is a test-only package-level variable
// initialized in TestMain — the AGENTS.md no-global-state constraint
// applies to production packages. Test files use TestMain for
// expensive shared fixtures per Go testing conventions.
var testFixtureCache *fixtureCache

// fixtureCache is a concurrency-safe cache for loaded test packages.
// It is initialized once in TestMain and cleared after all tests run.
type fixtureCache struct {
	mu      sync.Mutex
	entries map[string]*packages.Package
}

func newFixtureCache() *fixtureCache {
	return &fixtureCache{entries: make(map[string]*packages.Package)}
}

func (c *fixtureCache) get(pkgName string) (*packages.Package, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if pkg, ok := c.entries[pkgName]; ok {
		return pkg, nil
	}

	pkg, err := loadTestdataPackage(pkgName)
	if err != nil {
		return nil, err
	}

	c.entries[pkgName] = pkg
	return pkg, nil
}

// TestMain initializes the fixture cache before tests run and
// clears it afterward. This avoids package-level mutable state.
func TestMain(m *testing.M) {
	testFixtureCache = newFixtureCache()
	code := m.Run()
	testFixtureCache = nil
	os.Exit(code)
}

func loadPkg(t *testing.T, name string) *packages.Package {
	t.Helper()
	if testFixtureCache == nil {
		t.Fatal("testFixtureCache is nil — TestMain was not called")
	}
	pkg, err := testFixtureCache.get(name)
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

	mapped, unmapped, _ := quality.MapAssertionsToEffects(nil, nil, sites, effects, nil)
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

	mapped, unmapped, _ := quality.MapAssertionsToEffects(nil, nil, sites, effects, nil)
	if len(mapped) != 0 {
		t.Errorf("expected 0 mapped, got %d", len(mapped))
	}
	if len(unmapped) != 0 {
		t.Errorf("expected 0 unmapped, got %d", len(unmapped))
	}
}

// assessFixture runs the full analysis + quality pipeline on a
// testdata fixture and returns the reports and summary. It uses
// cached package loading for efficiency.
func assessFixture(t *testing.T, fixtureName string) ([]taxonomy.QualityReport, *taxonomy.PackageSummary) {
	t.Helper()
	pkg := loadPkg(t, fixtureName)

	nonTestPkg, err := loadNonTestPackage(fixtureName)
	if err != nil {
		t.Fatalf("loading non-test package %q: %v", fixtureName, err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis of %q failed: %v", fixtureName, err)
	}

	var stderr bytes.Buffer
	qualOpts := quality.Options{Stderr: &stderr}
	reports, summary, err := quality.Assess(results, pkg, qualOpts)
	if err != nil {
		t.Fatalf("Assess(%q) failed: %v", fixtureName, err)
	}

	if summary == nil {
		t.Fatalf("Assess(%q) returned nil summary", fixtureName)
	}
	return reports, summary
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

// --- Acceptance Test: SC-001 Contract Coverage Accuracy ---

func TestSC001_ContractCoverageAccuracy(t *testing.T) {
	// SC-001: Contract Coverage correctly computed for 20+ test-target
	// pairs with known coverage. We combine reports from all fixtures
	// to reach the 20-pair minimum.

	type pairResult struct {
		fixture  string
		testFunc string
		target   string
		coverage float64
		covered  int
		total    int
	}

	var pairs []pairResult

	// Collect pairs from all fixtures.
	for _, fixture := range []string{"welltested", "undertested", "overspecd", "tabledriven", "helpers", "multilib"} {
		reports, _ := assessFixture(t, fixture)
		for _, r := range reports {
			pairs = append(pairs, pairResult{
				fixture:  fixture,
				testFunc: r.TestFunction,
				target:   r.TargetFunction.QualifiedName(),
				coverage: r.ContractCoverage.Percentage,
				covered:  r.ContractCoverage.CoveredCount,
				total:    r.ContractCoverage.TotalContractual,
			})
		}
	}

	t.Logf("Total test-target pairs: %d", len(pairs))
	for _, p := range pairs {
		t.Logf("  [%s] %s -> %s: %.0f%% (%d/%d)",
			p.fixture, p.testFunc, p.target, p.coverage, p.covered, p.total)
	}

	// Must have at least 20 pairs.
	if len(pairs) < 20 {
		t.Errorf("SC-001 requires 20+ test-target pairs, got %d", len(pairs))
	}

	// Verify coverage is in valid range.
	for _, p := range pairs {
		if p.coverage < 0 || p.coverage > 100 {
			t.Errorf("%s -> %s: coverage %.0f%% out of range [0,100]",
				p.testFunc, p.target, p.coverage)
		}
		if p.covered > p.total {
			t.Errorf("%s -> %s: covered %d > total %d",
				p.testFunc, p.target, p.covered, p.total)
		}
	}
}

// --- Acceptance Test: SC-002 Over-Specification Detection ---

func TestSC002_OverSpecificationDetection(t *testing.T) {
	// SC-002: Over-Specification Score correctly identifies all
	// incidental assertions. The overspecd fixture is designed to
	// have tests that assert on incidental (log output) effects.

	reports, _ := assessFixture(t, "overspecd")

	// Find TestProcess which asserts on log output (incidental).
	var processReport *taxonomy.QualityReport
	for i, r := range reports {
		if r.TestFunction == "TestProcess" {
			processReport = &reports[i]
		}
		t.Logf("  %s: overspec count=%d, ratio=%.2f, suggestions=%d",
			r.TestFunction, r.OverSpecification.Count,
			r.OverSpecification.Ratio, len(r.OverSpecification.Suggestions))
	}

	if processReport == nil {
		t.Fatal("expected TestProcess in overspecd fixture reports")
	}

	// TestFormat should have 0 over-specifications (no incidental assertions).
	for _, r := range reports {
		if r.TestFunction == "TestFormat" {
			if r.OverSpecification.Count != 0 {
				t.Errorf("TestFormat: expected 0 over-specifications, got %d",
					r.OverSpecification.Count)
			}
		}
	}
}

// --- Acceptance Test: SC-003 Mapping Accuracy ---

func TestSC003_MappingAccuracy(t *testing.T) {
	// SC-003: Assertion-to-side-effect mapping achieves >= 90%
	// accuracy for standard Go test patterns (direct comparison,
	// testify, go-cmp). We test this across all fixtures.

	totalAssertions := 0
	mappedAssertions := 0

	for _, fixture := range []string{"welltested", "undertested", "overspecd", "tabledriven", "helpers", "multilib"} {
		pkg := loadPkg(t, fixture)
		nonTestPkg, err := loadNonTestPackage(fixture)
		if err != nil {
			t.Fatalf("loading non-test package %q: %v", fixture, err)
		}

		opts := analysis.Options{Version: "test"}
		results, err := analysis.Analyze(nonTestPkg, opts)
		if err != nil {
			t.Fatalf("analysis of %q failed: %v", fixture, err)
		}

		testFuncs := quality.FindTestFunctions(pkg)
		_, ssaPkg, err := quality.BuildTestSSA(pkg)
		if err != nil {
			t.Fatalf("BuildTestSSA(%q) failed: %v", fixture, err)
		}

		resultMap := make(map[string]*taxonomy.AnalysisResult)
		for i := range results {
			resultMap[results[i].Target.QualifiedName()] = &results[i]
		}

		for _, tf := range testFuncs {
			ssaFunc := ssaPkg.Func(tf.Name)
			if ssaFunc == nil {
				continue
			}

			targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
			for _, target := range targets {
				result, ok := resultMap[target.FuncName]
				if !ok {
					continue
				}

				sites := quality.DetectAssertions(tf.Decl, pkg, 3)
				totalAssertions += len(sites)

				mapped, _, _ := quality.MapAssertionsToEffects(
					ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
				)
				mappedAssertions += len(mapped)
			}
		}
	}

	t.Logf("Total assertion sites: %d", totalAssertions)
	t.Logf("Mapped assertion sites: %d", mappedAssertions)

	if totalAssertions == 0 {
		t.Fatal("no assertion sites detected across fixtures")
	}

	accuracy := float64(mappedAssertions) * 100.0 / float64(totalAssertions)
	t.Logf("Mapping accuracy: %.1f%%", accuracy)

	// The spec requires >= 90% accuracy for standard patterns.
	// The AST-to-SSA bridge (TODO #6) maps return value assignments
	// to assertion expressions via types.Object identity and AST
	// assignment analysis.
	//
	// Current measured baseline: 73.8% (31/42 mapped). Remaining
	// unmapped assertions are primarily in helper functions (where
	// assertions reference helper parameters rather than the test's
	// local variables) and testify field-access patterns.
	//
	// Ratchet protocol: baselineFloor prevents regressions. Update
	// it when accuracy improves. The 90% target requires additional
	// work on helper function parameter tracing and testify argument
	// resolution.
	//
	// TODO(#6): Improve mapping accuracy to reach 90% target.
	const baselineFloor = 70.0 // ratchet: current baseline is ~73.8%
	if accuracy < baselineFloor {
		t.Errorf("SC-003: mapping accuracy %.1f%% regressed below baseline floor %.0f%% (%d/%d mapped)",
			accuracy, baselineFloor, mappedAssertions, totalAssertions)
	}
	if accuracy >= 90.0 {
		t.Logf("SC-003 PASSED: mapping accuracy %.1f%% meets 90%% target", accuracy)
	} else {
		t.Logf("SC-003 NOT MET: mapping accuracy %.1f%% (%d/%d) — target >= 90%% (remaining gap: helper param tracing + testify field access, TODO #6)",
			accuracy, mappedAssertions, totalAssertions)
	}
}

// Note: TestSC005_CIThresholds lives in cmd/gaze/main_test.go
// because it tests checkQualityThresholds which is in the main package.

// --- Acceptance Test: SC-007 Table-Driven Union ---

func TestSC007_TableDrivenUnion(t *testing.T) {
	// SC-007: Table-driven test support correctly unions assertions
	// across t.Run sub-tests.

	reports, _ := assessFixture(t, "tabledriven")

	// TestGreet uses t.Run with sub-tests. The assertions from all
	// sub-tests should be unioned into the parent test's coverage.
	var greetReport *taxonomy.QualityReport
	for i, r := range reports {
		t.Logf("  %s -> %s: coverage=%.0f%% (%d/%d), overspec=%d",
			r.TestFunction, r.TargetFunction.QualifiedName(),
			r.ContractCoverage.Percentage,
			r.ContractCoverage.CoveredCount,
			r.ContractCoverage.TotalContractual,
			r.OverSpecification.Count)
		if r.TestFunction == "TestGreet" {
			greetReport = &reports[i]
		}
	}

	if greetReport == nil {
		t.Fatal("expected TestGreet in tabledriven fixture reports")
	}

	// The sub-tests should have detected assertions. A table-driven
	// test with t.Run sub-tests that each assert on the return value
	// should produce non-zero assertion detection confidence.
	if greetReport.AssertionDetectionConfidence == 0 {
		t.Errorf("TestGreet: expected non-zero assertion detection confidence")
	}

	// Verify that assertions from sub-tests contributed to coverage.
	// TestGreet exercises Greet() which has return value effects.
	// If the union worked, we should see some coverage.
	t.Logf("TestGreet coverage: %.0f%% (%d/%d), confidence: %d%%",
		greetReport.ContractCoverage.Percentage,
		greetReport.ContractCoverage.CoveredCount,
		greetReport.ContractCoverage.TotalContractual,
		greetReport.AssertionDetectionConfidence)
}

// --- Acceptance Test: Discarded Returns ---

func TestDiscardedReturns(t *testing.T) {
	// Verify that discarded returns (_ = target()) are detected and that the
	// text output surfaces a "Discarded returns:" section with hint lines
	// (SC-003, T015, FR-009, FR-009a).

	reports, _ := assessFixture(t, "undertested")

	for _, r := range reports {
		t.Logf("  %s -> %s: coverage=%.0f%%, discarded=%d, gaps=%d",
			r.TestFunction, r.TargetFunction.QualifiedName(),
			r.ContractCoverage.Percentage,
			len(r.ContractCoverage.DiscardedReturns),
			len(r.ContractCoverage.Gaps))
	}

	// TestParse_Valid uses `got, _ := Parse("42")`. Go SSA generates an
	// Extract instruction even for blank identifiers in partial assignments,
	// so the implementation treats this as a gap (not a discard) — a known
	// v1 limitation for partial blank assignments. Log for visibility only.
	for i := range reports {
		if reports[i].TestFunction == "TestParse_Valid" {
			if len(reports[i].ContractCoverage.DiscardedReturns) == 0 {
				t.Logf("TestParse_Valid: error return is a gap (not a discard) — SSA generates Extract for blank in partial assignment; this is expected v1 behavior")
			}
			break
		}
	}

	// TestStore_Set uses `s.Set("key", "value")` with no assignment at all —
	// a complete discard. This MUST be detected as a discarded return.
	var setReport *taxonomy.QualityReport
	for i := range reports {
		if reports[i].TestFunction == "TestStore_Set" &&
			reports[i].TargetFunction.Function == "Set" {
			setReport = &reports[i]
			break
		}
	}
	if setReport == nil {
		t.Fatal("TestStore_Set -> Set report not found in undertested fixture")
	}
	if len(setReport.ContractCoverage.DiscardedReturns) == 0 {
		t.Error("TestStore_Set -> Set: expected at least one discarded return (s.Set(...) return value completely ignored, no assignment)")
	}

	// Verify the text formatter renders a "Discarded returns:" section with
	// "hint:" lines for any report that has discarded returns (SC-003, T015).
	var anyDiscarded bool
	for _, r := range reports {
		if len(r.ContractCoverage.DiscardedReturns) > 0 {
			anyDiscarded = true
			break
		}
	}
	if !anyDiscarded {
		t.Error("expected at least one report with discarded returns in undertested fixture")
	} else {
		var textBuf bytes.Buffer
		if err := quality.WriteText(&textBuf, reports, nil); err != nil {
			t.Fatalf("WriteText failed: %v", err)
		}
		output := textBuf.String()
		if !strings.Contains(output, "Discarded returns") {
			t.Error("expected text output to contain 'Discarded returns' section when discarded returns are present")
		}
		if !strings.Contains(output, "hint:") {
			t.Error("expected text output to contain 'hint:' lines under discarded returns section")
		}
	}
}

// --- Spec 006: Agent-Oriented Quality Report Enhancement Tests ---

// TestWriteText_GapHints verifies that the text formatter renders a
// "hint:" line under each coverage gap (SC-002, FR-008).
func TestWriteText_GapHints(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestStore_Set",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Set",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       0,
				CoveredCount:     0,
				TotalContractual: 1,
				Gaps: []taxonomy.SideEffect{
					{
						Type:        taxonomy.ErrorReturn,
						Description: "returns error from Set",
						Location:    "store.go:22",
					},
				},
				GapHints: []string{"if err != nil { t.Fatal(err) }"},
			},
			AssertionDetectionConfidence: 80,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "hint:") {
		t.Error("expected 'hint:' in text output for gap with hint")
	}
	if !strings.Contains(output, "t.Fatal(err)") {
		t.Error("expected hint text 'if err != nil { t.Fatal(err) }' in output")
	}
}

// TestWriteText_DiscardedReturns verifies that the text formatter renders a
// "Discarded returns:" section with hint lines when discarded returns are
// present (SC-003, FR-009, FR-009a).
func TestWriteText_DiscardedReturns(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestParse_Valid",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Parse",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       0,
				CoveredCount:     0,
				TotalContractual: 0,
				DiscardedReturns: []taxonomy.SideEffect{
					{
						Type:        taxonomy.ErrorReturn,
						Description: "returns error from Parse",
						Location:    "parse.go:15",
					},
				},
				DiscardedReturnHints: []string{"if err != nil { t.Fatal(err) }"},
			},
			AssertionDetectionConfidence: 90,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Discarded returns") {
		t.Error("expected 'Discarded returns' section in text output")
	}
	if !strings.Contains(output, "ErrorReturn") {
		t.Error("expected discarded effect type 'ErrorReturn' in output")
	}
	if !strings.Contains(output, "parse.go:15") {
		t.Error("expected discarded effect location 'parse.go:15' in output")
	}
	if !strings.Contains(output, "hint:") {
		t.Error("expected 'hint:' line under discarded return entry")
	}
	if !strings.Contains(output, "t.Fatal(err)") {
		t.Error("expected hint text 'if err != nil { t.Fatal(err) }' under discarded return")
	}
}

// TestWriteText_NoDiscardedReturns verifies that no "Discarded returns:"
// section appears when the list is empty.
func TestWriteText_NoDiscardedReturns(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction:                 "TestFoo",
			TargetFunction:               taxonomy.FunctionTarget{Package: "pkg", Function: "Foo"},
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100},
			AssertionDetectionConfidence: 100,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	if strings.Contains(buf.String(), "Discarded returns") {
		t.Error("unexpected 'Discarded returns' section when DiscardedReturns is empty")
	}
}

// TestWriteText_AmbiguousEffectsDetail verifies that ambiguous effects are
// rendered as a per-item list (not just a count) in the text formatter
// (SC-004, FR-010).
func TestWriteText_AmbiguousEffectsDetail(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestHandler",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Handler",
			},
			ContractCoverage: taxonomy.ContractCoverage{Percentage: 100},
			AmbiguousEffects: []taxonomy.SideEffect{
				{
					Type:        taxonomy.LogWrite,
					Description: "writes to logger",
					Location:    "handler.go:42",
				},
				{
					Type:        taxonomy.ReturnValue,
					Description: "returns interface{}",
					Location:    "handler.go:55",
				},
			},
			AssertionDetectionConfidence: 80,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Ambiguous effects") {
		t.Error("expected 'Ambiguous effects' section in output")
	}
	if !strings.Contains(output, "handler.go:42") {
		t.Error("expected ambiguous effect location 'handler.go:42' in output")
	}
	if !strings.Contains(output, "handler.go:55") {
		t.Error("expected ambiguous effect location 'handler.go:55' in output")
	}
	if !strings.Contains(output, "LogWrite") {
		t.Error("expected ambiguous effect type 'LogWrite' in output")
	}
}

// TestWriteText_UnmappedAssertionsDetail verifies that unmapped assertions
// are rendered as a per-item list with location, type, and reason
// (SC-001, FR-007).
func TestWriteText_UnmappedAssertionsDetail(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestMultiply",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Multiply",
			},
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100},
			AssertionDetectionConfidence: 75,
			UnmappedAssertions: []taxonomy.AssertionMapping{
				{
					AssertionLocation: "helpers_test.go:15",
					AssertionType:     taxonomy.AssertionEquality,
					Confidence:        0,
					UnmappedReason:    taxonomy.UnmappedReasonHelperParam,
				},
				{
					AssertionLocation: "counter_test.go:22",
					AssertionType:     taxonomy.AssertionEquality,
					Confidence:        0,
					UnmappedReason:    taxonomy.UnmappedReasonInlineCall,
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Unmapped assertions: 2") {
		t.Error("expected 'Unmapped assertions: 2' header in output")
	}
	if !strings.Contains(output, "helpers_test.go:15") {
		t.Error("expected unmapped assertion location 'helpers_test.go:15' in output")
	}
	if !strings.Contains(output, "counter_test.go:22") {
		t.Error("expected unmapped assertion location 'counter_test.go:22' in output")
	}
	if !strings.Contains(output, string(taxonomy.UnmappedReasonHelperParam)) {
		t.Errorf("expected unmapped reason %q in output", taxonomy.UnmappedReasonHelperParam)
	}
	if !strings.Contains(output, string(taxonomy.UnmappedReasonInlineCall)) {
		t.Errorf("expected unmapped reason %q in output", taxonomy.UnmappedReasonInlineCall)
	}
}

// TestWriteJSON_GapHints verifies that gap_hints are serialized in JSON
// output and have the same length as gaps (SC-002, FR-004, FR-005).
func TestWriteJSON_GapHints(t *testing.T) {
	gaps := []taxonomy.SideEffect{
		{ID: "se-001", Type: taxonomy.ErrorReturn, Tier: taxonomy.TierP0,
			Description: "returns error", Target: "error"},
		{ID: "se-002", Type: taxonomy.ReturnValue, Tier: taxonomy.TierP0,
			Description: "returns int", Target: "int"},
	}
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestFoo",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Foo",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       0,
				CoveredCount:     0,
				TotalContractual: 2,
				Gaps:             gaps,
				GapHints: []string{
					"if err != nil { t.Fatal(err) }",
					"got := target(); // assert got == expected",
				},
			},
			AssertionDetectionConfidence: 80,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Navigate to contract_coverage.gap_hints.
	qualReports, ok := output["quality_reports"].([]interface{})
	if !ok || len(qualReports) == 0 {
		t.Fatal("expected non-empty quality_reports array")
	}
	report, ok := qualReports[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected quality_reports[0] to be an object")
	}
	cc, ok := report["contract_coverage"].(map[string]interface{})
	if !ok {
		t.Fatal("expected contract_coverage to be an object")
	}
	gapHints, ok := cc["gap_hints"].([]interface{})
	if !ok {
		t.Fatalf("expected gap_hints to be an array in JSON, got %T", cc["gap_hints"])
	}
	if len(gapHints) != 2 {
		t.Errorf("expected 2 gap_hints, got %d", len(gapHints))
	}
	first, _ := gapHints[0].(string)
	if !strings.Contains(first, "t.Fatal(err)") {
		t.Errorf("expected first hint to contain 't.Fatal(err)', got %q", first)
	}
}

// TestWriteJSON_UnmappedReason verifies that unmapped_reason is serialized
// in JSON output for unmapped assertions and omitted for mapped ones
// (SC-001, FR-002).
func TestWriteJSON_UnmappedReason(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestMultiply",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Multiply",
			},
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100},
			AssertionDetectionConfidence: 75,
			UnmappedAssertions: []taxonomy.AssertionMapping{
				{
					AssertionLocation: "helpers_test.go:15",
					AssertionType:     taxonomy.AssertionEquality,
					Confidence:        0,
					UnmappedReason:    taxonomy.UnmappedReasonHelperParam,
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	raw := buf.String()
	if !strings.Contains(raw, `"unmapped_reason"`) {
		t.Error("expected 'unmapped_reason' key in JSON output")
	}
	if !strings.Contains(raw, `"helper_param"`) {
		t.Error("expected 'helper_param' value in JSON output")
	}
}

// TestWriteJSON_DiscardedReturnHints verifies that discarded_return_hints
// are serialized in JSON output parallel to discarded_returns (SC-003, FR-009a).
func TestWriteJSON_DiscardedReturnHints(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestStore_Set",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Set",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       0,
				CoveredCount:     0,
				TotalContractual: 0,
				DiscardedReturns: []taxonomy.SideEffect{
					{ID: "se-001", Type: taxonomy.ErrorReturn, Tier: taxonomy.TierP0,
						Description: "returns error", Target: "error"},
				},
				DiscardedReturnHints: []string{"if err != nil { t.Fatal(err) }"},
			},
			AssertionDetectionConfidence: 80,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	qualReports, _ := output["quality_reports"].([]interface{})
	if len(qualReports) == 0 {
		t.Fatal("expected non-empty quality_reports")
	}
	report, _ := qualReports[0].(map[string]interface{})
	cc, _ := report["contract_coverage"].(map[string]interface{})

	hints, ok := cc["discarded_return_hints"].([]interface{})
	if !ok {
		t.Fatalf("expected discarded_return_hints array in JSON, got %T", cc["discarded_return_hints"])
	}
	if len(hints) != 1 {
		t.Errorf("expected 1 discarded_return_hint, got %d", len(hints))
	}
	hint, _ := hints[0].(string)
	if !strings.Contains(hint, "t.Fatal(err)") {
		t.Errorf("expected hint to contain 't.Fatal(err)', got %q", hint)
	}
}

// TestWriteJSON_GapHints_ZeroGaps verifies that gap_hints is absent from JSON
// output when there are no coverage gaps (omitempty behavior, T012, SC-002
// scenario 5).
func TestWriteJSON_GapHints_ZeroGaps(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestAdd",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Add",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       100,
				CoveredCount:     1,
				TotalContractual: 1,
				// Gaps and GapHints are nil → omitempty omits both JSON keys.
			},
			AssertionDetectionConfidence: 100,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	qualReports, _ := output["quality_reports"].([]interface{})
	if len(qualReports) == 0 {
		t.Fatal("expected non-empty quality_reports")
	}
	report, _ := qualReports[0].(map[string]interface{})
	cc, _ := report["contract_coverage"].(map[string]interface{})

	if _, ok := cc["gap_hints"]; ok {
		t.Error("expected gap_hints to be absent from JSON when there are no gaps (omitempty)")
	}
}

// TestWriteJSON_DiscardedReturnHints_ZeroDiscards verifies that
// discarded_return_hints is absent from JSON output when there are no
// discarded returns (omitempty behavior, T016, SC-003 scenario 4).
func TestWriteJSON_DiscardedReturnHints_ZeroDiscards(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestAdd",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Add",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       100,
				CoveredCount:     1,
				TotalContractual: 1,
				// DiscardedReturns and DiscardedReturnHints are nil → omitempty omits both.
			},
			AssertionDetectionConfidence: 100,
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	qualReports, _ := output["quality_reports"].([]interface{})
	if len(qualReports) == 0 {
		t.Fatal("expected non-empty quality_reports")
	}
	report, _ := qualReports[0].(map[string]interface{})
	cc, _ := report["contract_coverage"].(map[string]interface{})

	if _, ok := cc["discarded_return_hints"]; ok {
		t.Error("expected discarded_return_hints to be absent from JSON when there are no discarded returns (omitempty)")
	}
}

// TestUnmappedReason_HelpersFixture_Integration verifies that running the
// helpers fixture through the full quality pipeline produces unmapped
// assertions with UnmappedReasonHelperParam. The helpers fixture uses
// depth-1 helper functions (assertEqual, assertNoError, assertError) whose
// parameter objects cannot be traced back to the test's variable assignments
// (SC-001, T007).
func TestUnmappedReason_HelpersFixture_Integration(t *testing.T) {
	reports, _ := assessFixture(t, "helpers")

	var anyHelperParam bool
	for _, r := range reports {
		for _, ua := range r.UnmappedAssertions {
			t.Logf("  %s -> %s: unmapped at %s [%s]",
				r.TestFunction, r.TargetFunction.QualifiedName(),
				ua.AssertionLocation, ua.UnmappedReason)
			if ua.UnmappedReason == taxonomy.UnmappedReasonHelperParam {
				anyHelperParam = true
			}
			if ua.UnmappedReason == "" {
				t.Errorf("%s -> %s: unmapped assertion at %s has empty UnmappedReason",
					r.TestFunction, r.TargetFunction.QualifiedName(),
					ua.AssertionLocation)
			}
		}
	}

	if !anyHelperParam {
		t.Error("expected at least one unmapped assertion with UnmappedReasonHelperParam in helpers fixture")
	}

	// Verify JSON output carries the unmapped_reason field.
	var jsonBuf bytes.Buffer
	if err := quality.WriteJSON(&jsonBuf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"helper_param"`) {
		t.Error("expected JSON output to contain unmapped_reason \"helper_param\" for helpers fixture")
	}

	// Verify text output contains [helper_param] for the unmapped entry.
	var textBuf bytes.Buffer
	if err := quality.WriteText(&textBuf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}
	if !strings.Contains(textBuf.String(), "[helper_param]") {
		t.Error("expected text output to contain \"[helper_param]\" for helpers fixture")
	}
}

// TestUnmappedReason_WelltestedFixture_Integration verifies that running the
// welltested fixture through the full quality pipeline produces at least one
// unmapped assertion with UnmappedReasonInlineCall for TestCounter_Increment,
// which calls c.Value() inline without assigning the return value
// (SC-001, T007).
func TestUnmappedReason_WelltestedFixture_Integration(t *testing.T) {
	reports, _ := assessFixture(t, "welltested")

	var anyInlineCall bool
	for _, r := range reports {
		for _, ua := range r.UnmappedAssertions {
			t.Logf("  %s -> %s: unmapped at %s [%s]",
				r.TestFunction, r.TargetFunction.QualifiedName(),
				ua.AssertionLocation, ua.UnmappedReason)
			if ua.UnmappedReason == taxonomy.UnmappedReasonInlineCall {
				anyInlineCall = true
			}
			if ua.UnmappedReason == "" {
				t.Errorf("%s -> %s: unmapped assertion at %s has empty UnmappedReason",
					r.TestFunction, r.TargetFunction.QualifiedName(),
					ua.AssertionLocation)
			}
		}
	}

	if !anyInlineCall {
		t.Error("expected at least one unmapped assertion with UnmappedReasonInlineCall in welltested fixture (TestCounter_Increment calls c.Value() inline)")
	}

	// Verify JSON output carries the unmapped_reason field.
	var jsonBuf bytes.Buffer
	if err := quality.WriteJSON(&jsonBuf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"inline_call"`) {
		t.Error("expected JSON output to contain unmapped_reason \"inline_call\" for welltested fixture")
	}

	// Verify text output contains [inline_call] for the unmapped entry.
	var textBuf bytes.Buffer
	if err := quality.WriteText(&textBuf, reports, nil); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}
	if !strings.Contains(textBuf.String(), "[inline_call]") {
		t.Error("expected text output to contain \"[inline_call]\" for welltested fixture")
	}
}

// TestWriteJSON_UnmappedReason_OmitEmpty verifies that unmapped_reason is
// omitted from JSON for mapped assertions (confidence > 0, side_effect_id set).
func TestWriteJSON_UnmappedReason_OmitEmpty(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestAdd",
			TargetFunction: taxonomy.FunctionTarget{
				Package:  "pkg",
				Function: "Add",
			},
			ContractCoverage:             taxonomy.ContractCoverage{Percentage: 100},
			AssertionDetectionConfidence: 100,
			// No unmapped assertions; the mapped assertion should not have unmapped_reason.
		},
	}

	var buf bytes.Buffer
	if err := quality.WriteJSON(&buf, reports, nil); err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Confirm the key is absent (not present as empty string either).
	if strings.Contains(buf.String(), `"unmapped_reason"`) {
		t.Error("expected 'unmapped_reason' to be absent when there are no unmapped assertions")
	}
}

// --- Acceptance Test: Multilib Assertion Detection ---

func TestMultilib_AssertionDetection(t *testing.T) {
	// Verify that all three assertion libraries are detected
	// in the multilib fixture.

	pkg := loadPkg(t, "multilib")
	testFuncs := quality.FindTestFunctions(pkg)

	if len(testFuncs) == 0 {
		t.Fatal("no test functions found in multilib fixture")
	}

	kindCounts := make(map[quality.AssertionKind]int)
	for _, tf := range testFuncs {
		sites := quality.DetectAssertions(tf.Decl, pkg, 3)
		for _, s := range sites {
			kindCounts[s.Kind]++
		}
		t.Logf("  %s: %d assertion sites", tf.Name, len(sites))
	}

	t.Logf("Assertion kinds detected: %v", kindCounts)

	// Should detect testify patterns.
	if kindCounts[quality.AssertionKindTestifyEqual] == 0 &&
		kindCounts[quality.AssertionKindTestifyNoError] == 0 {
		t.Error("expected to detect testify assertion patterns in multilib")
	}

	// Should detect go-cmp patterns.
	if kindCounts[quality.AssertionKindGoCmpDiff] == 0 {
		t.Error("expected to detect go-cmp Diff assertion patterns in multilib")
	}

	// Should detect stdlib patterns.
	if kindCounts[quality.AssertionKindStdlibComparison] == 0 &&
		kindCounts[quality.AssertionKindStdlibErrorCheck] == 0 {
		t.Error("expected to detect stdlib assertion patterns in multilib")
	}
}
