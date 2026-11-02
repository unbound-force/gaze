package aireport

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/docscan/apidoc"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// AIOutcome carries the validated judgment for the report's AI slots.
//
// Exactly one field is expected to be set: Answer when the adapter
// produced a validated answer, or FailureNote when the adapter failed
// or its answer was rejected. The note already carries its prefix
// ("AI narrative unavailable: …" or "AI narrative rejected: …"), so
// the renderer only has to place it.
type AIOutcome struct {
	// Answer is the validated answer, or nil when there is none.
	Answer *AIAnswer `json:"answer,omitempty"`

	// FailureNote explains why no answer is available. It is empty
	// when Answer is set.
	FailureNote string `json:"failure_note,omitempty"`
}

// Report titles, one per selected-section set.
const (
	titleFullReport    = "🔍 Gaze Full Quality Report"
	titleCRAPReport    = "🔍 Gaze CRAP Report"
	titleQualityReport = "🔍 Gaze Quality Report"
)

// Step labels used in the "analysis unavailable" callouts.
const (
	stepCRAP           = "CRAP"
	stepQuality        = "Quality"
	stepClassification = "Classification"
	stepDocumentation  = "Documentation"
)

// noRecommendations is printed when the assessment produced none.
const noRecommendations = "No recommendations — no functions at or above the CRAP threshold."

// quadrantRow is one fixed row of the GazeCRAP quadrant distribution.
type quadrantRow struct {
	quadrant crap.Quadrant
	label    string
	marker   string
	meaning  string
}

// quadrantRows fixes the quadrant table's order, markers, and meanings.
var quadrantRows = []quadrantRow{
	{crap.Q1Safe, "Q1 — Safe", "🟢", "Low complexity, high contract coverage"},
	{crap.Q2ComplexButTested, "Q2 — Complex But Tested", "🟡", "High complexity, contracts verified"},
	{crap.Q3SimpleButUnderspecified, "Q3 — Needs Tests", "⚪", "Simple but underspecified"},
	{crap.Q4Dangerous, "Q4 — Dangerous", "🔴", "Complex and contracts not adequately verified"},
}

// fixStrategyOrder fixes the remediation table's row order.
var fixStrategyOrder = []crap.FixStrategy{
	crap.FixDecompose,
	crap.FixAddTests,
	crap.FixAddAssertions,
	crap.FixDecomposeAndTest,
}

// headerEscaper neutralizes Markdown and HTML markup in header values.
//
// Module paths and branch names are environment-derived, so a crafted
// reference name must not be able to inject a link, an HTML comment, or
// a new line into the report or the GitHub Step Summary.
var headerEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"`", "\\`",
	"*", "\\*",
	"_", "\\_",
	"[", "\\[",
	"]", "\\]",
	"<", "\\<",
	">", "\\>",
	"|", "\\|",
	"\r\n", " ",
	"\r", " ",
	"\n", " ",
)

// RenderMarkdown renders the deterministic Markdown report.
//
// The function is pure: it performs no I/O, reads no clock, and mutates
// nothing. Every byte outside the AI slots is derived from the payload,
// so the same payload always renders the same report. An empty
// Sections list on the payload's metadata renders all sections. Cell and
// header escaping follow research R5, the AI slot delimiters follow R6,
// and the layout stays within the width bound established in R18.
func RenderMarkdown(p *ReportPayload, outcome AIOutcome) (string, error) {
	if p == nil {
		return "", fmt.Errorf("render markdown: nil payload")
	}

	sections := assessmentSections(p)
	assessment := p.Assessment
	if assessment == nil {
		computed := Assess(p)
		assessment = &computed
	}

	var b strings.Builder
	writeTitle(&b, sections)
	writeMetadata(&b, p.Metadata)
	writeAISummarySlot(&b, outcome)
	writeCRAPSection(&b, p, sections)
	writeQualitySection(&b, p, sections)
	writeClassificationSection(&b, p, sections)
	writeHealthSection(&b, p, *assessment, sections, outcome)
	if !sections.Has(SectionHealth) {
		writeAIRecommendationsSlot(&b, outcome)
	}

	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

// writeTitle writes the report title for the selected sections.
func writeTitle(b *strings.Builder, sections SectionSet) {
	b.WriteString("# ")
	b.WriteString(reportTitle(sections))
	b.WriteString("\n\n")
}

// reportTitle returns the title for a selected-section set.
func reportTitle(sections SectionSet) string {
	switch {
	case sections == AllSections():
		return titleFullReport
	case sections == sectionSetOf(SectionCRAP):
		return titleCRAPReport
	case sections == sectionSetOf(SectionQuality):
		return titleQualityReport
	default:
		return titleQualityReport + " (sections: " + sectionList(sections) + ")"
	}
}

// sectionSetOf returns the set holding exactly one section.
func sectionSetOf(s Section) SectionSet {
	set, err := ParseSections([]string{string(s)})
	if err != nil {
		return SectionSet{}
	}
	return set
}

// sectionList renders the selected sections as a comma-space list.
func sectionList(sections SectionSet) string {
	names := make([]string, 0, len(sections.List()))
	for _, s := range sections.List() {
		names = append(names, string(s))
	}
	return strings.Join(names, ", ")
}

// writeMetadata writes the project, branch, version, and date header.
func writeMetadata(b *strings.Builder, m ReportMetadata) {
	b.WriteString("Project: ")
	b.WriteString(sanitizeHeaderValue(m.ModulePath))
	if m.Branch != "" {
		b.WriteString(" · Branch: ")
		b.WriteString(sanitizeHeaderValue(m.Branch))
	}
	b.WriteString("\nGaze Version: ")
	b.WriteString(sanitizeHeaderValue(m.GazeVersion))
	b.WriteString(" · Date: ")
	b.WriteString(sanitizeHeaderValue(m.Date))
	b.WriteString("\n\n")
}

// proseEscaper neutralizes markup in prose derived from analysis data.
// File paths, package paths, and adapter errors are attacker-influenced in
// pull-request CI, so brackets and angle brackets are escaped rather than
// trusted: an escaped "[x](y)" cannot become a link and an escaped "<b>"
// cannot become HTML. Line breaks flatten to spaces so the prose cannot
// start a new markdown block.
var proseEscaper = strings.NewReplacer(
	"\r\n", " ",
	"\n", " ",
	"\r", " ",
	"[", `\[`,
	"]", `\]`,
	"<", `\<`,
)

// sanitizeProse escapes an analysis-derived string for a non-table position.
func sanitizeProse(s string) string {
	return proseEscaper.Replace(s)
}

// sanitizeHeaderValue escapes markup and flattens line breaks.
func sanitizeHeaderValue(s string) string {
	return headerEscaper.Replace(s)
}

// writeAISummarySlot writes the AI summary slot when there is an
// outcome. A failure renders as a callout inside the slot.
func writeAISummarySlot(b *strings.Builder, outcome AIOutcome) {
	if outcome.Answer == nil && outcome.FailureNote == "" {
		return
	}

	b.WriteString("<!-- gaze:ai-summary -->\n### Summary\n\n")
	if outcome.Answer != nil {
		b.WriteString(sanitizeProse(strings.TrimSpace(outcome.Answer.Summary)))
		b.WriteString("\n")
	} else {
		b.WriteString("> ⚠️ ")
		b.WriteString(sanitizeProse(boundedFirstLine(outcome.FailureNote)))
		b.WriteString("\n")
	}
	b.WriteString("<!-- /gaze:ai-summary -->\n\n")
}

// writeAIRecommendationsSlot writes the AI recommendations slot when a
// validated answer is available.
func writeAIRecommendationsSlot(b *strings.Builder, outcome AIOutcome) {
	if outcome.Answer == nil {
		return
	}

	b.WriteString("<!-- gaze:ai-recommendations -->\n")
	b.WriteString("### AI-Prioritized Recommendations\n\n")
	for i, rec := range outcome.Answer.Recommendations {
		fmt.Fprintf(b, "%d. %s\n", i+1, sanitizeProse(strings.TrimSpace(rec)))
	}
	b.WriteString("<!-- /gaze:ai-recommendations -->\n\n")
}

// writeCallout writes a step-unavailable callout.
func writeCallout(b *strings.Builder, step, detail string) {
	b.WriteString("> ⚠️ ")
	b.WriteString(step)
	b.WriteString(" analysis unavailable: ")
	b.WriteString(sanitizeProse(boundedFirstLine(detail)))
	b.WriteString("\n\n")
}

// boundedFirstLine returns the first line of s, bounded to 200 bytes on
// a rune boundary.
func boundedFirstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return truncateOnRuneBoundary(s, maxCalloutDetailBytes)
}

// maxCalloutDetailBytes bounds a callout's detail text.
const maxCalloutDetailBytes = 200

// truncateOnRuneBoundary bounds s to at most max bytes, cutting back to the
// nearest rune boundary. It returns s unchanged when it already fits.
func truncateOnRuneBoundary(s string, max int) string {
	if len(s) <= max {
		return s
	}

	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return s[:cut]
}

// decodeStep decodes a step's raw JSON into dst, writing the
// unavailable callout and reporting false when the step failed or
// produced nothing usable.
func decodeStep(b *strings.Builder, step string, stepErr *string, raw json.RawMessage, dst any) bool {
	if stepErr != nil {
		writeCallout(b, step, *stepErr)
		return false
	}
	if len(raw) == 0 {
		writeCallout(b, step, "no analysis data")
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		writeCallout(b, step, err.Error())
		return false
	}
	return true
}

// writeCRAPSection writes the CRAP summary section.
func writeCRAPSection(b *strings.Builder, p *ReportPayload, sections SectionSet) {
	if !sections.Has(SectionCRAP) {
		return
	}

	b.WriteString("---\n\n## 📊 CRAP Summary\n\n")
	var report crap.Report
	if !decodeStep(b, stepCRAP, p.Errors.CRAP, p.CRAP, &report) {
		return
	}
	writeCRAPMetrics(b, report.Summary)
	writeWorstCRAPTable(b, report.Summary.WorstCRAP)
	writeQuadrantTable(b, report.Summary.QuadrantCounts)
	writeRemediationTable(b, report.Summary.FixStrategyCounts)
}

// writeCRAPMetrics writes the CRAP metric table.
func writeCRAPMetrics(b *strings.Builder, s crap.Summary) {
	b.WriteString("| Metric | Value |\n|--------|------:|\n")
	fmt.Fprintf(b, "| Total functions analyzed | %d |\n", s.TotalFunctions)
	fmt.Fprintf(b, "| Average complexity | %.1f |\n", s.AvgComplexity)
	fmt.Fprintf(b, "| Average line coverage | %.1f%% |\n", s.AvgLineCoverage)
	fmt.Fprintf(b, "| Average CRAP score | %.1f |\n", s.AvgCRAP)
	fmt.Fprintf(b, "| CRAPload | %d (functions ≥ threshold %g) |\n", s.CRAPload, s.CRAPThreshold)
	fmt.Fprintf(b, "| GazeCRAPload | %s |\n\n", gazeCRAPLoadCell(s))
}

// gazeCRAPLoadCell renders the GazeCRAPload metric value.
func gazeCRAPLoadCell(s crap.Summary) string {
	if s.GazeCRAPload == nil {
		return "⚪ N/A"
	}
	threshold := s.CRAPThreshold
	if s.GazeCRAPThreshold != nil {
		threshold = *s.GazeCRAPThreshold
	}
	return fmt.Sprintf("%d (functions ≥ threshold %g)", *s.GazeCRAPload, threshold)
}

// writeWorstCRAPTable writes the worst CRAP scores table in data order.
func writeWorstCRAPTable(b *strings.Builder, rows []crap.Score) {
	b.WriteString("### Top 5 Worst CRAP Scores\n\n")
	b.WriteString("| Function | CRAP | Complexity | Coverage | Fix Strategy | Location |\n")
	b.WriteString("|----------|-----:|-----------:|---------:|--------------|----------|\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| %s | %.1f | %d | %.1f%% | %s | %s |\n",
			tableName(row.Function), row.CRAP, row.Complexity, row.LineCoverage,
			escapeCell(fixStrategyName(row.FixStrategy)), escapeCell(location(row.File, row.Line)))
	}
	b.WriteString("\n")
}

// writeQuadrantTable writes the GazeCRAP quadrant distribution table.
func writeQuadrantTable(b *strings.Builder, counts map[crap.Quadrant]int) {
	b.WriteString("### GazeCRAP Quadrant Distribution\n\n")
	b.WriteString("| Quadrant | Count | Meaning |\n")
	b.WriteString("|----------|------:|---------|\n")
	for _, row := range quadrantRows {
		fmt.Fprintf(b, "| %s %s | %d | %s |\n", row.marker, row.label, counts[row.quadrant], row.meaning)
	}
	b.WriteString("\n")
}

// writeRemediationTable writes the remediation breakdown table.
func writeRemediationTable(b *strings.Builder, counts map[crap.FixStrategy]int) {
	b.WriteString("### Remediation Breakdown\n\n")
	b.WriteString("| Fix Strategy | Functions |\n")
	b.WriteString("|--------------|----------:|\n")
	for _, strategy := range fixStrategyOrder {
		fmt.Fprintf(b, "| %s | %d |\n", strategy, counts[strategy])
	}
	b.WriteString("\n")
}

// writeQualitySection writes the quality summary section.
func writeQualitySection(b *strings.Builder, p *ReportPayload, sections SectionSet) {
	if !sections.Has(SectionQuality) {
		return
	}

	b.WriteString("---\n\n## 🧪 Quality Summary\n\n")
	var out qualityOutput
	if !decodeStep(b, stepQuality, p.Errors.Quality, p.Quality, &out) {
		return
	}
	if out.Summary == nil {
		writeCallout(b, stepQuality, "no analysis data")
		return
	}
	writeQualityMetrics(b, *out.Summary)
	if out.Summary.TotalTests == 0 {
		b.WriteString("> ⚠️ Quality analysis found 0 tests for the selected packages.\n\n")
		return
	}
	writeWorstTestsTable(b, out.Summary.WorstCoverageTests)
	writeGapsTable(b, out.Summary.WorstCoverageTests)
	writeSSACallout(b, *out.Summary)
}

// writeQualityMetrics writes the quality metric table.
func writeQualityMetrics(b *strings.Builder, s taxonomy.PackageSummary) {
	b.WriteString("| Metric | Value |\n|--------|------:|\n")
	fmt.Fprintf(b, "| Tests analyzed | %d |\n", s.TotalTests)
	fmt.Fprintf(b, "| Average contract coverage | %.1f%% |\n", s.AverageContractCoverage)
	fmt.Fprintf(b, "| Over-specifications | %d |\n", s.TotalOverSpecifications)
	fmt.Fprintf(b, "| Assertion detection confidence | %d%% |\n", s.AssertionDetectionConfidence)
	fmt.Fprintf(b, "| Skipped tests | %d |\n\n", s.SkippedTests)
}

// writeWorstTestsTable writes the worst tests table in data order.
func writeWorstTestsTable(b *strings.Builder, tests []taxonomy.QualityReport) {
	b.WriteString("### Worst Tests by Contract Coverage\n\n")
	b.WriteString("| Test | Target | Contract Coverage | Gaps |\n")
	b.WriteString("|------|--------|------------------:|-----:|\n")
	for i := range tests {
		t := &tests[i]
		fmt.Fprintf(b, "| %s | %s | %.1f%% | %d |\n",
			tableName(t.TestFunction), tableName(t.TargetFunction.QualifiedName()),
			t.ContractCoverage.Percentage, len(t.ContractCoverage.Gaps))
	}
	b.WriteString("\n")
}

// writeGapsTable writes the coverage gaps table, in per-test order and
// capped at maxGapRows rows overall.
func writeGapsTable(b *strings.Builder, tests []taxonomy.QualityReport) {
	b.WriteString("### Coverage Gaps in Worst Tests\n\n")
	b.WriteString("| Target | Effect | Description | Location |\n")
	b.WriteString("|--------|--------|-------------|----------|\n")
	written := 0
	for i := range tests {
		written += writeGapRows(b, &tests[i], maxGapRows-written)
		if written >= maxGapRows {
			break
		}
	}
	b.WriteString("\n")
}

// maxGapRows bounds the coverage gaps table.
const maxGapRows = 10

// writeGapRows writes a test's gap rows up to limit, returning the
// number written.
func writeGapRows(b *strings.Builder, t *taxonomy.QualityReport, limit int) int {
	written := 0
	for i := range t.ContractCoverage.Gaps {
		if written >= limit {
			break
		}
		gap := &t.ContractCoverage.Gaps[i]
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n",
			tableName(t.TargetFunction.QualifiedName()), escapeCell(string(gap.Type)),
			escapeCell(gap.Description), escapeCell(gap.Location))
		written++
	}
	return written
}

// writeSSACallout writes the SSA degradation callout when applicable.
func writeSSACallout(b *strings.Builder, s taxonomy.PackageSummary) {
	if !s.SSADegraded {
		return
	}
	packages := sanitizeProse(boundedFirstLine(strings.Join(s.SSADegradedPackages, ", ")))
	fmt.Fprintf(b, "> ⚠️ SSA analysis degraded for %d package(s): %s\n\n", len(s.SSADegradedPackages), packages)
}

// writeClassificationSection writes the classification summary section.
func writeClassificationSection(b *strings.Builder, p *ReportPayload, sections SectionSet) {
	if !sections.Has(SectionClassification) {
		return
	}

	b.WriteString("---\n\n## 🏷️ Classification Summary\n\n")
	if p.Errors.Classify != nil {
		writeCallout(b, stepClassification, *p.Errors.Classify)
		return
	}

	contractual := p.Summary.Contractual
	ambiguous := p.Summary.Ambiguous
	incidental := p.Summary.Incidental
	total := contractual + ambiguous + incidental
	b.WriteString("| Classification | Count | % |\n|----------------|------:|--:|\n")
	fmt.Fprintf(b, "| Contractual | %d | %s |\n", contractual, share(contractual, total))
	fmt.Fprintf(b, "| Ambiguous | %d | %s |\n", ambiguous, share(ambiguous, total))
	fmt.Fprintf(b, "| Incidental | %d | %s |\n\n", incidental, share(incidental, total))
}

// share renders count as a one-decimal percentage of total.
func share(count, total int) string {
	if total <= 0 {
		return "0.0%"
	}
	return formatTenths(ratioTenths(count, total)) + "%"
}

// writeHealthSection writes the overall health section.
func writeHealthSection(b *strings.Builder, p *ReportPayload, assessment Assessment, sections SectionSet, outcome AIOutcome) {
	if !sections.Has(SectionHealth) {
		return
	}

	b.WriteString("---\n\n## 🏥 Overall Health Assessment\n\n")
	writeScorecardTable(b, assessment)
	writeDocumentationLine(b, p)
	b.WriteString("### Recommendations\n\n")
	writeRecommendationsList(b, assessment)
	writeAIRecommendationsSlot(b, outcome)
}

// writeScorecardTable writes the assessment scorecard table.
func writeScorecardTable(b *strings.Builder, assessment Assessment) {
	b.WriteString("### Summary Scorecard\n\n")
	b.WriteString("| Dimension | Grade | Details |\n")
	b.WriteString("|-----------|-------|---------|\n")
	for i := range assessment.Scorecard {
		row := &assessment.Scorecard[i]
		fmt.Fprintf(b, "| %s | %s | %s |\n",
			escapeCell(row.Label), escapeCell(gradeCell(row)), escapeCell(row.Detail))
	}
	b.WriteString("\n")
}

// gradeCell renders a scorecard row's Grade cell.
func gradeCell(row *ScorecardRow) string {
	if row.Grade == nil {
		return "⚪ N/A"
	}
	return row.Marker + " " + string(*row.Grade)
}

// writeDocumentationLine writes the documentation coverage line.
func writeDocumentationLine(b *strings.Builder, p *ReportPayload) {
	if p.Errors.Docscan != nil {
		writeCallout(b, stepDocumentation, *p.Errors.Docscan)
		return
	}

	var envelope apidoc.DocscanEnvelope
	if len(p.Docscan) > 0 {
		if err := json.Unmarshal(p.Docscan, &envelope); err != nil {
			writeCallout(b, stepDocumentation, err.Error())
			return
		}
	}
	if len(envelope.Documents) == 0 {
		b.WriteString("Documentation: no files found.\n\n")
		return
	}
	fmt.Fprintf(b, "Documentation: %d files scanned.\n\n", len(envelope.Documents))
}

// writeRecommendationsList writes the deterministic recommendations.
func writeRecommendationsList(b *strings.Builder, assessment Assessment) {
	if len(assessment.Recommendations) == 0 {
		b.WriteString(noRecommendations)
		b.WriteString("\n\n")
		return
	}
	for i := range assessment.Recommendations {
		rec := &assessment.Recommendations[i]
		fmt.Fprintf(b, "%d. %s %s\n", i+1, rec.Marker, sanitizeProse(rec.Text))
	}
	b.WriteString("\n")
}

// tableName renders a function or test name as a table-cell code span.
func tableName(s string) string {
	return escapeCell(codeSpan(s))
}

// escapeCell escapes a table cell: pipes, line breaks, and link or HTML
// markup.
func escapeCell(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "<", "\\<")
	return strings.ReplaceAll(s, ">", "\\>")
}

// fixStrategyName renders a fix strategy, or an empty cell when absent.
func fixStrategyName(strategy *crap.FixStrategy) string {
	if strategy == nil {
		return ""
	}
	return string(*strategy)
}

// location renders file:line.
func location(file string, line int) string {
	return fmt.Sprintf("%s:%d", file, line)
}
