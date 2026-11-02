package aireport

import (
	_ "embed"
	"strings"
)

// reportModePromptRaw is the embedded, tool-free judgment prompt used for
// headless gaze report runs. It is never scaffolded into a target project,
// so a project's interactive gaze-reporter.md cannot change what the report
// pipeline asks for.
//
//go:embed assets/report-mode-prompt.md
var reportModePromptRaw string

// ReportModePrompt returns the system prompt for headless report runs.
//
// The prompt asks for judgment only: Gaze renders every table, grade, and
// deterministic recommendation itself, and the model answers with exactly
// one JSON object carrying a summary and up to five recommendations in
// English. The result is validated by ParseAIAnswer before it is rendered.
func ReportModePrompt() string {
	return strings.TrimRight(reportModePromptRaw, "\n")
}
