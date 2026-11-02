package aireport

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// validAnswerJSON is the smallest answer that passes every rule.
const validAnswerJSON = `{"summary":"The project is in fair shape.","recommendations":["Add tests for ` + "`run`" + `."]}`

// TestParseAIAnswer_Valid covers the accepted shapes, including the
// nearest-passing case for every bound.
func TestParseAIAnswer_Valid(t *testing.T) {
	maxSummary := strings.Repeat("a", maxSummaryRunes)
	maxRec := strings.Repeat("b", maxRecRunes)
	fiveRecs := `["Add tests.","Add tests.","Add tests.","Add tests.","Add tests."]`
	fivePercent := `{"summary":"abcdefghijklmnopqrsΩ","recommendations":["Add tests."]}`
	cjkIdentifier := `{"summary":"The queue handler ` + "`处理消息`" + ` is the riskiest function in this report because its complexity and coverage combine badly.","recommendations":["Add tests."]}`

	cases := []struct {
		name string
		raw  string
		want AIAnswer
	}{
		{
			name: "minimal answer",
			raw:  validAnswerJSON,
			want: AIAnswer{
				Summary:         "The project is in fair shape.",
				Recommendations: []string{"Add tests for `run`."},
			},
		},
		{
			name: "values are trimmed",
			raw:  `{"summary":"  Trimmed.  ","recommendations":["  Add tests.  "]} `,
			want: AIAnswer{Summary: "Trimmed.", Recommendations: []string{"Add tests."}},
		},
		{
			name: "backtick fence",
			raw:  "```\n" + validAnswerJSON + "\n```",
			want: AIAnswer{
				Summary:         "The project is in fair shape.",
				Recommendations: []string{"Add tests for `run`."},
			},
		},
		{
			name: "json fence",
			raw:  "```json\n" + validAnswerJSON + "\n```",
			want: AIAnswer{
				Summary:         "The project is in fair shape.",
				Recommendations: []string{"Add tests for `run`."},
			},
		},
		{
			name: "emoji are ignored",
			raw:  `{"summary":"Health is fine 🙂.","recommendations":["Add tests 🙂."]}`,
			want: AIAnswer{Summary: "Health is fine 🙂.", Recommendations: []string{"Add tests 🙂."}},
		},
		{
			name: "inline emphasis and code are allowed",
			raw:  `{"summary":"Use ` + "`foo`" + ` only *carefully*.","recommendations":["Add tests."]}`,
			want: AIAnswer{Summary: "Use `foo` only *carefully*.", Recommendations: []string{"Add tests."}},
		},
		{
			name: "cjk identifier in code span with english prose",
			raw:  cjkIdentifier,
			want: AIAnswer{
				Summary:         "The queue handler `处理消息` is the riskiest function in this report because its complexity and coverage combine badly.",
				Recommendations: []string{"Add tests."},
			},
		},
		{
			name: "summary at the upper bound",
			raw:  `{"summary":"` + maxSummary + `","recommendations":["Add tests."]}`,
			want: AIAnswer{Summary: maxSummary, Recommendations: []string{"Add tests."}},
		},
		{
			name: "recommendation at the upper bound",
			raw:  `{"summary":"Fine.","recommendations":["` + maxRec + `"]}`,
			want: AIAnswer{Summary: "Fine.", Recommendations: []string{maxRec}},
		},
		{
			name: "five recommendations at the upper bound",
			raw:  `{"summary":"Fine.","recommendations":` + fiveRecs + `}`,
			want: AIAnswer{
				Summary:         "Fine.",
				Recommendations: []string{"Add tests.", "Add tests.", "Add tests.", "Add tests.", "Add tests."},
			},
		},
		{
			name: "exactly five percent non-latin",
			raw:  fivePercent,
			want: AIAnswer{Summary: "abcdefghijklmnopqrsΩ", Recommendations: []string{"Add tests."}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAIAnswer(tc.raw)
			if err != nil {
				t.Fatalf("ParseAIAnswer() error = %v, want nil", err)
			}

			if got.Summary != tc.want.Summary {
				t.Errorf("Summary = %q, want %q", got.Summary, tc.want.Summary)
			}

			if len(got.Recommendations) != len(tc.want.Recommendations) {
				t.Fatalf("Recommendations = %v, want %v", got.Recommendations, tc.want.Recommendations)
			}

			for i := range tc.want.Recommendations {
				if got.Recommendations[i] != tc.want.Recommendations[i] {
					t.Errorf("Recommendations[%d] = %q, want %q", i, got.Recommendations[i], tc.want.Recommendations[i])
				}
			}
		})
	}
}

// TestParseAIAnswer_Rejections covers every rule with a minimal
// violation and asserts the rule identifier.
func TestParseAIAnswer_Rejections(t *testing.T) {
	longSummary := `{"summary":"` + strings.Repeat("a", maxSummaryRunes+1) + `","recommendations":["Add tests."]}`
	longRec := `{"summary":"Fine.","recommendations":["` + strings.Repeat("b", maxRecRunes+1) + `"]}`
	sixRecs := `{"summary":"Fine.","recommendations":["a","b","c","d","e","f"]}`

	cases := []struct {
		name string
		rule string
		raw  string
	}{
		{"empty output", ruleEmptyOutput, ""},
		{"whitespace only", ruleEmptyOutput, "   \n\t "},
		{"prose before object", ruleSurroundingText, `Here is the answer: ` + validAnswerJSON},
		{"prose after object", ruleSurroundingText, validAnswerJSON + ` Hope that helps.`},
		{"fence plus trailing prose", ruleSurroundingText, "```\n" + validAnswerJSON + "\n```\nThanks."},
		{"not json", ruleNotJSON, `{summary: "x"}`},
		{"multiple values", ruleMultipleValues, validAnswerJSON + ` {}`},
		{"not an object", ruleNotObject, "```\n[\"summary\"]\n```"},
		{"duplicate field", ruleDuplicateField, `{"summary":"a","summary":"b","recommendations":["Add tests."]}`},
		{"unknown field", ruleUnknownField, `{"summary":"a","recommendations":["Add tests."],"notes":"x"}`},
		{"missing summary", ruleMissingField, `{"recommendations":["Add tests."]}`},
		{"missing recommendations", ruleMissingField, `{"summary":"a"}`},
		{"summary null", ruleWrongType, `{"summary":null,"recommendations":["Add tests."]}`},
		{"recommendations not strings", ruleWrongType, `{"summary":"a","recommendations":[1]}`},
		{"summary too long", ruleSummaryLength, longSummary},
		{"summary empty", ruleSummaryLength, `{"summary":"   ","recommendations":["Add tests."]}`},
		{"too many recommendations", ruleRecommendationCount, sixRecs},
		{"no recommendations", ruleRecommendationCount, `{"summary":"Fine.","recommendations":[]}`},
		{"recommendation too long", ruleRecommendationLength, longRec},
		{"recommendation empty", ruleRecommendationLength, `{"summary":"Fine.","recommendations":["  "]}`},
		{"recommendation line break", ruleRecommendationLength, `{"summary":"Fine.","recommendations":["Add tests.\nThen more."]}`},
		{"carriage return", ruleMarkup, `{"summary":"Line one\rline two","recommendations":["Add tests."]}`},
		{"heading", ruleMarkup, `{"summary":"# Heading","recommendations":["Add tests."]}`},
		{"blockquote", ruleMarkup, `{"summary":"> quoted","recommendations":["Add tests."]}`},
		{"table", ruleMarkup, `{"summary":"| a | b |","recommendations":["Add tests."]}`},
		{"fence", ruleMarkup, `{"summary":"` + "```x```" + `","recommendations":["Add tests."]}`},
		{"unordered list", ruleMarkup, `{"summary":"- item","recommendations":["Add tests."]}`},
		{"ordered list", ruleMarkup, `{"summary":"1. item","recommendations":["Add tests."]}`},
		{"thematic break", ruleMarkup, `{"summary":"---","recommendations":["Add tests."]}`},
		{"html tag", ruleMarkup, `{"summary":"a <a href=\"x\">link</a>","recommendations":["Add tests."]}`},
		{"html comment", ruleMarkup, `{"summary":"a <!-- hidden --> b","recommendations":["Add tests."]}`},
		{"inline link", ruleMarkup, `{"summary":"See [docs](https://x).","recommendations":["Add tests."]}`},
		{"reference link", ruleMarkup, `{"summary":"See [docs][ref].","recommendations":["Add tests."]}`},
		{"bare url", ruleMarkup, `{"summary":"See https://example.com for details.","recommendations":["Add tests."]}`},
		{"www autolink", ruleMarkup, `{"summary":"See www.example.com for details.","recommendations":["Add tests."]}`},
		{"recommendation markup", ruleMarkup, `{"summary":"Fine.","recommendations":["# Do the thing"]}`},
		{"non-english summary", ruleNonEnglish, `{"summary":"这是一个中文总结。","recommendations":["Add tests."]}`},
		{"non-english above limit", ruleNonEnglish, `{"summary":"abcdefghijklmnopqrΩ","recommendations":["Add tests."]}`},
		{"no letters", ruleNonEnglish, `{"summary":"12345 !!!","recommendations":["Add tests."]}`},
		{"non-english recommendation", ruleNonEnglish, `{"summary":"Fine.","recommendations":["中文建议。"]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAIAnswer(tc.raw)
			if err == nil {
				t.Fatalf("ParseAIAnswer() = nil error, want rejection %q", tc.rule)
			}

			var rejection *AnswerRejection
			if !errors.As(err, &rejection) {
				t.Fatalf("error type = %T, want *AnswerRejection", err)
			}

			if rejection.Rule != tc.rule {
				t.Errorf("Rule = %q, want %q (detail %q)", rejection.Rule, tc.rule, rejection.Detail)
			}

			if rejection.Detail == "" {
				t.Error("Detail is empty")
			}

			if len(rejection.Detail) > maxDetailBytes {
				t.Errorf("Detail is %d bytes, want <= %d", len(rejection.Detail), maxDetailBytes)
			}
		})
	}
}

// TestAnswerRejection_Error pins the error text.
func TestAnswerRejection_Error(t *testing.T) {
	rejection := &AnswerRejection{Rule: ruleSummaryLength, Detail: "summary is empty"}

	want := "AI answer rejected (rule summary_length): summary is empty"
	if got := rejection.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestParseAIAnswer_DetailDoesNotLeakContent verifies the detail
// carries counts only, never the answer's text.
func TestParseAIAnswer_DetailDoesNotLeakContent(t *testing.T) {
	secret := "SECRETSUMMARY"
	raw := `{"summary":"` + strings.Repeat(secret, 200) + `","recommendations":["Add tests."]}`

	_, err := ParseAIAnswer(raw)
	if err == nil {
		t.Fatal("ParseAIAnswer() = nil error, want rejection")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("error type = %T, want *AnswerRejection", err)
	}

	if strings.Contains(rejection.Detail, secret) {
		t.Errorf("Detail leaks answer content: %q", rejection.Detail)
	}
}

// TestParseAIAnswer_Bounds pins the exact boundary values.
func TestParseAIAnswer_Bounds(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		reject bool
	}{
		{"summary 1200", validAnswerWithSummary(strings.Repeat("a", 1200)), false},
		{"summary 1201", validAnswerWithSummary(strings.Repeat("a", 1201)), true},
		{"recommendation 400", `{"summary":"Fine.","recommendations":["` + strings.Repeat("b", 400) + `"]}`, false},
		{"recommendation 401", `{"summary":"Fine.","recommendations":["` + strings.Repeat("b", 401) + `"]}`, true},
		{"one recommendation", `{"summary":"Fine.","recommendations":["Add tests."]}`, false},
		{"five recommendations", `{"summary":"Fine.","recommendations":["a","b","c","d","e"]}`, false},
		{"six recommendations", `{"summary":"Fine.","recommendations":["a","b","c","d","e","f"]}`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseAIAnswer(tc.raw)
			if tc.reject && err == nil {
				t.Error("ParseAIAnswer() = nil error, want rejection")
			}
			if !tc.reject && err != nil {
				t.Errorf("ParseAIAnswer() error = %v, want nil", err)
			}
		})
	}
}

// validAnswerWithSummary builds an answer with the given summary text.
func validAnswerWithSummary(summary string) string {
	return `{"summary":"` + summary + `","recommendations":["Add tests."]}`
}

// TestParseAIAnswer_CodeSpanEnglishBound pins the non-English rule for code
// spans: non-Latin identifiers copied from the analysis data are allowed,
// while non-English prose hidden in backticks is rejected.
func TestParseAIAnswer_CodeSpanEnglishBound(t *testing.T) {
	identifier := "Decompose and test `(*队列处理器).处理消息` before any further changes; complexity 18 gives a CRAP score of 342."
	if _, err := ParseAIAnswer(validAnswerWithRecommendation(identifier)); err != nil {
		t.Fatalf("non-Latin identifier in a code span must be accepted: %v", err)
	}

	hidden := "This is prose `" + strings.Repeat("中文内容", 40) + "` and little else."
	_, err := ParseAIAnswer(validAnswerWithRecommendation(hidden))
	if err == nil {
		t.Fatal("expected non-English prose hidden in a code span to be rejected")
	}
	var rejection *AnswerRejection
	if !errors.As(err, &rejection) || rejection.Rule != ruleNonEnglish {
		t.Fatalf("Rule = %v, want %q", err, ruleNonEnglish)
	}
}

// validAnswerWithRecommendation builds an answer around one recommendation.
func validAnswerWithRecommendation(rec string) string {
	data, err := json.Marshal(map[string]any{
		"summary":         "The project carries a CRAPload of 7 across 12 functions.",
		"recommendations": []string{rec},
	})
	if err != nil {
		panic(err)
	}

	return string(data)
}
