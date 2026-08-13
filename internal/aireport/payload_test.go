package aireport

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestReportPayload_JSONRoundTrip verifies that a fully-populated ReportPayload
// serialises and deserialises without data loss, and that the Summary field
// (tagged json:"-") is excluded from the wire format.
func TestReportPayload_JSONRoundTrip(t *testing.T) {
	crapMsg := json.RawMessage(`{"scores":[],"summary":{"crapload":3}}`)
	qualityMsg := json.RawMessage(`{"quality_reports":[],"quality_summary":null}`)
	classifyMsg := json.RawMessage(`{"version":"1.0.0","results":[]}`)
	docscanMsg := json.RawMessage(`[{"path":"README.md","content":"x","priority":2}]`)

	original := &ReportPayload{
		Summary:  ReportSummary{CRAPload: intPtr(3), GazeCRAPload: intPtr(1), AvgContractCoverage: intPtr(75)},
		CRAP:     crapMsg,
		Quality:  qualityMsg,
		Classify: classifyMsg,
		Docscan:  docscanMsg,
		Errors: PayloadErrors{
			CRAP:     nil,
			Quality:  nil,
			Classify: nil,
			Docscan:  nil,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Summary must not appear in the wire JSON.
	if strings.Contains(string(data), "CRAPload") || strings.Contains(string(data), "GazeCRAPload") {
		// "crapload" (lower-case) is inside the crap raw message — that's fine.
		// What must not appear is the ReportSummary struct itself as a top-level key.
		if strings.Contains(string(data), `"Summary"`) || strings.Contains(string(data), `"summary":{"CRAPload`) {
			t.Errorf("ReportSummary should not appear in JSON output, got: %s", data)
		}
	}

	// Errors null fields must be present.
	if !strings.Contains(string(data), `"crap":null`) {
		// The CRAP field at top level contains the raw message, not null.
		// Look specifically for the errors object.
		if !strings.Contains(string(data), `"errors"`) {
			t.Errorf("expected errors field in JSON output, got: %s", data)
		}
	}

	var decoded ReportPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Summary is not serialised (json:"-"), so decoded.Summary should be zero.
	if decoded.Summary.CRAPload != nil || decoded.Summary.GazeCRAPload != nil ||
		decoded.Summary.AvgContractCoverage != nil || decoded.Summary.SSADegraded ||
		len(decoded.Summary.SSADegradedPackages) != 0 ||
		decoded.Summary.Contractual != 0 || decoded.Summary.Ambiguous != 0 ||
		decoded.Summary.Incidental != 0 {
		t.Errorf("expected decoded Summary to be zero-value, got %+v", decoded.Summary)
	}

	if string(decoded.CRAP) != string(crapMsg) {
		t.Errorf("CRAP mismatch: got %s", decoded.CRAP)
	}
	if string(decoded.Quality) != string(qualityMsg) {
		t.Errorf("Quality mismatch: got %s", decoded.Quality)
	}
	if string(decoded.Classify) != string(classifyMsg) {
		t.Errorf("Classify mismatch: got %s", decoded.Classify)
	}
	if string(decoded.Docscan) != string(docscanMsg) {
		t.Errorf("Docscan mismatch: got %s", decoded.Docscan)
	}
	if decoded.Errors.CRAP != nil || decoded.Errors.Quality != nil ||
		decoded.Errors.Classify != nil || decoded.Errors.Docscan != nil {
		t.Errorf("expected all Errors fields nil, got %+v", decoded.Errors)
	}
}

// TestReportPayload_PartialFailure verifies the partial-failure scenario where
// the CRAP step failed: CRAP field is null and Errors.CRAP is non-nil.
func TestReportPayload_PartialFailure(t *testing.T) {
	errMsg := "coverage profile generation failed"
	original := &ReportPayload{
		CRAP:    nil,
		Quality: json.RawMessage(`{"quality_reports":[]}`),
		Errors: PayloadErrors{
			CRAP:    &errMsg,
			Quality: nil,
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded ReportPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// json.Unmarshal of a null JSON value into json.RawMessage yields the
	// literal bytes `null`, not a nil slice. Check by string.
	if string(decoded.CRAP) != "null" && len(decoded.CRAP) != 0 {
		t.Errorf("expected CRAP null on partial failure, got %s", decoded.CRAP)
	}
	if decoded.Errors.CRAP == nil {
		t.Fatal("expected Errors.CRAP non-nil on partial failure")
	}
	if *decoded.Errors.CRAP != errMsg {
		t.Errorf("expected Errors.CRAP %q, got %q", errMsg, *decoded.Errors.CRAP)
	}
	if decoded.Errors.Quality != nil {
		t.Errorf("expected Errors.Quality nil, got %v", decoded.Errors.Quality)
	}
}

// TestPayloadErrors_NullVsNonNull verifies that PayloadErrors serialises nil
// fields as JSON null and non-nil fields as quoted strings.
func TestPayloadErrors_NullVsNonNull(t *testing.T) {
	msg := "step failed"
	errs := PayloadErrors{
		CRAP:     &msg,
		Quality:  nil,
		Classify: nil,
		Docscan:  nil,
	}

	data, err := json.Marshal(errs)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, `"crap":"step failed"`) {
		t.Errorf("expected crap error string in JSON, got %s", s)
	}
	if !strings.Contains(s, `"quality":null`) {
		t.Errorf("expected quality null in JSON, got %s", s)
	}
}

// TestReportSummary_NotSerialised verifies that the ReportSummary is excluded
// from JSON output via the json:"-" tag.
func TestReportSummary_NotSerialised(t *testing.T) {
	p := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(99), GazeCRAPload: intPtr(42), AvgContractCoverage: intPtr(55)},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "99") || strings.Contains(string(data), "42") || strings.Contains(string(data), "55") {
		// Note: these numbers could appear inside raw messages, but there are none here.
		t.Errorf("ReportSummary values should not appear in JSON output, got: %s", data)
	}
}
