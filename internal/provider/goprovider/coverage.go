package goprovider

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unbound-force/gaze/v2/internal/crap"
)

// GoLineCoverageProvider implements crap.LineCoverageProvider by
// wrapping Go's coverage tooling: go test -coverprofile for
// generation and crap.ParseCoverProfile for parsing.
type GoLineCoverageProvider struct {
	// Stderr receives warnings about partial coverage recovery
	// and file parsing issues. If nil, warnings are suppressed.
	Stderr io.Writer
	// Short passes -short to the internal go test invocation when true.
	Short bool
}

// NewLineCoverageProvider creates a new GoLineCoverageProvider with
// the given stderr writer for diagnostic output.
func NewLineCoverageProvider(stderr io.Writer) *GoLineCoverageProvider {
	return &GoLineCoverageProvider{Stderr: stderr}
}

// Coverage returns per-function line coverage for the packages
// matched by patterns, rooted at rootDir. When coverProfile is
// non-empty, the provider parses the pre-generated profile directly
// instead of running go test. When coverProfile is empty, the
// provider generates a temporary coverage profile via go test.
func (p *GoLineCoverageProvider) Coverage(patterns []string, rootDir string, coverProfile string) ([]crap.FuncCoverage, error) {
	profilePath := coverProfile
	if profilePath == "" {
		var err error
		profilePath, err = generateCoverProfile(rootDir, patterns, p.Short, p.Stderr)
		if err != nil {
			return nil, fmt.Errorf("generating coverage: %w", err)
		}
		defer func() { _ = os.Remove(profilePath) }()
	} else {
		// Validate user-supplied cover profile path.
		profilePath = filepath.Clean(profilePath)
		info, err := os.Stat(profilePath)
		if err != nil {
			return nil, fmt.Errorf("cover profile %q: %w", profilePath, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("cover profile %q is a directory, not a file", profilePath)
		}
	}

	return crap.ParseCoverProfile(profilePath, rootDir, p.Stderr)
}

// generateCoverProfile runs go test to produce a coverage profile.
// The profile is written to a temporary file to avoid clobbering
// any existing cover.out in the user's working directory.
//
// When go test exits non-zero but wrote a usable coverage profile
// (non-empty file), the profile is preserved and a warning is
// emitted to stderr. This supports partial coverage from runs where
// some packages fail but others produce valid coverage data.
// See design decision D1 in ci-gate-integrity.
func generateCoverProfile(moduleDir string, patterns []string, short bool, stderr io.Writer) (string, error) {
	profilePath, err := createTempProfile()
	if err != nil {
		return "", err
	}
	return runGoTestCoverage(profilePath, moduleDir, patterns, short, stderr)
}

// runGoTestCoverage executes go test -coverprofile and returns the
// profile path. When go test exits non-zero but wrote a usable
// coverage profile, the partial profile is preserved with a warning.
//
// The short parameter controls whether -short is passed to go test.
// When true, heavyweight tests guarded by testing.Short() are skipped.
// This is opt-in via the --test-short CLI flag — callers that want
// full coverage (including long-running tests) leave it false.
//
// Recursive subprocess prevention is handled by the GAZE_COVERAGE_RUN=1
// environment variable set in buildGoTestCmd, not by -short. Tests
// that detect this env var can skip themselves to prevent infinite
// subprocess chains when gaze analyzes its own test suite.
func runGoTestCoverage(profilePath, moduleDir string, patterns []string, short bool, stderr io.Writer) (string, error) {
	cmd := buildGoTestCmd(profilePath, moduleDir, patterns, short)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if profile was written despite non-zero exit.
		// go test -coverprofile writes coverage data per-package
		// as each completes, so partial profiles are usable even
		// when later packages fail.
		return recoverOrFail(profilePath, err, output, stderr)
	}

	return profilePath, nil
}

// buildGoTestCmd constructs the exec.Cmd for running go test with
// coverage profiling. The short parameter controls whether -short
// is included in the args. The command's environment includes
// GAZE_COVERAGE_RUN=1 (deduplicated) to prevent recursive subprocess
// chains when gaze analyzes its own test suite.
func buildGoTestCmd(profilePath, moduleDir string, patterns []string, short bool) *exec.Cmd {
	args := []string{"test"}
	if short {
		args = append(args, "-short")
	}
	args = append(args, "-coverprofile="+profilePath)
	args = append(args, patterns...)

	cmd := exec.Command("go", args...)
	cmd.Dir = moduleDir

	// Set GAZE_COVERAGE_RUN=1 to signal child processes that they
	// are running inside a gaze coverage collection. Filter any
	// existing entry to avoid duplicates.
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GAZE_COVERAGE_RUN=") {
			env = append(env, e)
		}
	}
	env = append(env, "GAZE_COVERAGE_RUN=1")
	cmd.Env = env

	return cmd
}

// recoverOrFail attempts to recover a partial coverage profile after
// a go test failure. Returns the profile path if recovery succeeds,
// or a descriptive error if the profile is missing or empty.
func recoverOrFail(profilePath string, testErr error, output []byte, stderr io.Writer) (string, error) {
	recovered, recoverErr := recoverPartialProfile(profilePath, testErr, stderr)
	if recoverErr != nil {
		return "", fmt.Errorf("go test failed and produced no coverage: %s\n%s", testErr, string(output))
	}
	return recovered, nil
}

// createTempProfile creates a temporary file for the coverage profile.
func createTempProfile() (string, error) {
	tmpFile, err := os.CreateTemp("", "gaze-cover-*.out")
	if err != nil {
		return "", fmt.Errorf("creating temp cover profile: %w", err)
	}
	path := tmpFile.Name()
	_ = tmpFile.Close()
	return path, nil
}

// recoverPartialProfile checks whether a coverage profile exists
// and has non-zero size after a go test failure. If the profile is
// usable, it emits a warning to stderr and returns the profile path.
// If the profile is missing or empty, it cleans up and returns an
// error. The stderr writer may be nil, in which case the warning
// is suppressed.
func recoverPartialProfile(profilePath string, testErr error, stderr io.Writer) (string, error) {
	info, statErr := os.Stat(profilePath)
	if statErr != nil || info.Size() == 0 {
		_ = os.Remove(profilePath)
		return "", fmt.Errorf("profile missing or empty after test failure")
	}
	// Profile exists with data — warn and continue.
	if stderr != nil {
		_, _ = fmt.Fprintf(stderr, "warning: go test exited with error (partial coverage used): %s\n", testErr)
	}
	return profilePath, nil
}
