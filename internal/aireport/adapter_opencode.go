package aireport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// headlessAgentName is the opencode agent the adapter installs in its temp
// directory. The name is deliberately distinct from gaze-reporter so opencode
// never falls back to a user agent file when the headless one is present.
const headlessAgentName = "gaze-headless"

// headlessAgentFrontmatter denies every tool to the headless agent. opencode
// silently falls back to its default agent when an agent file is missing or
// declares a subagent, so the agent is declared primary and every permission
// token is denied.
const headlessAgentFrontmatter = "---\n" +
	"description: Gaze headless judgment call (no tools)\n" +
	"mode: primary\n" +
	"permission:\n" +
	"  \"*\": deny\n" +
	"---\n"

// OpenCodeAdapter invokes the opencode CLI to produce the report's judgment
// portion.
//
// The system prompt is written as .opencode/agents/gaze-headless.md in a
// temporary directory passed via --dir, because opencode reads an agent's
// system prompt from that location. The agent denies all tools, so the model
// cannot explore the repository instead of answering.
//
// The adapter runs opencode with --format json and extracts only the final
// assistant text from the event stream. Narration and tool output never reach
// the caller, and every extracted answer still passes the runner's validator.
// The temporary directory is removed after the subprocess exits.
type OpenCodeAdapter struct {
	config AdapterConfig
}

// Compile-time check that OpenCodeAdapter implements AdapterValidator.
var _ AdapterValidator = &OpenCodeAdapter{}

// ValidateBinary implements AdapterValidator. It checks that the opencode
// binary is available on PATH (FR-007). Call this before running the analysis
// pipeline to give users an immediate error rather than failing after minutes
// of work.
func (a *OpenCodeAdapter) ValidateBinary() error {
	if _, err := exec.LookPath("opencode"); err != nil {
		return fmt.Errorf("opencode not found on PATH (FR-007): %w", err)
	}
	return nil
}

// Format implements AIAdapter. It creates a temp directory containing
// .opencode/agents/gaze-headless.md, invokes opencode run with the payload on
// stdin, and returns the final assistant text of the event stream.
//
// Arguments are passed as distinct Go strings — no shell interpolation.
// The temp directory is removed in a deferred cleanup after the subprocess
// exits. Returns an error when:
//   - opencode is not found on PATH (detected by ValidateBinary; also checked here as defense).
//   - The temp directory or agent file cannot be created.
//   - The subprocess exits non-zero.
//   - The stream reports an error, a tool ran, or carries no final text.
func (a *OpenCodeAdapter) Format(ctx context.Context, systemPrompt string, payload io.Reader) (string, error) {
	tmpDir, err := os.MkdirTemp("", "gaze-opencode-*")
	if err != nil {
		return "", fmt.Errorf("creating temp dir for opencode agent: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := writeHeadlessAgent(tmpDir, systemPrompt); err != nil {
		return "", err
	}

	outBytes, stderrBytes, err := runSubprocess(ctx, "opencode", openCodeArgs(tmpDir, a.config.Model), "", payload)
	if err != nil {
		return "", err
	}

	return extractAnswer(outBytes, stderrBytes)
}

// writeHeadlessAgent writes the tool-free agent file the adapter runs.
func writeHeadlessAgent(dir, systemPrompt string) error {
	agentsDir := filepath.Join(dir, ".opencode", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		return fmt.Errorf("creating .opencode/agents dir: %w", err)
	}

	agentFile := filepath.Join(agentsDir, headlessAgentName+".md")
	content := headlessAgentFrontmatter + systemPrompt
	if err := os.WriteFile(agentFile, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing agent file: %w", err)
	}

	return nil
}

// openCodeArgs builds the argv for a headless opencode run.
func openCodeArgs(tmpDir, model string) []string {
	args := []string{"run", "--dir", tmpDir, "--agent", headlessAgentName, "--format", "json"}
	if model != "" {
		args = append(args, "-m", model)
	}

	// The trailing empty string keeps opencode in headless (non-interactive) mode.
	return append(args, "")
}

// extractAnswer reduces the opencode event stream to the final assistant text,
// attaching the stderr snippet when the stream carried no final text so the
// user can see why.
func extractAnswer(stdout, stderr []byte) (string, error) {
	answer, err := extractFinalText(stdout)
	if err != nil {
		if errors.Is(errNoFinalText, err) {
			return "", fmt.Errorf("%w%s", err, formatStderrSuffix(stderr))
		}
		return "", err
	}

	return answer, nil
}
