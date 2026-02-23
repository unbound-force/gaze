package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/taxonomy"
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
		pkgPath: "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
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
		pkgPath: "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
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
		pkgPath:  "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
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
		pkgPath:  "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
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
		pkgPath:           "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
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
		pkgPath: "github.com/unbound-force/gaze/nonexistent/package",
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

// ---------------------------------------------------------------------------
// runDocscan tests
// ---------------------------------------------------------------------------

func TestRunDocscan_OutputsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runDocscan(docscanParams{
		pkgPath: ".",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("runDocscan() error: %v", err)
	}

	// Output should be a JSON array.
	var docs interface{}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &docs); jsonErr != nil {
		t.Errorf("docscan output is not valid JSON: %v\noutput:\n%s",
			jsonErr, stdout.String())
	}
}

func TestRunDocscan_EmptyPkg(t *testing.T) {
	// An empty/non-existent package path should not cause a crash;
	// docscan uses CWD for the repo root.
	var stdout, stderr bytes.Buffer
	err := runDocscan(docscanParams{
		pkgPath: ".",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("runDocscan() error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// runAnalyze --classify tests
// ---------------------------------------------------------------------------

func TestRunAnalyze_ClassifyFlag_TextFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath:  "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
		format:   "text",
		classify: true,
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err != nil {
		t.Fatalf("runAnalyze --classify error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "CLASSIFICATION") {
		t.Errorf("expected CLASSIFICATION column in text output, got:\n%s", output)
	}
}

func TestRunAnalyze_ClassifyFlag_JSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath:  "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
		format:   "json",
		classify: true,
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err != nil {
		t.Fatalf("runAnalyze --classify --format=json error: %v", err)
	}

	// Output should be valid JSON with classification fields.
	var parsed map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		t.Errorf("output is not valid JSON: %v\noutput:\n%s", err, stdout.String())
	}
}

func TestRunAnalyze_VerboseImpliesClassify(t *testing.T) {
	// --verbose without --classify should still produce classification output.
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		pkgPath: "github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
		format:  "text",
		verbose: true,
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("runAnalyze --verbose error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "CLASSIFICATION") {
		t.Errorf("--verbose should imply --classify, expected CLASSIFICATION column, got:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// loadConfig threshold override tests (REQUIRED 6 / RECOMMENDED 10)
// ---------------------------------------------------------------------------

// TestLoadConfig_ContractualThresholdOverride verifies that a positive
// contractual threshold value is applied to the config.
func TestLoadConfig_ContractualThresholdOverride(t *testing.T) {
	cfg, err := loadConfig("", 90, -1)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	if cfg.Classification.Thresholds.Contractual != 90 {
		t.Errorf("contractual threshold = %d, want 90",
			cfg.Classification.Thresholds.Contractual)
	}
	// Incidental should remain at the default (50) since we passed -1.
	if cfg.Classification.Thresholds.Incidental != 50 {
		t.Errorf("incidental threshold = %d, want 50 (default)",
			cfg.Classification.Thresholds.Incidental)
	}
}

// TestLoadConfig_IncidentalThresholdOverride verifies that a positive
// incidental threshold value is applied to the config.
func TestLoadConfig_IncidentalThresholdOverride(t *testing.T) {
	cfg, err := loadConfig("", -1, 30)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	// Contractual should remain at the default (80) since we passed -1.
	if cfg.Classification.Thresholds.Contractual != 80 {
		t.Errorf("contractual threshold = %d, want 80 (default)",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 30 {
		t.Errorf("incidental threshold = %d, want 30",
			cfg.Classification.Thresholds.Incidental)
	}
}

// TestLoadConfig_BothThresholdsOverride verifies that both thresholds
// can be overridden simultaneously.
func TestLoadConfig_BothThresholdsOverride(t *testing.T) {
	cfg, err := loadConfig("", 95, 35)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	if cfg.Classification.Thresholds.Contractual != 95 {
		t.Errorf("contractual threshold = %d, want 95",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 35 {
		t.Errorf("incidental threshold = %d, want 35",
			cfg.Classification.Thresholds.Incidental)
	}
}

// TestLoadConfig_NoOverride verifies that -1 sentinel leaves
// thresholds at their config/default values.
func TestLoadConfig_NoOverride(t *testing.T) {
	cfg, err := loadConfig("", -1, -1)
	if err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	// Should be the defaults from DefaultConfig().
	if cfg.Classification.Thresholds.Contractual != 80 {
		t.Errorf("contractual threshold = %d, want 80 (default)",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 50 {
		t.Errorf("incidental threshold = %d, want 50 (default)",
			cfg.Classification.Thresholds.Incidental)
	}
}

// TestLoadConfig_YAMLInvertedThresholdsRejected verifies that a .gaze.yaml
// file with inverted thresholds (contractual <= incidental) is rejected
// even when no CLI flags are provided. This distinguishes the YAML-source
// error from the CLI-source error tested below.
func TestLoadConfig_YAMLInvertedThresholdsRejected(t *testing.T) {
	dir := t.TempDir()
	cfgPath := dir + "/.gaze.yaml"
	content := []byte(`classification:
  thresholds:
    contractual: 50
    incidental: 60
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}

	_, err := loadConfig(cfgPath, -1, -1)
	if err == nil {
		t.Fatal("expected error for inverted YAML thresholds, got nil")
	}
	// Error should reference the config file path, not CLI flags.
	if !strings.Contains(err.Error(), "config file") {
		t.Errorf("error should mention 'config file', got: %s", err)
	}
}

// TestLoadConfig_ZeroThresholdRejected verifies that a threshold of 0
// is rejected with an error (prevents degenerate all-contractual state).
func TestLoadConfig_ZeroThresholdRejected(t *testing.T) {
	_, err := loadConfig("", 0, -1)
	if err == nil {
		t.Fatal("expected error for contractual-threshold=0, got nil")
	}
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "[1, 99]") {
		t.Errorf("unexpected error message: %s", err)
	}
}

// TestLoadConfig_InvertedThresholdsRejected verifies that contractual <= incidental
// is rejected with an error.
func TestLoadConfig_InvertedThresholdsRejected(t *testing.T) {
	// contractual=40 < incidental=60 — invalid.
	_, err := loadConfig("", 40, 60)
	if err == nil {
		t.Fatal("expected error for contractual=40 < incidental=60, got nil")
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

// ---------------------------------------------------------------------------
// runQuality tests (T052)
// ---------------------------------------------------------------------------

func TestRunQuality_InvalidFormat(t *testing.T) {
	err := runQuality(qualityParams{
		pkgPath: "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
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

func TestRunQuality_TextFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		pkgPath: "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		format:  "text",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	if out == "" {
		t.Error("expected non-empty text output")
	}
}

func TestRunQuality_JSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		pkgPath: "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		format:  "json",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify valid JSON.
	var output map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if _, ok := output["quality_reports"]; !ok {
		t.Error("expected 'quality_reports' key in JSON output")
	}
	if _, ok := output["quality_summary"]; !ok {
		t.Error("expected 'quality_summary' key in JSON output")
	}
}

func TestRunQuality_TargetFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		pkgPath:    "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		format:     "text",
		targetFunc: "Add",
		stdout:     &stdout,
		stderr:     &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunQuality_ThresholdPass(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Use maxOverSpecification threshold only — set high enough
	// to always pass. Contract coverage is non-zero but varies
	// with mapping improvements (TODO #6), so coverage thresholds
	// are not yet stable enough for CI enforcement.
	err := runQuality(qualityParams{
		pkgPath:              "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		format:               "text",
		maxOverSpecification: 100, // very high — should pass
		stdout:               &stdout,
		stderr:               &stderr,
	})
	if err != nil {
		t.Fatalf("expected threshold to pass, got: %v", err)
	}
}

func TestRunQuality_ThresholdFail(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		pkgPath:             "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		format:              "text",
		minContractCoverage: 100, // strict — contract coverage is below 100%
		stdout:              &stdout,
		stderr:              &stderr,
	})
	// With minContractCoverage=100%, the threshold should fail
	// because current SSA mapping produces <100% contract coverage.
	// If all tests somehow achieve 100% in the future, this test
	// should be updated to use a stricter fixture.
	if err == nil {
		t.Error("expected threshold failure with minContractCoverage=100%%")
	}
}

func TestRunQuality_BadPackage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		pkgPath: "github.com/nonexistent/package",
		format:  "text",
		stdout:  &stdout,
		stderr:  &stderr,
	})
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
}

// ---------------------------------------------------------------------------
// checkQualityThresholds tests (SC-005)
// ---------------------------------------------------------------------------

func TestSC005_CIThresholds(t *testing.T) {
	// SC-005: CI threshold enforcement correctly exits non-zero
	// when violated, across 10+ scenarios.

	reports := []taxonomy.QualityReport{
		{
			TestFunction:      "TestA",
			ContractCoverage:  taxonomy.ContractCoverage{Percentage: 80, CoveredCount: 4, TotalContractual: 5},
			OverSpecification: taxonomy.OverSpecificationScore{Count: 1},
		},
		{
			TestFunction:      "TestB",
			ContractCoverage:  taxonomy.ContractCoverage{Percentage: 60, CoveredCount: 3, TotalContractual: 5},
			OverSpecification: taxonomy.OverSpecificationScore{Count: 3},
		},
		{
			TestFunction:      "TestC",
			ContractCoverage:  taxonomy.ContractCoverage{Percentage: 100, CoveredCount: 5, TotalContractual: 5},
			OverSpecification: taxonomy.OverSpecificationScore{Count: 0},
		},
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:              3,
		AverageContractCoverage: 80,
		TotalOverSpecifications: 4,
	}

	tests := []struct {
		name                 string
		minContractCoverage  int
		maxOverSpecification int
		wantErr              bool
		errContains          string
	}{
		{name: "no_thresholds", wantErr: false},
		{name: "coverage_all_pass", minContractCoverage: 50, wantErr: false},
		{name: "coverage_one_fail", minContractCoverage: 70, wantErr: true, errContains: "TestB"},
		{name: "coverage_two_fail", minContractCoverage: 90, wantErr: true, errContains: "TestA"},
		{name: "coverage_strict", minContractCoverage: 100, wantErr: true, errContains: "TestA"},
		{name: "overspec_all_pass", maxOverSpecification: 5, wantErr: false},
		{name: "overspec_one_fail", maxOverSpecification: 2, wantErr: true, errContains: "TestB"},
		{name: "both_pass", minContractCoverage: 50, maxOverSpecification: 5, wantErr: false},
		{name: "coverage_pass_overspec_fail", minContractCoverage: 50, maxOverSpecification: 2, wantErr: true, errContains: "over-specification"},
		{name: "coverage_fail_overspec_pass", minContractCoverage: 90, maxOverSpecification: 5, wantErr: true, errContains: "contract coverage"},
		{name: "both_fail", minContractCoverage: 90, maxOverSpecification: 2, wantErr: true},
		{name: "zero_coverage_disabled", minContractCoverage: 0, maxOverSpecification: 2, wantErr: true, errContains: "over-specification"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			p := qualityParams{
				minContractCoverage:  tt.minContractCoverage,
				maxOverSpecification: tt.maxOverSpecification,
				stderr:               &stderr,
			}
			err := checkQualityThresholds(p, reports, summary)

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got: %v", tt.errContains, err)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// runSelfCheck tests (T055)
// ---------------------------------------------------------------------------

func TestRunSelfCheck_InvalidFormat(t *testing.T) {
	err := runSelfCheck(selfCheckParams{
		format: "xml",
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), `invalid format "xml"`) {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestRunSelfCheck_TextFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping self-check in short mode")
	}
	var stdout, stderr bytes.Buffer
	err := runSelfCheck(selfCheckParams{
		format: "text",
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("self-check text failed: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("expected non-empty text output")
	}
}

func TestRunSelfCheck_JSONFormat(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping self-check in short mode")
	}
	var stdout, stderr bytes.Buffer
	err := runSelfCheck(selfCheckParams{
		format: "json",
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("self-check json failed: %v", err)
	}

	// Verify valid JSON with expected structure.
	var output map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if _, ok := output["scores"]; !ok {
		t.Error("expected 'scores' key in JSON output")
	}
	if _, ok := output["summary"]; !ok {
		t.Error("expected 'summary' key in JSON output")
	}

	// Verify it analyzed functions.
	summary, ok := output["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("expected 'summary' to be an object")
	}
	totalFunctions, ok := summary["total_functions"].(float64)
	if !ok || totalFunctions == 0 {
		t.Errorf("expected non-zero total_functions, got %v", summary["total_functions"])
	}
}
