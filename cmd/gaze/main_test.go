package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jflowers/gaze/internal/crap"
)

// ---------------------------------------------------------------------------
// runAnalyze tests
// ---------------------------------------------------------------------------

func TestRunAnalyze_InvalidFormat(t *testing.T) {
	err := runAnalyze(analyzeParams{
		pkgPath: "./...",
		format:  "yaml",
		stdout:  &bytes.Buffer{},
		stderr:  &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), `invalid format "yaml"`) {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestRunAnalyze_TextFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath: "github.com/jflowers/gaze/internal/analysis/testdata/src/returns",
		format:  "text",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "SingleReturn") {
		t.Errorf("expected output to contain 'SingleReturn', got:\n%s", out)
	}
	if !strings.Contains(out, "ReturnValue") {
		t.Errorf("expected output to contain 'ReturnValue', got:\n%s", out)
	}
}

func TestRunAnalyze_JSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath: "github.com/jflowers/gaze/internal/analysis/testdata/src/returns",
		format:  "json",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output is valid JSON.
	var parsed map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
	if _, ok := parsed["results"]; !ok {
		t.Errorf("JSON output missing 'results' key")
	}
}

func TestRunAnalyze_FunctionFilter(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath:  "github.com/jflowers/gaze/internal/analysis/testdata/src/returns",
		format:   "text",
		function: "SingleReturn",
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "SingleReturn") {
		t.Errorf("expected output to contain 'SingleReturn', got:\n%s", out)
	}
	// Should contain exactly 1 function analyzed.
	if !strings.Contains(out, "1 function(s) analyzed") {
		t.Errorf("expected exactly 1 function analyzed, got:\n%s", out)
	}
}

func TestRunAnalyze_FunctionNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath:  "github.com/jflowers/gaze/internal/analysis/testdata/src/returns",
		format:   "text",
		function: "NonExistentFunc",
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err == nil {
		t.Fatal("expected error for non-existent function")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %s", err)
	}
}

func TestRunAnalyze_IncludeUnexported(t *testing.T) {
	// The returns testdata package only has exported functions,
	// so this just verifies the flag passes through without error.
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath:           "github.com/jflowers/gaze/internal/analysis/testdata/src/returns",
		format:            "text",
		includeUnexported: true,
		stdout:            &stdout,
		stderr:            &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunAnalyze_BadPackage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath: "github.com/jflowers/gaze/nonexistent/package",
		format:  "text",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
}

// ---------------------------------------------------------------------------
// writeCrapReport tests
// ---------------------------------------------------------------------------

func TestWriteCrapReport_JSON(t *testing.T) {
	rpt := &crap.Report{
		Scores: []crap.Score{
			{
				Package:      "pkg",
				Function:     "Foo",
				File:         "foo.go",
				Line:         10,
				Complexity:   5,
				LineCoverage: 80.0,
				CRAP:         5.04,
			},
		},
		Summary: crap.Summary{
			TotalFunctions:  1,
			AvgComplexity:   5.0,
			AvgLineCoverage: 80.0,
			AvgCRAP:         5.04,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	err := writeCrapReport(&buf, "json", rpt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestWriteCrapReport_Text(t *testing.T) {
	rpt := &crap.Report{
		Scores: []crap.Score{
			{
				Package:      "pkg",
				Function:     "Foo",
				File:         "foo.go",
				Line:         10,
				Complexity:   5,
				LineCoverage: 80.0,
				CRAP:         5.04,
			},
		},
		Summary: crap.Summary{
			TotalFunctions:  1,
			AvgComplexity:   5.0,
			AvgLineCoverage: 80.0,
			AvgCRAP:         5.04,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	err := writeCrapReport(&buf, "text", rpt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Foo") {
		t.Errorf("expected text output to contain function name 'Foo', got:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// printCISummary tests
// ---------------------------------------------------------------------------

func TestPrintCISummary_NoThresholds(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 5},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 0, 0)
	if buf.Len() != 0 {
		t.Errorf("expected no output when thresholds are 0, got: %q", buf.String())
	}
}

func TestPrintCISummary_CRAPloadPass(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 3},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 5, 0)
	out := buf.String()
	if !strings.Contains(out, "CRAPload: 3/5 (PASS)") {
		t.Errorf("expected PASS summary, got: %q", out)
	}
}

func TestPrintCISummary_CRAPloadFail(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 10},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 5, 0)
	out := buf.String()
	if !strings.Contains(out, "CRAPload: 10/5 (FAIL)") {
		t.Errorf("expected FAIL summary, got: %q", out)
	}
}

func TestPrintCISummary_GazeCRAPloadPass(t *testing.T) {
	gc := 2
	rpt := &crap.Report{
		Summary: crap.Summary{GazeCRAPload: &gc},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 0, 5)
	out := buf.String()
	if !strings.Contains(out, "GazeCRAPload: 2/5 (PASS)") {
		t.Errorf("expected GazeCRAPload PASS, got: %q", out)
	}
}

func TestPrintCISummary_GazeCRAPloadFail(t *testing.T) {
	gc := 10
	rpt := &crap.Report{
		Summary: crap.Summary{GazeCRAPload: &gc},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 0, 5)
	out := buf.String()
	if !strings.Contains(out, "GazeCRAPload: 10/5 (FAIL)") {
		t.Errorf("expected GazeCRAPload FAIL, got: %q", out)
	}
}

func TestPrintCISummary_BothThresholds(t *testing.T) {
	gc := 2
	rpt := &crap.Report{
		Summary: crap.Summary{
			CRAPload:     3,
			GazeCRAPload: &gc,
		},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 5, 5)
	out := buf.String()
	if !strings.Contains(out, "CRAPload: 3/5 (PASS)") {
		t.Errorf("expected CRAPload PASS in combined output, got: %q", out)
	}
	if !strings.Contains(out, "GazeCRAPload: 2/5 (PASS)") {
		t.Errorf("expected GazeCRAPload PASS in combined output, got: %q", out)
	}
	if !strings.Contains(out, " | ") {
		t.Errorf("expected pipe separator in combined output, got: %q", out)
	}
}

func TestPrintCISummary_GazeCRAPloadNil(t *testing.T) {
	// When GazeCRAPload is nil but maxGazeCrapload > 0, should
	// not print a GazeCRAPload line.
	rpt := &crap.Report{
		Summary: crap.Summary{
			CRAPload:     3,
			GazeCRAPload: nil,
		},
	}
	var buf bytes.Buffer
	printCISummary(&buf, rpt, 5, 5)
	out := buf.String()
	if strings.Contains(out, "GazeCRAPload") {
		t.Errorf("should not print GazeCRAPload when nil, got: %q", out)
	}
}

// ---------------------------------------------------------------------------
// checkCIThresholds tests
// ---------------------------------------------------------------------------

func TestCheckCIThresholds_AllPass(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 3},
	}
	err := checkCIThresholds(rpt, 5, 0)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckCIThresholds_NoLimits(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 100},
	}
	err := checkCIThresholds(rpt, 0, 0)
	if err != nil {
		t.Errorf("expected no error with no limits, got: %v", err)
	}
}

func TestCheckCIThresholds_CRAPloadExceeded(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 10},
	}
	err := checkCIThresholds(rpt, 5, 0)
	if err == nil {
		t.Fatal("expected error when CRAPload exceeds max")
	}
	if !strings.Contains(err.Error(), "CRAPload 10 exceeds maximum 5") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestCheckCIThresholds_GazeCRAPloadExceeded(t *testing.T) {
	gc := 10
	rpt := &crap.Report{
		Summary: crap.Summary{GazeCRAPload: &gc},
	}
	err := checkCIThresholds(rpt, 0, 5)
	if err == nil {
		t.Fatal("expected error when GazeCRAPload exceeds max")
	}
	if !strings.Contains(err.Error(), "GazeCRAPload 10 exceeds maximum 5") {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestCheckCIThresholds_GazeCRAPloadNil(t *testing.T) {
	rpt := &crap.Report{
		Summary: crap.Summary{GazeCRAPload: nil},
	}
	// Should not error even with maxGazeCrapload set, because
	// GazeCRAPload is nil (not computed).
	err := checkCIThresholds(rpt, 0, 5)
	if err != nil {
		t.Errorf("expected no error when GazeCRAPload is nil, got: %v", err)
	}
}

func TestCheckCIThresholds_CRAPloadAtBoundary(t *testing.T) {
	// CRAPload == maxCrapload should NOT trigger an error
	// (the check is strictly greater than).
	rpt := &crap.Report{
		Summary: crap.Summary{CRAPload: 5},
	}
	err := checkCIThresholds(rpt, 5, 0)
	if err != nil {
		t.Errorf("expected no error when CRAPload equals max, got: %v", err)
	}
}

func TestCheckCIThresholds_BothExceeded(t *testing.T) {
	gc := 10
	rpt := &crap.Report{
		Summary: crap.Summary{
			CRAPload:     10,
			GazeCRAPload: &gc,
		},
	}
	err := checkCIThresholds(rpt, 5, 5)
	if err == nil {
		t.Fatal("expected error when both thresholds exceeded")
	}
	// CRAPload check runs first, so the error should mention CRAPload.
	if !strings.Contains(err.Error(), "CRAPload") {
		t.Errorf("expected CRAPload error (checked first), got: %s", err)
	}
}

// ---------------------------------------------------------------------------
// runCrap tests (format validation only — full integration requires
// go test -coverprofile which is slow and tested elsewhere)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// schema command tests
// ---------------------------------------------------------------------------

func TestSchemaCmd_OutputsValidJSON(t *testing.T) {
	cmd := newSchemaCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("schema command failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Errorf("schema output is not valid JSON: %v", err)
	}
}

func TestSchemaCmd_ContainsSchemaFields(t *testing.T) {
	cmd := newSchemaCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	for _, field := range []string{
		`"$schema"`, `"title"`, `"AnalysisResult"`,
		`"FunctionTarget"`, `"SideEffect"`, `"Metadata"`,
	} {
		if !strings.Contains(output, field) {
			t.Errorf("schema output missing %s", field)
		}
	}
}

func TestRunCrap_InvalidFormat(t *testing.T) {
	err := runCrap(crapParams{
		patterns: []string{"./..."},
		format:   "xml",
		stdout:   &bytes.Buffer{},
		stderr:   &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), `invalid format "xml"`) {
		t.Errorf("unexpected error message: %s", err)
	}
}
