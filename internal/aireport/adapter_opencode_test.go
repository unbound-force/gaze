package aireport

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildFakeOpenCode compiles the fake_opencode binary and returns its path.
// The binary is placed in t.TempDir() and cleaned up automatically.
func buildFakeOpenCode(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess tests not supported on Windows in CI")
	}

	dir := t.TempDir()
	bin := filepath.Join(dir, "opencode")

	srcDir := filepath.Join("internal", "aireport", "testdata", "fake_opencode")
	cmd := exec.Command("go", "build", "-o", bin, "./"+srcDir)
	cmd.Dir = findModuleRoot(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building fake_opencode: %v\n%s", err, out)
	}
	return bin
}

// withOpenCodeOnPath temporarily adds the directory containing the fake opencode
// binary to PATH, restoring the original value on cleanup.
func withOpenCodeOnPath(t *testing.T, bin string) {
	t.Helper()
	origPath := os.Getenv("PATH")
	dir := filepath.Dir(bin)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
}

func TestOpenCodeAdapter_SuccessfulInvocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	got, err := adapter.Format(context.Background(), "system instructions", strings.NewReader(`{"crap":{}}`))
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if _, err := ParseAIAnswer(got); err != nil {
		t.Errorf("expected a valid answer from the fake stream, got %q: %v", got, err)
	}
}

func TestOpenCodeAdapter_AgentFileWrittenToTempDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	// The fake_opencode binary verifies that .opencode/agents/gaze-headless.md
	// exists under --dir. A successful invocation proves the file was written.
	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(context.Background(), "# System Prompt\n\nInstructions.", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Format failed (agent file may not have been written): %v", err)
	}
}

func TestOpenCodeAdapter_FrontmatterWritten(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	// The fake_opencode binary reads the agent file and asserts it starts with
	// "---", declares a primary agent, and denies every tool. A successful
	// invocation proves the adapter wrote the tool-free frontmatter.
	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(context.Background(), "prompt without frontmatter", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Format failed (frontmatter may not have been written): %v", err)
	}
}

func TestOpenCodeAdapter_ModelFlagPassed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	capturePath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("FAKE_OPENCODE_CAPTURE_ARGS", capturePath)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode", Model: "claude-3-5-sonnet"}}
	if _, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`)); err != nil {
		t.Fatalf("Format: %v", err)
	}

	args, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("reading captured args: %v", err)
	}
	if !strings.Contains(string(args), "claude-3-5-sonnet") {
		t.Errorf("expected the model flag in argv, got: %q", string(args))
	}
}

func TestOpenCodeAdapter_NoModelFlag_Succeeds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	// When cfg.Model is empty, no -m flag is passed.
	capturePath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("FAKE_OPENCODE_CAPTURE_ARGS", capturePath)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	if _, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`)); err != nil {
		t.Fatalf("Format: %v", err)
	}

	args, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("reading captured args: %v", err)
	}
	if strings.Contains(string(args), "-m\n") {
		t.Errorf("unexpected model flag in argv: %q", string(args))
	}
}

func TestOpenCodeAdapter_NotOnPath_ReturnsError(t *testing.T) {
	// Override PATH with an empty temp directory so opencode is not found.
	// No testing.Short() guard needed — no subprocess is compiled or started.
	t.Setenv("PATH", t.TempDir())

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error when opencode not on PATH")
	}
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Errorf("expected 'not found on PATH' in error, got: %v", err)
	}
}

func TestOpenCodeAdapter_NonZeroExit_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)

	// Create a shell wrapper named "opencode" that injects --exit-error after
	// the "run" subcommand arg so that the fake binary's flag parser sees it.
	// The adapter passes args as: run --dir ... --agent gaze-headless --format json ""
	// so $1="run" and the rest are flags. We forward $1 (run) then inject
	// --exit-error before the remaining args.
	wrapDir := t.TempDir()
	opencodePath := filepath.Join(wrapDir, "opencode")
	errWrapper := "#!/bin/sh\nshift\nexec \"" + bin + "\" run --exit-error \"$@\"\n"
	if err := os.WriteFile(opencodePath, []byte(errWrapper), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", wrapDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected error on non-zero exit")
	}
}

func TestOpenCodeAdapter_EmptyOutput_ReturnsError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)

	// Create a shell wrapper that injects --empty-output after the "run"
	// subcommand arg so that the fake binary's flag parser sees it correctly.
	wrapDir := t.TempDir()
	opencodePath := filepath.Join(wrapDir, "opencode")
	emptyWrapper := "#!/bin/sh\nshift\nexec \"" + bin + "\" run --empty-output \"$@\"\n"
	if err := os.WriteFile(opencodePath, []byte(emptyWrapper), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", wrapDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))
	if err == nil {
		t.Fatal("expected an error for an empty event stream")
	}
	if !strings.Contains(err.Error(), "no JSON events") {
		t.Errorf("expected a no-JSON-events error, got: %v", err)
	}
}

func TestOpenCodeAdapter_TempDirCleanedUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	tmpBase := os.TempDir()
	beforeEntries, _ := os.ReadDir(tmpBase)
	before := countGazeOpenCodeDirs(beforeEntries)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, _ = adapter.Format(context.Background(), "prompt", strings.NewReader(`{}`))

	afterEntries, _ := os.ReadDir(tmpBase)
	after := countGazeOpenCodeDirs(afterEntries)

	if after > before {
		t.Errorf("temp directories leaked: before=%d after=%d", before, after)
	}
}

func TestOpenCodeAdapter_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	// Cancel the context before calling Format so the subprocess is never
	// started or is immediately killed — no time.Sleep needed.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	_, err := adapter.Format(ctx, "prompt", strings.NewReader(`{}`))
	// Either context.Canceled or a subprocess kill error is acceptable.
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

// TestOpenCodeAdapter_ValidateBinary_NotOnPath verifies that ValidateBinary
// returns an error containing "FR-007" when opencode is not on PATH.
func TestOpenCodeAdapter_ValidateBinary_NotOnPath(t *testing.T) {
	// Isolate PATH so opencode is definitely not found.
	t.Setenv("PATH", t.TempDir())

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	err := adapter.ValidateBinary()
	if err == nil {
		t.Fatal("expected error when opencode not on PATH")
	}
	if !strings.Contains(err.Error(), "FR-007") {
		t.Errorf("expected FR-007 in ValidateBinary error, got: %v", err)
	}
}

// TestOpenCodeAdapter_ValidateBinary_OnPath verifies that ValidateBinary
// returns nil when opencode is on PATH.
func TestOpenCodeAdapter_ValidateBinary_OnPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}
	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	if err := adapter.ValidateBinary(); err != nil {
		t.Errorf("expected nil from ValidateBinary when opencode on PATH, got: %v", err)
	}
}

func countGazeOpenCodeDirs(entries []os.DirEntry) int {
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "gaze-opencode-") {
			count++
		}
	}
	return count
}

// replayOpenCode runs Format against the fake binary replaying a stream
// fixture, and returns the extracted answer.
func replayOpenCode(t *testing.T, stream string) (string, error) {
	t.Helper()

	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)
	t.Setenv("FAKE_OPENCODE_STREAM", filepath.Join("testdata", "opencode_streams", stream))

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}

	return adapter.Format(context.Background(), "system prompt", strings.NewReader(`{"crap":{}}`))
}

// TestOpenCodeAdapter_Replay_NarrationThenAnswer verifies that narration from
// earlier steps never reaches the caller.
func TestOpenCodeAdapter_Replay_NarrationThenAnswer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	got, err := replayOpenCode(t, "narration_then_answer.jsonl")
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if _, err := ParseAIAnswer(got); err != nil {
		t.Errorf("expected the final answer, got %q: %v", got, err)
	}
	if strings.Contains(got, "Let me") || strings.Contains(got, "I will") {
		t.Errorf("narration leaked into the answer: %q", got)
	}
}

// TestOpenCodeAdapter_Replay_AnswerOnly verifies the single-step stream.
func TestOpenCodeAdapter_Replay_AnswerOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	got, err := replayOpenCode(t, "answer_only.jsonl")
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if _, err := ParseAIAnswer(got); err != nil {
		t.Errorf("expected a valid answer, got %q: %v", got, err)
	}
}

// TestOpenCodeAdapter_Replay_ToolRefused verifies that a refused tool call
// does not fail the run.
func TestOpenCodeAdapter_Replay_ToolRefused(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	got, err := replayOpenCode(t, "tool_refused.jsonl")
	if err != nil {
		t.Fatalf("Format: %v", err)
	}
	if _, err := ParseAIAnswer(got); err != nil {
		t.Errorf("expected a valid answer, got %q: %v", got, err)
	}
}

// TestOpenCodeAdapter_Replay_ToolCompletedFailsClosed verifies that a tool
// which actually ran fails the run.
func TestOpenCodeAdapter_Replay_ToolCompletedFailsClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	_, err := replayOpenCode(t, "tool_completed.jsonl")
	if err == nil {
		t.Fatal("expected an error when a tool ran")
	}
	if !strings.Contains(err.Error(), "denies all tools") {
		t.Errorf("expected a denied-tools error, got: %v", err)
	}
}

// TestOpenCodeAdapter_Replay_ErrorEventFails verifies that a provider error
// surfaces with its message.
func TestOpenCodeAdapter_Replay_ErrorEventFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	_, err := replayOpenCode(t, "error_event.jsonl")
	if err == nil {
		t.Fatal("expected an error from the error event")
	}
	if !strings.Contains(err.Error(), "Incorrect API key") {
		t.Errorf("expected the provider message, got: %v", err)
	}
}

// TestOpenCodeAdapter_Replay_NotJSONFails verifies that a non-event stream
// fails closed instead of returning raw stdout.
func TestOpenCodeAdapter_Replay_NotJSONFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	_, err := replayOpenCode(t, "not_json.txt")
	if err == nil {
		t.Fatal("expected an error for a non-JSON stream")
	}
	if !strings.Contains(err.Error(), "no JSON events") {
		t.Errorf("expected a no-JSON-events error, got: %v", err)
	}
}

// TestOpenCodeAdapter_AgentFileIsToolFree verifies the exact agent file the
// adapter runs: tool-free frontmatter plus the frontmatter-free system prompt.
func TestOpenCodeAdapter_AgentFileIsToolFree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in -short mode")
	}

	bin := buildFakeOpenCode(t)
	withOpenCodeOnPath(t, bin)

	capturePath := filepath.Join(t.TempDir(), "agent.md")
	t.Setenv("FAKE_OPENCODE_CAPTURE_AGENT", capturePath)

	adapter := &OpenCodeAdapter{config: AdapterConfig{Name: "opencode"}}
	if _, err := adapter.Format(context.Background(), "JUDGMENT ONLY", strings.NewReader(`{}`)); err != nil {
		t.Fatalf("Format: %v", err)
	}

	data, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("reading captured agent file: %v", err)
	}

	content := string(data)
	for _, want := range []string{
		"description: Gaze headless judgment call (no tools)",
		"mode: primary",
		"\"*\": deny",
		"JUDGMENT ONLY",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("agent file missing %q:\n%s", want, content)
		}
	}
	if strings.Contains(content, "gaze-reporter") {
		t.Errorf("agent file must not reference the interactive reporter agent:\n%s", content)
	}

	info, err := os.Stat(capturePath)
	if err != nil {
		t.Fatalf("stat captured agent file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("agent file mode is %o, want 600", info.Mode().Perm())
	}
}
