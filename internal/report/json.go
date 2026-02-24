// Package report provides output formatters for Gaze analysis
// results in JSON and human-readable text formats.
package report

import (
	"encoding/json"
	"io"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// JSONReport is the top-level JSON output structure.
type JSONReport struct {
	Version string                    `json:"version"`
	Results []taxonomy.AnalysisResult `json:"results"`
}

// WriteJSON writes analysis results as formatted JSON to the writer.
// The version string is embedded in the JSON output; if empty,
// it defaults to "dev".
func WriteJSON(w io.Writer, results []taxonomy.AnalysisResult, version string) error {
	if results == nil {
		results = []taxonomy.AnalysisResult{}
	}
	if version == "" {
		version = "dev"
	}
	report := JSONReport{
		Version: version,
		Results: results,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
