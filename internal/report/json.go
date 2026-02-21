// Package report provides output formatters for Gaze analysis
// results in JSON and human-readable text formats.
package report

import (
	"encoding/json"
	"io"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// JSONReport is the top-level JSON output structure.
type JSONReport struct {
	Version string                    `json:"version"`
	Results []taxonomy.AnalysisResult `json:"results"`
}

// WriteJSON writes analysis results as formatted JSON to the writer.
func WriteJSON(w io.Writer, results []taxonomy.AnalysisResult) error {
	if results == nil {
		results = []taxonomy.AnalysisResult{}
	}
	report := JSONReport{
		Version: "0.1.0",
		Results: results,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
