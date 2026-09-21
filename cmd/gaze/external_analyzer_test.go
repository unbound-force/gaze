package main

import (
	"bytes"
	"context"
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

// TestCrapWithExternalAnalyzer_BypassesFindModuleRoot verifies that
// runCrap with --analyzer set does NOT call FindModuleRoot. This is
// the regression test for issue #250: gaze crap --analyzer fails
// with 'no go.mod found' for non-Go projects.
func TestCrapWithExternalAnalyzer_BypassesFindModuleRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"."},
		format:       "text",
		opts:         opts,
		moduleDir:    t.TempDir(), // directory without go.mod
		analyzerFlag: "nonexistent-analyzer",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent analyzer")
	}
	errMsg := err.Error()

	// The error must NOT be about FindModuleRoot — that's the bug.
	if strings.Contains(errMsg, "finding module root") {
		t.Errorf("error should not mention FindModuleRoot, got: %s", errMsg)
	}
	if strings.Contains(errMsg, "no go.mod found") {
		t.Errorf("error should not mention go.mod, got: %s", errMsg)
	}

	// The error MUST be about the analyzer binary (proving the
	// external analyzer path was reached).
	if !strings.Contains(errMsg, "discovering analyzer") && !strings.Contains(errMsg, "not found") {
		t.Errorf("error should be about analyzer discovery, got: %s", errMsg)
	}
}

// TestRunCrap_GoNativePath_FindModuleRootFailure verifies that runCrap
// without --analyzer, called from a directory without go.mod, returns
// an error with the "finding module root" wrapping format. This proves
// FindModuleRoot was moved into runCrap and the error format is preserved.
func TestRunCrap_GoNativePath_FindModuleRootFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:  []string{"."},
		format:    "text",
		opts:      opts,
		moduleDir: t.TempDir(), // directory without go.mod
		stdout:    &stdout,
		stderr:    &stderr,
	})
	if err == nil {
		t.Fatal("expected error when moduleDir has no go.mod")
	}
	if !strings.Contains(err.Error(), "finding module root") {
		t.Errorf("error should contain 'finding module root', got: %s", err)
	}
}

// TestQualityWithExternalAnalyzer_HappyPath verifies the full quality
// pipeline with an external analyzer that supports test_mapping.
// The fake analyzer provides:
//   - analyze: divide (ReturnValue+ErrorReturn), multiply (ReturnValue), add (no effects)
//   - test_mapping: test_multiply → multiply:ReturnValue, test_divide_basic → divide:ReturnValue,
//     test_divide_error → divide:ErrorReturn
//
// Expected quality report: 3 test functions. Per-test assertion-detection
// confidence is the fraction of that test's mapping rows with a recognized
// (non-empty) assertion_type: test_multiply=100, test_divide_basic=100,
// test_divide_error=0. Summary confidence is the arithmetic mean = 67.
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
			TestFunction                 string `json:"test_function"`
			AssertionCount               int    `json:"assertion_count"`
			AssertionDetectionConfidence int    `json:"assertion_detection_confidence"`
			ContractCoverage             struct {
				Percentage       float64 `json:"percentage"`
				CoveredCount     int     `json:"covered_count"`
				TotalContractual int     `json:"total_contractual"`
			} `json:"contract_coverage"`
		} `json:"quality_reports"`
		Summary struct {
			TotalTests                   int     `json:"total_tests"`
			AverageContractCoverage      float64 `json:"average_contract_coverage"`
			AssertionDetectionConfidence int     `json:"assertion_detection_confidence"`
			ClassificationCounts         struct {
				Contractual int `json:"contractual"`
				Incidental  int `json:"incidental"`
				Ambiguous   int `json:"ambiguous"`
			} `json:"classification_counts"`
		} `json:"quality_summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, stdout.String())
	}

	if len(output.QualityReports) != 3 {
		t.Fatalf("got %d quality reports, want 3", len(output.QualityReports))
	}

	// Per-test-function confidence: fraction of that test's mapping rows
	// with a recognized (non-empty) assertion_type.
	wantConfidence := map[string]int{
		"test_multiply":     100,
		"test_divide_basic": 100,
		"test_divide_error": 0,
	}
	seen := make(map[string]bool)
	for _, r := range output.QualityReports {
		want, ok := wantConfidence[r.TestFunction]
		if !ok {
			t.Errorf("unexpected test function %q in report", r.TestFunction)
			continue
		}
		seen[r.TestFunction] = true
		if r.AssertionCount != 1 {
			t.Errorf("%s AssertionCount = %d, want 1", r.TestFunction, r.AssertionCount)
		}
		if r.AssertionDetectionConfidence != want {
			t.Errorf("%s AssertionDetectionConfidence = %d, want %d",
				r.TestFunction, r.AssertionDetectionConfidence, want)
		}
	}
	if len(seen) != 3 {
		t.Errorf("got %d distinct test functions, want 3", len(seen))
	}

	// test_multiply targets multiply, which has 1 contractual effect
	// (ReturnValue) covered by 1 mapping → 100% contract coverage.
	for _, r := range output.QualityReports {
		if r.TestFunction != "test_multiply" {
			continue
		}
		if r.ContractCoverage.Percentage != 100 {
			t.Errorf("test_multiply ContractCoverage.Percentage = %g, want 100",
				r.ContractCoverage.Percentage)
		}
		if r.ContractCoverage.CoveredCount != 1 {
			t.Errorf("test_multiply CoveredCount = %d, want 1", r.ContractCoverage.CoveredCount)
		}
		if r.ContractCoverage.TotalContractual != 1 {
			t.Errorf("test_multiply TotalContractual = %d, want 1",
				r.ContractCoverage.TotalContractual)
		}
	}

	if output.Summary.TotalTests != 3 {
		t.Errorf("Summary.TotalTests = %d, want 3", output.Summary.TotalTests)
	}
	// Average over 3 reports: (100 + 50 + 50) / 3 ≈ 66.67.
	wantAvgCoverage := (100.0 + 50.0 + 50.0) / 3.0
	if output.Summary.AverageContractCoverage != wantAvgCoverage {
		t.Errorf("Summary.AverageContractCoverage = %g, want %g",
			output.Summary.AverageContractCoverage, wantAvgCoverage)
	}
	// Mean of per-report confidence: (100 + 100 + 0) / 3 → 67 (round half up).
	if output.Summary.AssertionDetectionConfidence != 67 {
		t.Errorf("Summary.AssertionDetectionConfidence = %d, want 67",
			output.Summary.AssertionDetectionConfidence)
	}

	// The headline feature: the analyzer's analyze response emits three
	// side effects, all classified "contractual" (divide:ReturnValue,
	// divide:ErrorReturn, multiply:ReturnValue), surfaced as a
	// classification_counts distribution in the quality summary.
	cc := output.Summary.ClassificationCounts
	if cc.Contractual != 3 || cc.Incidental != 0 || cc.Ambiguous != 0 {
		t.Errorf("classification_counts = %+v, want contractual=3 incidental=0 ambiguous=0", cc)
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
	if reason != "test_mapping_unavailable" {
		t.Errorf("quality_summary.reason = %q, want %q", reason, "test_mapping_unavailable")
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
	if reason != "test_mapping_error" {
		t.Errorf("quality_summary.reason = %q, want %q", reason, "test_mapping_error")
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
// is rejected when used with --analyzer or --language (Go-specific SSA feature).
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
	if !strings.Contains(err.Error(), "--target is not supported with --analyzer or --language") {
		t.Errorf("expected target rejection error, got: %s", err.Error())
	}
}

// TestQualityWithExternalAnalyzer_RejectsAIMapper verifies that
// --ai-mapper is rejected when used with --analyzer or --language.
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
	if !strings.Contains(err.Error(), "--ai-mapper is not supported with --analyzer or --language") {
		t.Errorf("expected ai-mapper rejection error, got: %s", err.Error())
	}
}

// TestQualityWithExternalAnalyzer_RejectsIncludeUnexported verifies that
// --include-unexported is rejected when used with --analyzer (Go-specific feature).
func TestQualityWithExternalAnalyzer_RejectsIncludeUnexported(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:          []string{"./..."},
		format:            "text",
		analyzerFlag:      "some-analyzer",
		includeUnexported: true,
		stdout:            &stdout,
		stderr:            &stderr,
	})
	if err == nil {
		t.Fatal("expected error for --include-unexported with --analyzer")
	}
	if !strings.Contains(err.Error(), "--include-unexported is not supported with --analyzer") {
		t.Errorf("expected include-unexported rejection error, got: %s", err.Error())
	}
}

// TestReportWithExternalAnalyzer_BypassesFindModuleRoot verifies that
// runReport with --analyzer set does NOT call FindModuleRoot. This is
// the regression test for issue #257: gaze report --analyzer fails
// with 'no go.mod found' for non-Go projects (same pattern as #250).
func TestReportWithExternalAnalyzer_BypassesFindModuleRoot(t *testing.T) {
	// Run from a temporary directory that has no go.mod, so
	// FindModuleRoot would fail if it were called.
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer

	err := runReport(reportParams{
		patterns:     []string{"."},
		format:       "json",
		analyzerFlag: "nonexistent-analyzer",
		languageFlag: "python",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent analyzer")
	}
	errMsg := err.Error()

	// The error must NOT be about FindModuleRoot — that's the bug.
	if strings.Contains(errMsg, "finding module root") {
		t.Errorf("error should not mention FindModuleRoot, got: %s", errMsg)
	}
	if strings.Contains(errMsg, "no go.mod found") {
		t.Errorf("error should not mention go.mod, got: %s", errMsg)
	}

	// The error MUST be about the analyzer binary (proving the
	// external analyzer path was reached).
	if !strings.Contains(errMsg, "discovering analyzer") && !strings.Contains(errMsg, "not found") {
		t.Errorf("error should be about analyzer discovery, got: %s", errMsg)
	}
}

// TestCrapWithLanguageOnly_TriggersExternalPath verifies that runCrap
// with only --language set (no --analyzer) dispatches to the external
// analyzer path rather than silently falling through to Go-native
// analysis. This is the regression test for the --language-only
// dispatch inconsistency fixed alongside issue #278.
func TestCrapWithLanguageOnly_TriggersExternalPath(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"."},
		format:       "text",
		opts:         opts,
		moduleDir:    t.TempDir(), // no go.mod — Go-native path would fail differently
		languageFlag: "zz-nodiscover-test-language",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error when --language resolves to no analyzer")
	}
	errMsg := err.Error()
	// The error must NOT be about FindModuleRoot — that would prove the
	// Go-native path was (incorrectly) taken.
	if strings.Contains(errMsg, "finding module root") {
		t.Errorf("error should not mention FindModuleRoot, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "no analyzer found for language") {
		t.Errorf("error should be about language discovery, got: %s", errMsg)
	}
}

// TestQualityWithLanguageOnly_TriggersExternalPath verifies that runQuality
// with only --language set (no --analyzer) dispatches to the external
// analyzer path rather than silently falling through to Go-native
// analysis. This is the runQuality equivalent of
// TestCrapWithLanguageOnly_TriggersExternalPath — runQuality is the path
// that produces classification_counts from the external analyzer.
func TestQualityWithLanguageOnly_TriggersExternalPath(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:     []string{"."},
		format:       "text",
		languageFlag: "zz-nodiscover-test-language",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error when --language resolves to no analyzer")
	}
	errMsg := err.Error()
	// The error must NOT be about FindModuleRoot — that would prove the
	// Go-native path was (incorrectly) taken.
	if strings.Contains(errMsg, "finding module root") {
		t.Errorf("error should not mention FindModuleRoot, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "no analyzer found for language") {
		t.Errorf("error should be about language discovery, got: %s", errMsg)
	}
}

// TestRunReport_GoNativePath_FindModuleRootFailure verifies that
// runReport without --analyzer, called from a directory without
// go.mod, returns an error with the "finding module root" wrapping
// format. This proves FindModuleRoot runs in the Go-native path.
func TestRunReport_GoNativePath_FindModuleRootFailure(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer

	err := runReport(reportParams{
		patterns: []string{"."},
		format:   "json",
		stdout:   &stdout,
		stderr:   &stderr,
	})
	if err == nil {
		t.Fatal("expected error when cwd has no go.mod")
	}
	if !strings.Contains(err.Error(), "finding module root") {
		t.Errorf("error should contain 'finding module root', got: %s", err)
	}
}

// TestReportWithExternalAnalyzer_DocscanPopulated verifies that the external
// report path wires the analyzer session into the docscan step so the payload
// carries a populated api_coverage section.
func TestReportWithExternalAnalyzer_DocscanPopulated(t *testing.T) {
	moduleDir := t.TempDir()
	goMod := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	var stderr bytes.Buffer
	analyzeFunc, cleanup, err := buildExternalReportAnalyzeFunc(
		context.Background(), fakeBinaryPath, "python", moduleDir, []string{"./..."}, &stderr)
	if err != nil {
		t.Fatalf("buildExternalReportAnalyzeFunc: %v", err)
	}
	defer cleanup()

	payload, err := analyzeFunc([]string{"./..."}, moduleDir)
	if err != nil {
		t.Fatalf("analyzeFunc: %v\nstderr: %s", err, stderr.String())
	}

	if payload.Docscan == nil {
		t.Fatal("payload.Docscan is nil, expected docscan envelope")
	}

	var env struct {
		APICoverage map[string]interface{} `json:"api_coverage"`
	}
	if err := json.Unmarshal(payload.Docscan, &env); err != nil {
		t.Fatalf("unmarshal docscan envelope: %v\nraw: %s", err, string(payload.Docscan))
	}
	if env.APICoverage == nil {
		t.Fatal("api_coverage is null, expected populated")
	}
	if env.APICoverage["source"] != "doc_coverage" {
		t.Errorf("api_coverage source = %v, want doc_coverage", env.APICoverage["source"])
	}
	if env.APICoverage["total_symbols"] != float64(3) {
		t.Errorf("api_coverage total_symbols = %v, want 3", env.APICoverage["total_symbols"])
	}
	if env.APICoverage["documented_symbols"] != float64(2) {
		t.Errorf("api_coverage documented_symbols = %v, want 2", env.APICoverage["documented_symbols"])
	}
}
