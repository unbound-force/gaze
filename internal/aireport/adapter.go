package aireport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// maxAdapterOutputBytes caps AI adapter output (subprocess stdout or HTTP
// response body) at 64 MiB to prevent OOM on unexpectedly large responses.
const maxAdapterOutputBytes = 64 << 20 // 64 MiB

// maxAdapterStderrBytes caps stderr included in error messages at 512 bytes
// to avoid leaking secrets from AI CLI output into error strings. The cap is
// applied to already-scrubbed stderr — runSubprocess passes the captured
// buffer through redactEnvValues before truncating — so no prefix of a
// sensitive value can survive by straddling the truncation boundary.
const maxAdapterStderrBytes = 512

// AIAdapter formats an analysis payload using an external AI CLI or API.
// Implementations must be safe to call with a context that may be cancelled.
type AIAdapter interface {
	// Format invokes the AI integration with the given system prompt and
	// JSON payload (from payload io.Reader), returning the formatted
	// markdown report or an error.
	Format(ctx context.Context, systemPrompt string, payload io.Reader) (string, error)
}

// runSubprocess executes an external CLI binary as a subprocess with
// the given arguments and payload on stdin. It handles binary
// resolution (exec.LookPath), stdout capture with size limiting
// (maxAdapterOutputBytes), stderr capture with truncation
// (maxAdapterStderrBytes), and error formatting.
//
// Stderr is redacted before it is surfaced: immediately after
// cmd.Wait() returns, the complete stderr buffer is passed through
// redactEnvValues with os.Environ() as the environ snapshot, and both
// the non-zero-exit error embed and the success-path returned stderr
// bytes carry the scrubbed content — the single redaction choke point
// for subprocess stderr (design D1 of the
// redact-env-values-from-adapter-stderr change), so consumers such as
// formatStderrSuffix inherit redaction automatically. The scrub runs
// before the maxAdapterStderrBytes truncation on every path, so a
// sensitive value cannot survive by straddling the truncation
// boundary. Reading os.Environ() after cmd.Wait() equals the
// environment the child inherited, because gaze blocks inside
// cmd.Wait() for the child's entire lifetime and never mutates its
// own environment during adapter execution (design D4).
//
// If cmdDir is non-empty, cmd.Dir is set to that directory. This is
// used by the Gemini adapter which places GEMINI.md in a temp
// directory and sets the subprocess working directory there.
//
// On success (exit code 0), both stdout and stderr bytes are returned.
// Callers can use stderr for diagnostic messages when stdout is empty
// (e.g., authentication errors that the subprocess writes to stderr
// while exiting 0).
//
// On failure (non-zero exit), an error is returned with stderr
// embedded in the error message; stdout and stderr bytes are nil.
//
// This function is shared by the three subprocess-based AI adapters
// (Claude, Gemini, OpenCode) to eliminate duplicated pipe setup,
// output limiting, and error handling code.
func runSubprocess(
	ctx context.Context,
	binaryName string,
	args []string,
	cmdDir string,
	payload io.Reader,
) ([]byte, []byte, error) {
	binPath, err := exec.LookPath(binaryName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s not found on PATH: %w", binaryName, err)
	}

	cmd := exec.CommandContext(ctx, binPath, args...)
	cmd.Stdin = payload
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("creating stdout pipe for %s: %w", binaryName, err)
	}
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("starting %s: %w", binaryName, err)
	}

	outBytes, readErr := io.ReadAll(io.LimitReader(stdoutPipe, maxAdapterOutputBytes))
	waitErr := cmd.Wait()

	// The stderr buffer is complete once Wait returns, and os.Environ()
	// read here equals the environment the child inherited: gaze blocks
	// inside Wait for the child's entire lifetime and never mutates its
	// own environment (design D4). Scrub once, before truncation and
	// before any bytes are surfaced, so a sensitive value straddling the
	// maxAdapterStderrBytes cap cannot survive (design D1/D3).
	scrubbedStderr := redactEnvValues(stderrBuf.String(), os.Environ())

	if waitErr != nil {
		stderrSnippet := scrubbedStderr
		if len(stderrSnippet) > maxAdapterStderrBytes {
			stderrSnippet = stderrSnippet[:maxAdapterStderrBytes] + "... (truncated)"
		}
		return nil, nil, fmt.Errorf("%s exited with error: %w\nstderr: %s", binaryName, waitErr, stderrSnippet)
	}
	if readErr != nil {
		return nil, nil, fmt.Errorf("reading %s output: %w", binaryName, readErr)
	}

	return outBytes, []byte(scrubbedStderr), nil
}

// formatStderrSuffix returns a "\nstderr: <snippet>" suffix for error
// messages when stderrBytes contains non-whitespace content. Returns
// an empty string when stderr is empty or whitespace-only. The snippet
// is truncated at maxAdapterStderrBytes to limit the size of stderr
// included in error messages.
func formatStderrSuffix(stderrBytes []byte) string {
	snippet := strings.TrimSpace(string(stderrBytes))
	if snippet == "" {
		return ""
	}
	if len(snippet) > maxAdapterStderrBytes {
		snippet = snippet[:maxAdapterStderrBytes] + "... (truncated)"
	}
	return "\nstderr: " + snippet
}

// AdapterConfig holds the user-specified AI adapter configuration.
type AdapterConfig struct {
	// Name is the adapter identifier: "claude", "gemini", "ollama", or "opencode".
	Name string

	// Model is the model name to use. Required for ollama; optional for
	// claude, gemini, and opencode (uses each CLI's default when empty).
	Model string

	// Timeout is the maximum duration to wait for the AI adapter to respond.
	// Applied to the subprocess or HTTP request context.
	// Default: 10 minutes.
	Timeout time.Duration

	// OllamaHost overrides the default ollama server URL.
	// Reads from OLLAMA_HOST env var when empty.
	OllamaHost string
}

// validAdapters is the exact allowlist of supported AI adapter names.
var validAdapters = map[string]bool{
	"claude":   true,
	"gemini":   true,
	"ollama":   true,
	"opencode": true,
}

// NewAdapter creates an AIAdapter for the given config. It returns an error
// if cfg.Name is not in the allowlist {"claude", "gemini", "ollama", "opencode"}.
func NewAdapter(cfg AdapterConfig) (AIAdapter, error) {
	if !validAdapters[cfg.Name] {
		return nil, fmt.Errorf(
			"unknown AI adapter %q: must be one of \"claude\", \"gemini\", \"ollama\", or \"opencode\"",
			cfg.Name,
		)
	}
	switch cfg.Name {
	case "claude":
		return &ClaudeAdapter{config: cfg}, nil
	case "gemini":
		return &GeminiAdapter{config: cfg}, nil
	case "ollama":
		return &OllamaAdapter{config: cfg}, nil
	case "opencode":
		return &OpenCodeAdapter{config: cfg}, nil
	}
	// Unreachable — validAdapters check above covers all cases.
	panic("aireport: unreachable adapter case")
}

// AdapterValidator is an optional interface that adapters may implement to
// perform pre-flight validation before the analysis pipeline runs. CLI-based
// adapters use it to verify the binary is on PATH before the analysis pipeline
// runs (e.g. FR-012 for claude/gemini, FR-007 for opencode).
type AdapterValidator interface {
	ValidateBinary() error
}

// ValidateAdapterBinary calls adapter.ValidateBinary() if the adapter
// implements AdapterValidator. Returns nil for adapters that don't (e.g.
// OllamaAdapter, FakeAdapter).
func ValidateAdapterBinary(adapter AIAdapter) error {
	if v, ok := adapter.(AdapterValidator); ok {
		return v.ValidateBinary()
	}
	return nil
}
