package mockprovider_test

import (
	"fmt"
	"io"
	"math"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/provider/mockprovider"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// testdataDir returns the absolute path to the testdata directory
// adjacent to this test file.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata")
}

// sampleFile returns the path to testdata/sample.go, a non-generated
// Go source file used as a fixture for isGeneratedFile compatibility.
func sampleFile() string {
	return filepath.Join(testdataDir(), "sample.go")
}

// floatClose returns true if a and b are within epsilon of each other.
func floatClose(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// Test 1: Mock complexity + coverage providers produce CRAP scores
// matching crap.Formula() for 3 functions with known values.
func TestMockProviders_CRAPScoresMatchFormula(t *testing.T) {
	file := sampleFile()

	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{
			{Package: "pkg", Function: "Simple", File: file, Line: 1, Complexity: 1},
			{Package: "pkg", Function: "Medium", File: file, Line: 10, Complexity: 10},
			{Package: "pkg", Function: "Complex", File: file, Line: 25, Complexity: 25},
		},
	}

	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{
			{File: file, FuncName: "Simple", StartLine: 1, EndLine: 5, Percentage: 100.0},
			{File: file, FuncName: "Medium", StartLine: 10, EndLine: 20, Percentage: 50.0},
			{File: file, FuncName: "Complex", StartLine: 25, EndLine: 50, Percentage: 0.0},
		},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(rpt.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(rpt.Scores))
	}

	// Verify each score matches crap.Formula.
	expected := map[string]struct {
		complexity int
		coverage   float64
	}{
		"Simple":  {1, 100.0},
		"Medium":  {10, 50.0},
		"Complex": {25, 0.0},
	}

	for _, score := range rpt.Scores {
		exp, ok := expected[score.Function]
		if !ok {
			t.Errorf("unexpected function %q", score.Function)
			continue
		}
		want := crap.Formula(exp.complexity, exp.coverage)
		if !floatClose(score.CRAP, want, 0.001) {
			t.Errorf("function %q: CRAP = %f, want %f (Formula(%d, %f))",
				score.Function, score.CRAP, want, exp.complexity, exp.coverage)
		}
	}
}

// Test 2: All four mock providers produce a valid crap.Report with
// correct quadrant classifications and fix strategies for 5 functions
// spanning all four quadrants.
func TestMockProviders_QuadrantClassificationAndFixStrategies(t *testing.T) {
	file := sampleFile()

	// Design 5 functions to cover all 4 quadrants. Quadrant
	// classification uses CRAP threshold=15 and GazeCRAP threshold=15:
	//
	// Q1 Safe: CRAP<15, GazeCRAP<15
	//   complexity=2, lineCov=100%, contractCov=100%
	//   CRAP = 4×0 + 2 = 2, GazeCRAP = 4×0 + 2 = 2
	//
	// Q2 ComplexButTested: CRAP≥15, GazeCRAP<15
	//   complexity=5, lineCov=0%, contractCov=100%
	//   CRAP = 25×1 + 5 = 30, GazeCRAP = 25×0 + 5 = 5
	//
	// Q3 SimpleButUnderspecified: CRAP<15, GazeCRAP≥15
	//   complexity=4, lineCov=100%, contractCov=0%
	//   CRAP = 16×0 + 4 = 4, GazeCRAP = 16×1 + 4 = 20
	//
	// Q4 Dangerous: CRAP≥15, GazeCRAP≥15
	//   complexity=20, lineCov=0%, contractCov=0%
	//   CRAP = 400 + 20 = 420, GazeCRAP = 400 + 20 = 420
	//
	// Healthy (below threshold): complexity=1, lineCov=100%, contractCov=100%
	//   CRAP = 1, GazeCRAP = 1
	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{
			{Package: "pkg", Function: "Safe", File: file, Line: 1, Complexity: 2},
			{Package: "pkg", Function: "ComplexTested", File: file, Line: 10, Complexity: 5},
			{Package: "pkg", Function: "SimpleUntested", File: file, Line: 20, Complexity: 4},
			{Package: "pkg", Function: "Dangerous", File: file, Line: 30, Complexity: 20},
			{Package: "pkg", Function: "Healthy", File: file, Line: 40, Complexity: 1},
		},
	}

	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{
			{File: file, FuncName: "Safe", StartLine: 1, EndLine: 5, Percentage: 100.0},
			{File: file, FuncName: "ComplexTested", StartLine: 10, EndLine: 18, Percentage: 0.0},
			{File: file, FuncName: "SimpleUntested", StartLine: 20, EndLine: 25, Percentage: 100.0},
			{File: file, FuncName: "Dangerous", StartLine: 30, EndLine: 38, Percentage: 0.0},
			{File: file, FuncName: "Healthy", StartLine: 40, EndLine: 45, Percentage: 100.0},
		},
	}

	contractProvider := &mockprovider.MockContractCoverageProvider{
		LookupFunc: func(pkg, function string) (crap.ContractCoverageInfo, bool) {
			switch function {
			case "Safe":
				return crap.ContractCoverageInfo{Percentage: 100.0}, true
			case "ComplexTested":
				return crap.ContractCoverageInfo{Percentage: 100.0}, true
			case "SimpleUntested":
				return crap.ContractCoverageInfo{Percentage: 0.0}, true
			case "Dangerous":
				return crap.ContractCoverageInfo{Percentage: 0.0}, true
			case "Healthy":
				return crap.ContractCoverageInfo{Percentage: 100.0}, true
			default:
				return crap.ContractCoverageInfo{}, false
			}
		},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.ContractCoverageProvider = contractProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(rpt.Scores) != 5 {
		t.Fatalf("expected 5 scores, got %d", len(rpt.Scores))
	}

	// Build lookup by function name.
	scoreMap := make(map[string]crap.Score)
	for _, s := range rpt.Scores {
		scoreMap[s.Function] = s
	}

	// Verify quadrant assignments.
	tests := []struct {
		function string
		quadrant crap.Quadrant
	}{
		{"Safe", crap.Q1Safe},
		{"ComplexTested", crap.Q2ComplexButTested},
		{"SimpleUntested", crap.Q3SimpleButUnderspecified},
		{"Dangerous", crap.Q4Dangerous},
		{"Healthy", crap.Q1Safe},
	}

	for _, tt := range tests {
		s, ok := scoreMap[tt.function]
		if !ok {
			t.Errorf("function %q not found in scores", tt.function)
			continue
		}
		if s.Quadrant == nil {
			t.Errorf("function %q: expected quadrant %q, got nil", tt.function, tt.quadrant)
			continue
		}
		if *s.Quadrant != tt.quadrant {
			t.Errorf("function %q: quadrant = %q, want %q", tt.function, *s.Quadrant, tt.quadrant)
		}
	}

	// Verify fix strategies exist for CRAPload functions.
	// Dangerous (complexity=20, coverage=0%) should have decompose_and_test.
	if s := scoreMap["Dangerous"]; s.FixStrategy == nil {
		t.Error("Dangerous: expected non-nil FixStrategy")
	} else if *s.FixStrategy != crap.FixDecomposeAndTest {
		t.Errorf("Dangerous: FixStrategy = %q, want %q", *s.FixStrategy, crap.FixDecomposeAndTest)
	}

	// Healthy (complexity=1, coverage=100%) should have no fix strategy.
	if s := scoreMap["Healthy"]; s.FixStrategy != nil {
		t.Errorf("Healthy: expected nil FixStrategy, got %q", *s.FixStrategy)
	}

	// Verify summary has quadrant counts.
	if rpt.Summary.QuadrantCounts == nil {
		t.Fatal("expected non-nil QuadrantCounts in summary")
	}
}

// Test 3: Empty provider results produce an empty report (no scores,
// zero-valued summary), not an error.
func TestMockProviders_EmptyResults(t *testing.T) {
	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{},
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("expected nil error for empty results, got: %v", err)
	}

	if len(rpt.Scores) != 0 {
		t.Errorf("expected 0 scores, got %d", len(rpt.Scores))
	}
	if rpt.Summary.TotalFunctions != 0 {
		t.Errorf("expected TotalFunctions=0, got %d", rpt.Summary.TotalFunctions)
	}
}

// Test 4: ContractCoverageProvider returning reason
// "all_effects_ambiguous" flows through to Score.ContractCoverageReason.
func TestMockProviders_AmbiguousEffectsReason(t *testing.T) {
	file := sampleFile()

	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{
			{Package: "pkg", Function: "Ambiguous", File: file, Line: 1, Complexity: 5},
		},
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{
			{File: file, FuncName: "Ambiguous", StartLine: 1, EndLine: 10, Percentage: 80.0},
		},
	}

	contractProvider := &mockprovider.MockContractCoverageProvider{
		LookupFunc: func(pkg, function string) (crap.ContractCoverageInfo, bool) {
			return crap.ContractCoverageInfo{
				Percentage:    0.0,
				Reason:        "all_effects_ambiguous",
				MinConfidence: 78,
				MaxConfidence: 79,
			}, true
		},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.ContractCoverageProvider = contractProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(rpt.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(rpt.Scores))
	}

	s := rpt.Scores[0]
	if s.ContractCoverageReason == nil {
		t.Fatal("expected non-nil ContractCoverageReason")
	}
	if *s.ContractCoverageReason != "all_effects_ambiguous" {
		t.Errorf("ContractCoverageReason = %q, want %q",
			*s.ContractCoverageReason, "all_effects_ambiguous")
	}
	if s.EffectConfidenceRange == nil {
		t.Fatal("expected non-nil EffectConfidenceRange")
	}
	if s.EffectConfidenceRange[0] != 78 || s.EffectConfidenceRange[1] != 79 {
		t.Errorf("EffectConfidenceRange = %v, want [78, 79]",
			*s.EffectConfidenceRange)
	}
}

// Test 5: ContractCoverageProvider returning error causes graceful
// degradation (CRAP scores computed without GazeCRAP).
func TestMockProviders_ContractProviderError(t *testing.T) {
	file := sampleFile()

	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{
			{Package: "pkg", Function: "Func1", File: file, Line: 1, Complexity: 5},
		},
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{
			{File: file, FuncName: "Func1", StartLine: 1, EndLine: 10, Percentage: 80.0},
		},
	}

	contractProvider := &mockprovider.MockContractCoverageProvider{
		Err: fmt.Errorf("quality pipeline unavailable"),
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.ContractCoverageProvider = contractProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("expected nil error (graceful degradation), got: %v", err)
	}

	if len(rpt.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(rpt.Scores))
	}

	// CRAP should be computed (line coverage available).
	s := rpt.Scores[0]
	expectedCRAP := crap.Formula(5, 80.0)
	if !floatClose(s.CRAP, expectedCRAP, 0.001) {
		t.Errorf("CRAP = %f, want %f", s.CRAP, expectedCRAP)
	}

	// GazeCRAP should be nil (contract coverage provider failed).
	if s.GazeCRAP != nil {
		t.Errorf("expected nil GazeCRAP (graceful degradation), got %f", *s.GazeCRAP)
	}
	if s.Quadrant != nil {
		t.Errorf("expected nil Quadrant (graceful degradation), got %q", *s.Quadrant)
	}
}

// HIGH 1a: Nil ComplexityProvider guard returns an error.
func TestAnalyze_NilComplexityProvider(t *testing.T) {
	opts := crap.DefaultOptions()
	opts.ComplexityProvider = nil
	opts.LineCoverageProvider = &mockprovider.MockLineCoverageProvider{}
	opts.Stderr = io.Discard

	_, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err == nil {
		t.Fatal("expected error for nil ComplexityProvider, got nil")
	}
	if !strings.Contains(err.Error(), "ComplexityProvider is required") {
		t.Errorf("error = %q, want substring %q", err.Error(), "ComplexityProvider is required")
	}
}

// HIGH 1b: Nil LineCoverageProvider guard returns an error.
func TestAnalyze_NilLineCoverageProvider(t *testing.T) {
	opts := crap.DefaultOptions()
	opts.ComplexityProvider = &mockprovider.MockComplexityProvider{}
	opts.LineCoverageProvider = nil
	opts.Stderr = io.Discard

	_, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err == nil {
		t.Fatal("expected error for nil LineCoverageProvider, got nil")
	}
	if !strings.Contains(err.Error(), "LineCoverageProvider is required") {
		t.Errorf("error = %q, want substring %q", err.Error(), "LineCoverageProvider is required")
	}
}

// HIGH 2: MockSideEffectAnalyzer returns configured data and errors.
func TestMockSideEffectAnalyzer_ReturnsConfiguredData(t *testing.T) {
	t.Run("returns configured results", func(t *testing.T) {
		want := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{
					Package:  "pkg",
					Function: "Foo",
				},
				SideEffects: []taxonomy.SideEffect{
					{Type: taxonomy.ReturnValue, Description: "returns int"},
				},
			},
		}

		mock := &mockprovider.MockSideEffectAnalyzer{
			Results: want,
		}

		got, err := mock.Analyze("pkg")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != len(want) {
			t.Fatalf("got %d results, want %d", len(got), len(want))
		}
		if got[0].Target.Function != "Foo" {
			t.Errorf("Function = %q, want %q", got[0].Target.Function, "Foo")
		}
		if len(got[0].SideEffects) != 1 {
			t.Fatalf("got %d side effects, want 1", len(got[0].SideEffects))
		}
		if got[0].SideEffects[0].Type != taxonomy.ReturnValue {
			t.Errorf("SideEffect.Type = %q, want %q",
				got[0].SideEffects[0].Type, taxonomy.ReturnValue)
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		mock := &mockprovider.MockSideEffectAnalyzer{
			Err: fmt.Errorf("analysis failed"),
		}

		_, err := mock.Analyze("pkg")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "analysis failed") {
			t.Errorf("error = %q, want substring %q", err.Error(), "analysis failed")
		}
	})
}

// HIGH 3: DegradedPackages from MockContractCoverageProvider flow
// through to Summary.SSADegradedPackages.
func TestMockProviders_DegradedPackagesFlowThrough(t *testing.T) {
	file := sampleFile()

	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{
			{Package: "pkg", Function: "Func1", File: file, Line: 1, Complexity: 5},
		},
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{
			{File: file, FuncName: "Func1", StartLine: 1, EndLine: 10, Percentage: 80.0},
		},
	}

	contractProvider := &mockprovider.MockContractCoverageProvider{
		LookupFunc: func(pkg, function string) (crap.ContractCoverageInfo, bool) {
			return crap.ContractCoverageInfo{Percentage: 50.0}, true
		},
		DegradedPackages: []string{"pkg/degraded"},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.ContractCoverageProvider = contractProvider
	opts.Stderr = io.Discard

	rpt, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(rpt.Summary.SSADegradedPackages) == 0 {
		t.Fatal("expected SSADegradedPackages to be populated, got empty")
	}

	found := false
	for _, pkg := range rpt.Summary.SSADegradedPackages {
		if pkg == "pkg/degraded" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("SSADegradedPackages = %v, want to contain %q",
			rpt.Summary.SSADegradedPackages, "pkg/degraded")
	}
}

// MEDIUM 1a: ComplexityProvider error propagates with context wrapping.
func TestMockProviders_ComplexityProviderError(t *testing.T) {
	complexityProvider := &mockprovider.MockComplexityProvider{
		Err: fmt.Errorf("gocyclo not found"),
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Results: []crap.FuncCoverage{},
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.Stderr = io.Discard

	_, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err == nil {
		t.Fatal("expected error from ComplexityProvider, got nil")
	}
	if !strings.Contains(err.Error(), "complexity analysis") {
		t.Errorf("error = %q, want substring %q", err.Error(), "complexity analysis")
	}
	if !strings.Contains(err.Error(), "gocyclo not found") {
		t.Errorf("error = %q, want substring %q", err.Error(), "gocyclo not found")
	}
}

// MEDIUM 1b: LineCoverageProvider error propagates with context wrapping.
func TestMockProviders_LineCoverageProviderError(t *testing.T) {
	complexityProvider := &mockprovider.MockComplexityProvider{
		Results: []crap.FunctionComplexity{},
	}
	coverageProvider := &mockprovider.MockLineCoverageProvider{
		Err: fmt.Errorf("go test timed out"),
	}

	opts := crap.DefaultOptions()
	opts.ComplexityProvider = complexityProvider
	opts.LineCoverageProvider = coverageProvider
	opts.Stderr = io.Discard

	_, err := crap.Analyze([]string{"./..."}, testdataDir(), opts)
	if err == nil {
		t.Fatal("expected error from LineCoverageProvider, got nil")
	}
	if !strings.Contains(err.Error(), "line coverage") {
		t.Errorf("error = %q, want substring %q", err.Error(), "line coverage")
	}
	if !strings.Contains(err.Error(), "go test timed out") {
		t.Errorf("error = %q, want substring %q", err.Error(), "go test timed out")
	}
}
