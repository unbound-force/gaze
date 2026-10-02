package aireport

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWriteHTML_AllStepsSucceed(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			TotalFunctions:      12,
			CRAPload:            intPtr(3),
			GazeCRAPload:        intPtr(2),
			AvgContractCoverage: intPtr(74),
			SkippedTests:        1,
			Contractual:         8,
			Ambiguous:           3,
			Incidental:          4,
			SSADegraded:         true,
			SSADegradedPackages: []string{"example.com/degraded"},
		},
		CRAP:     json.RawMessage(`{"scores":[{"function":"Risky","crap":31.5}]}`),
		Quality:  json.RawMessage(`{"quality_reports":[{"test_function":"TestRisky"}]}`),
		Classify: json.RawMessage(`{"results":[{"function":"Risky","label":"contractual"}]}`),
		Docscan:  json.RawMessage(`{"documents":[{"path":"README.md"}]}`),
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, payload); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	required := []string{
		"<!DOCTYPE html>", `<html lang="en">`, `<meta charset="utf-8">`,
		"<title>Gaze Combined Report</title>", "<body>", "</body>", "</html>",
		"Functions analyzed</span><strong>12", "CRAPload</span><strong>3",
		"GazeCRAPload</span><strong>2", "Average contract coverage</span><strong>74%",
		"Skipped tests</span><strong>1", "Contractual</span><strong>8",
		"Ambiguous</span><strong>3", "Incidental</span><strong>4",
		"SSA analysis degraded.", "example.com/degraded",
		`<h2 id="step-crap">CRAP</h2>`, `<h2 id="step-quality">Quality</h2>`,
		`<h2 id="step-classification">Classification</h2>`, `<h2 id="step-documentation">Documentation</h2>`,
		"Risky", "31.5", "TestRisky", "contractual", "README.md",
	}
	for _, value := range required {
		if !strings.Contains(html, value) {
			t.Errorf("expected combined HTML to contain %q", value)
		}
	}
	if strings.Count(html, "Pipeline output") != 4 {
		t.Errorf("expected four successful pipeline outputs, got %d", strings.Count(html, "Pipeline output"))
	}
}

func TestWriteHTML_PartialFailurePreservesSuccessfulSteps(t *testing.T) {
	qualityError := "quality analysis failed: no test mappings"
	payload := &ReportPayload{
		Summary: ReportSummary{
			TotalFunctions: 4,
			CRAPload:       intPtr(1),
			GazeCRAPload:   intPtr(1),
			Contractual:    2,
		},
		CRAP:     json.RawMessage(`{"step":"crap-success"}`),
		Classify: json.RawMessage(`{"step":"classification-success"}`),
		Docscan:  json.RawMessage(`{"step":"documentation-success"}`),
		Errors:   PayloadErrors{Quality: &qualityError},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, payload); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	for _, value := range []string{
		`<h2 id="step-quality">Quality</h2>`, "Step failed.", qualityError,
		"crap-success", "classification-success", "documentation-success",
	} {
		if !strings.Contains(html, value) {
			t.Errorf("expected partial-failure HTML to contain %q", value)
		}
	}
	if strings.Count(html, "Pipeline output") != 3 {
		t.Errorf("expected three successful pipeline outputs, got %d", strings.Count(html, "Pipeline output"))
	}
	if strings.Contains(html, "Average contract coverage</span><strong>0%") ||
		strings.Contains(html, "Skipped tests</span><strong>0") {
		t.Error("failed quality metrics must not be presented as measured zero values")
	}
}

func TestWriteHTML_UnavailableMetrics(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			TotalFunctions: 5,
			CRAPload:       intPtr(0),
		},
		CRAP:     json.RawMessage(`{}`),
		Quality:  json.RawMessage(`{}`),
		Classify: json.RawMessage(`{}`),
		Docscan:  json.RawMessage(`{}`),
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, payload); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	for _, available := range []string{
		"Functions analyzed</span><strong>5", "CRAPload</span><strong>0",
		"Skipped tests</span><strong>0", "Contractual</span><strong>0",
	} {
		if !strings.Contains(html, available) {
			t.Errorf("expected available zero metric %q", available)
		}
	}
	for _, unavailable := range []string{
		"GazeCRAPload</span><strong class=\"unavailable\">Unavailable",
		"Average contract coverage</span><strong class=\"unavailable\">Unavailable",
	} {
		if !strings.Contains(html, unavailable) {
			t.Errorf("expected unavailable metric marker %q", unavailable)
		}
	}
	if strings.Contains(html, "GazeCRAPload</span><strong>0") ||
		strings.Contains(html, "Average contract coverage</span><strong>0%") {
		t.Error("unavailable pointer metrics must not be presented as measured zero values")
	}
}

func TestWriteHTML_MalformedSuccessfulStepPayload(t *testing.T) {
	tests := []struct {
		name        string
		setPayload  func(*ReportPayload)
		stepContext string
	}{
		{name: "CRAP", setPayload: func(payload *ReportPayload) { payload.CRAP = json.RawMessage(`{"broken"`) }, stepContext: "formatting CRAP step JSON"},
		{name: "Quality", setPayload: func(payload *ReportPayload) { payload.Quality = json.RawMessage(`{"broken"`) }, stepContext: "formatting Quality step JSON"},
		{name: "Classification", setPayload: func(payload *ReportPayload) { payload.Classify = json.RawMessage(`{"broken"`) }, stepContext: "formatting Classification step JSON"},
		{name: "Documentation", setPayload: func(payload *ReportPayload) { payload.Docscan = json.RawMessage(`{"broken"`) }, stepContext: "formatting Documentation step JSON"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload := &ReportPayload{
				CRAP:     json.RawMessage(`{}`),
				Quality:  json.RawMessage(`{}`),
				Classify: json.RawMessage(`{}`),
				Docscan:  json.RawMessage(`{}`),
			}
			test.setPayload(payload)

			var output bytes.Buffer
			err := WriteHTML(&output, payload)
			if err == nil {
				t.Fatal("expected malformed successful step payload to fail")
			}
			if !strings.Contains(err.Error(), "preparing combined HTML report data") ||
				!strings.Contains(err.Error(), test.stepContext) {
				t.Errorf("expected operation and step context, got %v", err)
			}
			if output.Len() != 0 {
				t.Errorf("malformed payload must not produce a fabricated document, got %d bytes", output.Len())
			}
		})
	}
}

func TestWriteHTML_AdversarialEscapingAndSelfContainment(t *testing.T) {
	adversarialData := `</pre><script>alert(data)</script>&<img src=x onerror=alert(1)>`
	adversarialError := `</div><script>alert("error")</script>&<svg onload=alert(1)>`
	payload := &ReportPayload{
		Summary: ReportSummary{
			SSADegraded:         true,
			SSADegradedPackages: []string{adversarialData},
		},
		CRAP:     json.RawMessage(`{"value":"\u003c/script\u003e safe & data"}`),
		Quality:  json.RawMessage(`{"value":"` + adversarialData + `"}`),
		Classify: json.RawMessage(`{"value":"classification"}`),
		Errors:   PayloadErrors{Docscan: &adversarialError},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, payload); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if strings.Contains(html, adversarialData) || strings.Contains(html, adversarialError) ||
		strings.Contains(html, `<script>alert`) || strings.Contains(html, `<img src=x`) || strings.Contains(html, `<svg onload`) {
		t.Error("analyzer-derived or error markup was rendered as active HTML")
	}
	for _, escaped := range []string{
		`&lt;/pre&gt;&lt;script&gt;alert(data)&lt;/script&gt;&amp;&lt;img`,
		`&lt;/div&gt;&lt;script&gt;alert(&#34;error&#34;)&lt;/script&gt;&amp;&lt;svg`,
	} {
		if !strings.Contains(html, escaped) {
			t.Errorf("expected adversarial content to appear as escaped text: %q", escaped)
		}
	}
	for _, forbidden := range []string{"<link ", "<script", "<img", "<svg", "@import", "http://", "https://", "url("} {
		if strings.Contains(html, forbidden) {
			t.Errorf("self-contained HTML must not contain %q", forbidden)
		}
	}
	if strings.Count(html, "<style>") != 1 {
		t.Errorf("self-contained HTML should contain one inline style block, got %d", strings.Count(html, "<style>"))
	}
}

func TestWriteHTML_DeterministicBytes(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			TotalFunctions:      2,
			CRAPload:            intPtr(1),
			GazeCRAPload:        intPtr(1),
			AvgContractCoverage: intPtr(50),
		},
		CRAP:     json.RawMessage(`{"z":2,"a":1}`),
		Quality:  json.RawMessage(`[{"test":"TestOne"}]`),
		Classify: json.RawMessage(`{"labels":{"contractual":1}}`),
		Docscan:  json.RawMessage(`{"documents":[]}`),
	}

	var first bytes.Buffer
	if err := WriteHTML(&first, payload); err != nil {
		t.Fatalf("first WriteHTML call failed: %v", err)
	}
	var second bytes.Buffer
	if err := WriteHTML(&second, payload); err != nil {
		t.Fatalf("second WriteHTML call failed: %v", err)
	}

	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("identical ordered input produced different HTML bytes")
	}
}

func TestWriteHTML_WriterFailure(t *testing.T) {
	err := WriteHTML(combinedErrorWriter{}, &ReportPayload{})
	if err == nil {
		t.Fatal("expected WriteHTML to return the writer failure")
	}
	if !errors.Is(err, errCombinedWrite) {
		t.Errorf("expected wrapped writer error, got %v", err)
	}
	if !strings.Contains(err.Error(), "executing combined HTML report template") {
		t.Errorf("expected operation-specific context, got %v", err)
	}
}

var errCombinedWrite = errors.New("combined writer failed")

type combinedErrorWriter struct{}

func (combinedErrorWriter) Write([]byte) (int, error) {
	return 0, errCombinedWrite
}
