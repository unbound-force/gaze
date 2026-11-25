package goprovider

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- recoverPartialProfile Tests ---

func TestRecoverPartialProfile_ExistsWithData(t *testing.T) {
	// go test exits non-zero but profile exists with data.
	// Verify recoverPartialProfile returns the profile path and
	// emits a warning to stderr.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	err := os.WriteFile(profilePath, []byte("mode: set\nsome/pkg/file.go:1.1,5.1 1 1\n"), 0644)
	if err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	var stderr strings.Builder
	testErr := fmt.Errorf("exit status 1")

	got, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr != nil {
		t.Fatalf("expected nil error, got: %v", recoverErr)
	}
	if got != profilePath {
		t.Errorf("expected profile path %q, got %q", profilePath, got)
	}

	// Verify warning was emitted.
	warning := stderr.String()
	if !strings.Contains(warning, "warning:") {
		t.Errorf("expected warning in stderr, got %q", warning)
	}
	if !strings.Contains(warning, "partial coverage used") {
		t.Errorf("expected 'partial coverage used' in warning, got %q", warning)
	}
	if !strings.Contains(warning, "exit status 1") {
		t.Errorf("expected test error in warning, got %q", warning)
	}

	// Verify profile was NOT deleted.
	if _, statErr := os.Stat(profilePath); statErr != nil {
		t.Errorf("profile should still exist after recovery, got: %v", statErr)
	}
}

func TestRecoverPartialProfile_Missing(t *testing.T) {
	// go test exits non-zero and profile is missing.
	// Verify hard error returned.
	profilePath := filepath.Join(t.TempDir(), "nonexistent.out")
	testErr := fmt.Errorf("exit status 1")

	var stderr strings.Builder
	_, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr == nil {
		t.Fatal("expected error for missing profile, got nil")
	}

	// Verify no warning was emitted (error path, not warning path).
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output for missing profile, got %q", stderr.String())
	}
}

func TestRecoverPartialProfile_Empty(t *testing.T) {
	// go test exits non-zero and profile is empty (0 bytes).
	// Verify hard error returned and file is cleaned up.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "empty.out")
	err := os.WriteFile(profilePath, []byte{}, 0644)
	if err != nil {
		t.Fatalf("writing empty profile: %v", err)
	}

	var stderr strings.Builder
	testErr := fmt.Errorf("exit status 2")

	_, recoverErr := recoverPartialProfile(profilePath, testErr, &stderr)
	if recoverErr == nil {
		t.Fatal("expected error for empty profile, got nil")
	}

	// Verify file was cleaned up.
	if _, statErr := os.Stat(profilePath); !os.IsNotExist(statErr) {
		t.Errorf("expected empty profile to be removed, but it still exists")
	}

	// Verify no warning was emitted (error path, not warning path).
	if stderr.Len() != 0 {
		t.Errorf("expected no stderr output for empty profile, got %q", stderr.String())
	}
}

func TestRecoverPartialProfile_NilStderr(t *testing.T) {
	// Verify that nil stderr writer doesn't panic — warning is
	// suppressed when no writer is provided.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	err := os.WriteFile(profilePath, []byte("mode: set\n"), 0644)
	if err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	testErr := fmt.Errorf("exit status 1")

	got, recoverErr := recoverPartialProfile(profilePath, testErr, nil)
	if recoverErr != nil {
		t.Fatalf("expected nil error with nil stderr, got: %v", recoverErr)
	}
	if got != profilePath {
		t.Errorf("expected profile path %q, got %q", profilePath, got)
	}
}

// --- buildGoTestCmd Tests ---

func TestBuildGoTestCmd_ShortTrue(t *testing.T) {
	// Verify that buildGoTestCmd includes -short in args when
	// short=true.
	cmd := buildGoTestCmd("/tmp/cover.out", "/some/dir", []string{"./..."}, true)

	found := false
	for _, arg := range cmd.Args {
		if arg == "-short" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected -short in cmd.Args when short=true, got %v", cmd.Args)
	}
}

func TestBuildGoTestCmd_ShortFalse(t *testing.T) {
	// Verify that buildGoTestCmd omits -short from args when
	// short=false.
	cmd := buildGoTestCmd("/tmp/cover.out", "/some/dir", []string{"./..."}, false)

	for _, arg := range cmd.Args {
		if arg == "-short" {
			t.Errorf("expected no -short in cmd.Args when short=false, got %v", cmd.Args)
			break
		}
	}
}

func TestBuildGoTestCmd_EnvVar(t *testing.T) {
	// Verify that buildGoTestCmd sets GAZE_COVERAGE_RUN=1 in the
	// command's environment.
	cmd := buildGoTestCmd("/tmp/cover.out", "/some/dir", []string{"./..."}, false)

	found := false
	for _, e := range cmd.Env {
		if e == "GAZE_COVERAGE_RUN=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GAZE_COVERAGE_RUN=1 in cmd.Env")
	}
}

func TestBuildGoTestCmd_EnvVarDeduplication(t *testing.T) {
	// Verify that when GAZE_COVERAGE_RUN is already in the
	// environment, buildGoTestCmd produces exactly one entry.
	t.Setenv("GAZE_COVERAGE_RUN", "stale")

	cmd := buildGoTestCmd("/tmp/cover.out", "/some/dir", []string{"./..."}, false)

	count := 0
	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "GAZE_COVERAGE_RUN=") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 GAZE_COVERAGE_RUN entry, got %d", count)
	}

	// Verify the value is "1", not "stale".
	found := false
	for _, e := range cmd.Env {
		if e == "GAZE_COVERAGE_RUN=1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected GAZE_COVERAGE_RUN=1 (not stale value) in cmd.Env")
	}
}

func TestGenerateCoverProfile_NestedRunRefused(t *testing.T) {
	// Inside a gaze coverage collection, generateCoverProfile must
	// refuse before creating a profile or spawning go test. Otherwise
	// a test that generates coverage for its own module re-runs
	// itself without bound (each level runs `go test ./...` again).
	t.Setenv("GAZE_COVERAGE_RUN", "1")
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	// With no go on PATH, an attempted spawn would fail with an exec
	// error instead of ErrNestedCoverageRun.
	t.Setenv("PATH", "")

	path, err := generateCoverProfile(t.TempDir(), []string{"./..."}, false, io.Discard)
	if !errors.Is(err, ErrNestedCoverageRun) {
		t.Fatalf("generateCoverProfile error = %v, want ErrNestedCoverageRun", err)
	}
	if path != "" {
		t.Errorf("generateCoverProfile path = %q, want empty", path)
	}
	entries, readErr := os.ReadDir(tmp)
	if readErr != nil {
		t.Fatalf("reading TMPDIR: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected no temp profile to be created, found %d entries", len(entries))
	}
}

func TestGenerateCoverProfile_ProceedsWhenUnset(t *testing.T) {
	// The guard's negative direction: with GAZE_COVERAGE_RUN empty,
	// generation must proceed past the recursion guard. The empty-string
	// pin (rather than ambient unset) keeps this test correct inside the
	// E2E self-check, which spawns the test suite with GAZE_COVERAGE_RUN=1
	// set; an ambient-unset variant would fail there exactly when the
	// guard works.
	t.Setenv("GAZE_COVERAGE_RUN", "")
	t.Setenv("TMPDIR", t.TempDir())
	// With no go on PATH, a run that proceeds past the guard fails with
	// an exec error instead of recursing.
	t.Setenv("PATH", "")

	_, err := generateCoverProfile(t.TempDir(), []string{"./..."}, false, io.Discard)
	if err == nil {
		t.Fatal("expected error from generateCoverProfile with empty PATH (go test cannot spawn), got nil")
	}
	if errors.Is(err, ErrNestedCoverageRun) {
		t.Fatalf("generateCoverProfile error = %v, want a spawn error: the guard fired with GAZE_COVERAGE_RUN unset", err)
	}
}
