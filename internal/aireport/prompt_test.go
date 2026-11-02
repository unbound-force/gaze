package aireport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// promptRequiredPhrases are the contract phrases the report-mode prompt
// must carry (contracts/agents-and-prompts.md §A).
var promptRequiredPhrases = []string{
	`"summary"`,
	`"recommendations"`,
	"1200",
	"400",
	"Do not use tools",
	"English",
}

func TestReportModePrompt_Content(t *testing.T) {
	prompt := ReportModePrompt()
	if strings.TrimSpace(prompt) == "" {
		t.Fatal("ReportModePrompt() returned an empty prompt")
	}
	if strings.HasPrefix(strings.TrimSpace(prompt), "---") {
		t.Error("prompt must not start with YAML frontmatter")
	}
	for _, phrase := range promptRequiredPhrases {
		if !strings.Contains(prompt, phrase) {
			t.Errorf("prompt is missing %q", phrase)
		}
	}
	if !strings.Contains(prompt, "1 to 5") {
		t.Error(`prompt is missing the "1 to 5" recommendation bound`)
	}
}

func TestReportModePrompt_ExampleAnswerParses(t *testing.T) {
	prompt := ReportModePrompt()
	example, ok := fencedExampleAnswer(prompt)
	if !ok {
		t.Fatal("prompt has no ```json fenced example answer")
	}
	if _, err := ParseAIAnswer(example); err != nil {
		t.Fatalf("prompt example answer does not pass ParseAIAnswer: %v", err)
	}
}

// fencedExampleAnswer returns the body of the prompt's last ```json block.
func fencedExampleAnswer(prompt string) (string, bool) {
	const fence = "```json"
	start := strings.LastIndex(prompt, fence)
	if start < 0 {
		return "", false
	}
	rest := prompt[start+len(fence):]
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "```")
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(rest[:end]), true
}

func TestReportModePrompt_Length(t *testing.T) {
	prompt := ReportModePrompt()
	lines := strings.Count(strings.TrimRight(prompt, "\n"), "\n") + 1
	if lines > 40 {
		t.Errorf("prompt is %d lines, want at most 40", lines)
	}
}

func TestReportModePrompt_NotScaffolded(t *testing.T) {
	const marker = "Do not use tools, read files, or run commands."
	root := filepath.Join("..", "scaffold", "assets")

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), marker) {
			t.Errorf("%s contains the report-mode prompt; it must stay embedded only", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
}
