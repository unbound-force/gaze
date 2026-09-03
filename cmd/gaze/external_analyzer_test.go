package main

import (
	"bytes"
	"encoding/json"
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

// TestQualityWithExternalAnalyzer verifies the full external analyzer
// quality path: session init → analyze → classify_signals → test_mapping
// → contract coverage report. The fake analyzer provides:
//
//   - 3 functions: add, multiply, divide (from analyze response)
//   - classify_signals: signals for divide/ErrorReturn and multiply/ReturnValue
//   - test_mapping: maps test_multiply → multiply/ReturnValue
//
// The quality report should contain entries for each analyzed function
// with contract coverage data.
func TestQualityWithExternalAnalyzer(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Use a temp directory as the "module root" — the external
	// analyzer doesn't need a real Go module.
	moduleDir := t.TempDir()

	// Create a minimal go.mod so pattern resolution works.
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

	// Parse the JSON output to verify quality report structure.
	var result struct {
		Reports []json.RawMessage      `json:"quality_reports"`
		Summary map[string]interface{} `json:"quality_summary"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, stdout.String())
	}

	// Should have reports (one per analyzed function from the external analyzer).
	if len(result.Reports) == 0 {
		t.Fatal("no reports in quality output")
	}

	// Summary should have average_contract_coverage.
	if result.Summary == nil {
		t.Fatal("missing summary in quality output")
	}

	// TotalTests should be 0 (external analyzers don't provide test function data).
	totalTests, _ := result.Summary["total_tests"].(float64)
	if totalTests != 0 {
		t.Errorf("total_tests = %g, want 0 (external analyzers don't provide test data)", totalTests)
	}

	// Average contract coverage should be > 0 because the fake analyzer
	// maps test_multiply → multiply/ReturnValue, giving multiply non-zero
	// contract coverage.
	avgCoverage, _ := result.Summary["average_contract_coverage"].(float64)
	if avgCoverage <= 0 {
		t.Errorf("average_contract_coverage = %g, want > 0 (multiply has test coverage)", avgCoverage)
	}

	// Inspect individual reports: each should have a target function and
	// contract coverage. Find the multiply function which should have
	// non-zero coverage from the test_mapping.
	type reportEntry struct {
		TargetFunction struct {
			Function string `json:"function"`
			Package  string `json:"package"`
		} `json:"target_function"`
		ContractCoverage struct {
			Percentage float64 `json:"percentage"`
		} `json:"contract_coverage"`
	}
	var foundMultiply bool
	for _, raw := range result.Reports {
		var entry reportEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatalf("parsing report entry: %v", err)
		}
		if entry.TargetFunction.Function == "" {
			t.Error("report entry has empty target_function.function")
		}
		if entry.TargetFunction.Function == "multiply" {
			foundMultiply = true
			if entry.ContractCoverage.Percentage <= 0 {
				t.Errorf("multiply contract_coverage.percentage = %g, want > 0",
					entry.ContractCoverage.Percentage)
			}
		}
	}
	if !foundMultiply {
		t.Error("no report entry found for function 'multiply'")
	}

	// Stderr should mention the reduced report note.
	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "contract coverage only") {
		t.Errorf("stderr should mention reduced report, got: %s", stderrStr)
	}

	// Stderr should mention the external analyzer.
	if !strings.Contains(stderrStr, "fake-analyzer") {
		t.Errorf("stderr should mention analyzer name, got: %s", stderrStr)
	}
}

// TestBuildExternalQualityReports_NilSideEffects verifies that
// buildExternalQualityReports returns empty reports and an empty summary
// when the providers have no SideEffects analyzer (nil).
func TestBuildExternalQualityReports_NilSideEffects(t *testing.T) {
	var stderr bytes.Buffer
	providers := &adapter.Providers{
		// SideEffects is nil — no side effect data available.
	}
	reports, summary := buildExternalQualityReports(nil, providers, qualityParams{
		stderr: &stderr,
	})
	if len(reports) != 0 {
		t.Errorf("expected 0 reports, got %d", len(reports))
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.TotalTests != 0 {
		t.Errorf("TotalTests = %d, want 0", summary.TotalTests)
	}
	if summary.AverageContractCoverage != 0 {
		t.Errorf("AverageContractCoverage = %g, want 0", summary.AverageContractCoverage)
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
		t.Fatal("expected error when analyzer binary not found")
	}
	// The external analyzer path is now supported; error should be about
	// the binary not being found, not about the feature being unsupported.
	errMsg := err.Error()
	if bytes.Contains([]byte(errMsg), []byte("not yet supported")) {
		t.Errorf("quality --analyzer should no longer be rejected; got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "analyzer") {
		t.Errorf("error should mention analyzer, got: %s", errMsg)
	}
}
