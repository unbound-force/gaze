package scaffold

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractVersion verifies that extractVersion correctly parses
// version markers from the first line of scaffolded files.
func TestExtractVersion(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "valid version marker",
			content: "<!-- scaffolded by gaze v1.0.0 -->\n# Some content",
			want:    "v1.0.0",
		},
		{
			name:    "dev version marker",
			content: "<!-- scaffolded by gaze dev -->\n# Some content",
			want:    "dev",
		},
		{
			name:    "empty version in marker",
			content: "<!-- scaffolded by gaze  -->\n# Some content",
			want:    "",
		},
		{
			name:    "non-marker first line",
			content: "# Some other content\n<!-- scaffolded by gaze v1.0.0 -->",
			want:    "",
		},
		{
			name:    "empty file",
			content: "",
			want:    "",
		},
		{
			name:    "marker only no newline",
			content: "<!-- scaffolded by gaze v2.3.4 -->",
			want:    "v2.3.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.md")
			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("writing test file: %v", err)
				}
			} else {
				// Create empty file.
				if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
					t.Fatalf("writing empty test file: %v", err)
				}
			}

			got := extractVersion(path)
			if got != tt.want {
				t.Errorf("extractVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractVersion_NonexistentFile verifies extractVersion returns
// empty string for a file that does not exist.
func TestExtractVersion_NonexistentFile(t *testing.T) {
	got := extractVersion("/nonexistent/path/file.md")
	if got != "" {
		t.Errorf("extractVersion(nonexistent) = %q, want empty", got)
	}
}

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
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated files, got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.UpToDate) != 0 {
		t.Errorf("expected 0 up-to-date files, got %d: %v", len(result.UpToDate), result.UpToDate)
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

// TestRun_UpToDate verifies SC-002: gaze init reports files as
// up to date when they already have the current version marker.
func TestRun_UpToDate(t *testing.T) {
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

	// Second run with same version: all files should be up to date.
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
	if len(result.UpToDate) != 4 {
		t.Errorf("expected 4 up to date, got %d: %v", len(result.UpToDate), result.UpToDate)
	}
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated, got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}

	output := buf2.String()
	if !strings.Contains(output, "up to date:") {
		t.Errorf("summary should mention 'up to date:', got:\n%s", output)
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
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated, got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.UpToDate) != 0 {
		t.Errorf("expected 0 up to date, got %d: %v", len(result.UpToDate), result.UpToDate)
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
		Version:   "v0.1.0",
		Stdout:    &buf,
	})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	expected := "<!-- scaffolded by gaze v0.1.0 -->"

	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
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

	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
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

	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}

	if len(paths) != 4 {
		t.Fatalf("expected 4 embedded assets, got %d: %v", len(paths), paths)
	}

	for _, relPath := range paths {
		embedded, err := assetContent(relPath)
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
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
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

// TestPrintSummary_MixedScenario verifies FR-005 / FR-007: output
// correctly categorizes files into all four states and includes
// version transition details for updated files.
func TestPrintSummary_MixedScenario(t *testing.T) {
	r := &Result{
		Created:  []string{".opencode/command/new-cmd.md"},
		Updated:  []string{".opencode/agents/gaze-reporter.md"},
		UpToDate: []string{".opencode/command/gaze.md", ".opencode/command/classify-docs.md"},
		UpdatedFrom: map[string]string{
			".opencode/agents/gaze-reporter.md": "v1.0.0",
		},
	}

	var buf bytes.Buffer
	printSummary(&buf, r, "v2.0.0")
	output := buf.String()

	// Header should say "initialized" since files were changed.
	if !strings.Contains(output, "initialized:") {
		t.Errorf("expected 'initialized:' header, got:\n%s", output)
	}

	// Should contain all disposition labels.
	if !strings.Contains(output, "created:") {
		t.Errorf("expected 'created:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "updated:") {
		t.Errorf("expected 'updated:' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "up to date:") {
		t.Errorf("expected 'up to date:' in output, got:\n%s", output)
	}

	// Version transition for updated file.
	if !strings.Contains(output, "(v1.0.0 -> v2.0.0)") {
		t.Errorf("expected version transition '(v1.0.0 -> v2.0.0)' in output, got:\n%s", output)
	}

	// Summary count line.
	if !strings.Contains(output, "1 created") {
		t.Errorf("expected '1 created' in summary, got:\n%s", output)
	}
	if !strings.Contains(output, "1 updated") {
		t.Errorf("expected '1 updated' in summary, got:\n%s", output)
	}
	if !strings.Contains(output, "2 up to date") {
		t.Errorf("expected '2 up to date' in summary, got:\n%s", output)
	}
}

// TestPrintSummary_AllUpToDate verifies SC-002: when all files
// are current, the header says "already up to date" and no
// modification labels appear.
func TestPrintSummary_AllUpToDate(t *testing.T) {
	r := &Result{
		UpToDate: []string{
			".opencode/agents/gaze-reporter.md",
			".opencode/agents/doc-classifier.md",
			".opencode/command/gaze.md",
			".opencode/command/classify-docs.md",
		},
		UpdatedFrom: make(map[string]string),
	}

	var buf bytes.Buffer
	printSummary(&buf, r, "v1.0.0")
	output := buf.String()

	// Header should say "already up to date".
	if !strings.Contains(output, "already up to date:") {
		t.Errorf("expected 'already up to date:' header, got:\n%s", output)
	}

	// Should NOT contain "created:", "updated:", or "overwritten:".
	if strings.Contains(output, "created:") {
		t.Errorf("should not contain 'created:' when all up to date, got:\n%s", output)
	}
	if strings.Contains(output, "updated:") {
		t.Errorf("should not contain 'updated:' when all up to date, got:\n%s", output)
	}
	if strings.Contains(output, "overwritten:") {
		t.Errorf("should not contain 'overwritten:' when all up to date, got:\n%s", output)
	}

	// Summary should show "4 up to date".
	if !strings.Contains(output, "4 up to date") {
		t.Errorf("expected '4 up to date' in summary, got:\n%s", output)
	}
}

// TestRun_UpdatesOutdated verifies SC-001 / FR-002: gaze init
// updates all files when their version marker differs from the
// running version.
func TestRun_UpdatesOutdated(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: scaffold with v1.0.0.
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "v1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Second run with v2.0.0: all files should be updated.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "v2.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	if len(result.Updated) != 4 {
		t.Errorf("expected 4 updated, got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.UpToDate) != 0 {
		t.Errorf("expected 0 up to date, got %d: %v", len(result.UpToDate), result.UpToDate)
	}
	if len(result.Created) != 0 {
		t.Errorf("expected 0 created, got %d: %v", len(result.Created), result.Created)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}

	// Verify each file has the new version marker.
	expectedMarker := "<!-- scaffolded by gaze v2.0.0 -->"
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}
	for _, relPath := range paths {
		fullPath := filepath.Join(dir, ".opencode", relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}
		firstLine := strings.SplitN(string(content), "\n", 2)[0]
		if firstLine != expectedMarker {
			t.Errorf("file %s: expected first line %q, got %q", relPath, expectedMarker, firstLine)
		}
	}

	// Verify UpdatedFrom maps each file to "v1.0.0".
	for _, f := range result.Updated {
		oldVer, ok := result.UpdatedFrom[f]
		if !ok {
			t.Errorf("UpdatedFrom missing entry for %s", f)
			continue
		}
		if oldVer != "v1.0.0" {
			t.Errorf("UpdatedFrom[%s] = %q, want %q", f, oldVer, "v1.0.0")
		}
	}
}

// TestRun_DevAlwaysUpdates verifies FR-008: dev builds always
// update all existing scaffolded files regardless of on-disk marker.
func TestRun_DevAlwaysUpdates(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: scaffold with v1.0.0.
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "v1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Second run with dev version (empty string defaults to "dev"):
	// all files should be updated.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "", // defaults to "dev"
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() with dev returned error: %v", err)
	}

	if len(result.Updated) != 4 {
		t.Errorf("expected 4 updated (dev from v1.0.0), got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.UpToDate) != 0 {
		t.Errorf("expected 0 up to date, got %d: %v", len(result.UpToDate), result.UpToDate)
	}

	// Verify marker changes to "dev".
	expectedMarker := "<!-- scaffolded by gaze dev -->"
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}
	for _, relPath := range paths {
		fullPath := filepath.Join(dir, ".opencode", relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}
		firstLine := strings.SplitN(string(content), "\n", 2)[0]
		if firstLine != expectedMarker {
			t.Errorf("file %s: expected first line %q, got %q", relPath, expectedMarker, firstLine)
		}
	}

	// Third run with dev again: should still update (dev always updates).
	var buf3 bytes.Buffer
	result3, err := Run(Options{
		TargetDir: dir,
		Version:   "", // defaults to "dev"
		Stdout:    &buf3,
	})
	if err != nil {
		t.Fatalf("third Run() with dev returned error: %v", err)
	}

	if len(result3.Updated) != 4 {
		t.Errorf("expected 4 updated (dev-to-dev), got %d: %v", len(result3.Updated), result3.Updated)
	}
}

// TestRun_MissingMarkerTreatedAsOutdated verifies FR-006: files
// without a recognizable version marker are treated as outdated.
func TestRun_MissingMarkerTreatedAsOutdated(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: scaffold with v1.0.0.
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "v1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Overwrite one file with content that has no version marker.
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no asset paths found")
	}
	targetFile := filepath.Join(dir, ".opencode", paths[0])
	if err := os.WriteFile(targetFile, []byte("# No version marker here\n"), 0o644); err != nil {
		t.Fatalf("overwriting %s: %v", paths[0], err)
	}

	// Second run with v2.0.0: the file with no marker should be updated.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "v2.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	// All 4 files should be updated (3 have old v1.0.0, 1 has no marker).
	if len(result.Updated) != 4 {
		t.Errorf("expected 4 updated, got %d: %v", len(result.Updated), result.Updated)
	}

	// The file with no marker should have empty string in UpdatedFrom.
	noMarkerPath := filepath.Join(".opencode", paths[0])
	oldVer, ok := result.UpdatedFrom[noMarkerPath]
	if !ok {
		t.Errorf("UpdatedFrom missing entry for %s", noMarkerPath)
	} else if oldVer != "" {
		t.Errorf("UpdatedFrom[%s] = %q, want empty string (no marker)", noMarkerPath, oldVer)
	}

	// Verify the file now has the v2.0.0 marker.
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("reading %s: %v", targetFile, err)
	}
	expectedMarker := "<!-- scaffolded by gaze v2.0.0 -->"
	firstLine := strings.SplitN(string(content), "\n", 2)[0]
	if firstLine != expectedMarker {
		t.Errorf("file %s: expected first line %q, got %q", paths[0], expectedMarker, firstLine)
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
		_, err := os.Stat(filepath.Join(dir, "go.mod"))
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
