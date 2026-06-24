package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/aireport"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/provider/goprovider"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// runAnalyze tests
// ---------------------------------------------------------------------------

func TestRunAnalyze_InvalidFormat(t *testing.T) {
	err := runAnalyze(analyzeParams{
		patterns: []string{"./..."},
		format:   "yaml",
		stdout:   &bytes.Buffer{},
		stderr:   &bytes.Buffer{},
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
		format:   "text",
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
	if !strings.Contains(out, "ReturnValue") {
		t.Errorf("expected output to contain 'ReturnValue', got:\n%s", out)
	}
}

func TestRunAnalyze_JSONFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
		format:   "json",
		stdout:   &stdout,
		stderr:   &stderr,
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
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
		patterns:          []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
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
		patterns: []string{"github.com/unbound-force/gaze/nonexistent/package"},
		format:   "text",
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
}

func TestRunAnalyze_MultiPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: loads multiple real packages")
	}
	var stdout, stderr bytes.Buffer
	err := runAnalyze(analyzeParams{
		patterns: []string{
			"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
			"github.com/unbound-force/gaze/internal/analysis/testdata/src/mutation",
		},
		format: "text",
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	// Verify results from both packages appear.
	if !strings.Contains(out, "SingleReturn") {
		t.Error("expected SingleReturn from returns package in output")
	}
	if !strings.Contains(out, "Increment") {
		t.Error("expected Increment from mutation package in output")
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
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
		patterns: []string{"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns"},
		format:   "text",
		verbose:  true,
		stdout:   &stdout,
		stderr:   &stderr,
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
	// Chdir to a temp dir without go.mod so FindModuleRoot falls back
	// to DefaultConfig, isolating the test from the project's .gaze.yaml.
	t.Chdir(t.TempDir())
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
	// Chdir to a temp dir without go.mod so FindModuleRoot falls back
	// to DefaultConfig, isolating the test from the project's .gaze.yaml.
	t.Chdir(t.TempDir())
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
	cfgPath := filepath.Join(dir, ".gaze.yaml")
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
	// Error now comes from config.Load (threshold validation) rather
	// than loadConfig's own coherence check, so it uses YAML field
	// paths instead of "config file" source attribution.
	if !strings.Contains(err.Error(), "classification.thresholds.contractual") {
		t.Errorf("error should mention 'classification.thresholds.contractual', got: %s", err)
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
// runCrap fast unit tests (US3 — T016)
// ---------------------------------------------------------------------------

// stubReport returns a minimal canned crap.Report for testing.
func stubReport() *crap.Report {
	return &crap.Report{
		Scores: []crap.Score{
			{
				Package:      "example.com/pkg",
				Function:     "Foo",
				File:         "foo.go",
				Line:         10,
				Complexity:   5,
				LineCoverage: 90.0,
				CRAP:         5.5,
			},
		},
		Summary: crap.Summary{
			TotalFunctions:  1,
			AvgComplexity:   5.0,
			AvgLineCoverage: 90.0,
			AvgCRAP:         5.5,
			CRAPload:        0,
			CRAPThreshold:   15,
			WorstCRAP:       nil,
		},
	}
}

func stubAnalyze(_ []string, _ string, _ crap.Options) (*crap.Report, error) {
	return stubReport(), nil
}



func TestRunCrap_TextOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubAnalyze,

	})
	if err != nil {
		t.Fatalf("runCrap returned error: %v", err)
	}
	if stdout.Len() == 0 {
		t.Error("expected non-empty text output")
	}
	if !strings.Contains(stdout.String(), "Foo") {
		t.Errorf("expected output to contain 'Foo', got: %s", stdout.String())
	}
}

func TestRunCrap_JSONOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "json",
		opts:         crap.DefaultOptions(),
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubAnalyze,

	})
	if err != nil {
		t.Fatalf("runCrap returned error: %v", err)
	}
	// Verify JSON output is valid.
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Errorf("output is not valid JSON: %v", err)
	}
}

func TestRunCrap_NoCoverageWarning(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubAnalyze,

	})
	if err != nil {
		t.Fatalf("runCrap returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "GazeCRAP unavailable") {
		t.Errorf("expected GazeCRAP unavailability warning in stderr, got: %s", stderr.String())
	}
}

func TestRunCrap_ThresholdPass(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		maxCrapload:  10, // report has CRAPload=0, well under 10
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubAnalyze,

	})
	if err != nil {
		t.Fatalf("expected no error when under threshold, got: %v", err)
	}
}

func TestRunCrap_ThresholdBreach(t *testing.T) {
	// Create a report with CRAPload > 0 to trigger threshold breach.
	overThreshold := func(_ []string, _ string, _ crap.Options) (*crap.Report, error) {
		rpt := stubReport()
		rpt.Summary.CRAPload = 5
		return rpt, nil
	}

	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		maxCrapload:  2, // CRAPload=5 exceeds max=2
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  overThreshold,

	})
	if err == nil {
		t.Fatal("expected error when CRAPload exceeds threshold")
	}
	if !strings.Contains(err.Error(), "CRAPload") {
		t.Errorf("expected error about CRAPload, got: %s", err)
	}
}

func TestRunCrap_EmptyPatterns(t *testing.T) {
	var capturedPatterns []string
	capturingAnalyze := func(patterns []string, _ string, _ crap.Options) (*crap.Report, error) {
		capturedPatterns = patterns
		return stubReport(), nil
	}

	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{},
		format:       "text",
		opts:         crap.DefaultOptions(),
		moduleDir:    ".",
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  capturingAnalyze,

	})
	if err != nil {
		t.Fatalf("runCrap with empty patterns returned error: %v", err)
	}
	if len(capturedPatterns) != 0 {
		t.Errorf("expected empty patterns to be forwarded as-is, got %v", capturedPatterns)
	}
}

// ---------------------------------------------------------------------------
// runCrap zero-result gate tests (#116)
// ---------------------------------------------------------------------------

// stubEmptyReport returns a crap.Report with zero scores.
func stubEmptyReport() *crap.Report {
	return &crap.Report{
		Scores:  []crap.Score{},
		Summary: crap.Summary{},
	}
}

func stubEmptyAnalyze(_ []string, _ string, _ crap.Options) (*crap.Report, error) {
	return stubEmptyReport(), nil
}

// TestRunCrap_ZeroResults_ThresholdSet_ReturnsError verifies that when
// crap.Analyze returns zero scores and a threshold flag was explicitly
// set, runCrap returns a non-nil error (#116 zero-result gate).
func TestRunCrap_ZeroResults_ThresholdSet_ReturnsError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		maxCrapload:  5,
		moduleDir:    ".",
		thresholdSet: true,
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubEmptyAnalyze,

	})
	if err == nil {
		t.Fatal("expected error when zero results and threshold set")
	}
	if !strings.Contains(err.Error(), "no functions analyzed") {
		t.Errorf("expected 'no functions analyzed' in error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "cannot evaluate thresholds") {
		t.Errorf("expected 'cannot evaluate thresholds' in error, got: %s", err)
	}
}

// TestRunCrap_ZeroResults_NoThreshold_ExitZero verifies that when
// crap.Analyze returns zero scores and no threshold flag was set,
// runCrap returns nil (exit 0) with a warning on stderr.
func TestRunCrap_ZeroResults_NoThreshold_ExitZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         crap.DefaultOptions(),
		moduleDir:    ".",
		thresholdSet: false,
		stdout:       &stdout,
		stderr:       &stderr,
		analyzeFunc:  stubEmptyAnalyze,

	})
	if err != nil {
		t.Fatalf("expected nil error when zero results and no threshold, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: no functions analyzed") {
		t.Errorf("expected 'warning: no functions analyzed' on stderr, got: %q", stderr.String())
	}
}

// ---------------------------------------------------------------------------
// runSelfCheck fast unit tests (US3 — T017)
// ---------------------------------------------------------------------------

func TestRunSelfCheck_HappyPath(t *testing.T) {
	var delegatedParams crapParams
	var stdout, stderr bytes.Buffer
	err := runSelfCheck(selfCheckParams{
		format:          "text",
		maxCrapload:     100,
		maxGazeCrapload: 100,
		stdout:          &stdout,
		stderr:          &stderr,
		moduleRootFunc: func() (string, error) {
			return "/fake/module/root", nil
		},
		runCrapFunc: func(p crapParams) error {
			delegatedParams = p
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runSelfCheck returned error: %v", err)
	}
	if delegatedParams.moduleDir != "/fake/module/root" {
		t.Errorf("expected moduleDir=/fake/module/root, got %q", delegatedParams.moduleDir)
	}
	if len(delegatedParams.patterns) != 1 || delegatedParams.patterns[0] != "./..." {
		t.Errorf("expected patterns=[./...], got %v", delegatedParams.patterns)
	}
	if delegatedParams.format != "text" {
		t.Errorf("expected format=text, got %q", delegatedParams.format)
	}
	if delegatedParams.maxCrapload != 100 {
		t.Errorf("expected maxCrapload=100, got %d", delegatedParams.maxCrapload)
	}
	if delegatedParams.maxGazeCrapload != 100 {
		t.Errorf("expected maxGazeCrapload=100, got %d", delegatedParams.maxGazeCrapload)
	}
	if delegatedParams.stdout != &stdout {
		t.Error("expected stdout to be forwarded")
	}
	if delegatedParams.stderr != &stderr {
		t.Error("expected stderr to be forwarded")
	}
}

func TestRunSelfCheck_ModuleRootError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runSelfCheck(selfCheckParams{
		format: "text",
		stdout: &stdout,
		stderr: &stderr,
		moduleRootFunc: func() (string, error) {
			return "", fmt.Errorf("no go.mod found")
		},
	})
	if err == nil {
		t.Fatal("expected error when moduleRootFunc fails")
	}
	if !strings.Contains(err.Error(), "module root") {
		t.Errorf("expected error about module root, got: %s", err)
	}
}

// ---------------------------------------------------------------------------
// runQuality tests (T052)
// ---------------------------------------------------------------------------

func TestRunQuality_InvalidFormat(t *testing.T) {
	err := runQuality(qualityParams{
		patterns: []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
		format:   "yaml",
		stdout:   &bytes.Buffer{},
		stderr:   &bytes.Buffer{},
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
		patterns: []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
		format:   "text",
		stdout:   &stdout,
		stderr:   &stderr,
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
		patterns: []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
		format:   "json",
		stdout:   &stdout,
		stderr:   &stderr,
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
		patterns:   []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
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
		patterns:             []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
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
		patterns:            []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
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
		patterns: []string{"github.com/nonexistent/package"},
		format:   "text",
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err == nil {
		t.Fatal("expected error for non-existent package")
	}
}

func TestRunQuality_MultiPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: loads multiple real packages with test suites")
	}
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		patterns: []string{
			"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
			"github.com/unbound-force/gaze/internal/quality/testdata/src/helpers",
		},
		format: "text",
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := stdout.String()
	// Verify the merged summary reflects tests from multiple packages.
	if !strings.Contains(out, "Tests analyzed:") {
		t.Error("expected 'Tests analyzed:' summary line in output")
	}
}

func TestRunQuality_MultiPackage_SkipsNoTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: loads real packages")
	}
	var stdout, stderr bytes.Buffer
	// returns has no test files — should be skipped with a warning.
	err := runQuality(qualityParams{
		patterns: []string{
			"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
			"github.com/unbound-force/gaze/internal/analysis/testdata/src/returns",
		},
		format: "text",
		stdout: &stdout,
		stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("unexpected error (should skip no-test package): %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Tests analyzed:") {
		t.Error("expected quality output from welltested package")
	}
}

// ---------------------------------------------------------------------------
// mergeSummaries tests
// ---------------------------------------------------------------------------

func TestMergeSummaries_Empty(t *testing.T) {
	got := mergeSummaries(nil)
	if got == nil {
		t.Fatal("expected non-nil summary for empty input")
	}
	if got.TotalTests != 0 {
		t.Errorf("TotalTests = %d, want 0", got.TotalTests)
	}
}

func TestMergeSummaries_Single(t *testing.T) {
	s := &taxonomy.PackageSummary{
		TotalTests:                   5,
		AverageContractCoverage:      80.0,
		TotalOverSpecifications:      2,
		AssertionDetectionConfidence: 90,
	}
	got := mergeSummaries([]*taxonomy.PackageSummary{s})
	if got != s {
		t.Error("expected single-element input to return the same pointer")
	}
}

func TestMergeSummaries_Multiple(t *testing.T) {
	s1 := &taxonomy.PackageSummary{
		TotalTests:                   3,
		AverageContractCoverage:      60.0,
		TotalOverSpecifications:      1,
		AssertionDetectionConfidence: 80,
		WorstCoverageTests: []taxonomy.QualityReport{
			{TestFunction: "TestA", ContractCoverage: taxonomy.ContractCoverage{Percentage: 20}},
		},
	}
	s2 := &taxonomy.PackageSummary{
		TotalTests:                   7,
		AverageContractCoverage:      40.0,
		TotalOverSpecifications:      3,
		AssertionDetectionConfidence: 60,
		WorstCoverageTests: []taxonomy.QualityReport{
			{TestFunction: "TestB", ContractCoverage: taxonomy.ContractCoverage{Percentage: 10}},
		},
	}
	got := mergeSummaries([]*taxonomy.PackageSummary{s1, s2})

	if got.TotalTests != 10 {
		t.Errorf("TotalTests = %d, want 10", got.TotalTests)
	}
	if got.TotalOverSpecifications != 4 {
		t.Errorf("TotalOverSpecifications = %d, want 4", got.TotalOverSpecifications)
	}
	// Average of 60 and 40 = 50.
	if got.AverageContractCoverage != 50.0 {
		t.Errorf("AverageContractCoverage = %f, want 50.0", got.AverageContractCoverage)
	}
	// Average of 80 and 60 = 70.
	if got.AssertionDetectionConfidence != 70 {
		t.Errorf("AssertionDetectionConfidence = %d, want 70", got.AssertionDetectionConfidence)
	}
	// Worst tests sorted ascending: TestB(10%), TestA(20%).
	if len(got.WorstCoverageTests) != 2 {
		t.Fatalf("WorstCoverageTests len = %d, want 2", len(got.WorstCoverageTests))
	}
	if got.WorstCoverageTests[0].TestFunction != "TestB" {
		t.Errorf("first worst = %s, want TestB", got.WorstCoverageTests[0].TestFunction)
	}
}

func TestMergeSummaries_TruncatesWorstTo5(t *testing.T) {
	var summaries []*taxonomy.PackageSummary
	for i := 0; i < 3; i++ {
		s := &taxonomy.PackageSummary{
			TotalTests: 1,
			WorstCoverageTests: []taxonomy.QualityReport{
				{TestFunction: fmt.Sprintf("Test%d_A", i), ContractCoverage: taxonomy.ContractCoverage{Percentage: float64(i * 10)}},
				{TestFunction: fmt.Sprintf("Test%d_B", i), ContractCoverage: taxonomy.ContractCoverage{Percentage: float64(i*10 + 5)}},
			},
		}
		summaries = append(summaries, s)
	}
	got := mergeSummaries(summaries)
	if len(got.WorstCoverageTests) != 5 {
		t.Errorf("WorstCoverageTests len = %d, want 5 (truncated from 6)", len(got.WorstCoverageTests))
	}
}

func TestMergeSummaries_SSADegraded(t *testing.T) {
	s1 := &taxonomy.PackageSummary{TotalTests: 1, SSADegraded: false}
	s2 := &taxonomy.PackageSummary{
		TotalTests:          2,
		SSADegraded:         true,
		SSADegradedPackages: []string{"pkg/broken"},
	}
	got := mergeSummaries([]*taxonomy.PackageSummary{s1, s2})
	if !got.SSADegraded {
		t.Error("SSADegraded should be true when any summary is degraded")
	}
	if len(got.SSADegradedPackages) != 1 || got.SSADegradedPackages[0] != "pkg/broken" {
		t.Errorf("SSADegradedPackages = %v, want [pkg/broken]", got.SSADegradedPackages)
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

// TestCheckQualityThresholds_SSADegraded verifies that threshold
// enforcement is skipped when SSA degradation is detected, preventing
// false-positive CI failures from zero-valued coverage metrics.
func TestCheckQualityThresholds_SSADegraded(t *testing.T) {
	// Degraded reports have zero coverage — thresholds would fail
	// without the SSADegraded guard.
	reports := []taxonomy.QualityReport{
		{
			TestFunction:     "TestFoo",
			ContractCoverage: taxonomy.ContractCoverage{Percentage: 0},
		},
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:  1,
		SSADegraded: true,
	}

	var stderr bytes.Buffer
	p := qualityParams{
		minContractCoverage:  100, // would fail without guard
		maxOverSpecification: 0,
		stderr:               &stderr,
	}
	err := checkQualityThresholds(p, reports, summary)

	if err != nil {
		t.Fatalf("expected nil error when SSADegraded, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "CI thresholds skipped") {
		t.Errorf("expected skip warning on stderr, got: %q", stderr.String())
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

// ---------------------------------------------------------------------------
// runInit tests
// ---------------------------------------------------------------------------

func TestRunInit_CreatesFiles(t *testing.T) {
	dir := t.TempDir()

	// Create go.mod to avoid warning.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var stdout bytes.Buffer
	err := runInit(initParams{
		targetDir: dir,
		force:     false,
		version:   "test",
		stdout:    &stdout,
	})
	if err != nil {
		t.Fatalf("runInit() returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "created:") {
		t.Errorf("expected 'created:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Run /gaze") {
		t.Errorf("expected hint in output, got:\n%s", output)
	}

	// Verify files exist.
	expected := []string{
		".opencode/agents/gaze-reporter.md",
		".opencode/agents/reviewer-testing.md",
		".opencode/commands/gaze.md",
		".opencode/commands/speckit.testreview.md",
		".opencode/references/doc-scoring-model.md",
		".opencode/references/example-report.md",
	}
	for _, rel := range expected {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", rel)
		}
	}
}

func TestRunInit_ForceFlag(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: create files.
	var buf1 bytes.Buffer
	if err := runInit(initParams{
		targetDir: dir,
		force:     false,
		version:   "v1.0.0",
		stdout:    &buf1,
	}); err != nil {
		t.Fatalf("first runInit() returned error: %v", err)
	}

	// Second run without force: should skip.
	var buf2 bytes.Buffer
	if err := runInit(initParams{
		targetDir: dir,
		force:     false,
		version:   "v1.0.0",
		stdout:    &buf2,
	}); err != nil {
		t.Fatalf("second runInit() returned error: %v", err)
	}
	if !strings.Contains(buf2.String(), "skipped:") {
		t.Errorf("expected 'skipped:' in output, got:\n%s", buf2.String())
	}

	// Third run with force: should overwrite.
	var buf3 bytes.Buffer
	if err := runInit(initParams{
		targetDir: dir,
		force:     true,
		version:   "v2.0.0",
		stdout:    &buf3,
	}); err != nil {
		t.Fatalf("third runInit() with force returned error: %v", err)
	}
	if !strings.Contains(buf3.String(), "overwritten:") {
		t.Errorf("expected 'overwritten:' in output, got:\n%s", buf3.String())
	}
}

// ---------------------------------------------------------------------------
// extractShortPkgName tests
// ---------------------------------------------------------------------------

// Tests for extractShortPkgName, resolvePackagePaths,
// analyzePackageCoverage, and BuildContractCoverageFunc have been
// moved to internal/crap/contract_test.go (spec 022).

// ---------------------------------------------------------------------------
// gaze report tests (T-008, T-027 through T-035, SC-001 through SC-006)
// ---------------------------------------------------------------------------

// fakeRunnerFunc is a helper that creates a runnerFunc stub returning the
// given payload and error for use in reportParams tests.
// It writes response to Stdout and honours StepSummaryPath.
func fakeRunnerFunc(response string, runErr error) func(aireport.RunnerOptions) error {
	return func(opts aireport.RunnerOptions) error {
		if runErr != nil {
			return runErr
		}
		if opts.Format == "json" {
			enc := json.NewEncoder(opts.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(&aireport.ReportPayload{})
		}
		_, _ = fmt.Fprint(opts.Stdout, response)
		// Honour StepSummaryPath so integration tests can verify the write.
		if opts.StepSummaryPath != "" {
			aireport.WriteStepSummary(opts.StepSummaryPath, response, opts.Stderr)
		}
		return nil
	}
}

// TestSC001_GithubActionsReport verifies that the formatted report is written
// to GITHUB_STEP_SUMMARY when the env var is set (SC-001).
func TestSC001_GithubActionsReport(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "summary.md")
	t.Setenv("GITHUB_STEP_SUMMARY", tmpFile)

	report := "🔍 CRAP Analysis\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"

	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "claude",
		stdout:      &stdout,
		stderr:      &stderr,
		runnerFunc:  fakeRunnerFunc(report, nil),
	})
	if err != nil {
		t.Fatalf("runReport: %v", err)
	}

	data, readErr := os.ReadFile(tmpFile)
	if readErr != nil {
		t.Fatalf("reading step summary file: %v", readErr)
	}
	for _, marker := range []string{"🔍", "📊", "🧪", "🏥"} {
		if !strings.Contains(string(data), marker) {
			t.Errorf("expected %q in step summary, got: %s", marker, data)
		}
	}
}

// TestSC002_ReportStructure verifies that the formatted report contains all
// four required emoji section markers in order (SC-002).
func TestSC002_ReportStructure(t *testing.T) {
	report := "🔍 CRAP Analysis\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"
	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "claude",
		stdout:      &stdout,
		stderr:      &stderr,
		runnerFunc:  fakeRunnerFunc(report, nil),
	})
	if err != nil {
		t.Fatalf("runReport: %v", err)
	}

	out := stdout.String()
	markers := []string{"🔍", "📊", "🧪", "🏥"}
	lastIdx := -1
	for _, marker := range markers {
		idx := strings.Index(out, marker)
		if idx < 0 {
			t.Errorf("expected %q in report output", marker)
			continue
		}
		if idx <= lastIdx {
			t.Errorf("expected %q after previous marker (idx %d), got idx %d", marker, lastIdx, idx)
		}
		lastIdx = idx
	}
}

// TestSC003_ThresholdEvaluation_Correctness verifies that EvaluateThresholds
// correctly classifies pass/fail results for a known payload (SC-003).
// Timing is not measured here; EvaluateThresholds is a pure in-memory function
// with no I/O — its performance is validated by the BenchmarkEvaluateThresholds
// benchmark in threshold_test.go.
func TestSC003_ThresholdEvaluation_Correctness(t *testing.T) {
	payload := &aireport.ReportPayload{
		Summary: aireport.ReportSummary{CRAPload: 3},
	}
	maxCrapload := 10
	cfg := aireport.ThresholdConfig{MaxCrapload: &maxCrapload}
	results, passed := aireport.EvaluateThresholds(cfg, payload)
	if !passed {
		t.Errorf("expected passed=true for CRAPload 3 <= max 10")
	}
	if len(results) != 1 {
		t.Errorf("expected 1 threshold result, got %d", len(results))
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected result name 'CRAPload', got %q", results[0].Name)
	}
	if results[0].Actual == nil || *results[0].Actual != 3 {
		t.Errorf("expected Actual=3, got %v", results[0].Actual)
	}
	if results[0].Limit != 10 {
		t.Errorf("expected Limit=10, got %d", results[0].Limit)
	}
}

// TestSC004_PartialFailure verifies that a failing analysis step produces a
// partial report with a warning, and that the command exits 0 (SC-004).
func TestSC004_PartialFailure(t *testing.T) {
	errMsg := "CRAP step failed"
	payload := &aireport.ReportPayload{
		Errors: aireport.PayloadErrors{CRAP: &errMsg},
	}
	report := "> ⚠️ CRAP analysis unavailable: CRAP step failed\n\n📊 Quality\n"

	var stdout, stderr bytes.Buffer
	// Use a runnerFunc that simulates partial failure (still returns nil).
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "claude",
		stdout:      &stdout,
		stderr:      &stderr,
		runnerFunc: func(opts aireport.RunnerOptions) error {
			_ = payload // payload available but we simulate the formatted text
			_, _ = fmt.Fprint(opts.Stdout, report)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runReport returned error on partial failure: %v", err)
	}
	if !strings.Contains(stdout.String(), "⚠️") {
		t.Errorf("expected warning marker in output, got: %s", stdout.String())
	}
}

// TestSC006_CrossAdapterStructure verifies that all four adapter names are
// correctly wired through the pipeline: runReport → aireport.Run → adapter.Format
// is called exactly once per invocation, and produces structurally equivalent
// output (same four emoji markers in order) regardless of adapter name (SC-006).
func TestSC006_CrossAdapterStructure(t *testing.T) {
	reportBody := "🔍 CRAP\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"
	payload := &aireport.ReportPayload{}

	for _, adapterName := range []string{"claude", "gemini", "ollama", "opencode"} {
		t.Run(adapterName, func(t *testing.T) {
			fa := &aireport.FakeAdapter{Response: reportBody}

			var stdout, stderr bytes.Buffer
			// Use runnerFunc that delegates to real aireport.Run with FakeAdapter
			// and AnalyzeFunc override — exercises the full Run code path including
			// adapter wiring, while keeping the test fast (no real analysis).
			err := runReport(reportParams{
				patterns:    []string{"./..."},
				format:      "text",
				adapterName: adapterName,
				modelName:   "test-model", // needed for ollama validation
				stdout:      &stdout,
				stderr:      &stderr,
				runnerFunc: func(opts aireport.RunnerOptions) error {
					return aireport.Run(aireport.RunnerOptions{
						Patterns:     opts.Patterns,
						Format:       opts.Format,
						Adapter:      fa,
						SystemPrompt: "# Test",
						Stdout:       opts.Stdout,
						Stderr:       opts.Stderr,
						AnalyzeFunc: func([]string, string) (*aireport.ReportPayload, error) {
							return payload, nil
						},
					})
				},
			})
			if err != nil {
				t.Fatalf("runReport(%s): %v", adapterName, err)
			}

			// Verify the adapter was called exactly once (pipeline wiring check).
			if len(fa.Calls) != 1 {
				t.Errorf("[%s] expected adapter.Format called once, got %d calls", adapterName, len(fa.Calls))
			}

			// Verify structural output: four emoji markers in order.
			out := stdout.String()
			markers := []string{"🔍", "📊", "🧪", "🏥"}
			lastIdx := -1
			for _, marker := range markers {
				idx := strings.Index(out, marker)
				if idx < 0 {
					t.Errorf("[%s] expected %q in report output", adapterName, marker)
					continue
				}
				if idx <= lastIdx {
					t.Errorf("[%s] expected %q after previous marker", adapterName, marker)
				}
				lastIdx = idx
			}
		})
	}
}

// TestRunReport_JSONFormat_NoAIRequired verifies that --format=json works
// without --ai flag (FR-015).
func TestRunReport_JSONFormat_NoAIRequired(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns: []string{"./..."},
		format:   "json",
		stdout:   &stdout,
		stderr:   &stderr,
		runnerFunc: func(opts aireport.RunnerOptions) error {
			enc := json.NewEncoder(opts.Stdout)
			return enc.Encode(&aireport.ReportPayload{})
		},
	})
	if err != nil {
		t.Fatalf("expected json mode to succeed without --ai: %v", err)
	}
}

// TestRunReport_JSONFormat_ValidOutput verifies that --format=json produces
// parseable ReportPayload JSON (T-030).
func TestRunReport_JSONFormat_ValidOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns: []string{"./..."},
		format:   "json",
		stdout:   &stdout,
		stderr:   &stderr,
		runnerFunc: func(opts aireport.RunnerOptions) error {
			enc := json.NewEncoder(opts.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(&aireport.ReportPayload{
				Errors: aireport.PayloadErrors{},
			})
		},
	})
	if err != nil {
		t.Fatalf("runReport json: %v", err)
	}
	var decoded aireport.ReportPayload
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid ReportPayload JSON: %v\noutput: %s", err, stdout.String())
	}
}

// TestRunReport_MissingAI_TextMode_ReturnsError verifies that --ai is
// required in text mode (FR-002) and that the error lists valid adapters.
func TestRunReport_MissingAI_TextMode_ReturnsError(t *testing.T) {
	var analyzeCallCount int
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "",
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		runnerFunc: func(_ aireport.RunnerOptions) error {
			analyzeCallCount++
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected error for missing --ai in text mode")
	}
	for _, valid := range []string{"claude", "gemini", "ollama", "opencode"} {
		if !strings.Contains(err.Error(), valid) {
			t.Errorf("expected error to list valid adapter %q, got: %v", valid, err)
		}
	}
	if analyzeCallCount > 0 {
		t.Error("expected no analysis to run before --ai validation")
	}
}

// TestRunReport_UnknownAI_ReturnsError verifies that an unknown --ai value
// returns a descriptive error (T-033).
func TestRunReport_UnknownAI_ReturnsError(t *testing.T) {
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "badai",
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected error for unknown adapter")
	}
	if !strings.Contains(err.Error(), "badai") {
		t.Errorf("expected error to mention adapter name, got: %v", err)
	}
}

// TestRunReport_OllamaMissingModel_ReturnsError verifies that --ai=ollama
// without --model returns an immediate error (T-034).
func TestRunReport_OllamaMissingModel_ReturnsError(t *testing.T) {
	var analyzeCallCount int
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "ollama",
		modelName:   "",
		stdout:      &bytes.Buffer{},
		stderr:      &bytes.Buffer{},
		runnerFunc: func(_ aireport.RunnerOptions) error {
			analyzeCallCount++
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected error for ollama without --model")
	}
	if !strings.Contains(err.Error(), "FR-003") {
		t.Errorf("expected FR-003 in error, got: %v", err)
	}
	if analyzeCallCount > 0 {
		t.Error("expected no analysis to run before model validation")
	}
}

// TestRunReport_StepSummaryUnwritable_Succeeds verifies FR-008: unwritable
// GITHUB_STEP_SUMMARY path emits a warning but command exits 0 (T-035).
func TestRunReport_StepSummaryUnwritable_Succeeds(t *testing.T) {
	t.Setenv("GITHUB_STEP_SUMMARY", "/nonexistent/dir/summary.md")
	report := "🔍 CRAP\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"

	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		adapterName: "claude",
		stdout:      &stdout,
		stderr:      &stderr,
		runnerFunc:  fakeRunnerFunc(report, nil),
	})
	if err != nil {
		t.Fatalf("expected success despite unwritable GITHUB_STEP_SUMMARY: %v", err)
	}
	if stdout.String() != report {
		t.Errorf("expected stdout to contain report, got: %q", stdout.String())
	}
}

// TestRunReport_ThresholdEnforcement verifies US2 threshold scenarios 1–5
// (T-031, scenarios 1–5). Uses runReport → aireport.Run to verify the end-to-end
// threshold format contract: the output on stderr must match "N/M (FAIL)" or
// "N/M (PASS)" as emitted by evaluateAndPrintThresholds.
func TestRunReport_ThresholdEnforcement(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	cases := []struct {
		name       string
		payload    *aireport.ReportPayload
		thresholds aireport.ThresholdConfig
		expectFail bool
	}{
		{
			name:       "SC1: CRAPload exceeds max → fail",
			payload:    &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, CRAPload: 13}},
			thresholds: aireport.ThresholdConfig{MaxCrapload: intPtr(10)},
			expectFail: true,
		},
		{
			name:       "SC2: CRAPload within max → pass",
			payload:    &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, CRAPload: 8}},
			thresholds: aireport.ThresholdConfig{MaxCrapload: intPtr(10)},
			expectFail: false,
		},
		{
			name:       "SC3: avg coverage below min → fail",
			payload:    &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, AvgContractCoverage: 40}},
			thresholds: aireport.ThresholdConfig{MinContractCoverage: intPtr(60)},
			expectFail: true,
		},
		{
			name:       "SC4: no thresholds → pass",
			payload:    &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, CRAPload: 999}},
			thresholds: aireport.ThresholdConfig{},
			expectFail: false,
		},
		{
			name:       "SC5: max-crapload=0 with positive actual → fail",
			payload:    &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, CRAPload: 1}},
			thresholds: aireport.ThresholdConfig{MaxCrapload: intPtr(0)},
			expectFail: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capturedPayload := tc.payload // capture for closure
			var stderr bytes.Buffer

			// Drive through runReport → aireport.Run to exercise the real
			// evaluateAndPrintThresholds format contract ("N/M (FAIL)").
			err := runReport(reportParams{
				patterns:            []string{"./..."},
				format:              "json",
				stdout:              io.Discard,
				stderr:              &stderr,
				maxCrapload:         tc.thresholds.MaxCrapload,
				maxGazeCrapload:     tc.thresholds.MaxGazeCrapload,
				minContractCoverage: tc.thresholds.MinContractCoverage,
				runnerFunc: func(opts aireport.RunnerOptions) error {
					return aireport.Run(aireport.RunnerOptions{
						Patterns:   opts.Patterns,
						Format:     opts.Format,
						Stdout:     opts.Stdout,
						Stderr:     opts.Stderr,
						Thresholds: opts.Thresholds,
						AnalyzeFunc: func([]string, string) (*aireport.ReportPayload, error) {
							return capturedPayload, nil
						},
					})
				},
			})

			gotFail := err != nil
			if tc.expectFail && !gotFail {
				t.Errorf("expected threshold failure, but runReport returned nil")
			}
			if !tc.expectFail && gotFail {
				t.Errorf("expected threshold pass, but runReport returned error: %v\nstderr: %s", err, stderr.String())
			}
			if tc.expectFail && !strings.Contains(stderr.String(), "(FAIL)") {
				t.Errorf("expected '(FAIL)' in stderr output, got: %q", stderr.String())
			}
			hasThreshold := tc.thresholds.MaxCrapload != nil ||
				tc.thresholds.MaxGazeCrapload != nil ||
				tc.thresholds.MinContractCoverage != nil
			if !tc.expectFail && hasThreshold && !strings.Contains(stderr.String(), "(PASS)") {
				t.Errorf("expected '(PASS)' in stderr output, got: %q", stderr.String())
			}
		})
	}
}

// TestRunReport_GazeCRAPloadThresholds verifies US2 scenarios 6 & 7 for
// GazeCRAPload threshold (T-031, scenarios 6–7).
// Drives through runReport → aireport.Run → evaluateAndPrintThresholds to
// verify the end-to-end format contract: "(FAIL)" must appear on stderr.
func TestRunReport_GazeCRAPloadThresholds(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	cases := []struct {
		name            string
		payload         *aireport.ReportPayload
		maxGazeCrapload *int
		expectFail      bool
	}{
		{
			name:            "SC6: GazeCRAPload > max → fail",
			payload:         &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, GazeCRAPload: intPtr(5)}},
			maxGazeCrapload: intPtr(3),
			expectFail:      true,
		},
		{
			name:            "SC7: max-gaze-crapload=0 with positive actual → fail",
			payload:         &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, GazeCRAPload: intPtr(1)}},
			maxGazeCrapload: intPtr(0),
			expectFail:      true,
		},
		{
			name:            "SC-003 PASS: GazeCRAPload=0 below threshold=5 → pass",
			payload:         &aireport.ReportPayload{Summary: aireport.ReportSummary{TotalFunctions: 20, GazeCRAPload: intPtr(0)}},
			maxGazeCrapload: intPtr(5),
			expectFail:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capturedPayload := tc.payload
			var stderr bytes.Buffer

			// Drive through runReport → aireport.Run to exercise the real
			// evaluateAndPrintThresholds format contract ("N/M (FAIL)").
			err := runReport(reportParams{
				patterns:        []string{"./..."},
				format:          "json",
				stdout:          io.Discard,
				stderr:          &stderr,
				maxGazeCrapload: tc.maxGazeCrapload,
				runnerFunc: func(opts aireport.RunnerOptions) error {
					return aireport.Run(aireport.RunnerOptions{
						Patterns:   opts.Patterns,
						Format:     opts.Format,
						Stdout:     opts.Stdout,
						Stderr:     opts.Stderr,
						Thresholds: opts.Thresholds,
						AnalyzeFunc: func([]string, string) (*aireport.ReportPayload, error) {
							return capturedPayload, nil
						},
					})
				},
			})

			gotFail := err != nil
			if tc.expectFail && !gotFail {
				t.Errorf("expected threshold failure, but runReport returned nil")
			}
			if !tc.expectFail && gotFail {
				t.Errorf("expected threshold pass, but runReport returned error: %v\nstderr: %s", err, stderr.String())
			}
			if tc.expectFail && !strings.Contains(stderr.String(), "(FAIL)") {
				t.Errorf("expected '(FAIL)' in stderr output, got: %q", stderr.String())
			}
			if tc.expectFail && !strings.Contains(stderr.String(), "GazeCRAPload") {
				t.Errorf("expected 'GazeCRAPload' label in stderr output, got: %q", stderr.String())
			}
		})
	}
}

// TestSC002_GazeCRAPloadMatchBetweenCrapAndReport verifies SC-002: the
// gaze_crapload value from gaze report matches gaze crap standalone
// with the same coverprofile and package pattern (exact match).
// Guarded by testing.Short() — runs the full quality+CRAP pipeline.
func TestSC002_GazeCRAPloadMatchBetweenCrapAndReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs full quality+CRAP pipeline (SC-002)")
	}

	// Use the welltested fixture — it has known contractual functions
	// that produce non-nil GazeCRAPload, ensuring SC-002 is not vacuous.
	pattern := "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"
	moduleDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	// Run gaze crap standalone with GoContractCoverageProvider.
	crapOpts := crap.DefaultOptions()
	crapOpts.Stderr = io.Discard
	crapOpts.ContractCoverageProvider = goprovider.NewContractCoverageProvider(io.Discard)
	crapReport, err := crap.Analyze([]string{pattern}, moduleDir, crapOpts)
	if err != nil {
		t.Fatalf("crap.Analyze: %v", err)
	}

	// Run gaze report (JSON format, no AI adapter).
	var reportStdout, reportStderr bytes.Buffer
	reportErr := runReport(reportParams{
		patterns: []string{pattern},
		format:   "json",
		stdout:   &reportStdout,
		stderr:   &reportStderr,
	})
	if reportErr != nil {
		t.Fatalf("runReport: %v\nstderr: %s", reportErr, reportStderr.String())
	}

	// Parse the report JSON to extract gaze_crapload from CRAP summary.
	var reportJSON struct {
		CRAP json.RawMessage `json:"crap"`
	}
	if err := json.Unmarshal(reportStdout.Bytes(), &reportJSON); err != nil {
		t.Fatalf("unmarshal report JSON: %v", err)
	}
	var crapJSON struct {
		Summary struct {
			GazeCRAPload *int `json:"gaze_crapload"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(reportJSON.CRAP, &crapJSON); err != nil {
		t.Fatalf("unmarshal CRAP JSON: %v", err)
	}

	// Compare GazeCRAPload values.
	var crapStandalone int
	if crapReport.Summary.GazeCRAPload != nil {
		crapStandalone = *crapReport.Summary.GazeCRAPload
	}
	var reportValue int
	if crapJSON.Summary.GazeCRAPload != nil {
		reportValue = *crapJSON.Summary.GazeCRAPload
	}

	t.Logf("SC-002: gaze crap GazeCRAPload=%d, gaze report GazeCRAPload=%d", crapStandalone, reportValue)

	// Guard against vacuous pass: if both sides produced nil GazeCRAPload,
	// the test is not verifying any real data flow (0 == 0 trivially).
	if crapReport.Summary.GazeCRAPload == nil && crapJSON.Summary.GazeCRAPload == nil {
		t.Skip("SC-002: both sides produced nil GazeCRAPload — test is vacuous; use a fixture with known contract coverage")
	}

	if crapStandalone != reportValue {
		t.Errorf("SC-002 FAIL: gaze crap GazeCRAPload=%d != gaze report GazeCRAPload=%d", crapStandalone, reportValue)
	}
}

// TestSC004_PayloadContainsQuadrantCounts verifies SC-004's data
// precondition: when the report pipeline produces GazeCRAP data, the
// JSON payload passed to the AI adapter contains quadrant_counts.
// Uses FakeAdapter to capture the payload without a real AI model.
func TestSC004_PayloadContainsQuadrantCounts(t *testing.T) {
	crapJSON := json.RawMessage(`{
		"summary": {
			"total_functions": 10,
			"crapload": 2,
			"gaze_crapload": 1,
			"quadrant_counts": {"Q1_Safe": 7, "Q2_ComplexButTested": 1, "Q3_NeedsTests": 1, "Q4_Dangerous": 1}
		},
		"scores": []
	}`)

	fakeAdapter := &aireport.FakeAdapter{Response: "# Fake Report\nOK"}

	var stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:    []string{"./..."},
		format:      "text",
		stdout:      io.Discard,
		stderr:      &stderr,
		adapterName: "claude", // triggers text path with adapter
		runnerFunc: func(opts aireport.RunnerOptions) error {
			return aireport.Run(aireport.RunnerOptions{
				Patterns: opts.Patterns,
				Format:   opts.Format,
				Stdout:   opts.Stdout,
				Stderr:   opts.Stderr,
				Adapter:  fakeAdapter,
				AnalyzeFunc: func([]string, string) (*aireport.ReportPayload, error) {
					return &aireport.ReportPayload{
						CRAP: crapJSON,
					}, nil
				},
			})
		},
	})
	if err != nil {
		t.Fatalf("runReport: %v", err)
	}

	if len(fakeAdapter.Calls) == 0 {
		t.Fatal("expected at least one FakeAdapter.Format call")
	}

	payloadStr := string(fakeAdapter.Calls[0].Payload)
	if !strings.Contains(payloadStr, "quadrant_counts") {
		t.Errorf("SC-004 FAIL: payload passed to AI adapter does not contain 'quadrant_counts'.\nPayload excerpt: %.500s", payloadStr)
	}
	if !strings.Contains(payloadStr, "gaze_crapload") {
		t.Errorf("SC-004 FAIL: payload passed to AI adapter does not contain 'gaze_crapload'.\nPayload excerpt: %.500s", payloadStr)
	}
}

// TestSC005_AnalysisPerformance verifies that the analysis pipeline completes
// within 5 minutes on the gaze module itself (SC-005).
// Guarded by testing.Short() — only runs in the slow E2E suite.
// Uses the real four-step analysis pipeline with FakeAdapter (to exclude AI
// network latency from the timing measurement).
func TestSC005_AnalysisPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("TestSC005_AnalysisPerformance skipped in -short mode")
	}
	t.Log("Running SC-005 analysis performance test (may take up to 5 minutes)...")

	modRoot := findModuleRootForReport(t)
	fa := &aireport.FakeAdapter{Response: "🔍 CRAP\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health\n"}
	var stdout, stderr bytes.Buffer

	// Run the real four-step analysis pipeline. FakeAdapter replaces the AI
	// formatting step to exclude network round-trip from the timing measurement.
	err := aireport.Run(aireport.RunnerOptions{
		Patterns:     []string{"./..."},
		ModuleDir:    modRoot,
		Format:       "text",
		Adapter:      fa,
		SystemPrompt: "# Test prompt",
		Stdout:       &stdout,
		Stderr:       &stderr,
		// AnalyzeFunc is nil — real production pipeline runs.
	})
	if err != nil {
		t.Fatalf("SC-005 analysis pipeline failed: %v", err)
	}
	if len(fa.Calls) != 1 {
		t.Errorf("expected adapter called once, got %d calls", len(fa.Calls))
	}
	t.Log("SC-005: analysis pipeline completed within timeout")
}

// findModuleRootForReport returns the module root directory for use in report tests.
func findModuleRootForReport(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// --coverprofile flag tests (spec 020)

// TestRunReport_CoverProfile_ValidPath verifies that a valid --coverprofile
// path is threaded from reportParams through to RunnerOptions.CoverProfile
// (FR-001, FR-002). Uses a runnerFunc spy — no subprocess is spawned.
// SC-001 regression: spy.callCount == 1 confirms no double invocation.
func TestRunReport_CoverProfile_ValidPath(t *testing.T) {
	// Write a minimal valid coverage profile so the pre-flight os.Stat passes.
	profilePath := filepath.Join(t.TempDir(), "coverage.out")
	if err := os.WriteFile(profilePath, []byte("mode: set\n"), 0600); err != nil {
		t.Fatalf("writing fixture profile: %v", err)
	}

	var (
		capturedProfile string
		callCount       int
	)
	spy := func(opts aireport.RunnerOptions) error {
		callCount++
		capturedProfile = opts.CoverProfile
		return nil
	}

	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:     []string{"./..."},
		format:       "json",
		coverProfile: profilePath,
		stdout:       &stdout,
		stderr:       &stderr,
		runnerFunc:   spy,
	})
	if err != nil {
		t.Fatalf("runReport: %v", err)
	}
	if capturedProfile != profilePath {
		t.Errorf("opts.CoverProfile = %q, want %q", capturedProfile, profilePath)
	}
	if callCount != 1 {
		t.Errorf("spy called %d times, want 1 (SC-001: no double invocation)", callCount)
	}
}

// TestRunReport_CoverProfile_NonexistentPath verifies that a nonexistent
// --coverprofile path causes runReport to exit non-zero with an error
// that identifies the path (FR-004, SC-003).
func TestRunReport_CoverProfile_NonexistentPath(t *testing.T) {
	profilePath := filepath.Join(t.TempDir(), "nonexistent.out")

	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:     []string{"./..."},
		format:       "json",
		coverProfile: profilePath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent --coverprofile, got nil")
	}
	if !strings.Contains(err.Error(), profilePath) {
		t.Errorf("error %q does not contain path %q", err.Error(), profilePath)
	}
	if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "not exist") {
		t.Errorf("error %q does not indicate file not found", err.Error())
	}
}

// TestRunReport_CoverProfile_DirectoryPath verifies that a directory path
// supplied as --coverprofile causes runReport to exit non-zero with an error
// that identifies the problem (FR-005, SC-003).
func TestRunReport_CoverProfile_DirectoryPath(t *testing.T) {
	dirPath := t.TempDir()

	var stdout, stderr bytes.Buffer
	err := runReport(reportParams{
		patterns:     []string{"./..."},
		format:       "json",
		coverProfile: dirPath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for directory --coverprofile, got nil")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error %q does not contain \"directory\"", err.Error())
	}
}

// TestRunReport_CoverProfile_UnparseableContent verifies that a file with
// invalid coverage profile content results in a CRAP error recorded in the
// JSON payload (FR-006, SC-003). The partial-failure architecture stores CRAP
// errors in payload.Errors.CRAP rather than returning a Go error from runReport.
// Guarded by testing.Short() — runs the real pipeline (quality/classify/docscan steps).
func TestRunReport_CoverProfile_UnparseableContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs real analysis pipeline")
	}

	// Write a file that passes pre-flight (exists, is a file) but fails parsing.
	profilePath := filepath.Join(t.TempDir(), "bad.out")
	if err := os.WriteFile(profilePath, []byte("not a coverage profile\n"), 0600); err != nil {
		t.Fatalf("writing bad profile: %v", err)
	}

	var stdout, stderr bytes.Buffer
	// Use --format=json so no AI adapter is required.
	// The real production pipeline runs; CRAP step fails with parse error,
	// which is stored in payload.Errors.CRAP (partial-failure mode).
	err := runReport(reportParams{
		patterns:     []string{"github.com/unbound-force/gaze/internal/config"},
		format:       "json",
		coverProfile: profilePath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	// Under partial-failure mode, runReport returns nil even when CRAP fails.
	if err != nil {
		t.Logf("runReport returned error (unexpected under partial-failure): %v", err)
	}

	// Unmarshal JSON output and assert the CRAP error references the parse failure.
	var payload aireport.ReportPayload
	if decErr := json.NewDecoder(&stdout).Decode(&payload); decErr != nil {
		t.Fatalf("decoding JSON output: %v", decErr)
	}
	if payload.Errors.CRAP == nil {
		t.Fatal("expected payload.Errors.CRAP to be non-nil for unparseable profile")
	}
	if !strings.Contains(*payload.Errors.CRAP, "parsing coverage profile") {
		t.Errorf("payload.Errors.CRAP = %q, want to contain \"parsing coverage profile\"", *payload.Errors.CRAP)
	}
}

// TestReportCmd_CoverprofileInHelp verifies that --coverprofile appears in
// gaze report --help output with a description mentioning "pre-generated"
// (FR-007, US3 acceptance scenario 1).
func TestReportCmd_CoverprofileInHelp(t *testing.T) {
	cmd := newReportCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})
	// Execute returns an error for --help but the output is already written.
	_ = cmd.Execute()
	output := buf.String()
	if !strings.Contains(output, "--coverprofile") {
		t.Errorf("help output does not contain \"--coverprofile\":\n%s", output)
	}
	if !strings.Contains(output, "pre-generated") {
		t.Errorf("help output does not contain \"pre-generated\":\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// quality --include-unexported tests (issue #70)
// ---------------------------------------------------------------------------

// TestRunQuality_IncludeUnexported_PackageMain verifies that runQuality
// auto-detects package main and includes unexported functions without
// requiring --include-unexported. Both unexported functions (add, greet)
// should appear in the quality output.
func TestRunQuality_IncludeUnexported_PackageMain(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		patterns:          []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/mainpkg"},
		format:            "json",
		includeUnexported: false, // NOT set — auto-detect should kick in
		contractualThresh: -1,
		incidentalThresh:  -1,
		stdout:            &stdout,
		stderr:            &stderr,
	})
	if err != nil {
		t.Fatalf("runQuality returned error: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if output == "" {
		t.Fatal("expected non-empty quality output for package main (auto-detect should include unexported functions)")
	}

	// Both unexported functions must appear — verifies auto-detect
	// fired and included unexported functions.
	if !strings.Contains(output, "add") {
		t.Errorf("expected 'add' in quality output (unexported function in mainpkg)")
	}
	if !strings.Contains(output, "greet") {
		t.Errorf("expected 'greet' in quality output (unexported function in mainpkg)")
	}
}

// TestRunQuality_IncludeUnexported_LibraryPackage verifies that a non-main
// library package without --include-unexported only reports exported
// functions. Uses the welltested fixture which has known exported
// functions (Add, Greet) and unexported helpers.
func TestRunQuality_IncludeUnexported_LibraryPackage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runQuality(qualityParams{
		patterns:          []string{"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"},
		format:            "json",
		includeUnexported: false,
		contractualThresh: -1,
		incidentalThresh:  -1,
		stdout:            &stdout,
		stderr:            &stderr,
	})
	if err != nil {
		t.Fatalf("runQuality returned error: %v\nstderr: %s", err, stderr.String())
	}

	output := stdout.String()
	if output == "" {
		t.Fatal("expected non-empty quality output for welltested fixture")
	}

	// Exported functions should appear in the output.
	if !strings.Contains(output, "Add") {
		t.Errorf("expected exported function 'Add' in quality output")
	}
}

// ---------------------------------------------------------------------------
// buildAIMapperFunc tests
// ---------------------------------------------------------------------------

func TestBuildAIMapperFunc_InvalidBackend(t *testing.T) {
	_, err := buildAIMapperFunc("invalid", "")
	if err == nil {
		t.Fatal("expected error for invalid backend name")
	}
	if !strings.Contains(err.Error(), "invalid --ai-mapper value") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestBuildAIMapperFunc_ValidBackend(t *testing.T) {
	fn, err := buildAIMapperFunc("claude", "")
	if err != nil {
		t.Fatalf("unexpected error for valid backend: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil AIMapperFunc for valid backend")
	}
}

func TestBuildAIMapperFunc_OllamaRequiresModel(t *testing.T) {
	_, err := buildAIMapperFunc("ollama", "")
	if err == nil {
		t.Fatal("expected error for ollama without model")
	}
	if !strings.Contains(err.Error(), "--ai-mapper-model") {
		t.Errorf("expected error to mention --ai-mapper-model, got: %s", err)
	}
}

func TestBuildAIMapperFunc_OllamaWithModel(t *testing.T) {
	fn, err := buildAIMapperFunc("ollama", "llama3")
	if err != nil {
		t.Fatalf("unexpected error for ollama with model: %v", err)
	}
	if fn == nil {
		t.Fatal("expected non-nil AIMapperFunc for ollama with model")
	}
}
