package aireport

import (
	"strings"
	"testing"
)

// answerA is the validated answer carried by
// testdata/opencode_streams/answer_only.jsonl.
func answerA() AIAnswer {
	return AIAnswer{
		Summary: "The project carries a CRAPload of 7 across 12 functions, and 4 of them sit in the Q4 Dangerous quadrant. " +
			"The riskiest code is the queue message handler, which pairs complexity 18 with 0% line coverage for a CRAP score of 342. " +
			"Average contract coverage is 43%, and 9 of 15 classified side effects are still ambiguous, so many existing tests cannot be credited with verifying a contract. " +
			"Quality analysis ran in degraded mode for one package, and 2 tests were skipped because no target could be resolved.",
		Recommendations: []string{
			"Decompose and test `(*队列处理器).处理消息` — complexity 18 with 0% coverage gives a CRAP score of 342.",
			"Add assertions to `validateInput` — 50% line coverage but 0% contract coverage drives its GazeCRAP to 110.",
			"Decompose `(*Parser).parse|Pipe` — complexity 16 keeps it in Q4 Dangerous regardless of coverage.",
			"Document the contractual side effects of ambiguous functions so 9 ambiguous effects can be classified.",
		},
	}
}

// answerB is a second valid answer, distinct from answerA.
func answerB() AIAnswer {
	return AIAnswer{
		Summary:         "Coverage is concentrated in simple code while the complex paths stay untested.",
		Recommendations: []string{"Add tests for `loadConfig` — it has no line coverage."},
	}
}

// answerC is a third valid answer, distinct from answerA and answerB.
func answerC() AIAnswer {
	return AIAnswer{
		Summary: "Contract verification is thin: most classified effects remain ambiguous.",
		Recommendations: []string{
			"Strengthen assertions for `validateInput`.",
			"Decompose `(*Parser).parse|Pipe`.",
			"Document ambiguous effects.",
		},
	}
}

// markdownGoldenNames lists every golden markdown file the renderer tests
// pin. TestGoldenHelpers_SharedFiles uses it to catch an orphaned golden on
// disk, which would otherwise never be regenerated or reviewed.
var markdownGoldenNames = []string{
	"report-full.md",
	"report-full-ai.md",
	"report-full-rejected.md",
	"report-crap.md",
	"report-quality.md",
	"report-partial.md",
}

// TestRenderMarkdown_Golden pins the rendered bytes of every report shape.
func TestRenderMarkdown_Golden(t *testing.T) {
	rejected := AIOutcome{
		FailureNote: "AI narrative rejected: summary exceeds length bound (1201 > 1200 characters)",
	}

	cases := []struct {
		name     string
		fixture  string
		sections []string
		outcome  AIOutcome
		golden   string
	}{
		{"full-no-ai", "payload-full.json", nil, AIOutcome{}, "report-full.md"},
		{"full-ai", "payload-full.json", nil, AIOutcome{Answer: answerPtr(answerA())}, "report-full-ai.md"},
		{"full-rejected", "payload-full.json", nil, rejected, "report-full-rejected.md"},
		{"crap-only", "payload-full.json", []string{"crap"}, AIOutcome{}, "report-crap.md"},
		{"quality-only", "payload-full.json", []string{"quality"}, AIOutcome{}, "report-quality.md"},
		{"partial", "payload-partial.json", nil, AIOutcome{}, "report-partial.md"},
	}

	pinned := make(map[string]bool, len(cases))
	for _, tc := range cases {
		pinned[tc.golden] = true
		t.Run(tc.name, func(t *testing.T) {
			got := renderFixture(t, tc.fixture, tc.outcome, tc.sections...)
			assertGolden(t, tc.golden, got)
		})
	}

	// The declared name list and the case table must agree in both
	// directions, so neither can drift without a failure.
	for _, name := range markdownGoldenNames {
		if !pinned[name] {
			t.Errorf("markdownGoldenNames lists %s but no case renders it", name)
		}
	}
	if len(pinned) != len(markdownGoldenNames) {
		t.Errorf("case table pins %d goldens, markdownGoldenNames lists %d", len(pinned), len(markdownGoldenNames))
	}
}

// TestRenderMarkdown_ByteIdentity verifies SC-002: every valid answer
// differs from the no-AI report only inside the AI slots.
func TestRenderMarkdown_ByteIdentity(t *testing.T) {
	p := loadPayloadFixture(t, "payload-full.json")

	base, err := RenderMarkdown(p, AIOutcome{})
	if err != nil {
		t.Fatalf("RenderMarkdown(no AI): %v", err)
	}
	// Rule 11 normalizes the report's trailing newline, so the final
	// slot's trailing separator is stripped along with it; compare the
	// bodies with trailing newlines removed.
	want := strings.TrimRight(stripAISlots(base), "\n")

	answers := []struct {
		name   string
		answer AIAnswer
	}{
		{"answerA", answerA()},
		{"answerB", answerB()},
		{"answerC", answerC()},
	}

	for _, tc := range answers {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderMarkdown(p, AIOutcome{Answer: answerPtr(tc.answer)})
			if err != nil {
				t.Fatalf("RenderMarkdown(%s): %v", tc.name, err)
			}
			stripped := strings.TrimRight(stripAISlots(got), "\n")
			if stripped != want {
				t.Errorf("report outside AI slots differs with %s:\n%s", tc.name, goldenDiff(want, stripped))
			}
		})
	}
}

// answerPtr returns a pointer to a copy of answer.
func answerPtr(answer AIAnswer) *AIAnswer {
	return &answer
}
