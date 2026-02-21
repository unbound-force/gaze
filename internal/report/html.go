package report

import (
	"fmt"
	"io"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// WriteHTML writes analysis results as a self-contained HTML report
// with embedded SVG visualizations.
//
// Planned features:
//   - Function table with sortable columns
//   - Side effect tier breakdown (SVG pie/bar chart)
//   - Collapsible per-function detail sections
//   - Self-contained single-file HTML (embedded CSS/JS)
//   - Light/dark theme support
//
// This is not yet implemented. Use text or json format instead.
func WriteHTML(_ io.Writer, _ []taxonomy.AnalysisResult) error {
	return fmt.Errorf("HTML report format is not yet implemented")
}
