package crap

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// --- Task 3.4: LoadBaseline tests ---

func TestSC002_LoadBaseline_ValidJSON(t *testing.T) {
	input := `{
		"scores": [
			{
				"package": "crap",
				"function": "Analyze",
				"file": "internal/crap/analyze.go",
				"line": 91,
				"complexity": 5,
				"line_coverage": 80.0,
				"crap": 5.8,
				"gaze_crap": 7.2,
				"contract_coverage": 60.0
			}
		],
		"summary": {
			"total_functions": 1,
			"avg_complexity": 5.0,
			"avg_line_coverage": 80.0,
			"avg_crap": 5.8,
			"crapload": 0,
			"crap_threshold": 15,
			"worst_crap": []
		}
	}`

	r := strings.NewReader(input)
	report, err := LoadBaseline(r)
	if err != nil {
		t.Fatalf("LoadBaseline() error: %v", err)
	}

	if len(report.Scores) != 1 {
		t.Fatalf("len(Scores) = %d, want 1", len(report.Scores))
	}

	s := report.Scores[0]
	if s.Function != "Analyze" {
		t.Errorf("Function = %q, want %q", s.Function, "Analyze")
	}
	if s.File != "internal/crap/analyze.go" {
		t.Errorf("File = %q, want %q", s.File, "internal/crap/analyze.go")
	}
	if s.CRAP != 5.8 {
		t.Errorf("CRAP = %g, want 5.8", s.CRAP)
	}
	if s.GazeCRAP == nil || *s.GazeCRAP != 7.2 {
		t.Errorf("GazeCRAP = %v, want 7.2", s.GazeCRAP)
	}
	if report.Summary.TotalFunctions != 1 {
		t.Errorf("Summary.TotalFunctions = %d, want 1",
			report.Summary.TotalFunctions)
	}
}

func TestSC002_LoadBaseline_InvalidJSON(t *testing.T) {
	r := strings.NewReader(`{not valid json`)
	_, err := LoadBaseline(r)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "parsing baseline JSON") {
		t.Errorf("error = %q, want to contain %q",
			err, "parsing baseline JSON")
	}
}

func TestSC002_LoadBaseline_EmptyInput(t *testing.T) {
	r := strings.NewReader("")
	_, err := LoadBaseline(r)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}

	if !strings.Contains(err.Error(), "baseline is empty") {
		t.Errorf("error = %q, want to contain %q",
			err, "baseline is empty")
	}
}

func TestSC002_LoadBaseline_OlderVersion(t *testing.T) {
	// Baseline from an older gaze version without gaze_crap,
	// fix_strategy, or quadrant fields. Missing optional fields
	// should get zero/nil values (D5, R6).
	input := `{
		"scores": [
			{
				"package": "crap",
				"function": "Formula",
				"file": "internal/crap/crap.go",
				"line": 100,
				"complexity": 2,
				"line_coverage": 100.0,
				"crap": 2.0
			}
		],
		"summary": {
			"total_functions": 1,
			"avg_complexity": 2.0,
			"avg_line_coverage": 100.0,
			"avg_crap": 2.0,
			"crapload": 0,
			"crap_threshold": 15,
			"worst_crap": []
		}
	}`

	r := strings.NewReader(input)
	report, err := LoadBaseline(r)
	if err != nil {
		t.Fatalf("LoadBaseline() error: %v", err)
	}

	s := report.Scores[0]
	if s.GazeCRAP != nil {
		t.Errorf("GazeCRAP = %v, want nil (older version has no GazeCRAP)",
			*s.GazeCRAP)
	}
	if s.FixStrategy != nil {
		t.Errorf("FixStrategy = %v, want nil", *s.FixStrategy)
	}
	if s.Quadrant != nil {
		t.Errorf("Quadrant = %v, want nil", *s.Quadrant)
	}
}

func TestSC002_LoadBaseline_ExtraFields(t *testing.T) {
	// Baseline from a newer gaze version with unknown fields.
	// Standard json.Unmarshal ignores unknown fields (R6).
	input := `{
		"scores": [
			{
				"package": "crap",
				"function": "Formula",
				"file": "internal/crap/crap.go",
				"line": 100,
				"complexity": 2,
				"line_coverage": 100.0,
				"crap": 2.0,
				"some_future_field": "should be ignored",
				"another_future_field": 42
			}
		],
		"summary": {
			"total_functions": 1,
			"avg_complexity": 2.0,
			"avg_line_coverage": 100.0,
			"avg_crap": 2.0,
			"crapload": 0,
			"crap_threshold": 15,
			"worst_crap": [],
			"future_summary_field": true
		}
	}`

	r := strings.NewReader(input)
	report, err := LoadBaseline(r)
	if err != nil {
		t.Fatalf("LoadBaseline() error: %v", err)
	}

	if len(report.Scores) != 1 {
		t.Fatalf("len(Scores) = %d, want 1", len(report.Scores))
	}
	if report.Scores[0].CRAP != 2.0 {
		t.Errorf("CRAP = %g, want 2.0", report.Scores[0].CRAP)
	}
}

// --- Task 3.5: Compare tests ---

// helper to create a Score with the minimum required fields.
func makeScore(file, function string, crapScore float64, gazeCRAP *float64) Score {
	return Score{
		Package:  "test",
		Function: function,
		File:     file,
		Line:     1,
		CRAP:     crapScore,
		GazeCRAP: gazeCRAP,
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestSC003_Compare_CRAPRegression(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 12.5, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(result.Deltas))
	}

	d := result.Deltas[0]
	if d.Status != StatusRegression {
		t.Errorf("Status = %q, want %q", d.Status, StatusRegression)
	}

	wantDelta := 3.3
	if diff := d.CRAPDelta - wantDelta; diff > 0.01 || diff < -0.01 {
		t.Errorf("CRAPDelta = %g, want ~%g", d.CRAPDelta, wantDelta)
	}

	if result.Summary.Passed {
		t.Error("Summary.Passed = true, want false (regression)")
	}
	if result.Summary.Regressions != 1 {
		t.Errorf("Summary.Regressions = %d, want 1",
			result.Summary.Regressions)
	}
}

func TestSC003_Compare_CRAPImprovement(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 12.5, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 8.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(result.Deltas))
	}

	d := result.Deltas[0]
	if d.Status != StatusImprovement {
		t.Errorf("Status = %q, want %q", d.Status, StatusImprovement)
	}

	wantDelta := -4.5
	if diff := d.CRAPDelta - wantDelta; diff > 0.01 || diff < -0.01 {
		t.Errorf("CRAPDelta = %g, want ~%g", d.CRAPDelta, wantDelta)
	}

	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (improvement only)")
	}
	if result.Summary.Improvements != 1 {
		t.Errorf("Summary.Improvements = %d, want 1",
			result.Summary.Improvements)
	}
}

func TestSC003_Compare_WithinEpsilon(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.5, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(result.Deltas))
	}

	d := result.Deltas[0]
	if d.Status != StatusUnchanged {
		t.Errorf("Status = %q, want %q", d.Status, StatusUnchanged)
	}

	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (within epsilon)")
	}
	if result.Summary.Unchanged != 1 {
		t.Errorf("Summary.Unchanged = %d, want 1",
			result.Summary.Unchanged)
	}
}

func TestSC003_Compare_GazeCRAPRegression(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2,
				float64Ptr(14.1)),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2,
				float64Ptr(18.3)),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	d := result.Deltas[0]
	if d.Status != StatusRegression {
		t.Errorf("Status = %q, want %q (GazeCRAP regression)",
			d.Status, StatusRegression)
	}

	if d.GazeCRAPDelta == nil {
		t.Fatal("GazeCRAPDelta is nil, want non-nil")
	}

	wantGazeDelta := 4.2
	if diff := *d.GazeCRAPDelta - wantGazeDelta; diff > 0.01 || diff < -0.01 {
		t.Errorf("GazeCRAPDelta = %g, want ~%g",
			*d.GazeCRAPDelta, wantGazeDelta)
	}

	if result.Summary.Passed {
		t.Error("Summary.Passed = true, want false")
	}
}

func TestSC003_Compare_GazeCRAPSkipNoBaseline(t *testing.T) {
	// When baseline GazeCRAP is nil, current GazeCRAP should
	// not trigger a regression (D5).
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.2,
				float64Ptr(25.0)),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	d := result.Deltas[0]
	if d.Status != StatusUnchanged {
		t.Errorf("Status = %q, want %q (GazeCRAP skip, CRAP unchanged)",
			d.Status, StatusUnchanged)
	}

	if d.GazeCRAPDelta != nil {
		t.Errorf("GazeCRAPDelta = %v, want nil (baseline had no GazeCRAP)",
			*d.GazeCRAPDelta)
	}
}

func TestSC003_Compare_ConflictingSignals(t *testing.T) {
	// CRAP regresses but GazeCRAP improves. Regression MUST win
	// per SC-003.
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0,
				float64Ptr(18.0)),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 12.0,
				float64Ptr(14.0)),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	d := result.Deltas[0]
	if d.Status != StatusRegression {
		t.Errorf("Status = %q, want %q (CRAP regressed, GazeCRAP improved -> regression wins)",
			d.Status, StatusRegression)
	}
}

func TestSC004_Compare_NewFunctionBelowThreshold(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
			makeScore("internal/crap/helper.go", "helperFunc", 12.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.NewFunctions) != 1 {
		t.Fatalf("len(NewFunctions) = %d, want 1",
			len(result.NewFunctions))
	}

	if result.NewFunctions[0].Function != "helperFunc" {
		t.Errorf("NewFunctions[0].Function = %q, want %q",
			result.NewFunctions[0].Function, "helperFunc")
	}

	if result.Summary.NewViolations != 0 {
		t.Errorf("NewViolations = %d, want 0 (below threshold)",
			result.Summary.NewViolations)
	}
	if result.Summary.NewFunctions != 1 {
		t.Errorf("NewFunctions count = %d, want 1",
			result.Summary.NewFunctions)
	}
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true")
	}
}

func TestSC004_Compare_NewFunctionAboveThreshold(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
			makeScore("internal/crap/complex.go", "complexFunc", 42.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.NewFunctions) != 1 {
		t.Fatalf("len(NewFunctions) = %d, want 1",
			len(result.NewFunctions))
	}

	if result.Summary.NewViolations != 1 {
		t.Errorf("NewViolations = %d, want 1 (above threshold)",
			result.Summary.NewViolations)
	}
	if result.Summary.Passed {
		t.Error("Summary.Passed = true, want false (new violation)")
	}
}

func TestSC005_Compare_RemovedFunction(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
			makeScore("internal/crap/old.go", "oldFunc", 5.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 9.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.RemovedFunctions) != 1 {
		t.Fatalf("len(RemovedFunctions) = %d, want 1",
			len(result.RemovedFunctions))
	}

	if result.RemovedFunctions[0].Function != "oldFunc" {
		t.Errorf("RemovedFunctions[0].Function = %q, want %q",
			result.RemovedFunctions[0].Function, "oldFunc")
	}

	if result.Summary.RemovedFunctions != 1 {
		t.Errorf("Summary.RemovedFunctions = %d, want 1",
			result.Summary.RemovedFunctions)
	}

	// Removed functions do not count as failures (SC-005).
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (removed is informational)")
	}
}

func TestSC006_Compare_MixedResultsPass(t *testing.T) {
	// 0 regressions, 1 improvement, 1 new (below threshold),
	// 1 removed => pass.
	baseline := &Report{
		Scores: []Score{
			makeScore("a.go", "Improved", 15.0, nil),
			makeScore("a.go", "Unchanged", 5.0, nil),
			makeScore("b.go", "Removed", 3.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("a.go", "Improved", 10.0, nil),
			makeScore("a.go", "Unchanged", 5.0, nil),
			makeScore("c.go", "NewFunc", 12.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if result.Summary.Regressions != 0 {
		t.Errorf("Regressions = %d, want 0", result.Summary.Regressions)
	}
	if result.Summary.Improvements != 1 {
		t.Errorf("Improvements = %d, want 1", result.Summary.Improvements)
	}
	if result.Summary.Unchanged != 1 {
		t.Errorf("Unchanged = %d, want 1", result.Summary.Unchanged)
	}
	if result.Summary.NewFunctions != 1 {
		t.Errorf("NewFunctions = %d, want 1", result.Summary.NewFunctions)
	}
	if result.Summary.RemovedFunctions != 1 {
		t.Errorf("RemovedFunctions = %d, want 1",
			result.Summary.RemovedFunctions)
	}
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true")
	}
}

func TestSC006_Compare_MixedResultsFail(t *testing.T) {
	// 1 regression + 1 improvement + 1 new violation => fail.
	baseline := &Report{
		Scores: []Score{
			makeScore("a.go", "Regressed", 9.0, nil),
			makeScore("a.go", "Improved", 15.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("a.go", "Regressed", 14.0, nil),
			makeScore("a.go", "Improved", 10.0, nil),
			makeScore("c.go", "ComplexNew", 42.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if result.Summary.Regressions != 1 {
		t.Errorf("Regressions = %d, want 1", result.Summary.Regressions)
	}
	if result.Summary.Improvements != 1 {
		t.Errorf("Improvements = %d, want 1", result.Summary.Improvements)
	}
	if result.Summary.NewViolations != 1 {
		t.Errorf("NewViolations = %d, want 1", result.Summary.NewViolations)
	}
	if result.Summary.Passed {
		t.Error("Summary.Passed = true, want false")
	}
}

func TestSC006_Compare_EmptyBaseline(t *testing.T) {
	baseline := &Report{
		Scores: []Score{},
	}
	current := &Report{
		Scores: []Score{
			makeScore("a.go", "FuncA", 5.0, nil),
			makeScore("b.go", "FuncB", 10.0, nil),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.Deltas) != 0 {
		t.Errorf("len(Deltas) = %d, want 0 (empty baseline)",
			len(result.Deltas))
	}
	if len(result.NewFunctions) != 2 {
		t.Errorf("len(NewFunctions) = %d, want 2",
			len(result.NewFunctions))
	}
	if len(result.RemovedFunctions) != 0 {
		t.Errorf("len(RemovedFunctions) = %d, want 0",
			len(result.RemovedFunctions))
	}
	if result.Summary.NewFunctions != 2 {
		t.Errorf("NewFunctions = %d, want 2",
			result.Summary.NewFunctions)
	}
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (all below threshold)")
	}
}

func TestSC006_Compare_EmptyCurrent(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("a.go", "FuncA", 5.0, nil),
			makeScore("b.go", "FuncB", 10.0, nil),
		},
	}
	current := &Report{
		Scores: []Score{},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	if len(result.Deltas) != 0 {
		t.Errorf("len(Deltas) = %d, want 0 (empty current)",
			len(result.Deltas))
	}
	if len(result.NewFunctions) != 0 {
		t.Errorf("len(NewFunctions) = %d, want 0",
			len(result.NewFunctions))
	}
	if len(result.RemovedFunctions) != 2 {
		t.Errorf("len(RemovedFunctions) = %d, want 2",
			len(result.RemovedFunctions))
	}
	if result.Summary.RemovedFunctions != 2 {
		t.Errorf("RemovedFunctions = %d, want 2",
			result.Summary.RemovedFunctions)
	}
	// Removed functions are informational — pass.
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (removed is informational)")
	}
}

// TestSC003_Compare_GazeCRAPSkipZeroBaseline verifies that a
// baseline GazeCRAP of 0 is treated the same as nil (D5).
func TestSC003_Compare_GazeCRAPSkipZeroBaseline(t *testing.T) {
	baseline := &Report{
		Scores: []Score{
			makeScore("a.go", "Func", 9.0, float64Ptr(0.0)),
		},
	}
	current := &Report{
		Scores: []Score{
			makeScore("a.go", "Func", 9.0, float64Ptr(25.0)),
		},
	}

	result := Compare(baseline, current, CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	})

	d := result.Deltas[0]
	if d.Status != StatusUnchanged {
		t.Errorf("Status = %q, want %q (baseline GazeCRAP=0, skip per D5)",
			d.Status, StatusUnchanged)
	}
	if d.GazeCRAPDelta != nil {
		t.Errorf("GazeCRAPDelta = %v, want nil (baseline GazeCRAP=0)",
			*d.GazeCRAPDelta)
	}
}

// --- Task 6: Integration tests ---
//
// These tests exercise the full comparison flow: LoadBaseline +
// Compare + WriteComparisonJSON/Text using static fixtures. They
// validate end-to-end data flow without subprocess execution.
// No testing.Short() guard needed -- these read local JSON and
// run pure functions, completing in milliseconds.

// TestSC003_RegressionDetection validates the full comparison flow
// when CRAP scores regress between baseline and current. Loads the
// sample_baseline.json fixture, constructs a current Report with
// increased scores for two functions, runs Compare, and verifies
// the regression is detected in both the ComparisonResult and the
// JSON/text output (task 6.1).
func TestSC003_RegressionDetection(t *testing.T) {
	// Load the sample baseline fixture (task 6.5).
	f, err := os.Open("testdata/sample_baseline.json")
	if err != nil {
		t.Fatalf("opening sample baseline: %v", err)
	}
	defer f.Close()

	baseline, err := LoadBaseline(f)
	if err != nil {
		t.Fatalf("LoadBaseline() error: %v", err)
	}

	// Construct a "current" report with regressions:
	// - Analyze: CRAP 12.4 -> 18.0 (delta +5.6, regression)
	// - Formula: CRAP 1.0 -> 1.0 (unchanged)
	// - WriteText: CRAP 5.5 -> 3.0 (improvement)
	// - ComputeScore: CRAP 9.4 -> 9.6 (within epsilon 0.5, unchanged)
	// - AnalyzeP1Effects: CRAP 28.7 -> 35.0 (delta +6.3, regression)
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 18.0,
				float64Ptr(22.0)),
			makeScore("internal/crap/crap.go", "Formula", 1.0,
				float64Ptr(1.0)),
			makeScore("internal/crap/report.go", "WriteText", 3.0,
				float64Ptr(4.0)),
			makeScore("internal/classify/score.go", "ComputeScore", 9.6,
				float64Ptr(11.5)),
			makeScore("internal/analysis/p1.go", "AnalyzeP1Effects", 35.0,
				float64Ptr(50.0)),
		},
		Summary: Summary{
			TotalFunctions:  5,
			AvgComplexity:   7.0,
			AvgLineCoverage: 65.0,
			AvgCRAP:         13.32,
			CRAPload:        1,
			CRAPThreshold:   15,
		},
	}

	opts := CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	}

	result := Compare(baseline, current, opts)

	// Verify the comparison detects regressions.
	if result.Summary.Passed {
		t.Error("Summary.Passed = true, want false (regressions present)")
	}

	if result.Summary.Regressions != 2 {
		t.Errorf("Summary.Regressions = %d, want 2", result.Summary.Regressions)
	}

	// Verify the regressed functions appear with StatusRegression.
	regressedFuncs := make(map[string]FunctionDelta)
	for _, d := range result.Deltas {
		if d.Status == StatusRegression {
			regressedFuncs[d.Current.Function] = d
		}
	}

	if _, ok := regressedFuncs["Analyze"]; !ok {
		t.Error("Analyze should be classified as regression")
	}
	if _, ok := regressedFuncs["AnalyzeP1Effects"]; !ok {
		t.Error("AnalyzeP1Effects should be classified as regression")
	}

	// Verify the improvement is detected.
	if result.Summary.Improvements != 1 {
		t.Errorf("Summary.Improvements = %d, want 1", result.Summary.Improvements)
	}

	// Verify unchanged count (Formula + ComputeScore).
	if result.Summary.Unchanged != 2 {
		t.Errorf("Summary.Unchanged = %d, want 2", result.Summary.Unchanged)
	}

	// Verify JSON output contains regression data.
	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var jsonOutput map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	var comparison ComparisonSummary
	if err := json.Unmarshal(jsonOutput["comparison"], &comparison); err != nil {
		t.Fatalf("parsing comparison: %v", err)
	}
	if comparison.Passed {
		t.Error("JSON comparison.passed = true, want false")
	}
	if comparison.Regressions != 2 {
		t.Errorf("JSON comparison.regressions = %d, want 2",
			comparison.Regressions)
	}

	// Verify text output contains FAIL.
	var textBuf bytes.Buffer
	if err := WriteComparisonText(&textBuf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}
	textOutput := textBuf.String()
	if !strings.Contains(textOutput, "FAIL") {
		t.Error("text output missing FAIL header")
	}
	if !strings.Contains(textOutput, "Regressions:") {
		t.Error("text output missing Regressions section")
	}
}

// TestSC001_NoBaselineLoaded verifies that when no baseline is loaded,
// the normal WriteJSON output contains zero comparison fields. This
// validates the non-comparison code path at a higher level than the
// existing TestSC007_BackwardCompat (task 6.2).
func TestSC001_NoBaselineLoaded(t *testing.T) {
	// Build a report as if gaze crap ran without a baseline.
	rpt := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 12.4,
				float64Ptr(16.2)),
			makeScore("internal/crap/crap.go", "Formula", 1.0,
				float64Ptr(1.0)),
			makeScore("internal/crap/report.go", "WriteText", 5.5,
				float64Ptr(8.1)),
		},
		Summary: Summary{
			TotalFunctions:  3,
			AvgComplexity:   4.7,
			AvgLineCoverage: 85.8,
			AvgCRAP:         6.3,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	// Write using the normal (non-comparison) JSON path.
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rpt); err != nil {
		t.Fatalf("WriteJSON() error: %v", err)
	}

	outputStr := buf.String()

	// Verify no comparison-specific fields appear.
	forbiddenFields := []string{
		"baseline_crap",
		"crap_delta",
		"baseline_gaze_crap",
		"gaze_crap_delta",
		"new_functions",
		"removed_functions",
		`"comparison"`,
		`"status"`,
	}
	for _, field := range forbiddenFields {
		if strings.Contains(outputStr, field) {
			t.Errorf("non-comparison JSON output contains comparison field %q", field)
		}
	}

	// Verify the JSON is valid and has the expected structure.
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	requiredKeys := []string{"scores", "summary"}
	for _, key := range requiredKeys {
		if _, ok := parsed[key]; !ok {
			t.Errorf("missing expected key %q in non-comparison output", key)
		}
	}

	// Verify the scores parse correctly.
	var scores []Score
	if err := json.Unmarshal(parsed["scores"], &scores); err != nil {
		t.Fatalf("parsing scores: %v", err)
	}
	if len(scores) != 3 {
		t.Errorf("len(scores) = %d, want 3", len(scores))
	}
}

// TestSC001_ExplicitBaselineNotFound verifies that LoadBaseline
// errors when given invalid input, and that os.Open on a non-existent
// path returns an error. This tests the error handling for the
// --baseline /nonexistent scenario (task 6.3).
func TestSC001_ExplicitBaselineNotFound(t *testing.T) {
	// Test 1: os.Open on a non-existent file path returns an error.
	// This mirrors the CLI behavior when --baseline points to a
	// missing file.
	_, err := os.Open("testdata/nonexistent_baseline.json")
	if err == nil {
		t.Fatal("expected error opening non-existent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist error, got: %v", err)
	}

	// Test 2: LoadBaseline with invalid JSON content returns
	// a parse error.
	invalidJSON := strings.NewReader(`{"scores": [invalid]}`)
	_, err = LoadBaseline(invalidJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing baseline JSON") {
		t.Errorf("error = %q, want to contain %q",
			err, "parsing baseline JSON")
	}

	// Test 3: LoadBaseline with truncated/malformed content.
	truncated := strings.NewReader(`{"scores": [{"function": "Foo"`)
	_, err = LoadBaseline(truncated)
	if err == nil {
		t.Fatal("expected error for truncated JSON, got nil")
	}
	if !strings.Contains(err.Error(), "parsing baseline JSON") {
		t.Errorf("error = %q, want to contain %q",
			err, "parsing baseline JSON")
	}

	// Test 4: LoadBaseline with empty content returns an error.
	empty := strings.NewReader("")
	_, err = LoadBaseline(empty)
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
	if !strings.Contains(err.Error(), "baseline is empty") {
		t.Errorf("error = %q, want to contain %q",
			err, "baseline is empty")
	}
}

// TestSC006_ComparisonPasses validates the full comparison flow when
// baseline exists and no regressions are present. Loads the sample
// baseline fixture, constructs a current Report with identical or
// improved scores, and verifies Summary.Passed is true (task 6.4).
func TestSC006_ComparisonPasses(t *testing.T) {
	// Load the sample baseline fixture.
	f, err := os.Open("testdata/sample_baseline.json")
	if err != nil {
		t.Fatalf("opening sample baseline: %v", err)
	}
	defer f.Close()

	baseline, err := LoadBaseline(f)
	if err != nil {
		t.Fatalf("LoadBaseline() error: %v", err)
	}

	// Construct a "current" report with improvements and unchanged:
	// - Analyze: CRAP 12.4 -> 8.0 (improvement)
	// - Formula: CRAP 1.0 -> 1.0 (unchanged)
	// - WriteText: CRAP 5.5 -> 4.0 (improvement)
	// - ComputeScore: CRAP 9.4 -> 9.2 (within epsilon 0.5, unchanged)
	// - AnalyzeP1Effects: CRAP 28.7 -> 20.0 (improvement)
	// - NewHelper: new function with CRAP 5.0 (below threshold 30)
	current := &Report{
		Scores: []Score{
			makeScore("internal/crap/analyze.go", "Analyze", 8.0,
				float64Ptr(10.0)),
			makeScore("internal/crap/crap.go", "Formula", 1.0,
				float64Ptr(1.0)),
			makeScore("internal/crap/report.go", "WriteText", 4.0,
				float64Ptr(5.0)),
			makeScore("internal/classify/score.go", "ComputeScore", 9.2,
				float64Ptr(11.0)),
			makeScore("internal/analysis/p1.go", "AnalyzeP1Effects", 20.0,
				float64Ptr(30.0)),
			makeScore("internal/crap/helper.go", "NewHelper", 5.0, nil),
		},
		Summary: Summary{
			TotalFunctions:  6,
			AvgComplexity:   5.0,
			AvgLineCoverage: 80.0,
			AvgCRAP:         7.87,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	opts := CompareOptions{
		Epsilon:              0.5,
		NewFunctionThreshold: 30,
	}

	result := Compare(baseline, current, opts)

	// Verify the comparison passes.
	if !result.Summary.Passed {
		t.Error("Summary.Passed = false, want true (no regressions)")
	}

	if result.Summary.Regressions != 0 {
		t.Errorf("Summary.Regressions = %d, want 0",
			result.Summary.Regressions)
	}
	if result.Summary.NewViolations != 0 {
		t.Errorf("Summary.NewViolations = %d, want 0",
			result.Summary.NewViolations)
	}

	// Verify improvements are detected.
	if result.Summary.Improvements != 3 {
		t.Errorf("Summary.Improvements = %d, want 3",
			result.Summary.Improvements)
	}

	// Verify unchanged count (Formula + ComputeScore).
	if result.Summary.Unchanged != 2 {
		t.Errorf("Summary.Unchanged = %d, want 2",
			result.Summary.Unchanged)
	}

	// Verify new function count (NewHelper, below threshold).
	if result.Summary.NewFunctions != 1 {
		t.Errorf("Summary.NewFunctions = %d, want 1",
			result.Summary.NewFunctions)
	}

	// Verify JSON output shows passed.
	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var jsonOutput map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	var comparison ComparisonSummary
	if err := json.Unmarshal(jsonOutput["comparison"], &comparison); err != nil {
		t.Fatalf("parsing comparison: %v", err)
	}
	if !comparison.Passed {
		t.Error("JSON comparison.passed = false, want true")
	}

	// Verify text output shows PASS.
	var textBuf bytes.Buffer
	if err := WriteComparisonText(&textBuf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}
	textOutput := textBuf.String()
	if !strings.Contains(textOutput, "PASS") {
		t.Error("text output missing PASS header")
	}
	if strings.Contains(textOutput, "Regressions:") {
		t.Error("text output should not contain Regressions section")
	}

	// Verify improvements section appears in text output.
	if !strings.Contains(textOutput, "Improvements:") {
		t.Error("text output missing Improvements section")
	}

	// Verify new function section appears.
	if !strings.Contains(textOutput, "New Functions:") {
		t.Error("text output missing New Functions section")
	}
}
