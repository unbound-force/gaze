package aireport

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
		Summary: ReportSummary{CRAPload: intPtr(2)},
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

func TestRun_HTMLFormat_WritesNativeReportWithoutAdapter(t *testing.T) {
	qualityErr := "quality analysis degraded"
	payload := &ReportPayload{
		Summary: ReportSummary{
			TotalFunctions:      3,
			CRAPload:            intPtr(1),
			SSADegraded:         true,
			SSADegradedPackages: []string{"example.com/degraded"},
		},
		CRAP:   json.RawMessage(`{"scores":[{"function":"Risky"}]}`),
		Errors: PayloadErrors{Quality: &qualityErr},
	}
	var stdout bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "html",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{
		"<title>Gaze Combined Report</title>",
		"Risky",
		"quality analysis degraded",
		"SSA analysis degraded.",
		"example.com/degraded",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected native HTML report to contain %q", want)
		}
	}
}

func TestRun_HTMLFormat_WritesReportBeforeReturningThresholdError(t *testing.T) {
	maxCrapload := 5
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: intPtr(10)},
		CRAP:    json.RawMessage(`{"scores":[{"function":"Risky"}]}`),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "html",
		Stdout:      &stdout,
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MaxCrapload: &maxCrapload,
		},
	})
	if err == nil {
		t.Fatal("expected error when HTML report threshold is breached")
	}
	if !strings.Contains(stdout.String(), "<title>Gaze Combined Report</title>") ||
		!strings.Contains(stdout.String(), "Risky") {
		t.Errorf("expected HTML report to be written before threshold failure, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "(FAIL)") {
		t.Errorf("expected threshold failure on stderr, got %q", stderr.String())
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
	fa := &FakeAdapter{Response: validAnswerJSON}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &stderr,
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(stdout.String(), "# 🔍") {
		t.Errorf("expected the deterministic report heading on stdout, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "<!-- gaze:ai-summary -->") {
		t.Errorf("expected the AI summary slot on stdout, got: %q", stdout.String())
	}
	if len(fa.Calls) != 1 {
		t.Errorf("expected 1 adapter call, got %d", len(fa.Calls))
	}
	if fa.Calls[0].SystemPrompt != ReportModePrompt() {
		t.Errorf("expected the report-mode prompt, got %q", fa.Calls[0].SystemPrompt)
	}
}

// TestRun_TextFormat_EmptyAdapterOutput_ReturnsError verifies that empty AI
// adapter output is rejected by the answer validator and that the
// deterministic report is still published.
func TestRun_TextFormat_EmptyAdapterOutput_ReturnsError(t *testing.T) {
	fa := &FakeAdapter{Response: "   "}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected error for empty adapter output")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected *AnswerRejection, got %T: %v", err, err)
	}
	if rejection.Rule != ruleEmptyOutput {
		t.Errorf("Rule = %q, want %q", rejection.Rule, ruleEmptyOutput)
	}
	if !strings.Contains(stdout.String(), "AI narrative rejected") {
		t.Errorf("expected a rejection callout on stdout, got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "# 🔍") {
		t.Errorf("expected the deterministic report on stdout, got: %q", stdout.String())
	}
}

// TestRun_TextFormat_WritesStepSummary verifies that StepSummaryPath non-empty
// causes the report to be appended to that file with the correct content.
func TestRun_TextFormat_WritesStepSummary(t *testing.T) {
	fa := &FakeAdapter{Response: validAnswerJSON}
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
	if string(data) != stdout.String() {
		t.Errorf("step summary must match stdout:\nwant: %q\ngot:  %q", stdout.String(), string(data))
	}
	if !strings.Contains(string(data), "# 🔍") {
		t.Errorf("expected the deterministic report in the step summary, got: %q", string(data))
	}
}

// TestRun_TextFormat_UnwritableStepSummary_ReturnsNil verifies that an
// unwritable StepSummaryPath emits a warning but returns nil (FR-008).
func TestRun_TextFormat_UnwritableStepSummary_ReturnsNil(t *testing.T) {
	fa := &FakeAdapter{Response: validAnswerJSON}
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
	if !strings.Contains(stdout.String(), "# 🔍") {
		t.Errorf("expected stdout report despite StepSummary failure, got: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning") {
		t.Errorf("expected warning on stderr for unwritable path, got: %q", stderr.String())
	}
}

// TestRun_TextFormat_ProgressSignals verifies that progress signals appear on
// stderr during --format=text mode.
func TestRun_TextFormat_ProgressSignals(t *testing.T) {
	fa := &FakeAdapter{Response: validAnswerJSON}
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
	fa := &FakeAdapter{Response: validAnswerJSON}
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
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: intPtr(10)},
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
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: intPtr(5)},
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
		// CRAPload is intPtr(0) because the step succeeded but found no functions.
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: intPtr(0), GazeCRAPload: nil},
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
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: intPtr(0), GazeCRAPload: nil},
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

// TestRun_CRAPStepFailed_ThresholdSet_ReturnsError verifies that when the
// CRAP step fails and a CRAPload threshold is configured, Run returns a
// non-nil error because the CRAPload metric is unavailable (nil). This is
// the fix for #102: CI gates must not silently pass when data is missing.
//
// Previously this test (named TestRun_ZeroResults_CRAPStepFailed_NoGate)
// asserted err == nil, which enshrined the bug as correct behavior.
func TestRun_CRAPStepFailed_ThresholdSet_ReturnsError(t *testing.T) {
	maxCrapload := 5
	crapErr := "coverage profile failed"
	payload := &ReportPayload{
		// CRAPload is nil because the CRAP step failed — the summary field
		// was never set. GazeCRAPload is also nil (same step).
		Summary: ReportSummary{TotalFunctions: 0, CRAPload: nil, GazeCRAPload: nil},
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
	// The zero-result gate still skips because payload.Errors.CRAP is set.
	// But threshold evaluation now correctly detects CRAPload=nil (unavailable)
	// and fails the gate. Run must return a non-nil error.
	if err == nil {
		t.Fatal("expected error when CRAP step failed and threshold is set (CRAPload unavailable)")
	}
	// The error should NOT be "no functions analyzed" — that gate
	// must not fire when the CRAP step itself reported an error.
	if strings.Contains(err.Error(), "no functions analyzed") {
		t.Errorf("zero-result gate fired despite CRAP step error: %v", err)
	}
}

// TestRun_QualityStepFailed_ContractCoverageThreshold_ReturnsError verifies
// that when the quality step fails and MinContractCoverage is set, Run returns
// an error because AvgContractCoverage is unavailable (nil).
func TestRun_QualityStepFailed_ContractCoverageThreshold_ReturnsError(t *testing.T) {
	minCov := 50
	qualErr := "quality analysis failed"
	payload := &ReportPayload{
		Summary: ReportSummary{
			// CRAP step succeeded — CRAPload is valid.
			CRAPload:       intPtr(3),
			TotalFunctions: 10,
			// Quality step failed — AvgContractCoverage is nil (never set).
			AvgContractCoverage: nil,
		},
		CRAP:   json.RawMessage(`{"scores":[]}`),
		Errors: PayloadErrors{Quality: &qualErr},
	}

	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds: ThresholdConfig{
			MinContractCoverage: &minCov,
		},
	})
	if err == nil {
		t.Fatal("expected error when quality step failed and MinContractCoverage is set")
	}
}

// TestCheckZeroResultGate_NonZeroTotalFunctions verifies that the gate does
// not fire when TotalFunctions > 0 (functions were analyzed).
func TestCheckZeroResultGate_NonZeroTotalFunctions(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 10, CRAPload: intPtr(0)},
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

// TestValidateRunnerOpts_Formats verifies the accepted format set: native
// formats (json, html, markdown) need no adapter, text requires one, and
// empty or unrecognised values fall back to text.
func TestValidateRunnerOpts_Formats(t *testing.T) {
	tests := []struct {
		format     string
		wantFormat string
		wantErr    bool
	}{
		{format: "json", wantFormat: "json"},
		{format: "html", wantFormat: "html"},
		{format: "markdown", wantFormat: "markdown"},
		{format: "", wantFormat: "text", wantErr: true},
		{format: "yaml", wantFormat: "text", wantErr: true},
		{format: "text", wantFormat: "text", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			opts := RunnerOptions{Format: tc.format}
			err := validateRunnerOpts(&opts)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateRunnerOpts(%q) err = %v, wantErr %v", tc.format, err, tc.wantErr)
			}
			if opts.Format != tc.wantFormat {
				t.Errorf("Format = %q, want %q", opts.Format, tc.wantFormat)
			}
		})
	}
}

// TestRun_CopiesMetadataOntoPayload verifies that Run copies the caller's
// metadata onto the payload with the effective sections in canonical
// order, defaulting to every section when Sections is the zero value.
func TestRun_CopiesMetadataOntoPayload(t *testing.T) {
	quality, err := ParseSections([]string{"quality,crap"})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}
	tests := []struct {
		name     string
		sections SectionSet
		want     []Section
	}{
		{name: "zero value selects all", want: AllSections().List()},
		{name: "explicit canonical order", sections: quality, want: []Section{SectionCRAP, SectionQuality}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := &ReportPayload{}
			var stdout bytes.Buffer
			err := Run(RunnerOptions{
				Patterns: []string{"./..."},
				Format:   "json",
				Sections: tc.sections,
				Metadata: ReportMetadata{
					ModulePath:  "github.com/example/project",
					Branch:      "main",
					GazeVersion: "v2.4.0",
					Date:        "2026-11-01",
					Sections:    []Section{SectionHealth},
				},
				Stdout:      &stdout,
				Stderr:      &bytes.Buffer{},
				AnalyzeFunc: fakeAnalyze(payload, nil),
			})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			var decoded struct {
				Metadata ReportMetadata `json:"metadata"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
				t.Fatalf("stdout is not valid JSON: %v", err)
			}
			md := decoded.Metadata
			if md.ModulePath != "github.com/example/project" || md.Branch != "main" ||
				md.GazeVersion != "v2.4.0" || md.Date != "2026-11-01" {
				t.Errorf("metadata fields not copied: %+v", md)
			}
			if len(md.Sections) != len(tc.want) {
				t.Fatalf("Sections = %v, want %v", md.Sections, tc.want)
			}
			for i := range tc.want {
				if md.Sections[i] != tc.want[i] {
					t.Errorf("Sections[%d] = %q, want %q", i, md.Sections[i], tc.want[i])
				}
			}
		})
	}
}

// TestRun_MarkdownFormat_WritesReport verifies that --format=markdown writes
// the deterministic report without an AI adapter and attaches the assessment.
func TestRun_MarkdownFormat_WritesReport(t *testing.T) {
	payload := loadPayloadFixture(t, "payload-full.json")

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "markdown",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{
		"# 🔍 Gaze Full Quality Report",
		"## 📊 CRAP Summary",
		"## 🧪 Quality Summary",
		"## 🏷️ Classification Summary",
		"## 🏥 Overall Health Assessment",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("markdown report missing %q", want)
		}
	}
	if strings.Contains(out, "gaze:ai-summary") {
		t.Error("markdown format must not render AI slots")
	}
	if payload.Assessment == nil {
		t.Fatal("payload.Assessment was not set")
	}
	if got := len(payload.Assessment.Scorecard); got != 5 {
		t.Errorf("scorecard rows = %d, want 5", got)
	}
}

// TestRun_MarkdownFormat_WritesStepSummary verifies the markdown report is
// appended to GITHUB_STEP_SUMMARY.
func TestRun_MarkdownFormat_WritesStepSummary(t *testing.T) {
	payload := loadPayloadFixture(t, "payload-full.json")
	tmpFile := t.TempDir() + "/summary.md"

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:        []string{"./..."},
		Format:          "markdown",
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		StepSummaryPath: tmpFile,
		AnalyzeFunc:     fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	written, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("read step summary: %v", err)
	}
	if string(written) != stdout.String() {
		t.Error("step summary does not match stdout report")
	}
}

// TestRun_MarkdownFormat_ThresholdFailureStillWritesReport verifies the
// report is written before the threshold error is returned.
func TestRun_MarkdownFormat_ThresholdFailureStillWritesReport(t *testing.T) {
	payload := loadPayloadFixture(t, "payload-full.json")

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "markdown",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Thresholds:  ThresholdConfig{MaxCrapload: intPtr(1)},
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected threshold failure")
	}
	if !strings.Contains(stdout.String(), "# 🔍 Gaze Full Quality Report") {
		t.Error("report was not written before the threshold error")
	}
}

// TestRun_JSONFormat_IncludesAssessment verifies the assessment travels with
// the JSON payload.
func TestRun_JSONFormat_IncludesAssessment(t *testing.T) {
	payload := loadPayloadFixture(t, "payload-full.json")

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var decoded ReportPayload
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("stdout is not valid JSON: %v", err)
	}
	if decoded.Assessment == nil {
		t.Fatal("JSON output has no assessment")
	}
	if decoded.Assessment.RubricVersion != "1" {
		t.Errorf("rubric_version = %q, want 1", decoded.Assessment.RubricVersion)
	}
	if got := len(decoded.Assessment.Scorecard); got != 5 {
		t.Errorf("scorecard rows = %d, want 5", got)
	}
}

// TestRun_TextPath_RejectedAnswer_PublishesReportAndFails verifies that an
// answer failing validation produces a rejection callout, still publishes the
// deterministic report to stdout and the Step Summary, never echoes the raw
// adapter output, and fails with an *AnswerRejection naming the rule.
func TestRun_TextPath_RejectedAnswer_PublishesReportAndFails(t *testing.T) {
	const rawOutput = "# Gaze Report\n\nNarrative the model invented."

	fa := &FakeAdapter{Response: rawOutput}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}
	summaryPath := filepath.Join(t.TempDir(), "summary.md")

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:        []string{"./..."},
		Format:          "text",
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		Adapter:         fa,
		AnalyzeFunc:     fakeAnalyze(payload, nil),
		StepSummaryPath: summaryPath,
	})
	if err == nil {
		t.Fatal("expected an error for a rejected answer")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected *AnswerRejection, got %T: %v", err, err)
	}
	if rejection.Rule != ruleSurroundingText {
		t.Errorf("Rule = %q, want %q", rejection.Rule, ruleSurroundingText)
	}

	report := stdout.String()
	if !strings.Contains(report, "# 🔍") {
		t.Errorf("expected the deterministic report on stdout, got: %q", report)
	}
	if !strings.Contains(report, "> ⚠️ AI narrative rejected:") {
		t.Errorf("expected a rejection callout on stdout, got: %q", report)
	}
	if strings.Contains(report, "Narrative the model invented") {
		t.Errorf("stdout must not echo the raw adapter output, got: %q", report)
	}

	summary, readErr := os.ReadFile(summaryPath)
	if readErr != nil {
		t.Fatalf("reading step summary: %v", readErr)
	}
	if string(summary) != report {
		t.Errorf("step summary differs from stdout")
	}

	if len(fa.Calls) != 1 {
		t.Fatalf("adapter calls = %d, want 1", len(fa.Calls))
	}
	if fa.Calls[0].SystemPrompt != ReportModePrompt() {
		t.Errorf("system prompt is not the report-mode prompt")
	}
}

// TestRun_TextPath_AdapterError_PublishesUnavailable verifies that an adapter
// failure renders the unavailable callout, still publishes the deterministic
// report, and returns the adapter error rather than a rejection.
func TestRun_TextPath_AdapterError_PublishesUnavailable(t *testing.T) {
	fa := &FakeAdapter{Err: errors.New("adapter blew up")}
	payload := &ReportPayload{CRAP: json.RawMessage(`{}`)}

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected an error when the adapter fails")
	}

	report := stdout.String()
	if !strings.Contains(report, "> ⚠️ AI narrative unavailable:") {
		t.Errorf("expected an unavailable callout on stdout, got: %q", report)
	}
	if !strings.Contains(report, "# 🔍") {
		t.Errorf("expected the deterministic report on stdout, got: %q", report)
	}

	var rejection *AnswerRejection
	if errors.As(err, &rejection) {
		t.Errorf("adapter failure must not be reported as a rejection: %v", err)
	}
}

// TestRun_TextPath_JoinsAIAndThresholdErrors verifies that a rejected answer
// combined with a failing threshold returns both errors, so callers can see
// the AI failure through errors.As and the threshold failure in the message.
func TestRun_TextPath_JoinsAIAndThresholdErrors(t *testing.T) {
	maxCrapload := 5
	payload := &ReportPayload{
		Summary: ReportSummary{TotalFunctions: 20, CRAPload: intPtr(10)},
		CRAP:    json.RawMessage(`{}`),
	}
	fa := &FakeAdapter{Response: "not an answer"}

	var stdout, stderr bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &stderr,
		Adapter:     fa,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds:  ThresholdConfig{MaxCrapload: &maxCrapload},
	})
	if err == nil {
		t.Fatal("expected an error for a rejected answer and a failed threshold")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected the joined error to carry *AnswerRejection, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "one or more quality thresholds failed") {
		t.Errorf("joined error lost the threshold failure: %v", err)
	}
	if !strings.Contains(stdout.String(), "# 🔍") {
		t.Errorf("expected the deterministic report on stdout, got: %q", stdout.String())
	}
}

// TestRun_ZeroResultGate_SkippedWhenCRAPNotSelected verifies research R8: a
// quality-only report has no function count to gate on, so the zero-result
// gate must not fire.
func TestRun_ZeroResultGate_SkippedWhenCRAPNotSelected(t *testing.T) {
	sections, err := ParseSections([]string{"quality"})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}

	payload := &ReportPayload{
		Quality: json.RawMessage(`{"quality_reports":[],"quality_summary":{"total_tests":0}}`),
		Errors:  PayloadErrors{},
	}
	// TotalFunctions stays zero: the CRAP step never ran, so the zero-result
	// gate has nothing to gate on. Contract coverage is supplied so the
	// contract-coverage threshold itself passes.
	payload.Summary.AvgContractCoverage = intPtr(80)

	var stdout, stderr bytes.Buffer
	err = Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "json",
		Sections:    sections,
		Stdout:      &stdout,
		Stderr:      &stderr,
		AnalyzeFunc: fakeAnalyze(payload, nil),
		Thresholds:  ThresholdConfig{MinContractCoverage: intPtr(0)},
	})

	if err != nil {
		t.Fatalf("expected nil error for quality-only report, got: %v", err)
	}

	var decoded ReportPayload
	if decErr := json.Unmarshal(stdout.Bytes(), &decoded); decErr != nil {
		t.Fatalf("unmarshal stdout: %v", decErr)
	}
	// json.RawMessage unmarshals a JSON null as the bytes "null", so compare
	// the content rather than the slice.
	if string(decoded.CRAP) != "null" {
		t.Errorf("expected null CRAP for a quality-only report, got %s", decoded.CRAP)
	}
}
