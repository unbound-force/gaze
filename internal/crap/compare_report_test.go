package crap

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- Task 4.3: WriteComparisonJSON tests ---

// buildTestComparisonResult creates a ComparisonResult with all
// status types represented for test coverage.
func buildTestComparisonResult() *ComparisonResult {
	return &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("internal/crap/analyze.go", "Analyze", 12.5, float64Ptr(18.3)),
				makeScore("internal/crap/report.go", "WriteText", 8.0, float64Ptr(10.0)),
				makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				// New functions are in NewFunctions list, but also in
				// Report.Scores since the current report contains them.
				makeScore("internal/crap/helper.go", "helperFunc", 12.0, nil),
				makeScore("internal/crap/complex.go", "complexFunc", 42.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 5,
				AvgComplexity:  4.0,
				AvgLineCoverage: 70.0,
				AvgCRAP:        15.3,
				CRAPload:       1,
				CRAPThreshold:  15,
				WorstCRAP:      nil,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("internal/crap/analyze.go", "Analyze", 9.2, float64Ptr(14.1)),
				Current:   makeScore("internal/crap/analyze.go", "Analyze", 12.5, float64Ptr(18.3)),
				CRAPDelta: 3.3,
				GazeCRAPDelta: float64Ptr(4.2),
				Status:    StatusRegression,
			},
			{
				Baseline:  makeScore("internal/crap/report.go", "WriteText", 12.0, float64Ptr(15.0)),
				Current:   makeScore("internal/crap/report.go", "WriteText", 8.0, float64Ptr(10.0)),
				CRAPDelta: -4.0,
				GazeCRAPDelta: float64Ptr(-5.0),
				Status:    StatusImprovement,
			},
			{
				Baseline:  makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				Current:   makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				CRAPDelta: 0.0,
				Status:    StatusUnchanged,
			},
		},
		NewFunctions: []Score{
			makeScore("internal/crap/helper.go", "helperFunc", 12.0, nil),
			makeScore("internal/crap/complex.go", "complexFunc", 42.0, nil),
		},
		RemovedFunctions: []Score{
			makeScore("internal/crap/old.go", "oldFunc", 5.0, nil),
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Improvements:         1,
			NewFunctions:         1,
			NewViolations:        1,
			RemovedFunctions:     1,
			Unchanged:            1,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
			Passed:               false,
		},
	}
}

func TestSC007_WriteComparisonJSON_Structure(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify top-level keys exist.
	requiredKeys := []string{"scores", "new_functions", "removed_functions", "comparison", "summary"}
	for _, key := range requiredKeys {
		if _, ok := output[key]; !ok {
			t.Errorf("missing top-level key %q in JSON output", key)
		}
	}

	// Verify scores have delta fields.
	var scores []map[string]interface{}
	if err := json.Unmarshal(output["scores"], &scores); err != nil {
		t.Fatalf("parsing scores: %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("len(scores) = %d, want 3 (new functions should be in new_functions, not scores)", len(scores))
	}

	// Check the regression score has delta fields.
	regressionScore := scores[0]
	deltaFields := []string{"baseline_crap", "crap_delta", "status"}
	for _, field := range deltaFields {
		if _, ok := regressionScore[field]; !ok {
			t.Errorf("regression score missing field %q", field)
		}
	}

	// Verify new_functions has 2 entries.
	var newFuncs []map[string]interface{}
	if err := json.Unmarshal(output["new_functions"], &newFuncs); err != nil {
		t.Fatalf("parsing new_functions: %v", err)
	}
	if len(newFuncs) != 2 {
		t.Errorf("len(new_functions) = %d, want 2", len(newFuncs))
	}

	// Verify removed_functions has 1 entry.
	var removedFuncs []map[string]interface{}
	if err := json.Unmarshal(output["removed_functions"], &removedFuncs); err != nil {
		t.Fatalf("parsing removed_functions: %v", err)
	}
	if len(removedFuncs) != 1 {
		t.Errorf("len(removed_functions) = %d, want 1", len(removedFuncs))
	}
}

func TestSC007_WriteComparisonJSON_AllStatuses(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	outputStr := buf.String()

	// All 6 status values should appear in the output.
	expectedStatuses := []string{
		string(StatusRegression),
		string(StatusImprovement),
		string(StatusUnchanged),
		string(StatusNew),
		string(StatusNewViolation),
		string(StatusRemoved),
	}
	for _, status := range expectedStatuses {
		if !strings.Contains(outputStr, `"`+status+`"`) {
			t.Errorf("output missing status value %q", status)
		}
	}
}

func TestSC007_WriteComparisonJSON_ComparisonPassed(t *testing.T) {
	// Build a result that passes (no regressions, no violations).
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 8.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 12.0, nil),
				Current:   makeScore("a.go", "Func", 8.0, nil),
				CRAPDelta: -4.0,
				Status:    StatusImprovement,
			},
		},
		Summary: ComparisonSummary{
			Improvements: 1,
			Passed:       true,
			Epsilon:      0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	var comparison ComparisonSummary
	if err := json.Unmarshal(output["comparison"], &comparison); err != nil {
		t.Fatalf("parsing comparison: %v", err)
	}

	if !comparison.Passed {
		t.Error("comparison.passed = false, want true")
	}
}

func TestSC007_BackwardCompat_NoComparisonFields(t *testing.T) {
	// Render a normal crap.Report through WriteJSON (not
	// WriteComparisonJSON) and verify zero comparison fields
	// appear in the output.
	rpt := &Report{
		Scores: []Score{
			makeScore("a.go", "Func", 5.0, nil),
		},
		Summary: Summary{
			TotalFunctions: 1,
			AvgComplexity:  3.0,
			AvgLineCoverage: 80.0,
			AvgCRAP:        5.0,
			CRAPload:       0,
			CRAPThreshold:  15,
		},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, rpt); err != nil {
		t.Fatalf("WriteJSON() error: %v", err)
	}

	outputStr := buf.String()

	// These comparison-specific fields MUST NOT appear in normal
	// WriteJSON output.
	forbiddenFields := []string{
		"baseline_crap",
		"crap_delta",
		"baseline_gaze_crap",
		"gaze_crap_delta",
		"new_functions",
		"removed_functions",
		`"comparison"`,
	}
	for _, field := range forbiddenFields {
		if strings.Contains(outputStr, field) {
			t.Errorf("normal WriteJSON output contains comparison field %q — backward compatibility broken", field)
		}
	}
}

// --- Task 4.4: WriteComparisonText tests ---

func TestSC008_WriteComparisonText_PassHeader(t *testing.T) {
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 8.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 12.0, nil),
				Current:   makeScore("a.go", "Func", 8.0, nil),
				CRAPDelta: -4.0,
				Status:    StatusImprovement,
			},
		},
		Summary: ComparisonSummary{
			Improvements: 1,
			Passed:       true,
			Epsilon:      0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	if !strings.Contains(buf.String(), "PASS") {
		t.Error("output missing PASS header")
	}
}

func TestSC008_WriteComparisonText_FailHeader(t *testing.T) {
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 12.5, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 9.0, nil),
				Current:   makeScore("a.go", "Func", 12.5, nil),
				CRAPDelta: 3.5,
				Status:    StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions: 1,
			Passed:      false,
			Epsilon:     0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	if !strings.Contains(buf.String(), "FAIL") {
		t.Error("output missing FAIL header")
	}
}

func TestSC008_WriteComparisonText_EmptySections(t *testing.T) {
	// All unchanged — no regressions, improvements, new, or removed
	// sections should appear.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 5.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 5.0, nil),
				Current:   makeScore("a.go", "Func", 5.0, nil),
				CRAPDelta: 0.0,
				Status:    StatusUnchanged,
			},
		},
		Summary: ComparisonSummary{
			Unchanged: 1,
			Passed:    true,
			Epsilon:   0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Regressions:") {
		t.Error("empty regressions section should be omitted")
	}
	if strings.Contains(output, "Improvements:") {
		t.Error("empty improvements section should be omitted")
	}
	if strings.Contains(output, "New Functions") {
		t.Error("empty new functions section should be omitted")
	}
	if strings.Contains(output, "Removed Functions:") {
		t.Error("empty removed functions section should be omitted")
	}
}

func TestSC008_WriteComparisonText_RegressionTable(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	// Verify regression table appears with the regressed function.
	if !strings.Contains(output, "Regressions:") {
		t.Error("output missing 'Regressions:' section")
	}
	if !strings.Contains(output, "Analyze") {
		t.Error("regression table missing function name 'Analyze'")
	}
	// Verify delta format (+3.3).
	if !strings.Contains(output, "+3.3") {
		t.Error("regression table missing delta value '+3.3'")
	}
	// Verify improvements section also appears.
	if !strings.Contains(output, "Improvements:") {
		t.Error("output missing 'Improvements:' section")
	}
	// Verify new function violation appears.
	if !strings.Contains(output, "VIOLATION") {
		t.Error("output missing VIOLATION marker for new function above threshold")
	}
	// Verify removed functions section.
	if !strings.Contains(output, "Removed Functions:") {
		t.Error("output missing 'Removed Functions:' section")
	}
}
