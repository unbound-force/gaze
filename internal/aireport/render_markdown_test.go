package aireport

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

// renderFixture loads a payload fixture, optionally restricts its
// sections, and renders it.
func renderFixture(t *testing.T, name string, outcome AIOutcome, sections ...string) string {
	t.Helper()
	p := loadPayloadFixture(t, name)
	if len(sections) > 0 {
		set, err := ParseSections(sections)
		if err != nil {
			t.Fatalf("ParseSections(%v): %v", sections, err)
		}
		p.Metadata.Sections = set.List()
	}
	out, err := RenderMarkdown(p, outcome)
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	return out
}

// validAnswer is a valid AIAnswer used by slot tests.
func validAnswer() *AIAnswer {
	return &AIAnswer{
		Summary:         "The project is in fair shape.",
		Recommendations: []string{"Decompose `runQuality`.", "Add tests for `parseConfig`."},
	}
}

func TestRenderMarkdown_NilPayload(t *testing.T) {
	if _, err := RenderMarkdown(nil, AIOutcome{}); err == nil {
		t.Fatal("RenderMarkdown(nil) = nil error, want error")
	}
}

func TestRenderMarkdown_Titles(t *testing.T) {
	tests := []struct {
		name     string
		sections []string
		want     string
	}{
		{"all sections", []string{"crap", "quality", "classification", "health"}, "# 🔍 Gaze Full Quality Report"},
		{"crap only", []string{"crap"}, "# 🔍 Gaze CRAP Report"},
		{"quality only", []string{"quality"}, "# 🔍 Gaze Quality Report"},
		{"crap and quality", []string{"quality", "crap"}, "# 🔍 Gaze Quality Report (sections: crap, quality)"},
		{"classification and health", []string{"health", "classification"}, "# 🔍 Gaze Quality Report (sections: classification, health)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderFixture(t, "payload-full.json", AIOutcome{}, tt.sections...)
			if !strings.HasPrefix(out, tt.want+"\n") {
				t.Errorf("first line = %q, want %q", firstLine(out), tt.want)
			}
		})
	}
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func TestRenderMarkdown_MetadataBranchOmitted(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = ""
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Branch:") || strings.Contains(out, " · \n") {
		t.Errorf("empty Branch must be omitted with its separator; got header %q", headerOf(out))
	}
	if !strings.Contains(out, "Project: github.com/example/project\n") {
		t.Errorf("project line malformed; got header %q", headerOf(out))
	}
}

// headerOf returns the metadata header (the first four lines).
func headerOf(s string) string {
	lines := strings.SplitN(s, "\n", 5)
	if len(lines) < 3 {
		return s
	}
	return strings.Join(lines[:3], "\n")
}

func TestRenderMarkdown_MetadataEscaped(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Metadata.Branch = "ev[il](https://x) <!--"
	p.Metadata.ModulePath = "github.com/example/<script>"
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	header := headerOf(out)
	want := `Project: github.com/example/\<script\> · Branch: ev\[il\](https://x) \<!--`
	if !strings.Contains(header, want) {
		t.Errorf("header %q does not contain escaped values %q", header, want)
	}
	for _, raw := range []string{"[", "]", "<", ">"} {
		for _, idx := range allIndexes(header, raw) {
			if idx == 0 || header[idx-1] != '\\' {
				t.Errorf("header %q contains unescaped %q at byte %d", header, raw, idx)
			}
		}
	}
}

// allIndexes returns every byte offset of sub in s.
func allIndexes(s, sub string) []int {
	var out []int
	for i := 0; i < len(s); {
		j := strings.Index(s[i:], sub)
		if j < 0 {
			break
		}
		out = append(out, i+j)
		i += j + 1
	}
	return out
}

func TestRenderMarkdown_UnselectedSectionsOmitted(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{}, "crap")
	for _, heading := range []string{"## 🧪 Quality Summary", "## 🏷️ Classification Summary", "## 🏥 Overall Health Assessment"} {
		if strings.Contains(out, heading) {
			t.Errorf("crap-only report contains %q", heading)
		}
	}
	if !strings.Contains(out, "## 📊 CRAP Summary") {
		t.Error("crap-only report missing the CRAP section")
	}
}

func TestRenderMarkdown_SectionSeparators(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{})
	if got, want := strings.Count(out, "\n---\n"), 4; got != want {
		t.Errorf("separator count = %d, want %d", got, want)
	}
}

func TestRenderMarkdown_TrailingNewline(t *testing.T) {
	for _, sections := range [][]string{
		{"crap", "quality", "classification", "health"},
		{"crap"},
	} {
		out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}, sections...)
		if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
			t.Errorf("sections %v: report must end with exactly one newline", sections)
		}
	}
}

func TestRenderMarkdown_CellEscaping(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}

	longName := strings.Repeat("processGeneratedProtocolBufferMessageForTheExtremelyLongServiceName", 3)[:200]
	if !strings.Contains(out, codeSpan(longName)) {
		t.Error("200-rune function name was truncated")
	}
	if !strings.Contains(out, "``parse`raw`Literal``") {
		t.Error("backtick name is not in a double-backtick span")
	}
	if !strings.Contains(out, `parse\|Pipe`) {
		t.Error("pipe inside a function name was not escaped")
	}
	if !strings.Contains(out, "(*队列处理器).处理消息") {
		t.Error("CJK function name missing")
	}

	var report struct {
		Summary struct {
			Worst []struct {
				Function string `json:"function"`
			} `json:"worst_crap"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(p.CRAP, &report); err != nil {
		t.Fatal(err)
	}
	for _, row := range report.Summary.Worst {
		if strings.Contains(row.Function, "\n") {
			t.Errorf("fixture function %q unexpectedly contains a newline", row.Function)
		}
	}
}

func TestRenderMarkdown_NewlineInCell(t *testing.T) {
	if got, want := escapeCell("a\nb\rc\r\nd"), "a b c d"; got != want {
		t.Errorf("escapeCell = %q, want %q", got, want)
	}
	if got, want := escapeCell("a|b"), `a\|b`; got != want {
		t.Errorf("escapeCell = %q, want %q", got, want)
	}
}

func TestRenderMarkdown_FailedStepCallout(t *testing.T) {
	out := renderFixture(t, "payload-partial.json", AIOutcome{})
	if !strings.Contains(out, "## 🧪 Quality Summary") {
		t.Error("failed quality section must keep its heading")
	}
	want := "> ⚠️ Quality analysis unavailable: resolving packages for quality: go list ./...: exit status 1"
	if !strings.Contains(out, want) {
		t.Errorf("missing failed-step callout %q", want)
	}
}

func TestRenderMarkdown_CalloutBounded(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	detail := "first line\rsecond line"
	long := strings.Repeat("é", 200)
	errorText := detail
	p.Errors.CRAP = &errorText
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ CRAP analysis unavailable: first line\n") {
		t.Error("callout did not take only the first line")
	}

	errorText = long
	out, err = RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ CRAP analysis unavailable: "+strings.Repeat("é", 100)) {
		t.Error("callout did not truncate to 200 bytes on a rune boundary")
	}
}

func TestRenderMarkdown_GazeCRAPloadUnavailable(t *testing.T) {
	out := renderFixture(t, "payload-partial.json", AIOutcome{}, "crap")
	if !strings.Contains(out, "| GazeCRAPload | ⚪ N/A |") {
		t.Error("absent GazeCRAPload must render ⚪ N/A")
	}
}

func TestRenderMarkdown_ZeroTestsCallout(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	var quality struct {
		Reports []json.RawMessage `json:"quality_reports"`
		Summary map[string]any    `json:"quality_summary"`
	}
	if err := json.Unmarshal(p.Quality, &quality); err != nil {
		t.Fatal(err)
	}
	quality.Summary["total_tests"] = 0
	quality.Summary["worst_coverage_tests"] = []any{}
	raw, err := json.Marshal(quality)
	if err != nil {
		t.Fatal(err)
	}
	p.Quality = raw
	p.Metadata.Sections = sectionSetOf(SectionQuality).List()

	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "> ⚠️ Quality analysis found 0 tests for the selected packages.") {
		t.Error("missing zero-tests callout")
	}
	if strings.Contains(out, "### Worst Tests by Contract Coverage") {
		t.Error("zero-tests report must not render the worst-tests table")
	}
}

func TestRenderMarkdown_DocumentationNoFiles(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Docscan = json.RawMessage(`{"documents":[],"api_coverage":null}`)
	p.Metadata.Sections = sectionSetOf(SectionHealth).List()
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Documentation: no files found.\n") {
		t.Error("missing no-documents line")
	}
}

func TestRenderMarkdown_ClosedVocabulary(t *testing.T) {
	allowed := map[rune]bool{
		'🔍': true, '📊': true, '🧪': true, '🏷': true, '🏥': true,
		'🟢': true, '🟡': true, '🔴': true, '⚪': true, '⚠': true,
	}
	out := stripAISlots(renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}))
	for _, r := range out {
		if unicode.Is(unicode.So, r) && !allowed[r] {
			t.Errorf("report contains emoji %q outside the closed vocabulary", r)
		}
	}
}

func TestRenderMarkdown_AISlotPlacement(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()})
	summarySlot := strings.Index(out, "<!-- gaze:ai-summary -->")
	crapHeading := strings.Index(out, "## 📊 CRAP Summary")
	recsSlot := strings.Index(out, "<!-- gaze:ai-recommendations -->")
	deterministicRecs := strings.Index(out, "### Recommendations")

	if summarySlot < 0 || crapHeading < 0 || recsSlot < 0 || deterministicRecs < 0 {
		t.Fatalf("missing markers: summary=%d crap=%d recs=%d deterministic=%d", summarySlot, crapHeading, recsSlot, deterministicRecs)
	}
	if summarySlot > crapHeading {
		t.Error("summary slot must precede the first section")
	}
	if recsSlot < deterministicRecs {
		t.Error("recommendations slot must follow the deterministic recommendations")
	}
	if !strings.Contains(out, "1. Decompose `runQuality`.\n2. Add tests for `parseConfig`.\n") {
		t.Error("AI recommendations are not rendered as a numbered list")
	}
}

func TestRenderMarkdown_AIRecommendationsSlotWhenHealthNotSelected(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{Answer: validAnswer()}, "crap")
	if !strings.HasSuffix(strings.TrimRight(out, "\n"), "<!-- /gaze:ai-recommendations -->") {
		t.Error("recommendations slot must sit at the end when health is not selected")
	}
	if strings.Contains(out, "### Recommendations") {
		t.Error("deterministic recommendations must not render without the health section")
	}
}

func TestRenderMarkdown_ZeroOutcomeOmitsSlots(t *testing.T) {
	out := renderFixture(t, "payload-full.json", AIOutcome{})
	for _, marker := range []string{"gaze:ai-summary", "gaze:ai-recommendations", "AI-Prioritized"} {
		if strings.Contains(out, marker) {
			t.Errorf("zero outcome must omit AI slots; found %q", marker)
		}
	}
}

func TestRenderMarkdown_RejectedOutcome(t *testing.T) {
	outcome := AIOutcome{FailureNote: "AI narrative rejected: summary exceeds length bound (1342 > 1200 characters)"}
	out := renderFixture(t, "payload-full.json", outcome)
	if !strings.Contains(out, "> ⚠️ AI narrative rejected: summary exceeds length bound (1342 > 1200 characters)") {
		t.Error("missing rejection callout")
	}
	if strings.Contains(out, "gaze:ai-recommendations") {
		t.Error("rejection must not render a recommendations slot")
	}
	if strings.Contains(out, "### Summary\n\nThe project") {
		t.Error("rejection must not render an answer body")
	}
}

func TestRenderMarkdown_UnavailableOutcome(t *testing.T) {
	outcome := AIOutcome{FailureNote: "AI narrative unavailable: adapter exited with status 1"}
	out := renderFixture(t, "payload-full.json", outcome)
	if !strings.Contains(out, "> ⚠️ AI narrative unavailable: adapter exited with status 1") {
		t.Error("missing unavailable callout")
	}
	if strings.Contains(out, "gaze:ai-recommendations") {
		t.Error("unavailable outcome must not render a recommendations slot")
	}
}

func TestRenderMarkdown_NoRecommendations(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")
	p.Summary = ReportSummary{TotalFunctions: 1}
	p.CRAP = json.RawMessage(`{"scores":[],"summary":{"total_functions":1,"avg_complexity":1,"avg_line_coverage":100,"avg_crap":1,"crapload":0,"crap_threshold":15,"quadrant_counts":{},"fix_strategy_counts":{},"worst_crap":[],"worst_gaze_crap":[]}}`)
	p.Assessment = nil
	out, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No recommendations — no functions at or above the CRAP threshold.") {
		t.Error("missing empty-recommendations line")
	}
}
