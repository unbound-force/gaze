package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/crap"
)

// TestCrapWithExternalAnalyzer verifies that runCrap correctly uses
// an external analyzer binary via the --analyzer flag. The fake
// analyzer provides canned complexity and coverage data:
//
//   - add:      complexity=2, coverage=90%
//   - multiply: complexity=3, coverage=60%
//   - divide:   complexity=5, coverage=0%
//
// CRAP scores are computed from these values using the standard
// formula: CRAP(c,cov) = c² × (1 - cov)³ + c.
func TestCrapWithExternalAnalyzer(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Use a temp directory as the "module root" — the external
	// analyzer doesn't need a real Go module.
	moduleDir := t.TempDir()

	// Create a minimal go.mod so crap.Analyze can resolve patterns.
	// The external providers bypass Go tooling, but the framework
	// still validates the module directory.
	goMod := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "json",
		opts:         opts,
		moduleDir:    moduleDir,
		analyzerFlag: fakeBinaryPath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err != nil {
		t.Fatalf("runCrap with external analyzer: %v\nstderr: %s", err, stderr.String())
	}

	// Parse the JSON output to verify CRAP scores.
	var report crap.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, stdout.String())
	}

	if len(report.Scores) == 0 {
		t.Fatal("no scores in report")
	}

	// Build a map of function name → CRAP score for verification.
	scores := make(map[string]float64)
	for _, s := range report.Scores {
		scores[s.Function] = s.CRAP
	}

	// Verify CRAP scores match expected values from the fake data.
	// CRAP formula: c² × (1 - cov)³ + c
	//
	// add:      2² × (1 - 0.90)³ + 2 = 4 × 0.001 + 2 = 2.004
	// multiply: 3² × (1 - 0.60)³ + 3 = 9 × 0.064 + 3 = 3.576
	// divide:   5² × (1 - 0.00)³ + 5 = 25 × 1.0 + 5 = 30.0
	wantApprox := map[string]struct {
		min, max float64
	}{
		"add":      {1.5, 3.0},
		"multiply": {3.0, 4.5},
		"divide":   {29.0, 31.0},
	}

	for name, want := range wantApprox {
		got, ok := scores[name]
		if !ok {
			t.Errorf("function %q not found in scores", name)
			continue
		}
		if got < want.min || got > want.max {
			t.Errorf("%s CRAP = %g, want in [%g, %g]", name, got, want.min, want.max)
		}
	}

	// Verify the stderr mentions the external analyzer.
	stderrStr := stderr.String()
	if !bytes.Contains([]byte(stderrStr), []byte("fake-analyzer")) {
		t.Errorf("stderr should mention analyzer name, got: %s", stderrStr)
	}
}

// TestCrapWithExternalAnalyzer_NotFound verifies that a nonexistent
// analyzer binary produces a clear error.
func TestCrapWithExternalAnalyzer_NotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         opts,
		moduleDir:    t.TempDir(),
		analyzerFlag: "/nonexistent/analyzer",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent analyzer")
	}
}

// TestQualityWithExternalAnalyzer_HappyPath verifies the full quality
// pipeline with an external analyzer that supports test_mapping.
// The fake analyzer provides:
//   - analyze: divide (ReturnValue+ErrorReturn), multiply (ReturnValue), add (no effects)
//   - test_mapping: test_multiply → multiply:ReturnValue (confidence 80)
//
// Expected quality report: 1 test function (test_multiply), targeting
// multiply which has 1 contractual effect (ReturnValue). Coverage = 100%.
func TestQualityWithExternalAnalyzer_HappyPath(t *testing.T) {
	var stdout, stderr bytes.Buffer

	moduleDir := t.TempDir()
	goMod := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	err := runQuality(qualityParams{
		patterns:     []string{"./..."},
		format:       "json",
		analyzerFlag: fakeBinaryPath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err != nil {
		t.Fatalf("runQuality with external analyzer: %v\nstderr: %s", err, stderr.String())
	}

	// Parse JSON output.
	var output struct {
		QualityReports []struct {
			TestFunction     string `json:"test_function"`
			AssertionCount   int    `json:"assertion_count"`
			ContractCoverage struct {
				Percentage       float64 `json:"percentage"`
				CoveredCount     int     `json:"covered_count"`
				TotalContractual int     `json:"total_contractual"`
			} `json:"contract_coverage"`
		} `json:"quality_reports"`
		Summary struct {
			TotalTests              int     `json:"total_tests"`
			AverageContractCoverage float64 `json:"average_contract_coverage"`
		} `json:"quality_summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, stdout.String())
	}

	if len(output.QualityReports) != 1 {
		t.Fatalf("got %d quality reports, want 1", len(output.QualityReports))
	}

	r := output.QualityReports[0]
	if r.TestFunction != "test_multiply" {
		t.Errorf("TestFunction = %q, want %q", r.TestFunction, "test_multiply")
	}
	if r.AssertionCount != 1 {
		t.Errorf("AssertionCount = %d, want 1", r.AssertionCount)
	}
	// multiply has 1 contractual effect (ReturnValue), 1 mapping covers it → 100%.
	if r.ContractCoverage.Percentage != 100 {
		t.Errorf("ContractCoverage.Percentage = %g, want 100", r.ContractCoverage.Percentage)
	}
	if r.ContractCoverage.CoveredCount != 1 {
		t.Errorf("CoveredCount = %d, want 1", r.ContractCoverage.CoveredCount)
	}
	if r.ContractCoverage.TotalContractual != 1 {
		t.Errorf("TotalContractual = %d, want 1", r.ContractCoverage.TotalContractual)
	}

	if output.Summary.TotalTests != 1 {
		t.Errorf("Summary.TotalTests = %d, want 1", output.Summary.TotalTests)
	}
	if output.Summary.AverageContractCoverage != 100 {
		t.Errorf("Summary.AverageContractCoverage = %g, want 100", output.Summary.AverageContractCoverage)
	}

	// Verify stderr mentions the external analyzer.
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "fake-analyzer") {
		t.Errorf("stderr should mention analyzer name, got: %s", stderrStr)
	}
}

// TestQualityWithExternalAnalyzer_NoTestMapping_NoThresholds verifies
// that when the analyzer doesn't support test_mapping and no thresholds
// are set, the command succeeds with a zero-coverage report.
func TestQualityWithExternalAnalyzer_NoTestMapping_NoThresholds(t *testing.T) {
	var stdout, stderr bytes.Buffer

	moduleDir := t.TempDir()
	goMod := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Use --crash-after=test_mapping so the fake analyzer supports
	// test_mapping in capabilities but crashes if called. We need
	// to test the "no test_mapping capability" path, which requires
	// a fake that declares test_mapping: false.
	// Since we can't easily change the fake's capabilities, we test
	// the handler function directly.
	err := handleQualityNoTestMapping(qualityParams{
		stdout: &stdout,
		stderr: &stderr,
		format: "json",
	}, &adapter.Providers{AnalyzerName: "test-analyzer"})

	if err != nil {
		t.Fatalf("expected nil error with no thresholds, got: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "does not support test_mapping") {
		t.Errorf("expected test_mapping warning in stderr, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "test-analyzer") {
		t.Errorf("expected analyzer name in stderr, got: %s", stderrStr)
	}

	// Verify JSON output contains the reason field.
	if stdout.Len() == 0 {
		t.Fatal("expected non-empty stdout")
	}
	var jsonOutput map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	summaryRaw, ok := jsonOutput["quality_summary"]
	if !ok {
		t.Fatal("JSON output missing 'quality_summary' key")
	}
	var summaryMap map[string]any
	if err := json.Unmarshal(summaryRaw, &summaryMap); err != nil {
		t.Fatalf("invalid quality_summary JSON: %v", err)
	}
	reason, _ := summaryMap["reason"].(string)
	if reason != "test_mapping unavailable" {
		t.Errorf("quality_summary.reason = %q, want %q", reason, "test_mapping unavailable")
	}
}

// TestQualityWithExternalAnalyzer_NoTestMapping_WithThresholds verifies
// that when the analyzer doesn't support test_mapping and thresholds
// are set, the command returns an error.
func TestQualityWithExternalAnalyzer_NoTestMapping_WithThresholds(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := handleQualityNoTestMapping(qualityParams{
		stdout:              &stdout,
		stderr:              &stderr,
		format:              "json",
		minContractCoverage: 50,
	}, &adapter.Providers{AnalyzerName: "test-analyzer"})

	if err == nil {
		t.Fatal("expected error when thresholds are set but test_mapping unavailable")
	}
	if !strings.Contains(err.Error(), "quality thresholds cannot be evaluated") {
		t.Errorf("expected threshold evaluation error, got: %s", err.Error())
	}
}

// TestQualityWithExternalAnalyzer_TestMappingError_NoThresholds verifies
// that when test_mapping fails and no thresholds are set, the command
// succeeds with a zero-coverage report.
func TestQualityWithExternalAnalyzer_TestMappingError_NoThresholds(t *testing.T) {
	var stdout, stderr bytes.Buffer

	fetchErr := fmt.Errorf("connection refused")
	err := handleQualityTestMappingError(qualityParams{
		stdout: &stdout,
		stderr: &stderr,
		format: "json",
	}, &adapter.Providers{AnalyzerName: "test-analyzer"}, fetchErr)

	if err != nil {
		t.Fatalf("expected nil error with no thresholds, got: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "test_mapping failed") {
		t.Errorf("expected test_mapping failure warning in stderr, got: %s", stderrStr)
	}
	if !strings.Contains(stderrStr, "connection refused") {
		t.Errorf("expected underlying error in stderr, got: %s", stderrStr)
	}

	// Verify JSON output contains the reason field with error details.
	if stdout.Len() == 0 {
		t.Fatal("expected non-empty stdout")
	}
	var jsonOutput map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &jsonOutput); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	summaryRaw, ok := jsonOutput["quality_summary"]
	if !ok {
		t.Fatal("JSON output missing 'quality_summary' key")
	}
	var summaryMap map[string]any
	if err := json.Unmarshal(summaryRaw, &summaryMap); err != nil {
		t.Fatalf("invalid quality_summary JSON: %v", err)
	}
	reason, _ := summaryMap["reason"].(string)
	if !strings.Contains(reason, "test_mapping error") {
		t.Errorf("quality_summary.reason = %q, want it to contain %q", reason, "test_mapping error")
	}
	if !strings.Contains(reason, "connection refused") {
		t.Errorf("quality_summary.reason = %q, want it to contain %q", reason, "connection refused")
	}
}

// TestQualityWithExternalAnalyzer_TestMappingError_WithThresholds verifies
// that when test_mapping fails and thresholds are set, the command
// returns an error wrapping the original fetch error.
func TestQualityWithExternalAnalyzer_TestMappingError_WithThresholds(t *testing.T) {
	var stdout, stderr bytes.Buffer

	fetchErr := fmt.Errorf("connection refused")
	err := handleQualityTestMappingError(qualityParams{
		stdout:               &stdout,
		stderr:               &stderr,
		format:               "json",
		maxOverSpecification: 10,
	}, &adapter.Providers{AnalyzerName: "test-analyzer"}, fetchErr)

	if err == nil {
		t.Fatal("expected error when thresholds are set but test_mapping failed")
	}
	if !strings.Contains(err.Error(), "test_mapping failed") {
		t.Errorf("expected test_mapping failed error, got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("expected underlying error in wrapped message, got: %s", err.Error())
	}
}

// TestQualityWithExternalAnalyzer_BinaryNotFound verifies that --analyzer
// on gaze quality attempts to run the external analyzer and fails cleanly
// when the binary does not exist.
func TestQualityWithExternalAnalyzer_BinaryNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:     []string{"./..."},
		format:       "text",
		analyzerFlag: "some-analyzer",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent analyzer binary")
	}
	// The error should be about the analyzer not being found, NOT
	// about the flag being unsupported.
	errMsg := err.Error()
	if strings.Contains(errMsg, "not yet supported") {
		t.Errorf("--analyzer should be accepted for quality now, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "not found") && !strings.Contains(errMsg, "spawning") {
		t.Errorf("expected discovery/spawn error, got: %s", errMsg)
	}
}

// TestQualityWithExternalAnalyzer_RejectsTarget verifies that --target
// is rejected when used with --analyzer (Go-specific SSA feature).
func TestQualityWithExternalAnalyzer_RejectsTarget(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:     []string{"./..."},
		format:       "text",
		analyzerFlag: "some-analyzer",
		targetFunc:   "SomeFunc",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for --target with --analyzer")
	}
	if !strings.Contains(err.Error(), "--target is not supported with --analyzer") {
		t.Errorf("expected target rejection error, got: %s", err.Error())
	}
}

// TestQualityWithExternalAnalyzer_RejectsAIMapper verifies that
// --ai-mapper is rejected when used with --analyzer.
func TestQualityWithExternalAnalyzer_RejectsAIMapper(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:     []string{"./..."},
		format:       "text",
		analyzerFlag: "some-analyzer",
		aiMapper:     "claude",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for --ai-mapper with --analyzer")
	}
	if !strings.Contains(err.Error(), "--ai-mapper is not supported with --analyzer") {
		t.Errorf("expected ai-mapper rejection error, got: %s", err.Error())
	}
}
