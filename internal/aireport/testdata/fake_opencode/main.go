// Command fake_opencode is a test double for the opencode CLI used in
// OpenCodeAdapter integration tests.
//
// It accepts the subset of opencode's arguments the adapter passes:
//   - run (subcommand; must come first)
//   - --dir <path> (directory holding .opencode/agents/gaze-headless.md)
//   - --agent <name> (must equal gaze-headless)
//   - --format <value> (must equal json)
//   - -m / --model <name> (optional)
//   - --exit-error (exits non-zero to simulate subprocess failure)
//   - --empty-output (writes no events, simulating an unusable stream)
//
// Modes:
//   - Replay (FAKE_OPENCODE_STREAM=<file>): verifies the invocation, drains
//     stdin, and copies the named stream file to stdout verbatim.
//   - Capture (FAKE_OPENCODE_CAPTURE_AGENT=<path>): copies the agent file the
//     adapter wrote to the named path so a test can inspect it.
//   - Capture (FAKE_OPENCODE_CAPTURE_ARGS=<path>): writes the argv it received,
//     one argument per line, so a test can assert flag wiring.
//   - Default: writes a minimal valid event stream whose final text is a valid
//     answer, so callers exercise the extraction path end to end.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// defaultAnswer is the final assistant text emitted when no replay stream is
// configured. It is a valid answer for the report-mode validator.
const defaultAnswer = `{"summary":"The project carries a CRAPload of 7 across 12 functions, with 4 in the Q4 Dangerous quadrant.","recommendations":["Add tests for ` + "`processQueue`" + ` — it has 0% line coverage and complexity 8."]}`

// defaultStream is a minimal opencode 1.19.0 event stream around defaultAnswer.
func defaultStream() string {
	events := []map[string]any{
		{
			"type":      "step_start",
			"sessionID": "ses_fake",
			"part": map[string]any{
				"id":        "prt_f001",
				"messageID": "msg_f001",
				"type":      "step-start",
			},
		},
		{
			"type":      "text",
			"sessionID": "ses_fake",
			"part": map[string]any{
				"id":        "prt_f002",
				"messageID": "msg_f001",
				"type":      "text",
				"text":      defaultAnswer,
			},
		},
		{
			"type":      "step_finish",
			"sessionID": "ses_fake",
			"part": map[string]any{
				"id":        "prt_f003",
				"messageID": "msg_f001",
				"type":      "step-finish",
				"reason":    "stop",
			},
		},
	}

	var b strings.Builder
	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			continue
		}
		b.Write(data)
		b.WriteByte('\n')
	}

	return b.String()
}

func main() {
	args := os.Args[1:]

	if len(args) == 0 || args[0] != "run" {
		fail("expected 'run' as first argument")
	}
	args = args[1:]

	fs := flag.NewFlagSet("opencode", flag.ContinueOnError)
	var (
		dir       = fs.String("dir", "", "directory containing .opencode/agents/")
		agent     = fs.String("agent", "", "agent name")
		format    = fs.String("format", "", "output format")
		_         = fs.String("model", "", "model name")
		_         = fs.String("m", "", "model name (short)")
		exitError = fs.Bool("exit-error", false, "exit non-zero")
		emptyOut  = fs.Bool("empty-output", false, "write no events")
	)
	if err := fs.Parse(args); err != nil {
		fail(fmt.Sprintf("flag parse error: %v", err))
	}

	if *exitError {
		fail("simulated error")
	}

	agentFile, err := verifyInvocation(*dir, *agent, *format)
	if err != nil {
		fail(err.Error())
	}

	if capture := os.Getenv("FAKE_OPENCODE_CAPTURE_ARGS"); capture != "" {
		if err := os.WriteFile(capture, []byte(strings.Join(os.Args[1:], "\n")), 0o600); err != nil {
			fail(fmt.Sprintf("capturing args: %v", err))
		}
	}

	if capture := os.Getenv("FAKE_OPENCODE_CAPTURE_AGENT"); capture != "" {
		if err := copyFile(agentFile, capture); err != nil {
			fail(fmt.Sprintf("capturing agent file: %v", err))
		}
	}

	// Drain the payload so the parent never blocks on a broken pipe.
	_, _ = io.ReadAll(os.Stdin)

	if *emptyOut {
		return
	}

	if stream := os.Getenv("FAKE_OPENCODE_STREAM"); stream != "" {
		data, readErr := os.ReadFile(stream)
		if readErr != nil {
			fail(fmt.Sprintf("reading replay stream %s: %v", stream, readErr))
		}
		_, _ = os.Stdout.Write(data)
		return
	}

	_, _ = io.WriteString(os.Stdout, defaultStream())
}

// verifyInvocation checks the adapter's argv and agent file, returning the
// agent file path.
func verifyInvocation(dir, agent, format string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("--dir is required")
	}
	if agent != "gaze-headless" {
		return "", fmt.Errorf("expected --agent gaze-headless, got %q", agent)
	}
	if format != "json" {
		return "", fmt.Errorf("expected --format json, got %q", format)
	}

	agentFile := filepath.Join(dir, ".opencode", "agents", "gaze-headless.md")
	content, err := os.ReadFile(agentFile)
	if err != nil {
		return "", fmt.Errorf("agent file not found at %s: %v", agentFile, err)
	}

	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return "", fmt.Errorf("agent file missing frontmatter")
	}
	if !strings.Contains(text, "mode: primary") {
		return "", fmt.Errorf("agent file missing 'mode: primary'")
	}
	if !strings.Contains(text, `"*": deny`) {
		return "", fmt.Errorf("agent file missing a deny-all permission")
	}

	info, statErr := os.Stat(agentFile)
	if statErr == nil && info.Mode().Perm() != 0o600 {
		return "", fmt.Errorf("agent file mode is %o, want 600", info.Mode().Perm())
	}

	return agentFile, nil
}

// copyFile copies src to dst with 0600 permissions.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0o600)
}

// fail reports a fake-binary failure and exits with status 2 so tests can
// distinguish invocation errors from simulated CLI errors.
func fail(msg string) {
	fmt.Fprintf(os.Stderr, "fake_opencode: %s\n", msg)
	os.Exit(2)
}
