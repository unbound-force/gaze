package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_CreatesFiles verifies SC-001: gaze init creates exactly
// 4 files in the correct directories when run in an empty project.
func TestRun_CreatesFiles(t *testing.T) {
	dir := t.TempDir()

	// Create go.mod so no warning is printed.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "1.2.3",
		Stdout:    &buf,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(result.Created) != 4 {
		t.Errorf("expected 4 created files, got %d: %v", len(result.Created), result.Created)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped files, got %d: %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten files, got %d: %v", len(result.Overwritten), result.Overwritten)
	}

	// Verify all 4 expected files exist on disk.
	expected := []string{
		".opencode/agents/gaze-reporter.md",
		".opencode/agents/doc-classifier.md",
		".opencode/command/gaze.md",
		".opencode/command/classify-docs.md",
	}
	for _, rel := range expected {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", rel)
		}
	}

	// Verify summary mentions "created".
	output := buf.String()
	if !strings.Contains(output, "created:") {
		t.Errorf("summary should mention 'created:', got:\n%s", output)
	}
	if !strings.Contains(output, "Run /gaze in OpenCode") {
		t.Errorf("summary should contain hint, got:\n%s", output)
	}
}

// TestRun_SkipsExisting verifies SC-002: gaze init skips existing
// files and reports them when --force is not set.
func TestRun_SkipsExisting(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: create all files.
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Second run without --force: should skip all files.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	if len(result.Created) != 0 {
		t.Errorf("expected 0 created, got %d: %v", len(result.Created), result.Created)
	}
	if len(result.Skipped) != 4 {
		t.Errorf("expected 4 skipped, got %d: %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}

	output := buf2.String()
	if !strings.Contains(output, "skipped:") {
		t.Errorf("summary should mention 'skipped:', got:\n%s", output)
	}
	if !strings.Contains(output, "use --force to overwrite") {
		t.Errorf("summary should suggest --force, got:\n%s", output)
	}
}

// TestRun_ForceOverwrites verifies SC-003: gaze init --force
// overwrites all files and reports the overwrites.
func TestRun_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: create all files.
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Second run with --force: should overwrite all files.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Force:     true,
		Version:   "2.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() with force returned error: %v", err)
	}

	if len(result.Created) != 0 {
		t.Errorf("expected 0 created, got %d: %v", len(result.Created), result.Created)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped, got %d: %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Overwritten) != 4 {
		t.Errorf("expected 4 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}

	output := buf2.String()
	if !strings.Contains(output, "overwritten:") {
		t.Errorf("summary should mention 'overwritten:', got:\n%s", output)
	}
}

// TestRun_VersionMarker verifies SC-004: every scaffolded file
// contains the version marker as the first line.
func TestRun_VersionMarker(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "0.1.0",
		Stdout:    &buf,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	expected := "<!-- scaffolded by gaze 0.1.0 -->"

	paths, err := AssetPaths()
	if err != nil {
		t.Fatalf("AssetPaths() returned error: %v", err)
	}
	for _, relPath := range paths {
		fullPath := filepath.Join(dir, ".opencode", relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}

		firstLine := strings.SplitN(string(content), "\n", 2)[0]
		if firstLine != expected {
			t.Errorf("file %s: expected first line %q, got %q", relPath, expected, firstLine)
		}
	}
}

// TestRun_VersionMarker_Dev verifies that development builds use
// "dev" as the version string in the marker.
func TestRun_VersionMarker_Dev(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "", // empty defaults to "dev"
		Stdout:    &buf,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	expected := "<!-- scaffolded by gaze dev -->"
	path := filepath.Join(dir, ".opencode", "agents", "gaze-reporter.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	firstLine := strings.SplitN(string(content), "\n", 2)[0]
	if firstLine != expected {
		t.Errorf("expected first line %q, got %q", expected, firstLine)
	}
}

// TestRun_NoGoMod_PrintsWarning verifies US4-AS6: gaze init in a
// directory without go.mod prints a warning but still creates files.
func TestRun_NoGoMod_PrintsWarning(t *testing.T) {
	dir := t.TempDir()
	// Deliberately do NOT create go.mod.

	var buf bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	// Files should still be created.
	if len(result.Created) != 4 {
		t.Errorf("expected 4 created files, got %d", len(result.Created))
	}

	// Warning should be printed.
	output := buf.String()
	if !strings.Contains(output, "Warning: no go.mod found") {
		t.Errorf("expected go.mod warning, got:\n%s", output)
	}
}

// TestEmbeddedAssetsMatchSource verifies SC-005 / FR-017: the
// embedded assets in internal/scaffold/assets/ are identical to
// the corresponding files in .opencode/.
func TestEmbeddedAssetsMatchSource(t *testing.T) {
	// Find the project root by walking up from this test file's
	// directory until we find go.mod.
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("finding project root: %v", err)
	}

	paths, err := AssetPaths()
	if err != nil {
		t.Fatalf("AssetPaths() returned error: %v", err)
	}

	if len(paths) != 4 {
		t.Fatalf("expected 4 embedded assets, got %d: %v", len(paths), paths)
	}

	for _, relPath := range paths {
		embedded, err := AssetContent(relPath)
		if err != nil {
			t.Fatalf("reading embedded asset %s: %v", relPath, err)
		}

		sourcePath := filepath.Join(projectRoot, ".opencode", relPath)
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("reading source file %s: %v", sourcePath, err)
		}

		if !bytes.Equal(embedded, source) {
			t.Errorf("drift detected: internal/scaffold/assets/%s differs from .opencode/%s\n"+
				"Run: cp .opencode/%s internal/scaffold/assets/%s",
				relPath, relPath, relPath, relPath)
		}
	}
}

// TestAssetPaths_Returns4Files verifies the embedded asset manifest
// contains exactly 4 files.
func TestAssetPaths_Returns4Files(t *testing.T) {
	paths, err := AssetPaths()
	if err != nil {
		t.Fatalf("AssetPaths() returned error: %v", err)
	}

	expected := map[string]bool{
		"agents/gaze-reporter.md":  true,
		"agents/doc-classifier.md": true,
		"command/gaze.md":          true,
		"command/classify-docs.md": true,
	}

	if len(paths) != len(expected) {
		t.Fatalf("expected %d assets, got %d: %v", len(expected), len(paths), paths)
	}

	for _, p := range paths {
		if !expected[p] {
			t.Errorf("unexpected asset path: %s", p)
		}
	}
}

// findProjectRoot walks up the directory tree from the current
// working directory to find the project root (directory containing
// go.mod).
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
