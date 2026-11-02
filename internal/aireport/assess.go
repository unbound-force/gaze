package aireport

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/unbound-force/gaze/v2/internal/crap"
)

// Grade is a letter grade from the deterministic rubric
// (contracts/grade-rubric.md). The set is closed: A, A-, B+, B, B-,
// C+, C, C-, D, F.
type Grade string

// Grade values, best to worst.
const (
	GradeA      Grade = "A"
	GradeAMinus Grade = "A-"
	GradeBPlus  Grade = "B+"
	GradeB      Grade = "B"
	GradeBMinus Grade = "B-"
	GradeCPlus  Grade = "C+"
	GradeC      Grade = "C"
	GradeCMinus Grade = "C-"
	GradeD      Grade = "D"
	GradeF      Grade = "F"
)

// Severity is the severity bucket for a scorecard row or
// recommendation. Each severity maps to one marker emoji.
type Severity string

// Severity values.
const (
	// SeverityLow covers grades A, A-, B+ (marker 🟢).
	SeverityLow Severity = "low"
	// SeverityModerate covers grades B, B-, C+, C (marker 🟡).
	SeverityModerate Severity = "moderate"
	// SeverityCritical covers grades C-, D, F (marker 🔴).
	SeverityCritical Severity = "critical"
	// SeverityUnavailable marks a metric that could not be computed
	// (marker ⚪).
	SeverityUnavailable Severity = "unavailable"
)

// Assessment is the Go-computed evaluation of a report payload: the
// scorecard and the deterministic recommendation list.
type Assessment struct {
	// RubricVersion identifies the grade rubric revision ("1").
	RubricVersion string `json:"rubric_version"`

	// Scorecard always holds five rows in fixed order: crapload,
	// gaze_crapload, avg_line_coverage, contract_coverage, complexity.
	Scorecard []ScorecardRow `json:"scorecard"`

	// Recommendations holds 0..8 entries in deterministic order:
	// recommended_actions first, then ssa_degraded, skipped_tests,
	// ambiguous_classification.
	Recommendations []Recommendation `json:"recommendations"`
}

// ScorecardRow is one graded dimension of the health scorecard.
type ScorecardRow struct {
	// Dimension is the machine key (crapload, gaze_crapload,
	// avg_line_coverage, contract_coverage, complexity).
	Dimension string `json:"dimension"`

	// Label is the human-readable dimension name.
	Label string `json:"label"`

	// Value is the measured metric; nil when unavailable.
	Value *float64 `json:"value"`

	// Grade is the rubric grade; nil when unavailable.
	Grade *Grade `json:"grade"`

	// Severity is derived from Grade (unavailable when Grade is nil).
	Severity Severity `json:"severity"`

	// Marker is the severity emoji.
	Marker string `json:"marker"`

	// Detail is a short deterministic explanation of the value.
	Detail string `json:"detail"`
}

// Recommendation is one deterministic, prioritized action item. It
// renders as "{i+1}. {Marker} {Text}".
type Recommendation struct {
	// Severity is the recommendation's severity bucket.
	Severity Severity `json:"severity"`

	// Marker is the severity emoji.
	Marker string `json:"marker"`

	// Source names the rule that produced the recommendation
	// (fix_strategy, ssa_degraded, skipped_tests,
	// ambiguous_classification).
	Source string `json:"source"`

	// Text is the rendered recommendation sentence.
	Text string `json:"text"`

	// Function is the target function, when applicable.
	Function string `json:"function,omitempty"`

	// File is the target file, when applicable.
	File string `json:"file,omitempty"`

	// Line is the target line, when applicable.
	Line int `json:"line,omitempty"`

	// FixStrategy is the CRAP fix strategy, when applicable.
	FixStrategy string `json:"fix_strategy,omitempty"`
}

// N/A reasons used in scorecard details (contracts/grade-rubric.md).
const (
	reasonNotSelected             = "not selected"
	reasonAnalysisFailed          = "analysis failed"
	reasonNoFunctions             = "no functions analyzed"
	reasonGazeCRAPUnavailable     = "GazeCRAP unavailable"
	reasonNoTests                 = "no tests analyzed"
	maxRecommendations            = 8
	maxFixStrategyRecommendations = 5
)

// crapAnalysis is the subset of the CRAP step JSON the assessment
// decodes.
type crapAnalysis struct {
	Scores  []crap.Score `json:"scores"`
	Summary crap.Summary `json:"summary"`
}

// qualityAnalysis is the subset of the quality step JSON the
// assessment decodes.
type qualityAnalysis struct {
	Summary struct {
		TotalTests int `json:"total_tests"`
	} `json:"quality_summary"`
}

// crapInputs carries the decoded CRAP data plus the reason the
// CRAP-backed dimensions are unavailable (empty when available).
type crapInputs struct {
	data        crapAnalysis
	unavailable string
}

// qualityInputs carries the decoded quality data plus the reason
// contract coverage is unavailable (empty when available).
type qualityInputs struct {
	data        qualityAnalysis
	unavailable string
}

// Assess computes the deterministic assessment for a report payload:
// the five-row scorecard and the recommendation list. It performs no
// I/O and consults no clock, so identical payloads always produce
// identical assessments. A payload with no selected sections is
// treated as having all sections selected.
func Assess(p *ReportPayload) Assessment {
	sections := assessmentSections(p)
	crapIn := loadCRAPInputs(p, sections)
	qualityIn := loadQualityInputs(p, sections)
	return Assessment{
		RubricVersion: rubricVersion,
		Scorecard: []ScorecardRow{
			crapLoadRow(crapIn),
			gazeCRAPLoadRow(crapIn),
			lineCoverageRow(crapIn),
			contractCoverageRow(p, qualityIn),
			complexityRow(crapIn),
		},
		Recommendations: recommendations(p, crapIn, sections),
	}
}

// assessmentSections returns the payload's selected sections, falling
// back to all sections when none are recorded or the list is invalid.
func assessmentSections(p *ReportPayload) SectionSet {
	if p == nil || len(p.Metadata.Sections) == 0 {
		return AllSections()
	}
	raw := make([]string, len(p.Metadata.Sections))
	for i, section := range p.Metadata.Sections {
		raw[i] = string(section)
	}
	parsed, err := ParseSections(raw)
	if err != nil {
		return AllSections()
	}
	return parsed
}

// loadCRAPInputs decodes the CRAP step output and classifies why the
// CRAP-backed dimensions are unavailable.
func loadCRAPInputs(p *ReportPayload, sections SectionSet) crapInputs {
	if !sections.Has(SectionCRAP) {
		return crapInputs{unavailable: reasonNotSelected}
	}
	if p == nil || p.Errors.CRAP != nil || len(p.CRAP) == 0 {
		return crapInputs{unavailable: reasonAnalysisFailed}
	}
	var data crapAnalysis
	if err := json.Unmarshal(p.CRAP, &data); err != nil {
		return crapInputs{unavailable: reasonAnalysisFailed}
	}
	if data.Summary.TotalFunctions == 0 {
		return crapInputs{unavailable: reasonNoFunctions}
	}
	return crapInputs{data: data}
}

// loadQualityInputs decodes the quality step output and classifies why
// contract coverage is unavailable.
func loadQualityInputs(p *ReportPayload, sections SectionSet) qualityInputs {
	if !sections.Has(SectionQuality) {
		return qualityInputs{unavailable: reasonNotSelected}
	}
	if p == nil || p.Errors.Quality != nil || len(p.Quality) == 0 {
		return qualityInputs{unavailable: reasonAnalysisFailed}
	}
	var data qualityAnalysis
	if err := json.Unmarshal(p.Quality, &data); err != nil {
		return qualityInputs{unavailable: reasonAnalysisFailed}
	}
	if p.Summary.AvgContractCoverage == nil || data.Summary.TotalTests == 0 {
		return qualityInputs{unavailable: reasonNoTests}
	}
	return qualityInputs{data: data}
}

// crapLoadRow grades the share of functions at or above the CRAP
// threshold.
func crapLoadRow(in crapInputs) ScorecardRow {
	if in.unavailable != "" {
		return unavailableRow("crapload", "CRAPload", in.unavailable)
	}
	s := in.data.Summary
	tenths := ratioTenths(s.CRAPload, s.TotalFunctions)
	return gradedRow("crapload", "CRAPload", float64(tenths)/10,
		gradeLowerIsBetter(tenths, crapLoadBounds),
		fmt.Sprintf("%d/%d functions (%s%%) at or above threshold %g",
			s.CRAPload, s.TotalFunctions, formatTenths(tenths), s.CRAPThreshold))
}

// gazeCRAPLoadRow grades the share of functions at or above the
// GazeCRAP threshold.
func gazeCRAPLoadRow(in crapInputs) ScorecardRow {
	if in.unavailable != "" {
		return unavailableRow("gaze_crapload", "GazeCRAPload", in.unavailable)
	}
	s := in.data.Summary
	if s.GazeCRAPload == nil {
		return unavailableRow("gaze_crapload", "GazeCRAPload", reasonGazeCRAPUnavailable)
	}
	threshold := s.CRAPThreshold
	if s.GazeCRAPThreshold != nil {
		threshold = *s.GazeCRAPThreshold
	}
	tenths := ratioTenths(*s.GazeCRAPload, s.TotalFunctions)
	return gradedRow("gaze_crapload", "GazeCRAPload", float64(tenths)/10,
		gradeLowerIsBetter(tenths, gazeCRAPLoadBounds),
		fmt.Sprintf("%d/%d functions (%s%%) at or above threshold %g",
			*s.GazeCRAPload, s.TotalFunctions, formatTenths(tenths), threshold))
}

// lineCoverageRow grades the average line coverage.
func lineCoverageRow(in crapInputs) ScorecardRow {
	if in.unavailable != "" {
		return unavailableRow("avg_line_coverage", "Avg Line Coverage", in.unavailable)
	}
	tenths, ok := displayTenths(in.data.Summary.AvgLineCoverage)
	if !ok {
		return unavailableRow("avg_line_coverage", "Avg Line Coverage", reasonAnalysisFailed)
	}
	return gradedRow("avg_line_coverage", "Avg Line Coverage", float64(tenths)/10,
		gradeHigherIsBetter(tenths, lineCoverageBounds),
		fmt.Sprintf("%s%% average line coverage", formatTenths(tenths)))
}

// contractCoverageRow grades the average contract coverage.
func contractCoverageRow(p *ReportPayload, in qualityInputs) ScorecardRow {
	if in.unavailable != "" {
		return unavailableRow("contract_coverage", "Contract Coverage", in.unavailable)
	}
	value := *p.Summary.AvgContractCoverage
	return gradedRow("contract_coverage", "Contract Coverage", float64(value),
		gradeHigherIsBetter(value, contractCoverageBounds),
		fmt.Sprintf("%d%% average contract coverage across %d tests",
			value, in.data.Summary.TotalTests))
}

// complexityRow grades the average cyclomatic complexity.
func complexityRow(in crapInputs) ScorecardRow {
	if in.unavailable != "" {
		return unavailableRow("complexity", "Complexity", in.unavailable)
	}
	tenths, ok := displayTenths(in.data.Summary.AvgComplexity)
	if !ok {
		return unavailableRow("complexity", "Complexity", reasonAnalysisFailed)
	}
	return gradedRow("complexity", "Complexity", float64(tenths)/10,
		gradeLowerIsBetter(tenths, complexityBounds),
		fmt.Sprintf("%s average cyclomatic complexity", formatTenths(tenths)))
}

// gradedRow builds an available scorecard row.
func gradedRow(dimension, label string, value float64, grade Grade, detail string) ScorecardRow {
	severity := severityForGrade(grade)
	return ScorecardRow{
		Dimension: dimension,
		Label:     label,
		Value:     &value,
		Grade:     &grade,
		Severity:  severity,
		Marker:    markerForSeverity(severity),
		Detail:    detail,
	}
}

// unavailableRow builds an N/A scorecard row.
func unavailableRow(dimension, label, reason string) ScorecardRow {
	return ScorecardRow{
		Dimension: dimension,
		Label:     label,
		Severity:  SeverityUnavailable,
		Marker:    markerForSeverity(SeverityUnavailable),
		Detail:    reason,
	}
}

// recommendations builds the deterministic recommendation list: the
// fix-strategy items first, then the data conditions, capped at 8.
func recommendations(p *ReportPayload, in crapInputs, sections SectionSet) []Recommendation {
	recs := fixStrategyRecommendations(in)
	recs = append(recs, conditionRecommendations(p, sections)...)
	if len(recs) > maxRecommendations {
		recs = recs[:maxRecommendations]
	}
	return recs
}

// fixStrategyRecommendations renders the first five recommended
// actions. An action whose score cannot be joined by (package,
// function, file, line) is skipped rather than rendered with a
// fabricated coverage value.
func fixStrategyRecommendations(in crapInputs) []Recommendation {
	if in.unavailable != "" {
		return nil
	}
	actions := in.data.Summary.RecommendedActions
	if len(actions) > maxFixStrategyRecommendations {
		actions = actions[:maxFixStrategyRecommendations]
	}
	recs := make([]Recommendation, 0, len(actions))
	for _, action := range actions {
		score, ok := findScore(in.data.Scores, action)
		if !ok {
			continue
		}
		if rec, ok := fixStrategyRecommendation(action, score); ok {
			recs = append(recs, rec)
		}
	}
	return recs
}

// findScore joins a recommended action to its score entry.
func findScore(scores []crap.Score, action crap.RecommendedAction) (crap.Score, bool) {
	for _, score := range scores {
		if score.Package == action.Package &&
			score.Function == action.Function &&
			score.File == action.File &&
			score.Line == action.Line {
			return score, true
		}
	}
	return crap.Score{}, false
}

// fixStrategyRecommendation renders one fix-strategy recommendation.
// It reports false for an unknown fix strategy.
func fixStrategyRecommendation(action crap.RecommendedAction, score crap.Score) (Recommendation, bool) {
	severity := SeverityModerate
	if score.LineCoverage == 0 || (score.Quadrant != nil && *score.Quadrant == crap.Q4Dangerous) {
		severity = SeverityCritical
	}
	text, ok := fixStrategyText(action, score)
	if !ok {
		return Recommendation{}, false
	}
	return Recommendation{
		Severity:    severity,
		Marker:      markerForSeverity(severity),
		Source:      "fix_strategy",
		Text:        text,
		Function:    action.Function,
		File:        action.File,
		Line:        action.Line,
		FixStrategy: string(action.FixStrategy),
	}, true
}

// fixStrategyText applies the exact text template for a fix strategy.
func fixStrategyText(action crap.RecommendedAction, score crap.Score) (string, bool) {
	fn := codeSpan(action.Function)
	location := fmt.Sprintf("%s:%d", action.File, action.Line)
	switch action.FixStrategy {
	case crap.FixAddTests:
		return fmt.Sprintf("Add tests for %s — %.1f%% line coverage, complexity %d, CRAP %.1f (%s).",
			fn, score.LineCoverage, action.Complexity, action.CRAP, location), true
	case crap.FixAddAssertions:
		gaze := ""
		if action.GazeCRAP != nil {
			gaze = fmt.Sprintf("; GazeCRAP %.1f", *action.GazeCRAP)
		}
		return fmt.Sprintf("Strengthen assertions for %s — tests execute it without verifying its contracts%s (%s).",
			fn, gaze, location), true
	case crap.FixDecompose:
		return fmt.Sprintf("Decompose %s — complexity %d keeps CRAP at %.1f even at full coverage (%s).",
			fn, action.Complexity, action.CRAP, location), true
	case crap.FixDecomposeAndTest:
		return fmt.Sprintf("Decompose %s, then add tests — complexity %d, %.1f%% line coverage, CRAP %.1f (%s).",
			fn, action.Complexity, score.LineCoverage, action.CRAP, location), true
	default:
		return "", false
	}
}

// conditionRecommendations renders the data-condition recommendations
// in their fixed order: ssa_degraded, skipped_tests,
// ambiguous_classification.
func conditionRecommendations(p *ReportPayload, sections SectionSet) []Recommendation {
	if p == nil {
		return nil
	}
	recs := make([]Recommendation, 0, 3)
	if rec, ok := ssaDegradedRecommendation(p.Summary); ok {
		recs = append(recs, rec)
	}
	if rec, ok := skippedTestsRecommendation(p.Summary); ok {
		recs = append(recs, rec)
	}
	if rec, ok := ambiguousClassificationRecommendation(p, sections); ok {
		recs = append(recs, rec)
	}
	return recs
}

// ssaDegradedRecommendation reports SSA degradation, when present.
func ssaDegradedRecommendation(summary ReportSummary) (Recommendation, bool) {
	if !summary.SSADegraded {
		return Recommendation{}, false
	}
	return newRecommendation(SeverityModerate, "ssa_degraded", fmt.Sprintf(
		"Resolve SSA degradation in %d package(s) — quality and contract data are partial there.",
		len(summary.SSADegradedPackages))), true
}

// skippedTestsRecommendation reports skipped tests, when present.
func skippedTestsRecommendation(summary ReportSummary) (Recommendation, bool) {
	if summary.SkippedTests <= 0 {
		return Recommendation{}, false
	}
	return newRecommendation(SeverityLow, "skipped_tests", fmt.Sprintf(
		"Map %d skipped test(s) to targets with %s — no target function could be inferred.",
		summary.SkippedTests, codeSpan("gaze quality --target"))), true
}

// ambiguousClassificationRecommendation reports an ambiguous-majority
// classification distribution, when the classification step ran.
func ambiguousClassificationRecommendation(p *ReportPayload, sections SectionSet) (Recommendation, bool) {
	if !sections.Has(SectionClassification) || p.Errors.Classify != nil {
		return Recommendation{}, false
	}
	ambiguous := p.Summary.Ambiguous
	decided := p.Summary.Contractual + p.Summary.Incidental
	total := ambiguous + decided
	if total == 0 || ambiguous <= decided {
		return Recommendation{}, false
	}
	return newRecommendation(SeverityLow, "ambiguous_classification", fmt.Sprintf(
		"Document contracts for ambiguous side effects — %d of %d effects (%s%%) are ambiguous.",
		ambiguous, total, formatTenths(ratioTenths(ambiguous, total)))), true
}

// newRecommendation builds a condition recommendation.
func newRecommendation(severity Severity, source, text string) Recommendation {
	return Recommendation{
		Severity: severity,
		Marker:   markerForSeverity(severity),
		Source:   source,
		Text:     text,
	}
}

// codeSpan renders s as inline code, widening to a double-backtick
// span when s itself contains a backtick.
func codeSpan(s string) string {
	if strings.Contains(s, "`") {
		return "``" + s + "``"
	}
	return "`" + s + "`"
}
