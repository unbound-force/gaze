package aireport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// AIAnswer is the only judgment shape accepted from an AI adapter.
//
// It carries the free-form parts of the report — a narrative and a
// prioritized list of recommendations — and nothing else. Every table,
// grade, metric, and deterministic recommendation is rendered by Gaze.
type AIAnswer struct {
	// Summary is the bounded, trimmed narrative.
	Summary string `json:"summary"`

	// Recommendations are the trimmed, prioritized recommendation texts,
	// in the order the model produced them.
	Recommendations []string `json:"recommendations"`
}

// AnswerRejection reports why an adapter's output was not accepted.
//
// Rule names the validation rule that failed, and Detail explains the
// failure with counts, field names, and indices only — never with
// content copied from the answer.
type AnswerRejection struct {
	// Rule is the identifier of the validation rule that failed.
	Rule string `json:"rule"`

	// Detail is a bounded, content-free explanation of the failure.
	Detail string `json:"detail"`
}

// Error renders the rejection in the documented form.
func (r *AnswerRejection) Error() string {
	return fmt.Sprintf("AI answer rejected (rule %s): %s", r.Rule, r.Detail)
}

// Validation rule identifiers, in normative evaluation order.
const (
	ruleEmptyOutput          = "empty_output"
	ruleSurroundingText      = "surrounding_text"
	ruleNotJSON              = "not_json"
	ruleMultipleValues       = "multiple_values"
	ruleNotObject            = "not_object"
	ruleDuplicateField       = "duplicate_field"
	ruleUnknownField         = "unknown_field"
	ruleMissingField         = "missing_field"
	ruleWrongType            = "wrong_type"
	ruleSummaryLength        = "summary_length"
	ruleRecommendationCount  = "recommendation_count"
	ruleRecommendationLength = "recommendation_length"
	ruleMarkup               = "markup"
	ruleNonEnglish           = "non_english"
)

// Answer bounds. They are published in docs/reference/cli/report.md.
const (
	maxSummaryRunes    = 1200
	minRecs            = 1
	maxRecs            = 5
	maxRecRunes        = 400
	maxNonLatinPercent = 5
	maxDetailBytes     = 200
)

// rejectionDetailBytes bounds a rejection's detail text.
const rejectionDetailBytes = maxDetailBytes

// reject builds a bounded *AnswerRejection.
func reject(rule, format string, args ...any) error {
	detail := fmt.Sprintf(format, args...)
	return &AnswerRejection{Rule: rule, Detail: truncateDetail(detail)}
}

// truncateDetail bounds s to rejectionDetailBytes on a rune boundary.
func truncateDetail(s string) string {
	return truncateOnRuneBoundary(s, rejectionDetailBytes)
}

// ParseAIAnswer validates raw adapter output and returns the answer.
//
// The validation pipeline is normative: the first failing rule wins, and
// the rules run in the order documented in contracts/ai-answer.md. The
// only normalization is TrimSpace for bound checks; accepted values are
// stored trimmed. Nothing is truncated, translated, repaired, or retried.
// The bounds and rule set are the values fixed in research R4; the markup
// and non-English rules follow research R5.
func ParseAIAnswer(raw string) (AIAnswer, error) {
	if strings.TrimSpace(raw) == "" {
		return AIAnswer{}, reject(ruleEmptyOutput, "AI returned no output")
	}

	body, err := unwrapAnswer(raw)
	if err != nil {
		return AIAnswer{}, err
	}

	object, err := decodeAnswerObject(body)
	if err != nil {
		return AIAnswer{}, err
	}

	summary, recs, err := decodeAnswerMembers(object)
	if err != nil {
		return AIAnswer{}, err
	}

	summary, recs, err = checkAnswerBounds(summary, recs)
	if err != nil {
		return AIAnswer{}, err
	}

	if err := checkAnswerMarkup(summary, recs); err != nil {
		return AIAnswer{}, err
	}

	if err := checkAnswerEnglish(summary, recs); err != nil {
		return AIAnswer{}, err
	}

	return AIAnswer{Summary: summary, Recommendations: recs}, nil
}

// unwrapAnswer enforces the surrounding_text rule: a bare JSON object, or
// exactly one fenced block whose body is the object.
func unwrapAnswer(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		return trimmed, nil
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) >= 3 && isFenceOpening(lines[0]) &&
		strings.TrimSpace(lines[len(lines)-1]) == "```" {
		return strings.Join(lines[1:len(lines)-1], "\n"), nil
	}

	return "", reject(ruleSurroundingText,
		"answer must be a single JSON object (optionally in one code fence) with no surrounding text")
}

// isFenceOpening reports whether line opens a JSON code fence.
func isFenceOpening(line string) bool {
	switch strings.TrimSpace(line) {
	case "```", "```json", "```JSON":
		return true
	default:
		return false
	}
}

// decodeAnswerObject enforces not_json, multiple_values, and not_object.
func decodeAnswerObject(body string) (json.RawMessage, error) {
	dec := json.NewDecoder(strings.NewReader(body))

	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		return nil, reject(ruleNotJSON, "answer is not valid JSON")
	}

	if _, err := dec.Token(); err != io.EOF {
		return nil, reject(ruleMultipleValues, "answer contains more than one JSON value")
	}

	if len(raw) == 0 || raw[0] != '{' {
		return nil, reject(ruleNotObject, "answer must be a JSON object")
	}

	return raw, nil
}

// memberValues holds the two accepted members as decoded JSON values.
type memberValues struct {
	summary json.RawMessage
	recs    json.RawMessage
}

// decodeAnswerMembers enforces duplicate_field, unknown_field,
// missing_field, and wrong_type.
func decodeAnswerMembers(object json.RawMessage) (string, []string, error) {
	values, err := walkAnswerMembers(object)
	if err != nil {
		return "", nil, err
	}

	summary, err := decodeSummaryValue(values.summary)
	if err != nil {
		return "", nil, err
	}

	recs, err := decodeRecommendationsValue(values.recs)
	if err != nil {
		return "", nil, err
	}

	return summary, recs, nil
}

// walkAnswerMembers walks the object's top-level members, detecting
// duplicates and unknown names and recording the two accepted values.
func walkAnswerMembers(object json.RawMessage) (memberValues, error) {
	dec := json.NewDecoder(bytes.NewReader(object))

	var values memberValues
	seen := make(map[string]bool, 2)

	if _, err := dec.Token(); err != nil {
		return values, reject(ruleNotJSON, "answer is not valid JSON")
	}

	for dec.More() {
		if err := readAnswerMember(dec, &values, seen); err != nil {
			return values, err
		}
	}

	if err := requireAnswerMembers(seen); err != nil {
		return values, err
	}

	return values, nil
}

// readAnswerMember reads one object member into values, rejecting duplicate
// and unknown fields.
func readAnswerMember(dec *json.Decoder, values *memberValues, seen map[string]bool) error {
	key, err := dec.Token()
	if err != nil {
		return reject(ruleNotJSON, "answer is not valid JSON")
	}

	name, _ := key.(string)
	if seen[name] {
		return reject(ruleDuplicateField, "field %q appears more than once", name)
	}
	seen[name] = true

	if name != "summary" && name != "recommendations" {
		return reject(ruleUnknownField, "unknown field %q", name)
	}

	var value json.RawMessage
	if err := dec.Decode(&value); err != nil {
		return reject(ruleNotJSON, "answer is not valid JSON")
	}

	if name == "summary" {
		values.summary = value
	} else {
		values.recs = value
	}

	return nil
}

// requireAnswerMembers rejects objects missing a required member.
func requireAnswerMembers(seen map[string]bool) error {
	if !seen["summary"] {
		return reject(ruleMissingField, `missing field "summary"`)
	}
	if !seen["recommendations"] {
		return reject(ruleMissingField, `missing field "recommendations"`)
	}

	return nil
}

// decodeSummaryValue enforces the summary member's type.
func decodeSummaryValue(raw json.RawMessage) (string, error) {
	if isJSONNull(raw) {
		return "", reject(ruleWrongType, `field "summary" must be a string`)
	}

	var summary string
	if err := json.Unmarshal(raw, &summary); err != nil {
		return "", reject(ruleWrongType, `field "summary" must be a string`)
	}

	return summary, nil
}

// decodeRecommendationsValue enforces the recommendations member's type.
func decodeRecommendationsValue(raw json.RawMessage) ([]string, error) {
	if isJSONNull(raw) {
		return nil, reject(ruleWrongType, `field "recommendations" must be an array of strings`)
	}

	var recs []string
	if err := json.Unmarshal(raw, &recs); err != nil {
		return nil, reject(ruleWrongType, `field "recommendations" must be an array of strings`)
	}

	return recs, nil
}

// isJSONNull reports whether a raw JSON value is the null literal.
func isJSONNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// checkAnswerBounds enforces summary_length, recommendation_count, and
// recommendation_length, returning the trimmed values.
func checkAnswerBounds(summary string, recs []string) (string, []string, error) {
	summary = strings.TrimSpace(summary)
	if err := checkSummaryLength(summary); err != nil {
		return "", nil, err
	}

	if len(recs) < minRecs || len(recs) > maxRecs {
		return "", nil, reject(ruleRecommendationCount,
			"recommendation count %d outside bounds %d–%d", len(recs), minRecs, maxRecs)
	}

	trimmed := make([]string, len(recs))
	for i, rec := range recs {
		rec = strings.TrimSpace(rec)
		if err := checkRecommendationLength(i, rec); err != nil {
			return "", nil, err
		}
		trimmed[i] = rec
	}

	return summary, trimmed, nil
}

// checkSummaryLength enforces the summary's rune bounds.
func checkSummaryLength(summary string) error {
	n := utf8.RuneCountInString(summary)
	switch {
	case n == 0:
		return reject(ruleSummaryLength, "summary is empty")
	case n > maxSummaryRunes:
		return reject(ruleSummaryLength,
			"summary exceeds length bound (%d > %d characters)", n, maxSummaryRunes)
	default:
		return nil
	}
}

// checkRecommendationLength enforces one recommendation's rune bounds and
// single-line requirement.
func checkRecommendationLength(index int, rec string) error {
	n := utf8.RuneCountInString(rec)
	switch {
	case n == 0:
		return reject(ruleRecommendationLength, "recommendation %d is empty", index+1)
	case n > maxRecRunes:
		return reject(ruleRecommendationLength,
			"recommendation %d exceeds length bound (%d > %d characters)", index+1, n, maxRecRunes)
	case strings.ContainsAny(rec, "\n\r"):
		return reject(ruleRecommendationLength, "recommendation %d contains a line break", index+1)
	default:
		return nil
	}
}

// checkAnswerMarkup enforces the markup rule on every accepted field.
func checkAnswerMarkup(summary string, recs []string) error {
	if detail, ok := markupViolation(summary); ok {
		return reject(ruleMarkup, "summary contains disallowed markup (%s)", detail)
	}

	for i, rec := range recs {
		if detail, ok := markupViolation(rec); ok {
			return reject(ruleMarkup, "recommendation %d contains disallowed markup (%s)", i+1, detail)
		}
	}

	return nil
}

// markupViolation reports the first structural markup form found in s.
func markupViolation(s string) (string, bool) {
	if strings.ContainsRune(s, '\r') {
		return "carriage return", true
	}
	if containsHTMLMarkup(s) {
		return "HTML tag", true
	}
	if strings.Contains(s, "](") || strings.Contains(s, "][") {
		return "link", true
	}
	if strings.Contains(s, "://") || containsFold(s, "www.") {
		return "link", true
	}

	for _, line := range strings.Split(s, "\n") {
		if detail, ok := lineMarkupViolation(line); ok {
			return detail, true
		}
	}

	return "", false
}

// containsHTMLMarkup reports whether s contains an HTML tag or comment
// opening.
func containsHTMLMarkup(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] != '<' {
			continue
		}
		c := s[i+1]
		if c == '!' || c == '/' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}

	return false
}

// containsFold reports whether s contains substr, ignoring ASCII case.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), substr)
}

// lineMarkupViolation reports the markup form that starts a line.
func lineMarkupViolation(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	switch {
	case strings.HasPrefix(trimmed, "#"):
		return "heading", true
	case strings.HasPrefix(trimmed, ">"):
		return "blockquote", true
	case strings.HasPrefix(trimmed, "\\|"), strings.HasPrefix(trimmed, "|"):
		return "table", true
	case strings.HasPrefix(trimmed, "```"), strings.HasPrefix(trimmed, "~~~"):
		return "fence", true
	case isListItem(trimmed):
		return "list", true
	case isThematicBreak(trimmed):
		return "thematic break", true
	default:
		return "", false
	}
}

// isListItem reports whether line opens a Markdown list item.
func isListItem(line string) bool {
	for _, marker := range []string{"- ", "* ", "+ "} {
		if strings.HasPrefix(line, marker) {
			return true
		}
	}

	i := 0
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	if i == 0 || i+1 >= len(line) {
		return false
	}

	return (line[i] == '.' || line[i] == ')') && line[i+1] == ' '
}

// isThematicBreak reports whether line consists only of one repeated
// thematic-break character.
func isThematicBreak(line string) bool {
	if line == "" {
		return false
	}

	c := line[0]
	if c != '-' && c != '=' && c != '*' && c != '_' {
		return false
	}

	for i := 0; i < len(line); i++ {
		if line[i] != c {
			return false
		}
	}

	return true
}

// checkAnswerEnglish enforces the non_english rule on every accepted field.
func checkAnswerEnglish(summary string, recs []string) error {
	if err := checkFieldEnglish("summary", summary); err != nil {
		return err
	}

	for i, rec := range recs {
		if err := checkFieldEnglish(fmt.Sprintf("recommendation %d", i+1), rec); err != nil {
			return err
		}
	}

	return nil
}

// checkFieldEnglish enforces the non-Latin ratio bound for one field.
//
// The ratio compares non-Latin letters written as prose (outside inline code
// spans) against the prose letters. Identifiers copied from the analysis data
// may be non-Latin without tripping the rule. To stop an answer hiding
// non-English prose inside backticks, the non-Latin volume inside code spans
// must stay within the prose volume.
func checkFieldEnglish(field, s string) error {
	counts := codeSpanLetterCounts(s)
	if counts.lettersOutside == 0 {
		return reject(ruleNonEnglish, "non-English text ratio above limit in %s (no letters found)", field)
	}

	if counts.nonLatinOutside*100 > maxNonLatinPercent*counts.lettersOutside {
		return reject(ruleNonEnglish,
			"non-English text ratio above limit in %s (%d%% > %d%%)",
			field, counts.nonLatinOutside*100/counts.lettersOutside, maxNonLatinPercent)
	}

	if counts.nonLatinInside > counts.lettersOutside {
		return reject(ruleNonEnglish,
			"non-English text ratio above limit in %s (code spans carry more non-Latin letters than the prose)",
			field)
	}

	return nil
}

// letterCounts holds the letter counts the English rule needs.
type letterCounts struct {
	lettersOutside  int
	nonLatinOutside int
	nonLatinInside  int
}

// codeSpanLetterCounts counts prose letters, non-Latin prose letters, and
// non-Latin letters inside inline code spans.
func codeSpanLetterCounts(s string) letterCounts {
	var counts letterCounts
	inside := false
	spanTicks := 0
	ticks := 0

	for _, r := range s {
		if r == '`' {
			ticks++
			continue
		}

		if ticks > 0 {
			inside, spanTicks = toggleCodeSpan(inside, spanTicks, ticks)
			ticks = 0
		}

		if !unicode.IsLetter(r) {
			continue
		}

		if inside {
			if !unicode.Is(unicode.Latin, r) {
				counts.nonLatinInside++
			}
			continue
		}

		counts.lettersOutside++
		if !unicode.Is(unicode.Latin, r) {
			counts.nonLatinOutside++
		}
	}

	return counts
}

// toggleCodeSpan updates the code-span state after a run of backticks.
func toggleCodeSpan(inside bool, spanTicks, ticks int) (bool, int) {
	if !inside {
		return true, ticks
	}

	if ticks == spanTicks {
		return false, 0
	}

	return inside, spanTicks
}
