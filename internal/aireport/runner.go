package aireport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/provider/goprovider"
)

// RunnerOptions configures the report pipeline runner.
type RunnerOptions struct {
	// Patterns is the package pattern(s) to analyze (e.g., "./...").
	Patterns []string

	// ModuleDir is the root of the Go module being analyzed.
	ModuleDir string

	// Adapter is the AI adapter to use for formatting.
	// Required when Format is "text"; ignored for native formats.
	Adapter AIAdapter

	// AdapterCfg holds adapter-specific configuration (timeout, model, etc.).
	AdapterCfg AdapterConfig

	// Format is "text" (default), "json", "html", or "markdown".
	// Only "text" consults the AI adapter.
	Format string

	// Sections selects the report sections to analyse and render.
	// The zero value selects every section (AllSections).
	Sections SectionSet

	// Metadata is the report header metadata. Run copies it onto the
	// payload before any output path, with Sections replaced by the
	// effective section list in canonical order.
	Metadata ReportMetadata

	// Stdout receives the formatted report or combined native payload.
	Stdout io.Writer

	// Stderr receives progress signals, threshold summaries, and warnings.
	Stderr io.Writer

	// Thresholds holds the CI gate configuration. EvaluateThresholds is
	// called after the pipeline completes. Results are printed to Stderr.
	// When any threshold fails, Run returns a non-nil error.
	Thresholds ThresholdConfig

	// StepSummaryPath is the value of $GITHUB_STEP_SUMMARY, if set.
	// Empty means Step Summary output is disabled.
	StepSummaryPath string

	// CoverProfile is the path to a pre-generated Go coverage profile.
	// When non-empty, the CRAP analysis step uses this file directly
	// instead of spawning go test internally (FR-001, FR-002).
	// Empty string means "generate internally" (default behavior, FR-003).
	CoverProfile string

	// TestShort passes -short to the internal go test invocation when true.
	TestShort bool

	// Context is the caller's context, propagated to the analysis steps
	// (notably the docscan analyzer invocation) for cancellation and
	// timeout control. When nil, context.Background() is used.
	Context context.Context

	// AnalyzeFunc overrides the analysis pipeline for testing.
	// When nil, the production pipeline is called.
	AnalyzeFunc func(patterns []string, moduleDir string) (*ReportPayload, error)
}

// validateRunnerOpts applies defaults and validates required fields.
// In text mode, verifies the adapter is non-nil and the binary is on PATH.
func validateRunnerOpts(opts *RunnerOptions) error {
	switch opts.Format {
	case "text", "json", "html", "markdown":
	default:
		opts.Format = "text"
	}
	if opts.Format == "text" && opts.Adapter == nil {
		return fmt.Errorf("text format requires a non-nil Adapter")
	}
	if opts.Format == "text" {
		if err := ValidateAdapterBinary(opts.Adapter); err != nil {
			return err
		}
	}
	return nil
}

// runJSONPath writes the payload as formatted JSON to stdout and
// evaluates thresholds.
func runJSONPath(payload *ReportPayload, opts RunnerOptions) error {
	enc := json.NewEncoder(opts.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return evaluateAndPrintThresholds(opts.Thresholds, payload, opts.Stderr)
}

// runHTMLPath writes the payload as native HTML and evaluates thresholds.
func runHTMLPath(payload *ReportPayload, opts RunnerOptions) error {
	if err := WriteHTML(opts.Stdout, payload); err != nil {
		return err
	}
	return evaluateAndPrintThresholds(opts.Thresholds, payload, opts.Stderr)
}

// runMarkdownPath renders the deterministic Markdown report, writes it
// to stdout and optionally to GITHUB_STEP_SUMMARY, and evaluates
// thresholds. Markdown never consults the AI adapter.
func runMarkdownPath(payload *ReportPayload, opts RunnerOptions) error {
	return publishReport(payload, opts, AIOutcome{}, nil)
}

// runTextPath asks the AI adapter for the judgment portion of the report,
// validates it against the answer contract, and publishes the deterministic
// report with the validated judgment in its AI slots.
//
// The deterministic report is published even when the adapter fails or its
// answer is rejected: the failure becomes an AIOutcome callout, and the AI
// error is returned alongside any threshold error so CI always shows the
// analysis while still failing the job (FR-020, FR-026).
func runTextPath(payload *ReportPayload, opts RunnerOptions) error {
	_, _ = fmt.Fprintln(opts.Stderr, "Formatting report...")

	payloadBytes, err := payload.CompactForAI()
	if err != nil {
		return fmt.Errorf("compacting payload for AI adapter: %w", err)
	}
	_, _ = fmt.Fprintf(opts.Stderr, "Payload: %d bytes (compact)\n", len(payloadBytes))

	outcome, aiErr := askAdapter(opts, payloadBytes)

	return publishReport(payload, opts, outcome, aiErr)
}

// runnerContext returns the caller's context, falling back to a background
// context when the caller supplied none.
func runnerContext(opts RunnerOptions) context.Context {
	if opts.Context != nil {
		return opts.Context
	}

	return context.Background()
}

// askAdapter runs the adapter with the embedded report-mode prompt and
// validates its output. A non-nil error means the judgment portion is
// unavailable; the returned AIOutcome still carries the callout to render.
func askAdapter(opts RunnerOptions, payloadBytes []byte) (AIOutcome, error) {
	timeout := opts.AdapterCfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(runnerContext(opts), timeout)
	defer cancel()

	raw, err := opts.Adapter.Format(ctx, ReportModePrompt(), bytes.NewReader(payloadBytes))
	if err != nil {
		wrapped := fmt.Errorf("AI formatting failed: %w", err)
		return AIOutcome{FailureNote: "AI narrative unavailable: " + boundedFirstLine(wrapped.Error())}, wrapped
	}

	answer, err := ParseAIAnswer(raw)
	if err != nil {
		return AIOutcome{FailureNote: "AI narrative rejected: " + rejectionDetail(err)}, err
	}

	return AIOutcome{Answer: &answer}, nil
}

// rejectionDetail returns the rule detail of a rejection error, falling back
// to a bounded message for any other error type.
func rejectionDetail(err error) string {
	var rejection *AnswerRejection
	if errors.As(err, &rejection) {
		return rejection.Detail
	}

	return boundedFirstLine(err.Error())
}

// publishReport renders the markdown report with outcome, writes it to stdout
// and the Step Summary, then evaluates thresholds. The AI error and the
// threshold error are joined so both surface to the caller.
func publishReport(payload *ReportPayload, opts RunnerOptions, outcome AIOutcome, aiErr error) error {
	report, err := RenderMarkdown(payload, outcome)
	if err != nil {
		return errors.Join(aiErr, fmt.Errorf("rendering markdown report: %w", err))
	}

	if _, err := fmt.Fprint(opts.Stdout, report); err != nil {
		return errors.Join(aiErr, fmt.Errorf("writing report to stdout: %w", err))
	}

	if opts.StepSummaryPath != "" {
		_, _ = fmt.Fprintln(opts.Stderr, "Writing Step Summary...")
		WriteStepSummary(opts.StepSummaryPath, report, opts.Stderr)
	}

	return errors.Join(aiErr, evaluateAndPrintThresholds(opts.Thresholds, payload, opts.Stderr))
}

// Run executes the report pipeline according to opts.
//
// In --format=json or --format=html mode it assembles a ReportPayload from the four analysis
// steps (CRAP, Quality, Classification, Docscan) and writes the combined
// native output to opts.Stdout. Each step's failure is recorded in PayloadErrors
// and the remaining steps still run (partial-failure mode).
//
// In --format=text mode (default) it additionally calls opts.Adapter.Format
// with the system prompt and JSON payload to produce a markdown report, then
// writes that to opts.Stdout and optionally to $GITHUB_STEP_SUMMARY.
//
// Returns an error when:
//   - The AI adapter binary is not on PATH (FR-012, text mode only; checked before analysis).
//   - No packages match the patterns (FR-013).
//   - The AI adapter returns empty output (FR-016, text mode only).
//   - The AI adapter invocation fails (text mode only).
func Run(opts RunnerOptions) error {
	if err := validateRunnerOpts(&opts); err != nil {
		return err
	}

	ctx := runnerContext(opts)

	analyzeFunc := opts.AnalyzeFunc
	if analyzeFunc == nil {
		analyzeFunc = func(patterns []string, moduleDir string) (*ReportPayload, error) {
			return runProductionPipeline(ctx, patterns, moduleDir, opts.CoverProfile, opts.TestShort, opts.Stderr, pipelineStepFuncs{}, sectionSteps(opts.Sections))
		}
	}

	payload, err := analyzeFunc(opts.Patterns, opts.ModuleDir)
	if err != nil {
		return err
	}
	applyMetadata(payload, opts)
	setAssessment(payload)

	// Zero-result gate (#116): when the CRAP step succeeded but
	// produced zero scores and any threshold is configured, fail
	// before threshold evaluation. A CI gate that passes when
	// nothing was measured provides false assurance.
	// The gate applies only when the CRAP step ran: a report that never
	// selected CRAP has no function count to gate on (FR-029, SC-011;
	// research R8).
	if sectionSteps(opts.Sections).CRAP {
		if err := checkZeroResultGate(payload, opts.Thresholds); err != nil {
			return err
		}
	}

	switch opts.Format {
	case "json":
		return runJSONPath(payload, opts)
	case "html":
		return runHTMLPath(payload, opts)
	case "markdown":
		return runMarkdownPath(payload, opts)
	default:
		return runTextPath(payload, opts)
	}
}

// sectionSteps returns the analysis steps a section selection requires.
// A zero SectionSet selects every section, matching applyMetadata.
func sectionSteps(s SectionSet) StepSet {
	if s == (SectionSet{}) {
		s = AllSections()
	}

	return s.Steps()
}

// applyMetadata copies opts.Metadata onto payload with Sections set to
// the effective section list in canonical order. A zero opts.Sections
// selects every section. A nil payload is left untouched.
func applyMetadata(payload *ReportPayload, opts RunnerOptions) {
	if payload == nil {
		return
	}
	sections := opts.Sections
	if sections == (SectionSet{}) {
		sections = AllSections()
	}
	md := opts.Metadata
	md.Sections = sections.List()
	payload.Metadata = md
}

// setAssessment computes the report's health assessment and attaches
// it to the payload, so every output format carries the same grades.
// A nil payload is left untouched.
func setAssessment(payload *ReportPayload) {
	if payload == nil {
		return
	}
	assessment := Assess(payload)
	payload.Assessment = &assessment
}

// checkZeroResultGate returns an error when the CRAP step succeeded
// (no error recorded) but produced zero analyzed functions, and any
// threshold is configured. This prevents CI gates from silently
// passing when the package pattern matched nothing (#116).
//
// The check uses TotalFunctions == 0 as the "zero results" signal,
// which distinguishes "no functions found" from "all functions are
// healthy" (CRAPload == 0 with TotalFunctions > 0).
func checkZeroResultGate(payload *ReportPayload, cfg ThresholdConfig) error {
	if payload == nil {
		return nil
	}
	// Only relevant when a threshold is configured.
	if cfg.MaxCrapload == nil && cfg.MaxGazeCrapload == nil && cfg.MinContractCoverage == nil {
		return nil
	}
	// Only relevant when the CRAP step succeeded (no error).
	if payload.Errors.CRAP != nil {
		return nil
	}
	// Check for zero results: no functions were analyzed.
	if payload.Summary.TotalFunctions != 0 {
		return nil
	}
	return fmt.Errorf("no functions analyzed — cannot evaluate thresholds (check package patterns)")
}

// evaluateAndPrintThresholds evaluates cfg thresholds against payload and
// prints results to stderr. Returns a non-nil error if any threshold failed.
// When cfg has no thresholds set, it is a no-op.
func evaluateAndPrintThresholds(cfg ThresholdConfig, payload *ReportPayload, stderr io.Writer) error {
	results, allPassed := EvaluateThresholds(cfg, payload)
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		actualStr := "N/A"
		if r.Actual != nil {
			actualStr = fmt.Sprintf("%d", *r.Actual)
		}
		_, _ = fmt.Fprintf(stderr, "%s: %s/%d (%s)\n", r.Name, actualStr, r.Limit, status)
	}
	if !allPassed {
		return fmt.Errorf("one or more quality thresholds failed")
	}
	return nil
}

// errString converts an error to a *string for PayloadErrors.
func errString(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}

// pipelineStepFuncs holds injectable step function references for
// runProductionPipeline. When a field is nil, the real step function
// is used. This enables unit testing of the orchestration logic
// (partial failures, error capture, payload assembly) without running
// real analysis.
type pipelineStepFuncs struct {
	crapStep     func([]string, string, string, io.Writer, crap.ContractCoverageProvider, bool) (*crapStepResult, error)
	qualityStep  func([]string, string, io.Writer, ...qualityPipelineDeps) (*qualityStepResult, error)
	classifyStep func([]string, string, io.Writer, ...qualityPipelineDeps) (*classifyStepResult, error)
	docscanStep  func(context.Context, string, io.Writer) (json.RawMessage, error)
}

// runProductionPipeline runs the selected analysis steps and returns a
// ReportPayload. Each step failure is recorded as a non-nil PayloadErrors
// field; remaining selected steps still run. Unselected steps leave both
// their payload field and their error entry nil.
//
// coverProfile is forwarded to runCRAPStep (FR-001, FR-002). Empty string
// means generate coverage internally (FR-003).
//
// The steps parameter allows injection of fake step functions for testing.
// Pass pipelineStepFuncs{} (zero value) for production behavior. The
// selected parameter comes from SectionSet.Steps().
func runProductionPipeline(ctx context.Context, patterns []string, moduleDir string, coverProfile string, testShort bool, stderr io.Writer, steps pipelineStepFuncs, selected StepSet) (*ReportPayload, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no package patterns specified")
	}

	steps = withDefaultSteps(steps)
	payload := &ReportPayload{}

	if selected.CRAP {
		runCRAPInto(payload, patterns, moduleDir, coverProfile, testShort, stderr, steps)
	}
	if selected.Quality {
		runQualityInto(payload, patterns, moduleDir, stderr, steps)
	}
	if selected.Classify {
		runClassifyInto(payload, patterns, moduleDir, stderr, steps)
	}
	if selected.Docscan {
		runDocscanInto(payload, ctx, moduleDir, stderr, steps)
	}

	return payload, nil
}

// withDefaultSteps fills in the real step functions for any nil field.
func withDefaultSteps(steps pipelineStepFuncs) pipelineStepFuncs {
	if steps.crapStep == nil {
		steps.crapStep = runCRAPStep
	}
	if steps.qualityStep == nil {
		steps.qualityStep = runQualityStep
	}
	if steps.classifyStep == nil {
		steps.classifyStep = runClassifyStep
	}
	if steps.docscanStep == nil {
		// The Go-native pipeline has no external analyzer session. The
		// language-aware docscan path is reached directly from the CLI
		// external-analyzer report path (runExternalReportCRAP), so here
		// the step always runs with a nil session (heuristic-only).
		steps.docscanStep = func(ctx context.Context, moduleDir string, stderr io.Writer) (json.RawMessage, error) {
			return RunDocscanStep(ctx, moduleDir, nil, stderr)
		}
	}

	return steps
}

// runCRAPInto runs the CRAP step and records its results on payload.
func runCRAPInto(payload *ReportPayload, patterns []string, moduleDir, coverProfile string, testShort bool, stderr io.Writer, steps pipelineStepFuncs) {
	// Build the contract coverage provider for GazeCRAP scoring (spec 022).
	// GoContractCoverageProvider wraps the quality pipeline per-package to
	// build a lookup closure. Best-effort: returns nil if all packages fail.
	ccProvider := goprovider.NewContractCoverageProvider(stderr)

	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (CRAP)")
	crapRes, err := steps.crapStep(patterns, moduleDir, coverProfile, stderr, ccProvider, testShort)
	if err != nil {
		payload.Errors.CRAP = errString(err)
		return
	}

	payload.CRAP = crapRes.JSON
	payload.Summary.CRAPload = crapRes.CRAPload
	payload.Summary.GazeCRAPload = crapRes.GazeCRAPload
	payload.Summary.TotalFunctions = crapRes.TotalFunctions
	if len(crapRes.SSADegradedPackages) > 0 {
		payload.Summary.SSADegraded = true
		payload.Summary.SSADegradedPackages = append(
			payload.Summary.SSADegradedPackages, crapRes.SSADegradedPackages...)
	}
}

// runQualityInto runs the quality step and records its results on payload.
func runQualityInto(payload *ReportPayload, patterns []string, moduleDir string, stderr io.Writer, steps pipelineStepFuncs) {
	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (Quality)")
	qualRes, err := steps.qualityStep(patterns, moduleDir, stderr)
	if err != nil {
		payload.Errors.Quality = errString(err)
		return
	}

	payload.Quality = qualRes.JSON
	payload.Summary.AvgContractCoverage = qualRes.AvgContractCoverage
	payload.Summary.SkippedTests = qualRes.SkippedTests
	payload.Summary.SkippedTestNames = qualRes.SkippedTestNames
	payload.Summary.SSADegraded = payload.Summary.SSADegraded || qualRes.SSADegraded
	payload.Summary.SSADegradedPackages = append(
		payload.Summary.SSADegradedPackages, qualRes.SSADegradedPackages...)
}

// runClassifyInto runs the classification step and records its results.
func runClassifyInto(payload *ReportPayload, patterns []string, moduleDir string, stderr io.Writer, steps pipelineStepFuncs) {
	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (Classification)")
	classifyRes, err := steps.classifyStep(patterns, moduleDir, stderr)
	if err != nil {
		payload.Errors.Classify = errString(err)
		return
	}

	payload.Classify = classifyRes.JSON
	payload.Summary.Contractual = classifyRes.Contractual
	payload.Summary.Ambiguous = classifyRes.Ambiguous
	payload.Summary.Incidental = classifyRes.Incidental
}

// runDocscanInto runs the documentation scan step and records its result.
func runDocscanInto(payload *ReportPayload, ctx context.Context, moduleDir string, stderr io.Writer, steps pipelineStepFuncs) {
	_, _ = fmt.Fprintln(stderr, "Scanning documentation...")
	docscanJSON, err := steps.docscanStep(ctx, moduleDir, stderr)
	if err != nil {
		payload.Errors.Docscan = errString(err)
		return
	}

	payload.Docscan = docscanJSON
}
