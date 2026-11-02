package aireport

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// TestIncident_NarrationReplay_ExitsNonZero replays the CI incident stream
// (FR-036, SC-001): a model that narrates instead of answering must fail the
// run with rule not_json while the deterministic report is still published,
// with no narration reaching the report.
//
// No testing.Short() guard: this is the required incident regression gate, so
// it must run in the standard short suite even though it builds and starts the
// fake opencode binary.
func TestIncident_NarrationReplay_ExitsNonZero(t *testing.T) {
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)
	t.Setenv("FAKE_OPENCODE_STREAM", filepath.Join("testdata", "opencode_streams", "incident.jsonl"))

	payload := loadPayloadFixture(t, "payload-full.json")
	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Adapter:     adapter,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected a non-nil error for the incident narration stream")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected *AnswerRejection, got %T: %v", err, err)
	}
	if rejection.Rule != ruleNotJSON {
		t.Errorf("Rule = %q, want %q", rejection.Rule, ruleNotJSON)
	}

	report := stdout.String()
	for _, heading := range []string{
		"## 📊 CRAP Summary",
		"## 🧪 Quality Summary",
		"## 🏷️ Classification Summary",
		"## 🏥 Overall Health Assessment",
		"> ⚠️ AI narrative rejected:",
	} {
		if !strings.Contains(report, heading) {
			t.Errorf("stdout missing %q", heading)
		}
	}

	for _, narration := range []string{"Let me", "I will", "首先", "CI 源确实存在"} {
		if strings.Contains(report, narration) {
			t.Errorf("stdout contains narration %q", narration)
		}
	}

	// CJK is legitimate in the analysis-derived text (function names), so the
	// guard is that no CJK line appears beyond a deterministic-render baseline
	// or the rejection callout.
	baseline, renderErr := RenderMarkdown(payload, AIOutcome{})
	if renderErr != nil {
		t.Fatalf("rendering baseline: %v", renderErr)
	}
	allowed := map[string]bool{}
	for _, line := range cjkLines(baseline) {
		allowed[line] = true
	}
	for _, line := range cjkLines(report) {
		if allowed[line] || strings.Contains(line, "AI narrative rejected") {
			continue
		}
		t.Errorf("CJK outside the rejection callout: %q", line)
	}
}

// TestIncident_ChineseAnswer_ExitsNonZero replays an answer written in Chinese
// and requires the run to fail with rule non_english.
//
// No testing.Short() guard: this is the required incident regression gate, so
// it must run in the standard short suite even though it builds and starts the
// fake opencode binary.
func TestIncident_ChineseAnswer_ExitsNonZero(t *testing.T) {
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)
	t.Setenv("FAKE_OPENCODE_STREAM", filepath.Join("testdata", "opencode_streams", "chinese_answer.jsonl"))

	payload := loadPayloadFixture(t, "payload-full.json")
	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}

	var stdout bytes.Buffer
	err := Run(RunnerOptions{
		Patterns:    []string{"./..."},
		Format:      "text",
		Stdout:      &stdout,
		Stderr:      &bytes.Buffer{},
		Adapter:     adapter,
		AnalyzeFunc: fakeAnalyze(payload, nil),
	})
	if err == nil {
		t.Fatal("expected a non-nil error for a Chinese answer")
	}

	var rejection *AnswerRejection
	if !errors.As(err, &rejection) {
		t.Fatalf("expected *AnswerRejection, got %T: %v", err, err)
	}
	if rejection.Rule != ruleNonEnglish {
		t.Errorf("Rule = %q, want %q", rejection.Rule, ruleNonEnglish)
	}
	if !strings.Contains(stdout.String(), "> ⚠️ AI narrative rejected:") {
		t.Errorf("expected a rejection callout on stdout")
	}
}

// cjkLines returns the lines of s that contain a Han character.
func cjkLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		for _, r := range line {
			if unicode.Is(unicode.Han, r) {
				lines = append(lines, line)
				break
			}
		}
	}
	return lines
}
