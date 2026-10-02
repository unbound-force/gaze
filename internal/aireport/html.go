package aireport

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
)

//go:embed report.html.tmpl
var reportTemplateFS embed.FS

type reportHTMLData struct {
	Summary reportHTMLSummary
	Steps   []reportHTMLStep
}

type reportHTMLSummary struct {
	TotalFunctions      reportHTMLInt
	CRAPload            reportHTMLInt
	GazeCRAPload        reportHTMLInt
	AvgContractCoverage reportHTMLInt
	SkippedTests        reportHTMLInt
	Classification      reportHTMLClassification
	SSADegraded         bool
	SSADegradedPackages []string
}

type reportHTMLInt struct {
	Available bool
	Value     int
}

type reportHTMLClassification struct {
	Available   bool
	Contractual int
	Ambiguous   int
	Incidental  int
}

type reportHTMLStep struct {
	ID     string
	Name   string
	JSON   string
	Error  string
	Failed bool
}

// WriteHTML writes a combined pipeline payload as a complete, self-contained
// HTML document to w. Headline values come from ReportPayload.Summary, while
// successful step payloads are validated and deterministically indented as
// JSON. Failed steps render their recorded error instead. Dynamic values remain
// plain strings so html/template applies contextual escaping. It returns an
// error for malformed successful step payloads or HTML rendering failures.
func WriteHTML(w io.Writer, payload *ReportPayload) error {
	data, err := buildReportHTMLData(payload)
	if err != nil {
		return fmt.Errorf("preparing combined HTML report data: %w", err)
	}

	tmpl, err := template.ParseFS(reportTemplateFS, "report.html.tmpl")
	if err != nil {
		return fmt.Errorf("parsing combined HTML report template: %w", err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing combined HTML report template: %w", err)
	}
	return nil
}

func buildReportHTMLData(payload *ReportPayload) (reportHTMLData, error) {
	if payload == nil {
		return reportHTMLData{}, fmt.Errorf("report payload is nil")
	}

	data := reportHTMLData{
		Summary: reportHTMLSummary{
			TotalFunctions:      reportHTMLInt{Available: payload.Errors.CRAP == nil, Value: payload.Summary.TotalFunctions},
			CRAPload:            reportHTMLIntValue(payload.Summary.CRAPload, payload.Errors.CRAP == nil),
			GazeCRAPload:        reportHTMLIntValue(payload.Summary.GazeCRAPload, payload.Errors.CRAP == nil),
			AvgContractCoverage: reportHTMLIntValue(payload.Summary.AvgContractCoverage, payload.Errors.Quality == nil),
			SkippedTests:        reportHTMLInt{Available: payload.Errors.Quality == nil, Value: payload.Summary.SkippedTests},
			Classification: reportHTMLClassification{
				Available:   payload.Errors.Classify == nil,
				Contractual: payload.Summary.Contractual,
				Ambiguous:   payload.Summary.Ambiguous,
				Incidental:  payload.Summary.Incidental,
			},
			SSADegraded:         payload.Summary.SSADegraded,
			SSADegradedPackages: payload.Summary.SSADegradedPackages,
		},
	}

	steps := []struct {
		id      string
		name    string
		payload json.RawMessage
		err     *string
	}{
		{id: "crap", name: "CRAP", payload: payload.CRAP, err: payload.Errors.CRAP},
		{id: "quality", name: "Quality", payload: payload.Quality, err: payload.Errors.Quality},
		{id: "classification", name: "Classification", payload: payload.Classify, err: payload.Errors.Classify},
		{id: "documentation", name: "Documentation", payload: payload.Docscan, err: payload.Errors.Docscan},
	}
	for _, step := range steps {
		view, err := buildReportHTMLStep(step.id, step.name, step.payload, step.err)
		if err != nil {
			return reportHTMLData{}, err
		}
		data.Steps = append(data.Steps, view)
	}

	return data, nil
}

func buildReportHTMLStep(id, name string, payload json.RawMessage, stepErr *string) (reportHTMLStep, error) {
	view := reportHTMLStep{ID: id, Name: name}
	if stepErr != nil {
		view.Failed = true
		view.Error = *stepErr
		return view, nil
	}
	if len(payload) == 0 {
		return view, nil
	}

	var formatted bytes.Buffer
	if err := json.Indent(&formatted, payload, "", "  "); err != nil {
		return reportHTMLStep{}, fmt.Errorf("formatting %s step JSON: %w", name, err)
	}
	view.JSON = formatted.String()
	return view, nil
}

func reportHTMLIntValue(value *int, stepSucceeded bool) reportHTMLInt {
	if value == nil || !stepSucceeded {
		return reportHTMLInt{}
	}
	return reportHTMLInt{Available: true, Value: *value}
}
