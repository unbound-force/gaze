package aireport

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openCodeStreamsDir holds the recorded opencode event streams.
const openCodeStreamsDir = "testdata/opencode_streams"

// readStream reads one recorded event stream.
func readStream(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(openCodeStreamsDir, name))
	if err != nil {
		t.Fatalf("reading stream %s: %v", name, err)
	}

	return data
}

// TestExtractFinalText_Fixtures runs the extractor over every recorded
// stream. The fixtures are the contract: the adapter must return the final
// answer and nothing else, and it must fail closed on the streams that show
// a tool running or the provider erroring.
func TestExtractFinalText_Fixtures(t *testing.T) {
	cases := []struct {
		name     string
		contains string
		wantErr  string
	}{
		{name: "answer_only.jsonl", contains: `"summary"`},
		{name: "narration_then_answer.jsonl", contains: `"summary"`},
		{name: "tool_refused.jsonl", contains: `"summary"`},
		{name: "chinese_answer.jsonl", contains: "该项目"},
		{name: "summary_too_long.jsonl", contains: `"summary"`},
		{name: "incident.jsonl", contains: "CI 源确实存在"},
		{
			name:    "tool_completed.jsonl",
			wantErr: `although the gaze-headless agent denies all tools`,
		},
		{
			name:    "error_event.jsonl",
			wantErr: "opencode reported an error: APIError: Incorrect API key provided.",
		},
		{
			name:    "not_json.txt",
			wantErr: "opencode produced no JSON events",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := extractFinalText(readStream(t, tc.name))

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("extractFinalText(%s) = %q, want error", tc.name, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("extractFinalText(%s) error: %v", tc.name, err)
			}
			if !strings.Contains(got, tc.contains) {
				t.Fatalf("answer = %q, want it to contain %q", got, tc.contains)
			}
		})
	}
}

// TestExtractFinalText_NarrationDropped proves only the final step's text
// survives, so narration between tool calls never reaches the report.
func TestExtractFinalText_NarrationDropped(t *testing.T) {
	got, _, err := extractFinalText(readStream(t, "narration_then_answer.jsonl"))
	if err != nil {
		t.Fatalf("extractFinalText error: %v", err)
	}
	if strings.Contains(got, "Let me") || strings.Contains(got, "I will") {
		t.Fatalf("answer = %q, want no narration", got)
	}
}

// TestExtractFinalText_Synthetic covers the stream shapes that only occur
// in practice: a missing step_finish, a rewritten part, filtered parts, and
// CRLF framing.
func TestExtractFinalText_Synthetic(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "missing step_finish falls back to last text",
			in:   textEventLine("p1", "m1", "first") + textEventLine("p2", "m2", "second"),
			want: "second",
		},
		{
			name: "duplicate part id keeps the rewritten text",
			in:   textEventLine("p1", "m1", "first") + textEventLine("p1", "m1", "second"),
			want: "second",
		},
		{
			name: "synthetic and ignored parts are dropped",
			in: `{"type":"text","part":{"id":"p1","messageID":"m1","type":"text","text":"hidden","synthetic":true}}` + "\n" +
				`{"type":"text","part":{"id":"p2","messageID":"m1","type":"text","text":"hidden","ignored":true}}` + "\n" +
				textEventLine("p3", "m1", "kept") + stepFinishLine("m1"),
			want: "kept",
		},
		{
			name: "crlf framing is tolerated",
			in: strings.ReplaceAll(
				textEventLine("p1", "m1", "kept")+stepFinishLine("m1"), "\n", "\r\n"),
			want: "kept",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := extractFinalText([]byte(tc.in))
			if err != nil {
				t.Fatalf("extractFinalText error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("answer = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestExtractFinalText_Errors covers the two failures the extractor owns.
func TestExtractFinalText_Errors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr string
	}{
		{
			name:    "empty input",
			in:      "",
			wantErr: "opencode produced no JSON events",
		},
		{
			name:    "blank final text",
			in:      textEventLine("p1", "m1", "   ") + stepFinishLine("m1"),
			wantErr: "opencode produced no final assistant text",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := extractFinalText([]byte(tc.in))
			if err == nil {
				t.Fatal("extractFinalText = nil error, want a failure")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// textEventLine renders one assistant text event.
func textEventLine(id, messageID, text string) string {
	return `{"type":"text","part":{"id":"` + id + `","messageID":"` + messageID +
		`","type":"text","text":"` + text + `"}}` + "\n"
}

// stepFinishLine renders one step_finish event.
func stepFinishLine(messageID string) string {
	return `{"type":"step_finish","part":{"id":"finish","messageID":"` + messageID +
		`","type":"step-finish"}}` + "\n"
}

// TestParseEvents_MixedJunkCounts pins the tolerance contract: junk lines
// (plain text, JSON of the wrong shape, CRLF-framed noise) are skipped and
// counted without discarding the valid events around them.
func TestParseEvents_MixedJunkCounts(t *testing.T) {
	stream := "not json at all\n" +
		textEventLine("p1", "m1", "first") +
		"[1,2]\n" +
		"trailing junk\r\n" +
		textEventLine("p2", "m1", "second") +
		stepFinishLine("m1")

	events, skipped, err := parseEvents([]byte(stream))
	if err != nil {
		t.Fatalf("parseEvents error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("events = %d, want 3 (a junk line must not drop its neighbors)", len(events))
	}
	if skipped != 3 {
		t.Fatalf("skipped = %d, want 3", skipped)
	}

	answer, skipped, err := extractFinalText([]byte(stream))
	if err != nil {
		t.Fatalf("extractFinalText error: %v", err)
	}
	if answer != "first\nsecond" {
		t.Fatalf("answer = %q, want %q", answer, "first\nsecond")
	}
	if skipped != 3 {
		t.Fatalf("extractFinalText skipped = %d, want 3", skipped)
	}
}

// TestExtractAnswer_SkippedLinesCounted proves the skip count surfaces on
// the errNoFinalText error, so a mostly-corrupt stream cannot degrade
// silently.
func TestExtractAnswer_SkippedLinesCounted(t *testing.T) {
	stream := "garbage line\n" +
		textEventLine("p1", "m1", "   ") +
		stepFinishLine("m1")

	_, err := extractAnswer([]byte(stream), nil)
	if err == nil {
		t.Fatal("extractAnswer = nil error, want errNoFinalText")
	}
	if !errors.Is(err, errNoFinalText) {
		t.Fatalf("error = %v, want errNoFinalText (sentinel must survive wrapping)", err)
	}
	if !strings.Contains(err.Error(), "skipped 1") {
		t.Fatalf("error = %q, want it to contain the skip count", err)
	}
}
