package adapter_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/adapter"
	"github.com/unbound-force/gaze/v2/internal/config"
)

// TestDiscover_CLIFlagOverridesConfig verifies that --analyzer flag
// takes precedence over config and PATH convention.
func TestDiscover_CLIFlagOverridesConfig(t *testing.T) {
	cfg := &config.GazeConfig{
		Analyzers: config.AnalyzersConfig{
			"python": config.AnalyzerEntry{
				Command: "snake-eyes",
				Args:    []string{"--stdio"},
			},
		},
	}

	binary, args, err := adapter.Discover("my-analyzer", "python", cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "my-analyzer" {
		t.Errorf("binary = %q, want %q", binary, "my-analyzer")
	}
	if len(args) != 1 || args[0] != "--stdio" {
		t.Errorf("args = %v, want [--stdio]", args)
	}
}

// TestDiscover_ConfigLookup verifies that config-based discovery
// works when no CLI flag is set.
func TestDiscover_ConfigLookup(t *testing.T) {
	cfg := &config.GazeConfig{
		Analyzers: config.AnalyzersConfig{
			"python": config.AnalyzerEntry{
				Command: "snake-eyes",
				Args:    []string{"--stdio", "--verbose"},
			},
		},
	}

	binary, args, err := adapter.Discover("", "python", cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "snake-eyes" {
		t.Errorf("binary = %q, want %q", binary, "snake-eyes")
	}
	if len(args) != 2 || args[0] != "--stdio" || args[1] != "--verbose" {
		t.Errorf("args = %v, want [--stdio --verbose]", args)
	}
}

// TestDiscover_PATHFallback verifies that the PATH convention
// (gaze-analyzer-<language>) is used when no flag or config exists.
func TestDiscover_PATHFallback(t *testing.T) {
	// Create a temporary directory with a fake binary.
	tmpDir := t.TempDir()

	fakeBin := filepath.Join(tmpDir, "gaze-analyzer-python")
	if runtime.GOOS == "windows" {
		fakeBin += ".exe"
	}
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing fake binary: %v", err)
	}

	// Add tmpDir to PATH.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

	binary, args, err := adapter.Discover("", "python", nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "gaze-analyzer-python" {
		t.Errorf("binary = %q, want %q", binary, "gaze-analyzer-python")
	}
	if len(args) != 1 || args[0] != "--stdio" {
		t.Errorf("args = %v, want [--stdio]", args)
	}
}

// TestDiscover_NoAnalyzerFound verifies that when no analyzer is
// found at any tier, empty values are returned (Go default behavior).
func TestDiscover_NoAnalyzerFound(t *testing.T) {
	binary, args, err := adapter.Discover("", "python", nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "" {
		t.Errorf("binary = %q, want empty", binary)
	}
	if args != nil {
		t.Errorf("args = %v, want nil", args)
	}
}

// TestDiscover_NoLanguage verifies that when no language is provided
// and no CLI flag is set, discovery returns empty (skips tiers 2+3).
func TestDiscover_NoLanguage(t *testing.T) {
	cfg := &config.GazeConfig{
		Analyzers: config.AnalyzersConfig{
			"python": config.AnalyzerEntry{
				Command: "snake-eyes",
				Args:    []string{"--stdio"},
			},
		},
	}

	binary, args, err := adapter.Discover("", "", cfg)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "" {
		t.Errorf("binary = %q, want empty", binary)
	}
	if args != nil {
		t.Errorf("args = %v, want nil", args)
	}
}

// TestDiscover_CLIFlagWithoutLanguage verifies that CLI flag works
// even without a language parameter.
func TestDiscover_CLIFlagWithoutLanguage(t *testing.T) {
	binary, args, err := adapter.Discover("custom-analyzer", "", nil)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if binary != "custom-analyzer" {
		t.Errorf("binary = %q, want %q", binary, "custom-analyzer")
	}
	if len(args) != 1 || args[0] != "--stdio" {
		t.Errorf("args = %v, want [--stdio]", args)
	}
}
