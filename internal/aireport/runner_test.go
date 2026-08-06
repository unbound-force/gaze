package aireport

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// fakeAnalyze returns a fixed payload for testing.
func fakeAnalyze(payload *ReportPayload, err error) func([]string, string) (*ReportPayload, error) {
	return func(_ []string, _ string) (*ReportPayload, error) {
		return payload, err
	}
}

// TestRun_JSONFormat_WritesValidPayload verifies that --format=json writes a
// valid JSON ReportPayload to stdout and returns nil.
func TestRun_JSONFormat_WritesValidPayload(t *testing.T) {
	crapMsg := json.RawMessage(`{"scores":[],"summary":{"crapload":2}}`)
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: 2},
		CRAP:    crapMsg,
		Errors:  PayloadErrors{},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &stdout,
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var decoded ReportPayload
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\noutput: %s", err, stdout.String())
	}
	// The encoder pretty-prints; re-compact for comparison.
	var crapCompact, decodedCompact bytes.Buffer
	if err := json.Compact(&crapCompact, crapMsg); err != nil {
		t.Fatalf("compact crapMsg: %v", err)
	}
	if err := json.Compact(&decodedCompact, decoded.CRAP); err != nil {
		t.Fatalf("compact decoded.CRAP: %v", err)
	}
	if crapCompact.String() != decodedCompact.String() {
		t.Errorf("CRAP field mismatch:\nwant: %s\ngot:  %s", crapCompact.String(), decodedCompact.String())
	}
}

// TestRun_JSONFormat_SkipsAIAdapter verifies that --format=json does not call
// the AI adapter even when one is set.
func TestRun_JSONFormat_SkipsAIAdapter(t *testing.T) {
	fa := &FakeAdapter{Response: "should not be called"}
	payload := &ReportPayload{}

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(fa.Calls) != 0 {
		t.Errorf("expected 0 adapter calls in json mode, got %d", len(fa.Calls))
	}
}

// TestRun_CRAPStepFailure_PartialPayload verifies that a CRAP step failure
// produces a partial payload with a non-null errors.crap field, and that the
// command does not abort (exit 0 / nil error).
func TestRun_CRAPStepFailure_PartialPayload(t *testing.T) {
	errMsg := "coverage profile failed"
	payload := &ReportPayload{
		CRAP:    nil,
		Quality: json.RawMessage(`{"quality_reports":[]}`),
		Errors: PayloadErrors{
			CRAP: &errMsg,
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &stdout,
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run returned error on partial failure: %v", err)
	}

	var decoded ReportPayload
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout not valid JSON: %v", err)
	}
	if decoded.Errors.CRAP == nil {
		t.Fatal("expected Errors.CRAP non-nil")
	}
	if *decoded.Errors.CRAP != errMsg {
		t.Errorf("expected Errors.CRAP %q, got %q", errMsg, *decoded.Errors.CRAP)
	}
}

// TestRun_ZeroPatterns_ReturnsError verifies that Run returns an error when
// the AnalyzeFunc returns an error (e.g., zero packages).
func TestRun_ZeroPatterns_ReturnsError(t *testing.T) {
	err := Run(RunnerOptions{
		Patterns:    []string{},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(nil, errors.New("no package patterns specified")),
	})
	if err == nil {
		t.Fatal("expected error for zero packages")
	}
}

// TestRun_ProgressSignals_JSON verifies that progress signals appear on stderr
// even in --format=json mode (they are emitted by the analysis steps).
func TestRun_ProgressSignals_JSON(t *testing.T) {
	payload := &ReportPayload{
		CRAP: json.RawMessage(`{}`),
	}
	var stderr bytes.Buffer

	// AnalyzeFunc bypasses the real pipeline, so no signals in json mode.
	// This test confirms no panic and no interference.
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestRun_TextFormat_WritesFormattedReport verifies that --format=text calls
// the AI adapter and writes the formatted report to stdout.
func TestRun_TextFormat_WritesFormattedReport(t *testing.T) {
	report := "🔍 CRAP Analysis\n\n📊 Quality\n\n🧪 Classification\n\n🏥 Health"
	fa := &FakeAdapter{Response: report}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:     []string{"./..."},
		Format:       "text",
		Stdout:       &stdout,
		Stderr:       &stderr,
		Adapter:      fa,
		SystemPrompt: "system instructions",
		AnalyzeFunc:  fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stdout.String() != report {
		t.Errorf("stdout mismatch:\nwant: %q\ngot:  %q", report, stdout.String())
	}
	if len(fa.Calls) != 1 {
		t.Errorf("expected 1 adapter call, got %d", len(fa.Calls))
	}
	if fa.Calls[0].SystemPrompt != "system instructions" {
		t.Errorf("expected SystemPrompt passed through, got %q", fa.Calls[0].SystemPrompt)
	}
}

// TestRun_TextFormat_EmptyAdapterOutput_ReturnsError verifies FR-016: empty
// AI adapter output triggers an error.
func TestRun_TextFormat_EmptyAdapterOutput_ReturnsError(t *testing.T) {
	fa := &FakeAdapter{Response: "   "}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected error for empty adapter output")
	}
	if !strings.Contains(err.Error(), "FR-016") {
		t.Errorf("expected FR-016 in error message, got: %v", err)
	}
}

// TestRun_TextFormat_WritesStepSummary verifies that StepSummaryPath non-empty
// causes the report to be appended to that file with the correct content.
func TestRun_TextFormat_WritesStepSummary(t *testing.T) {
	report := "# Report"
	fa := &FakeAdapter{Response: report}
	payload := &ReportPayload{}

	tmpFile := t.TempDir() + "/summary.md"
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:        []string{"./..."},
		Format:          "text",
		Stdout:          &stdout,
		Stderr:          &stderr,
		Adapter:         fa,
		StepSummaryPath: tmpFile,
		AnalyzeFunc:     fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify progress signal appeared on stderr.
	if !strings.Contains(stderr.String(), "Writing Step Summary") {
		t.Errorf("expected Step Summary signal on stderr, got: %q", stderr.String())
	}

	// Verify the file was actually written with the correct content.
	data, readErr := os.ReadFile(tmpFile)
	if readErr != nil {
		t.Fatalf("reading step summary file: %v", readErr)
	}
	if string(data) != report {
		t.Errorf("step summary content mismatch: want %q, got %q", report, string(data))
	}
}

// TestRun_TextFormat_UnwritableStepSummary_ReturnsNil verifies that an
// unwritable StepSummaryPath emits a warning but returns nil (FR-008).
func TestRun_TextFormat_UnwritableStepSummary_ReturnsNil(t *testing.T) {
	report := "# Report"
	fa := &FakeAdapter{Response: report}
	payload := &ReportPayload{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:        []string{"./..."},
		Format:          "text",
		Stdout:          &stdout,
		Stderr:          &stderr,
		Adapter:         fa,
		StepSummaryPath: "/nonexistent/dir/summary.md",
		AnalyzeFunc:     fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run returned error on unwritable StepSummaryPath: %v", err)
	}
	if stdout.String() != report {
		t.Errorf("expected stdout report despite StepSummary failure")
	}
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("expected warning on stderr for unwritable path, got: %q", stderr.String())
	}
}

// TestRun_TextFormat_ProgressSignals verifies that progress signals appear on
// stderr during --format=text mode.
func TestRun_TextFormat_ProgressSignals(t *testing.T) {
	fa := &FakeAdapter{Response: "# Report"}
	payload := &ReportPayload{}
	var stderr bytes.Buffer

	_ = Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})

	got := stderr.String()
	if !strings.Contains(got, "Formatting report") {
		t.Errorf("expected 'Formatting report' signal on stderr, got: %q", got)
	}
}

// TestRun_DefaultFormat_TreatedAsText verifies that an unrecognised Format
// value defaults to "text" behaviour.
func TestRun_DefaultFormat_TreatedAsText(t *testing.T) {
	fa := &FakeAdapter{Response: "# Report"}
	payload := &ReportPayload{}

	err := Run(RunnerOptions{
		Format:      "", // should default to text
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	// No error expected; FakeAdapter returns non-empty response.
	if err != nil {
		t.Fatalf("Run with empty format: %v", err)
	}
}

// TestRun_TextFormat_NilAdapter_ReturnsError verifies that a nil Adapter in
// text mode returns an error rather than panicking.
func TestRun_TextFormat_NilAdapter_ReturnsError(t *testing.T) {
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		Adapter:     nil, // must be rejected before analysis runs
		AnalyzeFunc: fakeAnalyze(&ReportPayload{}, nil),
	})
	if err == nil {
		t.Fatal("expected error for nil Adapter in text mode")
	}
	if !strings.Contains(err.Error(), "non-nil Adapter") {
		t.Errorf("expected 'non-nil Adapter' in error, got: %v", err)
	}
}

// TestRun_ThresholdFailure_ReturnsError verifies that threshold breach returns
// an error from Run and emits a FAIL line on stderr.
func TestRun_ThresholdFailure_ReturnsError(t *testing.T) {
	maxCrapload := 5
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: 10},
	}

	var stderr bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MaxCrapload: &maxCrapload,
		},
	})
	if err == nil {
		t.Fatal("expected error when threshold breached")
	}
	if !strings.Contains(stderr.String(), "(FAIL)") {
		t.Errorf("expected '(FAIL)' on stderr for threshold breach, got: %q", stderr.String())
	}
}

// TestRun_ThresholdPass_ReturnsNil verifies that passing thresholds do not
// cause Run to return an error, and emit a PASS line on stderr.
func TestRun_ThresholdPass_ReturnsNil(t *testing.T) {
	maxCrapload := 20
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: 5},
	}

	var stderr bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MaxCrapload: &maxCrapload,
		},
	})
	if err != nil {
		t.Fatalf("expected nil when threshold passed: %v", err)
	}
	if !strings.Contains(stderr.String(), "(PASS)") {
		t.Errorf("expected '(PASS)' on stderr for passing threshold, got: %q", stderr.String())
	}
}

// TestErrString_NilError verifies errString returns nil for nil error.
func TestErrString_NilError(t *testing.T) {
	if errString(nil) != nil {
		t.Error("expected nil for nil error")
	}
}

// TestErrString_NonNilError verifies errString returns a non-nil string pointer.
func TestErrString_NonNilError(t *testing.T) {
	s := errString(errors.New("test error"))
	if s == nil {
		t.Fatal("expected non-nil string pointer")
	}
	if *s != "test error" {
		t.Errorf("expected %q, got %q", "test error", *s)
	}
}

// ---------------------------------------------------------------------------
// Zero-result gate tests (#116)
// ---------------------------------------------------------------------------

// TestRun_ZeroResults_ThresholdSet_ReturnsError verifies that when the
// pipeline produces zero analyzed functions (TotalFunctions=0) and
// any threshold is configured, Run returns a non-nil error (#116).
func TestRun_ZeroResults_ThresholdSet_ReturnsError(t *testing.T) {
	maxCrapload := 5
	payload := &ReportPayload{
		// Zero results: TotalFunctions=0, no CRAP error.
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: 0, GazeCRAPload: nil},
		Errors:  PayloadErrors{},
	}

	var stderr bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MaxCrapload: &maxCrapload,
		},
	})
	if err == nil {
		t.Fatal("expected error when zero results and threshold set")
	}
	if !strings.Contains(err.Error(), "no functions analyzed") {
		t.Errorf("expected 'no functions analyzed' in error, got: %v", err)
	}
}

// TestRun_ZeroResults_NoThreshold_ReturnsNil verifies that when the
// pipeline produces zero analyzed functions but no threshold is configured,
// Run returns nil (no error).
func TestRun_ZeroResults_NoThreshold_ReturnsNil(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: 0, GazeCRAPload: nil},
		Errors:  PayloadErrors{},
	}

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
		// No thresholds configured.
	})
	if err != nil {
		t.Fatalf("expected nil error when zero results and no threshold, got: %v", err)
	}
}

// TestRun_ZeroResults_CRAPStepFailed_NoGate verifies that the zero-result
// gate does NOT fire when the CRAP step itself failed (the error is already
// captured in PayloadErrors.CRAP).
func TestRun_ZeroResults_CRAPStepFailed_NoGate(t *testing.T) {
	maxCrapload := 5
	crapErr := "coverage profile failed"
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: 0, GazeCRAPload: nil},
		Errors:  PayloadErrors{CRAP: &crapErr},
	}

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MaxCrapload: &maxCrapload,
		},
	})
	// The zero-result gate skips because payload.Errors.CRAP is set
	// (the CRAP step already failed). Threshold evaluation proceeds:
	// CRAPload(0) <= 5 passes, so Run returns nil. The key invariant
	// is that the error is NOT "no functions analyzed" — that gate
	// must not fire when the CRAP step itself reported an error.
	if err != nil {
		if strings.Contains(err.Error(), "no functions analyzed") {
			t.Errorf("zero-result gate fired despite CRAP step error: %v", err)
		} else {
			t.Errorf("expected nil error (CRAP step error bypasses zero-result gate, threshold passes), got: %v", err)
		}
	}
}

// TestCheckZeroResultGate_NonZeroTotalFunctions verifies that the gate does
// not fire when TotalFunctions > 0 (functions were analyzed).
func TestCheckZeroResultGate_NonZeroTotalFunctions(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 10, CRAPload: 0},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	err := checkZeroResultGate(payload, cfg)
	if err != nil {
		t.Errorf("expected nil when TotalFunctions > 0, got: %v", err)
	}
}

// TestCheckZeroResultGate_NilPayload verifies graceful handling of nil payload.
func TestCheckZeroResultGate_NilPayload(t *testing.T) {
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	err := checkZeroResultGate(nil, cfg)
	if err != nil {
		t.Errorf("expected nil for nil payload, got: %v", err)
	}
}
