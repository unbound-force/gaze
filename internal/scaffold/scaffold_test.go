package scaffold

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
		".opencode/agents/divisor-gaze-testing.md",
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

// TestRun_SkipsExisting verifies that gaze init skips tool-owned
// files when their content is identical to the embedded version
// (no --force needed). All nine .opencode/ assets are tool-owned
// (gaze-command-tool-owned), so no user-owned .opencode/ file can
// be skipped and the --force hint must not print.
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

	// Second run without --force: every .opencode/ asset is
	// tool-owned with identical content, so all 9 files land in
	// Skipped (0 user-owned + 9 tool-owned identical).
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
	if strings.Contains(output, "use --force to overwrite") {
		t.Errorf("summary should not mention the --force hint (no user-owned .opencode/ file can be skipped), got:\n%s", output)
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
		"agents/divisor-gaze-testing.md":  true,
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

// TestAssetPaths_ExcludesCISubtree verifies the ci/ subtree — the
// CI workflow template consumed by the dedicated write path, not a
// mirrored .opencode/ asset (design D8) — never appears in the
// mirrored asset manifest.
func TestAssetPaths_ExcludesCISubtree(t *testing.T) {
	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}
	for _, p := range paths {
		if strings.HasPrefix(p, "ci/") {
			t.Errorf("mirrored asset path %q must not come from the ci/ subtree", p)
		}
	}
}

// TestRun_PlainInit_NeverWritesOpencodeCIDir verifies no .opencode/ci/
// mirror directory is ever produced by a plain init run, even though
// the embedded assets tree contains the ci/ template subtree.
func TestRun_PlainInit_NeverWritesOpencodeCIDir(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go 1.24")

	var buf bytes.Buffer
	if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", Stdout: &buf}); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opencode", "ci")); !os.IsNotExist(err) {
		t.Error("plain init must never write a .opencode/ci/ mirror of the CI template subtree")
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
	// should be overwritten (Updated), the other 5 tool-owned files
	// skipped (identical), and 2 user-owned files skipped.
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
	// 2 user-owned + 5 identical tool-owned = 7 skipped.
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
		{"commands/gaze.md", true},
		// Tool-owned: DCP configuration.
		{"dcp.jsonc", true},
		// Tool-owned: gaze-managed agent files.
		{"agents/divisor-gaze-testing.md", true},
		{"agents/gaze-reporter.md", true},
		// Tool-owned: gaze-test-generator agent file.
		{"agents/gaze-test-generator.md", true},
		// User-owned: agents outside the tool-owned list.
		{"agents/custom.md", false},
		// Legacy retired name must not match the tool-owned list.
		{"agents/reviewer-testing.md", false},
		// User-owned: non-embedded command paths. commands/gaze-local.md
		// is the /gaze customization escape hatch (gaze-command-tool-owned
		// design D2): gaze never writes that path.
		{"commands/gaze-local.md", false},
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

// TestRun_OverwritesCustomizedGazeReporter verifies FR-034: the
// interactive gaze-reporter agent is tool-owned, so a customized copy
// is replaced on the next gaze init even without --force.
func TestRun_OverwritesCustomizedGazeReporter(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf1 bytes.Buffer
	if _, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1}); err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Simulate a local customization of the agent prompt.
	agentPath := filepath.Join(dir, ".opencode", "agents", "gaze-reporter.md")
	if err := os.WriteFile(agentPath, []byte("# custom\n"), 0o644); err != nil {
		t.Fatalf("customizing agent file: %v", err)
	}

	var buf2 bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf2})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	var updated bool
	for _, f := range result.Updated {
		if strings.HasSuffix(f, filepath.Join("agents", "gaze-reporter.md")) {
			updated = true
		}
	}
	if !updated {
		t.Errorf("expected agents/gaze-reporter.md in Updated, got %v", result.Updated)
	}

	got, err := os.ReadFile(agentPath)
	if err != nil {
		t.Fatalf("reading agent file: %v", err)
	}
	embedded, err := assetContent("agents/gaze-reporter.md")
	if err != nil {
		t.Fatalf("reading embedded asset: %v", err)
	}
	want := insertMarkerAfterFrontmatter(embedded, versionMarker("1.0.0"))
	if !bytes.Equal(got, want) {
		t.Errorf("agent file was not restored to the embedded copy (got %d bytes, want %d bytes)",
			len(got), len(want))
	}
}

// TestRun_OverwritesCustomizedGazeCommand verifies the
// gaze-command-tool-owned "Customized copy is overwritten" scenario:
// commands/gaze.md is tool-owned, so a customized copy is replaced on
// the next gaze init even without --force.
func TestRun_OverwritesCustomizedGazeCommand(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf1 bytes.Buffer
	if _, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1}); err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}

	// Simulate a local customization of the /gaze command.
	cmdPath := filepath.Join(dir, ".opencode", "commands", "gaze.md")
	if err := os.WriteFile(cmdPath, []byte("# custom\n"), 0o644); err != nil {
		t.Fatalf("customizing command file: %v", err)
	}

	var buf2 bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf2})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	var updated bool
	for _, f := range result.Updated {
		if f == filepath.Join(".opencode", "commands", "gaze.md") {
			updated = true
		}
	}
	if !updated {
		t.Errorf("expected commands/gaze.md in Updated, got %v", result.Updated)
	}

	got, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Fatalf("reading command file: %v", err)
	}
	embedded, err := assetContent("commands/gaze.md")
	if err != nil {
		t.Fatalf("reading embedded asset: %v", err)
	}
	want := insertMarkerAfterFrontmatter(embedded, versionMarker("1.0.0"))
	if !bytes.Equal(got, want) {
		t.Errorf("command file was not restored to the embedded copy (got %d bytes, want %d bytes)",
			len(got), len(want))
	}
}

// TestRun_HealsDuplicateGazeCommandMarkers verifies the
// gaze-command-tool-owned "Duplicate version markers are healed"
// scenario (design D4): a commands/gaze.md carrying the survey's
// observed dev + v1.8.0 duplicate markers (issue #353) and stale
// body content is rewritten by the tool-owned update path —
// handleToolOwnedFile's wholesale rewrite of the embedded asset
// plus a single current marker — without --force.
func TestRun_HealsDuplicateGazeCommandMarkers(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// Pre-write a stale copy shaped like the survey's findings: old
	// frontmatter, both observed marker lines, and a body that
	// predates the current embedded asset.
	cmdPath := filepath.Join(dir, ".opencode", "commands", "gaze.md")
	if err := os.MkdirAll(filepath.Dir(cmdPath), 0o755); err != nil {
		t.Fatalf("creating commands directory: %v", err)
	}
	stale := `---
description: >
  Old description from a previous release.
agent: gaze-reporter
---

<!-- scaffolded by gaze dev -->
<!-- scaffolded by gaze v1.8.0 -->

# Stale /gaze command body

This body predates the current embedded asset.
`
	if err := os.WriteFile(cmdPath, []byte(stale), 0o644); err != nil {
		t.Fatalf("pre-writing stale command file: %v", err)
	}

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	var updated bool
	for _, f := range result.Updated {
		if f == filepath.Join(".opencode", "commands", "gaze.md") {
			updated = true
		}
	}
	if !updated {
		t.Errorf("expected commands/gaze.md in Updated, got %v", result.Updated)
	}

	got, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Fatalf("reading healed command file: %v", err)
	}

	// Exactly one marker, carrying the current run's version.
	s := string(got)
	if count := strings.Count(s, "<!-- scaffolded by gaze"); count != 1 {
		t.Errorf("expected exactly 1 marker, got %d", count)
	}
	if !strings.Contains(s, "<!-- scaffolded by gaze v2.4.0 -->") {
		t.Errorf("expected current v2.4.0 marker, not found")
	}
	if strings.Contains(s, "gaze dev -->") {
		t.Errorf("old dev marker should have been healed away")
	}
	if strings.Contains(s, "gaze v1.8.0 -->") {
		t.Errorf("old v1.8.0 marker should have been healed away")
	}

	// The healed file is exactly the embedded asset rendered with the
	// current marker — no stale body content remains.
	embedded, err := assetContent("commands/gaze.md")
	if err != nil {
		t.Fatalf("reading embedded asset: %v", err)
	}
	want := insertMarkerAfterFrontmatter(embedded, versionMarker("v2.4.0"))
	if !bytes.Equal(got, want) {
		t.Errorf("healed command file does not match the embedded copy (got %d bytes, want %d bytes)",
			len(got), len(want))
	}
	if strings.Contains(s, "Stale /gaze command body") {
		t.Errorf("stale body content should not survive the update")
	}
}

// TestRun_GazeLocalEscapeHatch verifies the gaze-command-tool-owned
// "Custom command escape hatch" scenarios (design D2): the scaffold's
// asset walk only iterates embedded asset paths, so a customized
// .opencode/commands/gaze-local.md is user-owned by construction —
// byte-identical after every gaze init, with or without --force, and
// never reported in any Result list.
func TestRun_GazeLocalEscapeHatch(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	// Pre-create a customized escape-hatch command shaped like the
	// documented migration: a renamed, locally edited copy of the
	// scaffolded commands/gaze.md, old marker included — so the test
	// also pins that no marker-based sweep may touch the path.
	escapePath := filepath.Join(dir, ".opencode", "commands", "gaze-local.md")
	if err := os.MkdirAll(filepath.Dir(escapePath), 0o755); err != nil {
		t.Fatalf("creating commands directory: %v", err)
	}
	custom := `---
description: >
  Customized local /gaze command.
agent: gaze-reporter
---

<!-- scaffolded by gaze v1.8.0 -->

# Custom /gaze-local command

Renamed from the scaffolded commands/gaze.md and edited locally;
this copy must survive every gaze init, --force included.
`
	if err := os.WriteFile(escapePath, []byte(custom), 0o644); err != nil {
		t.Fatalf("pre-writing custom command file: %v", err)
	}

	// escapeSuffix matches the Result lists' display-path format
	// (filepath.Join(".opencode", relPath)) should the escape-hatch
	// path ever incorrectly appear in one.
	escapeSuffix := filepath.Join("commands", "gaze-local.md")

	// assertUntouched enforces both halves of the escape-hatch
	// contract for one Run: byte-identical file content, and no
	// Result list mentions the path.
	assertUntouched := func(label string, result *Result, before []byte) {
		after, err := os.ReadFile(escapePath)
		if err != nil {
			t.Fatalf("%s: reading escape-hatch file: %v", label, err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("%s: escape-hatch file content changed (got %d bytes, want %d bytes)",
				label, len(after), len(before))
		}
		for _, list := range []struct {
			name  string
			paths []string
		}{
			{"Created", result.Created},
			{"Skipped", result.Skipped},
			{"Overwritten", result.Overwritten},
			{"Updated", result.Updated},
			{"Removed", result.Removed},
		} {
			for _, f := range list.paths {
				if strings.HasSuffix(f, escapeSuffix) {
					t.Errorf("%s: escape-hatch file must not appear in Result.%s, got %v",
						label, list.name, list.paths)
				}
			}
		}
	}

	// First run without --force: the nine embedded assets are created
	// around the pre-existing escape-hatch file.
	before, err := os.ReadFile(escapePath)
	if err != nil {
		t.Fatalf("reading escape-hatch file before non-force run: %v", err)
	}
	var buf1 bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1})
	if err != nil {
		t.Fatalf("non-force Run() returned error: %v", err)
	}
	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d: %v", len(result.Created), result.Created)
	}
	assertUntouched("non-force run", result, before)

	// Second run with --force: all nine embedded assets are
	// overwritten; the escape hatch must survive force too.
	before, err = os.ReadFile(escapePath)
	if err != nil {
		t.Fatalf("reading escape-hatch file before force run: %v", err)
	}
	var buf2 bytes.Buffer
	result, err = Run(Options{TargetDir: dir, Version: "1.0.0", Force: true, Stdout: &buf2})
	if err != nil {
		t.Fatalf("force Run() returned error: %v", err)
	}
	if len(result.Overwritten) != 9 {
		t.Errorf("expected 9 overwritten files, got %d: %v", len(result.Overwritten), result.Overwritten)
	}
	assertUntouched("force run", result, before)
}

// TestRun_GazeCommandUpdateTip verifies the gaze-command-tool-owned
// "Escape-hatch guidance on update" requirement (design D3): when
// gaze init updates commands/gaze.md, the summary prints a one-line
// tip naming commands/gaze-local.md as the path that preserves a
// customized command. The tip must not print when the file is
// skipped as identical, created on a first run, or overwritten via
// --force — the update is the only outcome where the opt-out lesson
// is actionable. gaze-local.md is the stable assertion substring: no
// embedded asset path, summary line, or notice carries it, so its
// presence in stdout is attributable to the tip alone.
func TestRun_GazeCommandUpdateTip(t *testing.T) {
	const tipSubstring = "gaze-local.md"

	// gazeCmdDisplayPath is commands/gaze.md in the Result lists'
	// display-path form (filepath.Join(".opencode", relPath)).
	gazeCmdDisplayPath := filepath.Join(".opencode", "commands", "gaze.md")

	// inList reports whether the /gaze command's display path
	// appears in the given Result list.
	inList := func(list []string) bool {
		for _, f := range list {
			if f == gazeCmdDisplayPath {
				return true
			}
		}
		return false
	}

	// customizeGazeCommand overwrites the scaffolded /gaze command
	// with a local customization, so the next run sees a content diff
	// for that file (and only that file).
	customizeGazeCommand := func(t *testing.T, dir string) {
		t.Helper()
		cmdPath := filepath.Join(dir, ".opencode", "commands", "gaze.md")
		if err := os.WriteFile(cmdPath, []byte("# custom\n"), 0o644); err != nil {
			t.Fatalf("customizing commands/gaze.md: %v", err)
		}
	}

	t.Run("prints on update", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "")

		var buf1 bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1}); err != nil {
			t.Fatalf("first Run() returned error: %v", err)
		}

		customizeGazeCommand(t, dir)

		var buf2 bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf2})
		if err != nil {
			t.Fatalf("second Run() returned error: %v", err)
		}

		// Precondition: the customized command was updated, and it is
		// the only diff — the other eight assets are identical.
		if !inList(result.Updated) {
			t.Errorf("expected commands/gaze.md in Updated, got Created=%v Skipped=%v Overwritten=%v Updated=%v",
				result.Created, result.Skipped, result.Overwritten, result.Updated)
		}
		if len(result.Updated) != 1 {
			t.Errorf("expected exactly 1 updated file, got %d: %v", len(result.Updated), result.Updated)
		}

		// The tip prints exactly once — one line naming the
		// escape-hatch path.
		output := buf2.String()
		if n := strings.Count(output, tipSubstring); n != 1 {
			t.Errorf("expected exactly 1 escape-hatch tip naming %q, got %d, output:\n%s",
				tipSubstring, n, output)
		}
	})

	t.Run("silent on identical skip", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "")

		var buf1 bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1}); err != nil {
			t.Fatalf("first Run() returned error: %v", err)
		}

		var buf2 bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf2})
		if err != nil {
			t.Fatalf("second Run() returned error: %v", err)
		}

		// Precondition: nothing changed — gaze.md is skipped as
		// byte-identical alongside the other eight assets.
		if !inList(result.Skipped) {
			t.Errorf("expected commands/gaze.md in Skipped, got Created=%v Skipped=%v Overwritten=%v Updated=%v",
				result.Created, result.Skipped, result.Overwritten, result.Updated)
		}
		if len(result.Skipped) != 9 {
			t.Errorf("expected 9 skipped files, got %d: %v", len(result.Skipped), result.Skipped)
		}

		output := buf2.String()
		if strings.Contains(output, tipSubstring) {
			t.Errorf("escape-hatch tip must not print when commands/gaze.md is skipped as identical, got:\n%s", output)
		}
	})

	t.Run("silent on first-run creation", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "")

		var buf bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		// Precondition: gaze.md lands in Created, not Updated.
		if !inList(result.Created) {
			t.Errorf("expected commands/gaze.md in Created, got Created=%v Skipped=%v Overwritten=%v Updated=%v",
				result.Created, result.Skipped, result.Overwritten, result.Updated)
		}
		if len(result.Created) != 9 {
			t.Errorf("expected 9 created files, got %d: %v", len(result.Created), result.Created)
		}

		output := buf.String()
		if strings.Contains(output, tipSubstring) {
			t.Errorf("escape-hatch tip must not print on first-run creation, got:\n%s", output)
		}
	})

	t.Run("silent on force overwrite", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "")

		var buf1 bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf1}); err != nil {
			t.Fatalf("first Run() returned error: %v", err)
		}

		customizeGazeCommand(t, dir)

		var buf2 bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Force: true, Stdout: &buf2})
		if err != nil {
			t.Fatalf("second Run() with force returned error: %v", err)
		}

		// Precondition: the customized command was overwritten via
		// --force — explicitly requested — not updated.
		if !inList(result.Overwritten) {
			t.Errorf("expected commands/gaze.md in Overwritten, got Created=%v Skipped=%v Overwritten=%v Updated=%v",
				result.Created, result.Skipped, result.Overwritten, result.Updated)
		}
		if len(result.Overwritten) != 9 {
			t.Errorf("expected 9 overwritten files, got %d: %v", len(result.Overwritten), result.Overwritten)
		}

		output := buf2.String()
		if strings.Contains(output, tipSubstring) {
			t.Errorf("escape-hatch tip must not print when commands/gaze.md is overwritten via --force, got:\n%s", output)
		}
	})
}

// TestGazeReporterAgent_NoRenderingRules verifies FR-031: the
// interactive /gaze agent consumes the deterministic renderer and
// carries no rubric, emoji, or table templates of its own.
func TestGazeReporterAgent_NoRenderingRules(t *testing.T) {
	content, err := assetContent("agents/gaze-reporter.md")
	if err != nil {
		t.Fatalf("reading embedded agent asset: %v", err)
	}
	text := string(content)

	forbidden := []string{
		"Scoring Consistency Rules",
		"±",
		"\U0001F7E2", // green circle
		"\U0001F7E1", // yellow circle
		"\U0001F534", // red circle
		"\u26AA",     // white circle
		"Q1 — Safe",
		"Q2 — Complex But Tested",
		"Q3 — Needs Tests",
		"Q4 — Dangerous",
	}
	for _, phrase := range forbidden {
		if strings.Contains(text, phrase) {
			t.Errorf("agent prompt must not contain %q", phrase)
		}
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		for _, prefix := range []string{"| Metric |", "| Quadrant |", "| Dimension |", "| Grade |"} {
			if strings.HasPrefix(trimmed, prefix) {
				t.Errorf("agent prompt must not contain rendering table %q: %q", prefix, trimmed)
			}
		}
		if reGradeRow.MatchString(trimmed) {
			t.Errorf("agent prompt must not contain a grade table row: %q", trimmed)
		}
	}

	// Frontmatter must be unchanged: subagent mode and the same tools.
	for _, want := range []string{
		"mode: subagent",
		"  read: true",
		"  bash: true",
		"  write: false",
		"  edit: false",
		"  webfetch: false",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("agent prompt frontmatter must contain %q", want)
		}
	}

	// It must consume the deterministic renderer in all three modes.
	for _, want := range []string{
		"gaze report --format=markdown",
		"--sections=crap",
		"--sections=quality",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("agent prompt must reference %q", want)
		}
	}
}

// reGradeRow matches a markdown table row whose first cell is a grade.
var reGradeRow = regexp.MustCompile(`^\|\s*(A|A-|B\+|B|B-|C\+|C|C-|D|F)\s*\|`)

// TestExampleReportMatchesGolden verifies DE-007 Layer 2: the example
// report shipped with the agent is the renderer's own golden output, so
// the documented shape cannot drift from the implementation.
func TestExampleReportMatchesGolden(t *testing.T) {
	content, err := assetContent("references/example-report.md")
	if err != nil {
		t.Fatalf("reading embedded example report: %v", err)
	}
	text := string(content)

	if n := strings.Count(text, "```markdown"); n != 1 {
		t.Fatalf("expected exactly one ```markdown block, found %d", n)
	}
	const opener = "```markdown\n"
	start := strings.Index(text, opener)
	body := text[start+len(opener):]
	end := strings.Index(body, "\n```")
	if end < 0 {
		t.Fatal("markdown block is not closed")
	}
	got := body[:end+1]

	goldenPath := filepath.Join("..", "..", "internal", "aireport", "testdata", "golden", "report-full.md")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden report: %v", err)
	}
	if got != string(want) {
		t.Errorf("example report block does not match %s (got %d bytes, want %d bytes)",
			goldenPath, len(got), len(want))
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

// TestRun_ReScaffoldNonForceSingleMarker verifies the non-force
// re-init path from issue #279 after the gaze-command-tool-owned
// flip: every embedded .opencode/ asset is tool-owned, so a second
// gaze init with a new version and no --force updates every
// marker-bearing file on diff, each left with exactly one marker
// carrying the new version, while the marker-less dcp.jsonc is
// skipped as byte-identical. No embedded asset takes the user-owned
// skip path anymore; the untouched-user-file scenario now belongs to
// the escape-hatch test (TestRun_GazeLocalEscapeHatch).
func TestRun_ReScaffoldNonForceSingleMarker(t *testing.T) {
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

	// Second run with version "v1.8.0" and no --force: every
	// embedded asset takes the tool-owned path. The eight Markdown
	// files are updated on diff (replacing the old marker); the
	// marker-less dcp.jsonc is skipped as byte-identical.
	var buf2 bytes.Buffer
	result, err := Run(Options{
		TargetDir: dir,
		Version:   "v1.8.0",
		Stdout:    &buf2,
	})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}

	paths, err := assetPaths()
	if err != nil {
		t.Fatalf("assetPaths() returned error: %v", err)
	}

	// Every Markdown file must have exactly one marker with the
	// new version, and zero markers with the old version.
	mdChecked := 0
	for _, relPath := range paths {
		// Non-Markdown files skip the HTML marker (it would be
		// invalid syntax in formats like JSONC); dcp.jsonc is
		// covered by the Result assertions below.
		if !strings.HasSuffix(relPath, ".md") {
			continue
		}
		mdChecked++
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
	if mdChecked != 8 {
		t.Errorf("expected to verify all 8 marker-bearing Markdown assets, checked %d", mdChecked)
	}

	// All nine embedded assets took the tool-owned path on the
	// non-force re-run: the eight Markdown files were updated on
	// diff, commands/gaze.md included (the pre-flip user-owned
	// exception), and the byte-identical dcp.jsonc was skipped.
	if len(result.Updated) != 8 {
		t.Errorf("expected 8 updated files, got %d: %v", len(result.Updated), result.Updated)
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped file, got %d: %v", len(result.Skipped), result.Skipped)
	}
	updatedMap := make(map[string]bool)
	for _, f := range result.Updated {
		updatedMap[f] = true
	}
	if !updatedMap[filepath.Join(".opencode", "commands", "gaze.md")] {
		t.Errorf("expected commands/gaze.md in updated list, got %v", result.Updated)
	}
	skippedMap := make(map[string]bool)
	for _, f := range result.Skipped {
		skippedMap[f] = true
	}
	if !skippedMap[filepath.Join(".opencode", "dcp.jsonc")] {
		t.Errorf("expected dcp.jsonc in skipped list, got %v", result.Skipped)
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
