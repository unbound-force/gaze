// Package aireport implements the AI-powered CI quality report pipeline
// for the gaze report subcommand. It orchestrates gaze's four analysis
// operations, assembles a combined JSON payload, and delegates formatting
// to an external AI adapter (claude, gemini, ollama, or opencode).
package aireport

import "encoding/json"

// ReportSummary holds pre-extracted threshold-relevant metric values from
// the analysis pipeline. It is populated during Run() so that
// EvaluateThresholds can compare typed int values without unmarshalling
// the raw JSON fields. ReportSummary is internal to the pipeline; it is
// NOT serialised to the --format=json output (json:"-").
type ReportSummary struct {
	// TotalFunctions is the total number of functions analyzed by the
	// CRAP step. Zero means no functions were found (e.g., package
	// pattern matched nothing). Used by the zero-result gate (#116).
	TotalFunctions int

	// CRAPload is the number of functions whose CRAP score meets or exceeds
	// the configured threshold (from crap.Report.Summary.CRAPload).
	// Nil means the metric was not computed (e.g., CRAP step failed).
	CRAPload *int

	// GazeCRAPload is the number of Q4 functions (high complexity, low
	// coverage) from crap.Report.Summary.GazeCRAPload. Nil means the
	// metric was not computed (e.g., no contract coverage callback).
	GazeCRAPload *int

	// AvgContractCoverage is the average contract coverage percentage across
	// all assessed packages (from quality.Summary.AvgContractCoverage).
	// Nil means the metric was not computed (e.g., quality step failed).
	AvgContractCoverage *int

	// SSADegraded is true when SSA construction failed for one or
	// more packages and quality results are partial.
	SSADegraded bool

	// SSADegradedPackages lists the package paths where SSA
	// construction failed.
	SSADegradedPackages []string

	// Contractual is the number of side effects classified as
	// contractual across all analyzed packages.
	Contractual int

	// Ambiguous is the number of side effects classified as
	// ambiguous across all analyzed packages.
	Ambiguous int

	// Incidental is the number of side effects classified as
	// incidental across all analyzed packages.
	Incidental int

	// SkippedTests is the number of test functions that were skipped
	// because no target function could be resolved (e.g., BDD/Ginkgo
	// suites using dynamic dispatch).
	SkippedTests int

	// SkippedTestNames lists the names of skipped test functions.
	SkippedTestNames []string
}

// ReportPayload is the combined analysis data passed to the AI adapter
// and written to stdout in --format=json mode.
//
// Dual serialization paths:
//   - json.Marshal produces the full payload for --format=json output,
//     preserving all fields (signals, full score/side-effect arrays, docscan
//     content).
//   - CompactForAI produces a reduced payload for the AI adapter text path,
//     dropping the unbounded arrays (CRAP scores, classify results), bounding
//     quality reports to the most actionable entries, and projecting side
//     effects to self-contained objects (id, type, tier, location,
//     description, target) while preserving the bounded worst-offender lists
//     to fit within model context windows.
type ReportPayload struct {
	// Summary holds pre-extracted threshold-relevant values, populated during
	// pipeline execution. Used by EvaluateThresholds to avoid unmarshalling
	// the raw JSON fields. Not serialised in --format=json output.
	Summary ReportSummary `json:"-"`

	// Metadata identifies the report subject (module, branch, gaze
	// version, date, selected sections). It feeds the deterministic
	// markdown header.
	Metadata ReportMetadata `json:"metadata"`

	// CRAP holds the raw JSON from gaze crap --format=json.
	// Nil when the CRAP analysis step failed.
	CRAP json.RawMessage `json:"crap"`

	// Quality holds the raw JSON from gaze quality --format=json.
	// Nil when the quality analysis step failed.
	Quality json.RawMessage `json:"quality"`

	// Classify holds the raw JSON from gaze analyze --classify --format=json.
	// Nil when the classification step failed.
	Classify json.RawMessage `json:"classify"`

	// Docscan holds the raw JSON from the docscan step, a docscan envelope
	// ({"documents": [...], "api_coverage": ...}). Nil when the docscan step
	// failed.
	Docscan json.RawMessage `json:"docscan"`

	// Errors records step-level failures. A nil pointer value means the
	// step succeeded. A non-nil pointer is the error message string.
	Errors PayloadErrors `json:"errors"`

	// Assessment holds the Go-computed grades, severities, and
	// deterministic recommendations. Nil until the assessment is
	// computed; serialised as null in that case.
	Assessment *Assessment `json:"assessment"`
}

// ReportMetadata identifies the subject of a report. All values are
// produced by gaze itself (never by an AI adapter).
type ReportMetadata struct {
	// ModulePath is the Go module path from go.mod.
	ModulePath string `json:"module_path"`

	// Branch is the CI branch name when known; omitted otherwise.
	Branch string `json:"branch,omitempty"`

	// GazeVersion is the gaze build version that produced the report.
	GazeVersion string `json:"gaze_version"`

	// Date is the report date in YYYY-MM-DD (UTC).
	Date string `json:"date"`

	// Sections lists the selected report sections in canonical order.
	Sections []Section `json:"sections"`
}

// PayloadErrors records per-step failure messages for ReportPayload.
// A nil field means the step succeeded; a non-nil field is the error
// message string.
type PayloadErrors struct {
	CRAP     *string `json:"crap"`
	Quality  *string `json:"quality"`
	Classify *string `json:"classify"`
	Docscan  *string `json:"docscan"`
}

// ThresholdConfig holds the CI gate thresholds for gaze report.
// A nil field means the threshold was not provided on the command line
// and is disabled. A non-nil field with value 0 means "fail if any
// function exceeds this" — zero is a valid live threshold (not "disabled").
type ThresholdConfig struct {
	// MaxCrapload fails the command if the project CRAPload exceeds this value.
	MaxCrapload *int

	// MaxGazeCrapload fails the command if the GazeCRAPload exceeds this value.
	MaxGazeCrapload *int

	// MinContractCoverage fails the command if average contract coverage
	// (as a percentage integer) is below this value.
	MinContractCoverage *int
}

// ThresholdResult records the evaluation outcome for one threshold.
type ThresholdResult struct {
	// Name is the human-readable threshold label (e.g., "CRAPload").
	Name string

	// Actual is the measured value extracted from the ReportPayload.
	// Nil means the metric was unavailable (e.g., GazeCRAPload not computed).
	Actual *int

	// Limit is the configured threshold value.
	Limit int

	// Passed is true when Actual is within the acceptable range.
	Passed bool
}
