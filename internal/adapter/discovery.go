package adapter

import (
	"fmt"
	"os/exec"
	"regexp"

	"github.com/unbound-force/gaze/v2/internal/config"
)

// validLanguage matches language identifiers: lowercase letters,
// digits, and hyphens only (e.g., "python", "type-script", "go").
var validLanguage = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Discover resolves the external analyzer binary and arguments using
// a three-tier discovery mechanism (design decision D5):
//
//  1. CLI flag: --analyzer <name> overrides everything.
//  2. Config: .gaze.yaml analyzers.<language>.command.
//  3. PATH convention: gaze-analyzer-<language>.
//
// Returns the binary name/path and args to pass to NewSession. If no
// analyzer is found at any tier, returns empty strings and nil args
// (no error) — the caller should fall back to Go providers.
//
// The language parameter is required for tiers 2 and 3. If
// analyzerFlag is non-empty, language is not needed (tier 1 wins).
//
// Trust model: the discovered binary is spawned as a subprocess with
// access to the project directory. This is the same trust model as
// tools specified in Makefile, .goreleaser.yaml, or go:generate
// directives — the user explicitly configures which binary to run
// via CLI flag or .gaze.yaml, and the binary runs with the user's
// permissions. No sandboxing is applied.
func Discover(analyzerFlag, language string, cfg *config.GazeConfig) (binary string, args []string, err error) {
	// Tier 1: CLI flag overrides everything.
	if analyzerFlag != "" {
		return analyzerFlag, []string{"--stdio"}, nil
	}

	// Validate language before using it in binary name construction.
	if language != "" && !validLanguage.MatchString(language) {
		return "", nil, fmt.Errorf("invalid language identifier %q: must match [a-z0-9-]", language)
	}

	// Tier 2: Config-based lookup.
	if language != "" && cfg != nil && cfg.Analyzers != nil {
		if entry, ok := cfg.Analyzers[language]; ok && entry.Command != "" {
			return entry.Command, entry.Args, nil
		}
	}

	// Tier 3: PATH convention — gaze-analyzer-<language>.
	if language != "" {
		conventionName := "gaze-analyzer-" + language
		if _, lookErr := exec.LookPath(conventionName); lookErr == nil {
			return conventionName, []string{"--stdio"}, nil
		}
	}

	// No analyzer found — caller should use Go providers.
	return "", nil, nil
}
