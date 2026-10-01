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

// TestRun_CreatesFiles verifies that gaze init creates exactly 9
// files (agents, commands, reference files, and DCP config) in the
// correct directories when run in an empty project.
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

	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d: %v", len(result.Created), result.Created)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("expected 0 skipped files, got %d: %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten files, got %d: %v", len(result.Overwritten), result.Overwritten)
	}
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated files, got %d: %v", len(result.Updated), result.Updated)
	}

	// Verify all expected files exist on disk.
	expected := []string{
		".opencode/agents/gaze-reporter.md",
		".opencode/agents/reviewer-testing.md",
		".opencode/commands/gaze.md",
		".opencode/commands/speckit.testreview.md",
		".opencode/references/doc-scoring-model.md",
		".opencode/references/example-report.md",
		".opencode/dcp.jsonc",
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
	if !strings.Contains(output, "Run /gaze") {
		t.Errorf("summary should contain hint, got:\n%s", output)
	}
}

// TestRun_SkipsExisting verifies that gaze init skips user-owned
// files and skips tool-owned reference files when their content is
// identical to the embedded version (no --force needed).
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

	// Second run without --force: tool-owned files have identical
	// content, so all 9 files land in Skipped (3 user-owned +
	// 6 tool-owned identical).
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
	if len(result.Skipped) != 9 {
		t.Errorf("expected 9 skipped, got %d: %v", len(result.Skipped), result.Skipped)
	}
	if len(result.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated, got %d: %v", len(result.Updated), result.Updated)
	}

	output := buf2.String()
	if !strings.Contains(output, "skipped:") {
		t.Errorf("summary should mention 'skipped:', got:\n%s", output)
	}
	if !strings.Contains(output, "3 files skipped (use --force to overwrite)") {
		t.Errorf("summary should mention 3 user-owned files skipped, got:\n%s", output)
	}
}

// TestRun_ForceOverwrites verifies that gaze init --force
// overwrites all files (user-owned and tool-owned) and reports
// the overwrites.
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
	if len(result.Overwritten) != 9 {
		t.Errorf("expected 9 overwritten, got %d: %v", len(result.Overwritten), result.Overwritten)
	}
	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated, got %d: %v", len(result.Updated), result.Updated)
	}

	output := buf2.String()
	if !strings.Contains(output, "overwritten:") {
		t.Errorf("summary should mention 'overwritten:', got:\n%s", output)
	}
}

// TestRun_VersionMarker verifies that every scaffolded file
// contains the version marker. Files with YAML frontmatter have
// the marker inserted after the closing delimiter. Files without
// frontmatter (e.g., reference files) have the marker appended.
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

		s := string(content)

		// Non-Markdown files skip the HTML marker (it would be
		// invalid syntax in formats like JSONC).
		if !strings.HasSuffix(relPath, ".md") {
			if strings.Contains(s, expected) {
				t.Errorf("file %s: non-Markdown file should NOT contain HTML marker", relPath)
			}
			continue
		}

		// Marker must be present in Markdown files.
		if !strings.Contains(s, expected) {
			t.Errorf("file %s: marker %q not found in content", relPath, expected)
		}

		// Files with YAML frontmatter: marker must appear after
		// the closing delimiter. Files without frontmatter (e.g.,
		// reference files): marker is appended at the end.
		firstLine := strings.SplitN(s, "\n", 2)[0]
		if firstLine == "---" {
			// Frontmatter file: verify marker is after closing delimiter.
			closingIdx := strings.Index(s[4:], "\n---\n")
			if closingIdx < 0 {
				t.Fatalf("file %s: no closing frontmatter delimiter found", relPath)
			}
			markerIdx := strings.Index(s, expected)
			frontmatterEnd := closingIdx + 4 + len("\n---\n")
			if markerIdx < frontmatterEnd {
				t.Errorf("file %s: marker appears before frontmatter end (marker at %d, frontmatter ends at %d)",
					relPath, markerIdx, frontmatterEnd)
			}
		}
		// Non-frontmatter Markdown files: marker presence already verified above.
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

		s := string(content)

		// Non-Markdown files skip the HTML marker.
		if !strings.HasSuffix(relPath, ".md") {
			if strings.Contains(s, expected) {
				t.Errorf("file %s: non-Markdown file should NOT contain HTML marker", relPath)
			}
			continue
		}

		if !strings.Contains(s, expected) {
			t.Errorf("file %s: marker %q not found in content", relPath, expected)
		}

		// Files with frontmatter must start with "---".
		// Reference files without frontmatter skip this check.
		firstLine := strings.SplitN(s, "\n", 2)[0]
		if strings.HasPrefix(relPath, "agents/") || strings.HasPrefix(relPath, "commands/") {
			if firstLine != "---" {
				t.Errorf("file %s: expected first line %q (frontmatter), got %q", relPath, "---", firstLine)
			}
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
	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d", len(result.Created))
	}

	// Warning should be printed.
	output := buf.String()
	if !strings.Contains(output, "Warning: no go.mod found") {
		t.Errorf("expected go.mod warning, got:\n%s", output)
	}
}

// TestEmbeddedAssetsMatchSource verifies that the embedded assets
// in internal/scaffold/assets/ are identical to the corresponding
// files in .opencode/. This prevents drift between the scaffold
// copies and the live copies used by OpenCode.
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

	if len(paths) != 9 {
		t.Fatalf("expected 9 embedded assets, got %d: %v", len(paths), paths)
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

// TestAssetPaths_Returns9Files verifies the embedded asset manifest
// contains exactly 9 files.
func TestAssetPaths_Returns9Files(t *testing.T) {
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}

	expected := map[string]bool{
		"agents/gaze-reporter.md":         true,
		"agents/gaze-test-generator.md":   true,
		"agents/reviewer-testing.md":      true,
		"commands/gaze-fix.md":            true,
		"commands/gaze.md":                true,
		"commands/speckit.testreview.md":  true,
		"dcp.jsonc":                       true,
		"references/doc-scoring-model.md": true,
		"references/example-report.md":    true,
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

// TestRun_OverwriteOnDiff_ToolOwned verifies that tool-owned files
// (references and specific commands) are overwritten when their
// content differs from the embedded version, while user-owned
// files are skipped.
func TestRun_OverwriteOnDiff_ToolOwned(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run: create all 9 files.
	var buf1 bytes.Buffer
	result1, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}
	if len(result1.Created) != 9 {
		t.Fatalf("expected 9 created files, got %d: %v", len(result1.Created), result1.Created)
	}

	// Second run without --force: all files should be skipped
	// (tool-owned files have identical content).
	var buf2 bytes.Buffer
	result2, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}
	if len(result2.Skipped) != 9 {
		t.Errorf("expected 9 skipped, got %d: %v", len(result2.Skipped), result2.Skipped)
	}
	if len(result2.Updated) != 0 {
		t.Errorf("expected 0 updated, got %d: %v", len(result2.Updated), result2.Updated)
	}

	// Modify a reference file and a tool-owned command file on disk.
	refPath := filepath.Join(dir, ".opencode", "references", "example-report.md")
	if err := os.WriteFile(refPath, []byte("modified content\n"), 0o644); err != nil {
		t.Fatalf("modifying reference file: %v", err)
	}
	cmdPath := filepath.Join(dir, ".opencode", "commands", "speckit.testreview.md")
	if err := os.WriteFile(cmdPath, []byte("modified command\n"), 0o644); err != nil {
		t.Fatalf("modifying command file: %v", err)
	}

	// Third run without --force: the 2 modified tool-owned files
	// should be overwritten (Updated), the other 4 tool-owned files
	// skipped (identical), and 3 user-owned files skipped.
	var buf3 bytes.Buffer
	result3, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf3,
	})
	if err != nil {
		t.Fatalf("third Run() returned error: %v", err)
	}

	if len(result3.Updated) != 2 {
		t.Errorf("expected 2 updated, got %d: %v", len(result3.Updated), result3.Updated)
	}
	// 3 user-owned + 4 identical tool-owned = 7 skipped.
	if len(result3.Skipped) != 7 {
		t.Errorf("expected 7 skipped, got %d: %v", len(result3.Skipped), result3.Skipped)
	}
	if len(result3.Created) != 0 {
		t.Errorf("expected 0 created, got %d: %v", len(result3.Created), result3.Created)
	}
	if len(result3.Overwritten) != 0 {
		t.Errorf("expected 0 overwritten, got %d: %v", len(result3.Overwritten), result3.Overwritten)
	}

	// Verify the reference file on disk was restored.
	restored, readErr := os.ReadFile(refPath)
	if readErr != nil {
		t.Fatalf("reading restored reference file: %v", readErr)
	}
	embedded, embErr := assetContent("references/example-report.md")
	if embErr != nil {
		t.Fatalf("reading embedded asset: %v", embErr)
	}
	expectedContent := insertMarkerAfterFrontmatter(embedded, versionMarker("1.0.0"))
	if !bytes.Equal(restored, expectedContent) {
		t.Errorf("restored reference content does not match embedded asset (got %d bytes, want %d bytes)",
			len(restored), len(expectedContent))
	}

	// Verify the command file on disk was restored.
	restoredCmd, readErr := os.ReadFile(cmdPath)
	if readErr != nil {
		t.Fatalf("reading restored command file: %v", readErr)
	}
	embeddedCmd, embErr := assetContent("commands/speckit.testreview.md")
	if embErr != nil {
		t.Fatalf("reading embedded command asset: %v", embErr)
	}
	expectedCmd := insertMarkerAfterFrontmatter(embeddedCmd, versionMarker("1.0.0"))
	if !bytes.Equal(restoredCmd, expectedCmd) {
		t.Errorf("restored command content does not match embedded asset (got %d bytes, want %d bytes)",
			len(restoredCmd), len(expectedCmd))
	}

	// Verify the summary mentions "updated".
	output := buf3.String()
	if !strings.Contains(output, "updated:") {
		t.Errorf("summary should mention 'updated:', got:\n%s", output)
	}
	if !strings.Contains(output, "content changed") {
		t.Errorf("summary should mention 'content changed', got:\n%s", output)
	}
}

// TestRun_OverwriteOnDiff_SkipsIdentical verifies that tool-owned
// reference files with identical content to the embedded version
// appear in result.Skipped, not result.Updated.
func TestRun_OverwriteOnDiff_SkipsIdentical(t *testing.T) {
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

	// Second run with same version: reference files should be
	// skipped (content identical), not updated.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "1.0.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	if len(result.Updated) != 0 {
		t.Errorf("expected 0 updated (identical content), got %d: %v", len(result.Updated), result.Updated)
	}

	// All 9 files should be skipped.
	if len(result.Skipped) != 9 {
		t.Errorf("expected 9 skipped, got %d: %v", len(result.Skipped), result.Skipped)
	}

	// Verify tool-owned files are specifically in the skipped list.
	skippedMap := make(map[string]bool)
	for _, f := range result.Skipped {
		skippedMap[f] = true
	}
	for _, toolOwned := range []string{
		filepath.Join(".opencode", "references", "doc-scoring-model.md"),
		filepath.Join(".opencode", "references", "example-report.md"),
		filepath.Join(".opencode", "commands", "speckit.testreview.md"),
	} {
		if !skippedMap[toolOwned] {
			t.Errorf("expected %s in skipped list", toolOwned)
		}
	}
}

// TestIsToolOwned verifies the explicit file list that determines
// tool-owned vs user-owned classification.
func TestIsToolOwned(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Tool-owned: references directory (prefix match).
		{"references/doc-scoring-model.md", true},
		{"references/example-report.md", true},
		{"references/any-future-file.md", true},
		// Tool-owned: explicit command files.
		{"commands/speckit.testreview.md", true},
		{"commands/gaze-fix.md", true},
		// Tool-owned: DCP configuration.
		{"dcp.jsonc", true},
		// User-owned: agents.
		{"agents/gaze-reporter.md", false},
		{"agents/reviewer-testing.md", false},
		// User-owned: other commands.
		{"commands/gaze.md", false},
		// Edge cases.
		{"commands/speckit.analyze.md", false},
		{"references", false},
	}
	for _, tc := range cases {
		got := isToolOwned(tc.path)
		if got != tc.want {
			t.Errorf("isToolOwned(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// TestInsertMarkerAfterFrontmatter exercises the frontmatter-aware
// marker insertion function with edge cases.
func TestInsertMarkerAfterFrontmatter(t *testing.T) {
	marker := "<!-- marker -->\n"
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty input",
			input: "",
			want:  marker,
		},
		{
			name:  "no frontmatter",
			input: "# Hello\nWorld\n",
			want:  "# Hello\nWorld\n" + marker,
		},
		{
			name:  "unclosed frontmatter",
			input: "---\nkey: val\n",
			want:  "---\nkey: val\n" + marker,
		},
		{
			name:  "well-formed frontmatter",
			input: "---\nkey: val\n---\n# Body\n",
			want:  "---\nkey: val\n---\n" + marker + "# Body\n",
		},
		{
			name:  "body contains dashes",
			input: "---\nk: v\n---\n# H\n---\nmore\n",
			want:  "---\nk: v\n---\n" + marker + "# H\n---\nmore\n",
		},
		{
			name:  "frontmatter only no body",
			input: "---\nk: v\n---\n",
			want:  "---\nk: v\n---\n" + marker,
		},
		{
			name:  "empty marker",
			input: "---\nk: v\n---\n# Body\n",
			want:  "---\nk: v\n---\n# Body\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := marker
			if tc.name == "empty marker" {
				m = ""
			}
			got := string(insertMarkerAfterFrontmatter([]byte(tc.input), m))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestInsertMarkerAfterFrontmatter_ReplacesExistingMarker verifies
// that re-scaffolding a file with a different version replaces the
// existing marker instead of appending a second one (regression test
// for issue #279).
func TestInsertMarkerAfterFrontmatter_ReplacesExistingMarker(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "replaces marker after frontmatter",
			input: "---\ntitle: test\n---\n<!-- scaffolded by gaze dev -->\n# Agent\n",
			want:  "---\ntitle: test\n---\n<!-- scaffolded by gaze v1.8.0 -->\n# Agent\n",
		},
		{
			name:  "replaces marker without frontmatter",
			input: "# Ref\ncontent\n<!-- scaffolded by gaze dev -->\n",
			want:  "# Ref\ncontent\n<!-- scaffolded by gaze v1.8.0 -->\n",
		},
		{
			name:  "removes duplicate markers",
			input: "---\nk: v\n---\n<!-- scaffolded by gaze dev -->\n<!-- scaffolded by gaze v1.7.0 -->\n# Body\n",
			want:  "---\nk: v\n---\n<!-- scaffolded by gaze v1.8.0 -->\n# Body\n",
		},
		{
			name:  "no existing marker",
			input: "---\nk: v\n---\n# Body\n",
			want:  "---\nk: v\n---\n<!-- scaffolded by gaze v1.8.0 -->\n# Body\n",
		},
	}
	newMarker := "<!-- scaffolded by gaze v1.8.0 -->\n"
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := string(insertMarkerAfterFrontmatter([]byte(tc.input), newMarker))
			if result != tc.want {
				t.Errorf("got:\n%s\nwant:\n%s", result, tc.want)
			}
			count := strings.Count(result, "<!-- scaffolded by gaze")
			if count != 1 {
				t.Errorf("expected exactly 1 marker, got %d", count)
			}
		})
	}
}

// TestRemoveExistingMarker verifies the marker stripping helper.
func TestRemoveExistingMarker(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips single marker",
			input: "---\nk: v\n---\n<!-- scaffolded by gaze dev -->\n# Body\n",
			want:  "---\nk: v\n---\n# Body\n",
		},
		{
			name:  "strips multiple markers",
			input: "---\nk: v\n---\n<!-- scaffolded by gaze dev -->\n<!-- scaffolded by gaze v1.8.0 -->\n# Body\n",
			want:  "---\nk: v\n---\n# Body\n",
		},
		{
			name:  "no marker present",
			input: "---\nk: v\n---\n# Body\n",
			want:  "---\nk: v\n---\n# Body\n",
		},
		{
			name:  "marker at end of file",
			input: "# Ref\ncontent\n<!-- scaffolded by gaze v1.0.0 -->\n",
			want:  "# Ref\ncontent\n",
		},
		{
			name:  "preserves non-marker HTML comments",
			input: "<!-- not a gaze marker -->\n<!-- scaffolded by gaze dev -->\n",
			want:  "<!-- not a gaze marker -->\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(removeExistingMarker([]byte(tc.input)))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestRun_ReScaffoldSingleMarker verifies the end-to-end scenario
// from issue #279: running gaze init twice with different versions
// produces exactly one marker per file, not two.
func TestRun_ReScaffoldSingleMarker(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// First run with version "dev".
	var buf1 bytes.Buffer
	_, err := Run(Options{
		TargetDir: dir,
		Version:   "dev",
		Stdout:    &buf1,
	})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Second run with version "v1.8.0" and --force to overwrite
	// user-owned files too.
	var buf2 bytes.Buffer
	_, err = Run(Options{
		TargetDir: dir,
		Force:     true,
		Version:   "v1.8.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	// Every Markdown file must have exactly one marker with the
	// new version, and zero markers with the old version.
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}
	for _, relPath := range paths {
		if !strings.HasSuffix(relPath, ".md") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, ".opencode", relPath))
		if err != nil {
			t.Fatalf("reading %s: %v", relPath, err)
		}
		s := string(content)

		count := strings.Count(s, "<!-- scaffolded by gaze")
		if count != 1 {
			t.Errorf("file %s: expected exactly 1 marker, got %d", relPath, count)
		}
		if !strings.Contains(s, "<!-- scaffolded by gaze v1.8.0 -->") {
			t.Errorf("file %s: expected v1.8.0 marker, not found", relPath)
		}
		if strings.Contains(s, "gaze dev -->") {
			t.Errorf("file %s: old dev marker should have been replaced", relPath)
		}
	}
}

// TestProtectTagPlacement verifies that embedded command files
// contain correctly placed <protect> tags for DCP context
// preservation. Asserts tag count per file, balanced pairs, no
// tags in YAML frontmatter, no nesting, and own-line placement.
func TestProtectTagPlacement(t *testing.T) {
	cases := []struct {
		file          string
		expectedPairs int
	}{
		{"commands/gaze-fix.md", 1},
		{"commands/speckit.testreview.md", 1},
		{"commands/gaze.md", 1},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			content, err := assetContent(tc.file)
			if err != nil {
				t.Fatalf("reading embedded asset %s: %v", tc.file, err)
			}
			s := string(content)
			lines := strings.Split(s, "\n")

			// 1. Correct tag count.
			openCount := strings.Count(s, "<protect>")
			closeCount := strings.Count(s, "</protect>")
			if openCount != tc.expectedPairs {
				t.Errorf("expected %d <protect> tags, got %d", tc.expectedPairs, openCount)
			}

			// 2. Balanced pairs.
			if openCount != closeCount {
				t.Errorf("unbalanced tags: %d <protect> vs %d </protect>", openCount, closeCount)
			}

			// 3. No tags within YAML frontmatter.
			if strings.HasPrefix(s, "---\n") {
				closingIdx := strings.Index(s[4:], "\n---\n")
				if closingIdx >= 0 {
					frontmatter := s[:closingIdx+4+len("\n---\n")]
					if strings.Contains(frontmatter, "<protect>") || strings.Contains(frontmatter, "</protect>") {
						t.Errorf("protect tag found inside YAML frontmatter")
					}
				}
			}

			// 4. No nested tags.
			depth := 0
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "<protect>" {
					depth++
					if depth > 1 {
						t.Errorf("nested <protect> tag detected (depth %d)", depth)
					}
				}
				if trimmed == "</protect>" {
					depth--
					if depth < 0 {
						t.Errorf("</protect> without matching <protect>")
					}
				}
			}
			if depth != 0 {
				t.Errorf("unclosed <protect> tags: final depth %d", depth)
			}

			// 5. Tags appear on their own line.
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if strings.Contains(line, "<protect>") && trimmed != "<protect>" {
					t.Errorf("line %d: <protect> tag not on its own line: %q", i+1, line)
				}
				if strings.Contains(line, "</protect>") && trimmed != "</protect>" {
					t.Errorf("line %d: </protect> tag not on its own line: %q", i+1, line)
				}
			}
		})
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
