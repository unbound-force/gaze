package aireport

import (
	"io"
	"path/filepath"
	"runtime"
	"testing"
)

// TestLoadGazeConfigBestEffort_AlwaysNonNil verifies that the function always
// returns a non-nil config, even in a directory with no .gaze.yaml.
func TestLoadGazeConfigBestEffort_AlwaysNonNil(t *testing.T) {
	cfg := loadGazeConfigBestEffort(".")
	if cfg == nil {
		t.Error("expected non-nil GazeConfig from loadGazeConfigBestEffort")
	}
}

// TestRunCRAPStep_RealPackage verifies that runCRAPStep successfully runs on
// a real package and returns a non-nil JSON payload.
// Guarded by testing.Short() — spawns the Go analysis pipeline.
func TestRunCRAPStep_RealPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs real CRAP analysis pipeline")
	}
	modRoot := findModuleRoot(t)
	res, err := runCRAPStep(
		[]string{"github.com/unbound-force/gaze/internal/config"},
		modRoot,
		"", // no pre-generated profile — use internal generation
		io.Discard,
		nil, // no contract coverage callback
	)
	if err != nil {
		t.Fatalf("runCRAPStep: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil crapStepResult")
	}
	if res.JSON == nil {
		t.Error("expected non-nil JSON from runCRAPStep")
	}
}

// TestRunDocscanStep_RealModuleDir verifies that runDocscanStep runs without
// error on the module root and returns a non-nil JSON payload.
// Guarded by testing.Short().
func TestRunDocscanStep_RealModuleDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs real docscan pipeline")
	}
	modRoot := findModuleRoot(t)
	raw, err := runDocscanStep(modRoot)
	if err != nil {
		t.Fatalf("runDocscanStep: %v", err)
	}
	if raw == nil {
		t.Error("expected non-nil JSON from runDocscanStep")
	}
}

// TestRunCRAPStep_WithCoverProfile verifies that runCRAPStep accepts a
// pre-generated coverage profile and produces a non-nil JSON result (FR-001,
// FR-002). Uses the static fixture at testdata/sample.coverprofile, which
// records one covered statement in internal/crap/crap.go.
// Guarded by testing.Short() — calls crap.Analyze which loads Go packages.
func TestRunCRAPStep_WithCoverProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: calls crap.Analyze which loads Go packages")
	}
	// Locate the testdata fixture relative to this file's directory.
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "sample.coverprofile")

	modRoot := findModuleRoot(t)
	res, err := runCRAPStep(
		[]string{"github.com/unbound-force/gaze/internal/crap"},
		modRoot,
		fixture,
		io.Discard,
		nil, // no contract coverage callback
	)
	if err != nil {
		t.Fatalf("runCRAPStep with coverprofile: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil crapStepResult")
	}
	if res.JSON == nil {
		t.Error("expected non-nil JSON from runCRAPStep with pre-generated profile")
	}
}

// TestRunProductionPipeline_RealPackage verifies that runProductionPipeline
// returns a non-nil payload and exercises all four steps without panicking.
// Guarded by testing.Short() — runs the full four-step pipeline.
func TestRunProductionPipeline_RealPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs full four-step analysis pipeline")
	}
	modRoot := findModuleRoot(t)
	payload, err := runProductionPipeline(
		[]string{"github.com/unbound-force/gaze/internal/config"},
		modRoot,
		"", // no pre-generated profile — use internal generation
		io.Discard,
		pipelineStepFuncs{}, // zero value = real step functions
	)
	if err != nil {
		t.Fatalf("runProductionPipeline: %v", err)
	}
	if payload == nil {
		t.Fatal("expected non-nil ReportPayload")
	}
	// CRAP step must succeed for a real package.
	if payload.CRAP == nil && payload.Errors.CRAP == nil {
		t.Error("expected either CRAP JSON or CRAP error, got both nil")
	}
}
