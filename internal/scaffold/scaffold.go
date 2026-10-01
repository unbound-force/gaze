// Package scaffold embeds distributable OpenCode agent and command
// files and writes them to a target project directory.
package scaffold

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed assets
var assets embed.FS

// Options configures the scaffold operation.
type Options struct {
	// TargetDir is the root directory to scaffold into.
	// Defaults to the current working directory.
	TargetDir string

	// Force overwrites existing files when true.
	// When false, existing files are skipped.
	Force bool

	// Version is the gaze version string to embed in the
	// version marker comment. Set by ldflags at build time.
	// Defaults to "dev" for development builds.
	Version string

	// Stdout is the writer for summary output.
	// Defaults to os.Stdout.
	Stdout io.Writer
}

// Result reports what the scaffold operation did.
type Result struct {
	// Created lists files that were written for the first time.
	Created []string

	// Skipped lists files that already existed and were not
	// overwritten (Force was false).
	Skipped []string

	// Overwritten lists files that existed and were replaced
	// (Force was true).
	Overwritten []string

	// Updated lists tool-owned files that existed with different
	// content and were overwritten via overwrite-on-diff (Force
	// was false, but content differed from the embedded version).
	Updated []string
}

// isToolOwned reports whether the given relative path (under
// .opencode/) is a tool-owned file. Tool-owned files use
// overwrite-on-diff behavior: they are replaced when their
// content differs from the embedded version, even without --force.
// User-owned files (agents/, most of commands/) retain
// skip-if-present behavior.
//
// Ownership is determined by an explicit list: all files under
// references/ are tool-owned by directory convention, and specific
// command files are tool-owned by exact match. This approach is
// necessary because the commands/ directory contains both
// user-owned files (gaze.md) and tool-owned files
// (speckit.testreview.md).
func isToolOwned(relPath string) bool {
	if strings.HasPrefix(relPath, "references/") {
		return true
	}
	switch relPath {
	case "commands/speckit.testreview.md",
		"agents/gaze-test-generator.md",
		"commands/gaze-fix.md",
		"dcp.jsonc":
		return true
	}
	return false
}

// versionMarker returns the version marker comment to embed in
// scaffolded Markdown files. Non-Markdown files skip the marker
// (see processAssetFile).
func versionMarker(version string) string {
	if version == "" {
		version = "dev"
	}
	return fmt.Sprintf("<!-- scaffolded by gaze %s -->\n", version)
}

// removeExistingMarker strips any "<!-- scaffolded by gaze ... -->"
// line from content. This prevents marker duplication when
// re-scaffolding with a different version (issue #279).
func removeExistingMarker(content []byte) []byte {
	const prefix = "<!-- scaffolded by gaze "
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) && strings.HasSuffix(trimmed, "-->") {
			continue
		}
		out = append(out, line)
	}
	return []byte(strings.Join(out, "\n"))
}

// insertMarkerAfterFrontmatter inserts the version marker after
// the YAML frontmatter closing delimiter (---). If no frontmatter
// is found, the marker is appended to the end of the content.
//
// Any existing "<!-- scaffolded by gaze ... -->" marker is removed
// first to prevent duplication when re-scaffolding with a different
// version.
//
// YAML frontmatter must start with "---\n" on the very first line
// and end with "\n---\n". Prepending a marker before the opening
// "---" breaks frontmatter parsing in tools like OpenCode, which
// causes agent tool configurations and command delegation to be
// silently ignored.
//
// Constraint: YAML frontmatter content MUST NOT contain a line
// that is exactly "---". If it does, the first such line will be
// incorrectly treated as the closing delimiter. All current
// embedded assets satisfy this constraint.
func insertMarkerAfterFrontmatter(content []byte, marker string) []byte {
	// Strip any existing version marker to avoid duplication when
	// re-scaffolding with a different version (issue #279).
	content = removeExistingMarker(content)
	s := string(content)

	// Check for YAML frontmatter: must start with "---\n".
	if !strings.HasPrefix(s, "---\n") {
		// No frontmatter — append marker at the end.
		return append(content, []byte(marker)...)
	}

	// Find the closing "---" delimiter (skip the opening one).
	closeIdx := strings.Index(s[4:], "\n---\n")
	if closeIdx < 0 {
		// Malformed frontmatter — append marker at the end.
		return append(content, []byte(marker)...)
	}

	// Insert marker after the closing "---\n".
	// closeIdx is relative to s[4:], so the absolute position of
	// the closing "\n---\n" is closeIdx+4. The line ends at
	// closeIdx+4+len("\n---\n") = closeIdx+8.
	insertAt := closeIdx + 4 + len("\n---\n")
	out := make([]byte, 0, len(content)+len(marker))
	out = append(out, content[:insertAt]...)
	out = append(out, []byte(marker)...)
	out = append(out, content[insertAt:]...)
	return out
}

// applyDefaults sets zero-valued Options fields to their defaults.
func applyDefaults(opts *Options) error {
	if opts.TargetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		opts.TargetDir = cwd
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	return nil
}

// handleToolOwnedFile compares the existing file at outPath with the
// new content. If they differ, the file is overwritten. Returns an
// action string ("updated" or "skipped") and any error.
func handleToolOwnedFile(outPath string, content []byte, displayPath string) (string, error) {
	existing, err := os.ReadFile(outPath)
	if err != nil {
		return "", fmt.Errorf("reading existing %s: %w", displayPath, err)
	}
	if bytes.Equal(existing, content) {
		return "skipped", nil
	}
	// Content differs — overwrite. Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory for %s: %w", displayPath, err)
	}
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return "", fmt.Errorf("updating %s: %w", displayPath, err)
	}
	return "updated", nil
}

// writeNewFile creates parent directories and writes content to
// outPath. Returns an action string ("overwritten" or "created")
// based on whether the file previously existed.
func writeNewFile(outPath string, content []byte, exists bool, displayPath string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return "", fmt.Errorf("creating directory %s: %w", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, content, 0o644); err != nil {
		return "", fmt.Errorf("creating %s: %w", displayPath, err)
	}
	if exists {
		return "overwritten", nil
	}
	return "created", nil
}

// processAssetFile handles a single embedded asset: checks existence,
// reads content, inserts the version marker for Markdown files (non-
// Markdown files skip the marker), and writes or skips based on
// force/tool-ownership semantics. Returns the action taken
// ("created", "overwritten", "updated", "skipped") or an error.
func processAssetFile(embeddedPath, relPath string, opts Options, marker string) (string, error) {
	outPath := filepath.Join(opts.TargetDir, ".opencode", relPath)
	displayPath := filepath.Join(".opencode", relPath)

	_, statErr := os.Stat(outPath)
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return "", fmt.Errorf("checking %s: %w", displayPath, statErr)
	}
	exists := statErr == nil

	content, err := assets.ReadFile(embeddedPath)
	if err != nil {
		return "", fmt.Errorf("reading embedded asset %s: %w", embeddedPath, err)
	}

	// Skip the HTML version marker for non-Markdown files.
	// The marker (<!-- scaffolded by gaze vX.Y.Z -->) is an HTML
	// comment that is invalid in formats like JSONC.
	var out []byte
	if strings.HasSuffix(relPath, ".md") {
		out = insertMarkerAfterFrontmatter(content, marker)
	} else {
		out = content
	}

	if exists && !opts.Force {
		if isToolOwned(relPath) {
			return handleToolOwnedFile(outPath, out, displayPath)
		}
		return "skipped", nil
	}

	return writeNewFile(outPath, out, exists, displayPath)
}

// Run scaffolds OpenCode agent, command, reference, and configuration
// files into the target directory. It creates .opencode/agents/,
// .opencode/commands/, and .opencode/references/ subdirectories
// and writes the embedded quality-reporting files and DCP config.
//
// Markdown files are prepended with a version marker comment:
//
//	<!-- scaffolded by gaze vX.Y.Z -->
//
// Non-Markdown files (e.g., dcp.jsonc) skip the marker because the
// HTML comment syntax is invalid in those formats.
//
// Files are classified as user-owned or tool-owned via
// isToolOwned(). If a user-owned file already exists
// and opts.Force is false, the file is skipped. Tool-owned files
// use overwrite-on-diff: they are replaced when their content
// differs from the embedded version, even without --force. If
// opts.Force is true, all files are overwritten regardless of
// ownership.
//
// Run returns a Result summarizing what was created, skipped,
// overwritten, or updated.
func Run(opts Options) (*Result, error) {
	if err := applyDefaults(&opts); err != nil {
		return nil, err
	}

	// Check for go.mod and warn if absent.
	goModPath := filepath.Join(opts.TargetDir, "go.mod")
	if _, err := os.Stat(goModPath); errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintln(opts.Stdout, "Warning: no go.mod found in current directory.")
		_, _ = fmt.Fprintln(opts.Stdout, "Gaze works best in a Go module root.")
		_, _ = fmt.Fprintln(opts.Stdout)
	}

	result := &Result{}
	marker := versionMarker(opts.Version)

	err := fs.WalkDir(assets, "assets", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath := strings.TrimPrefix(p, "assets/")
		action, err := processAssetFile(p, relPath, opts, marker)
		if err != nil {
			return err
		}

		displayPath := filepath.Join(".opencode", relPath)
		switch action {
		case "created":
			result.Created = append(result.Created, displayPath)
		case "overwritten":
			result.Overwritten = append(result.Overwritten, displayPath)
		case "updated":
			result.Updated = append(result.Updated, displayPath)
		case "skipped":
			result.Skipped = append(result.Skipped, displayPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	printSummary(opts.Stdout, result)
	return result, nil
}

// printSummary writes a human-readable summary of the scaffold
// operation to w.
func printSummary(w io.Writer, r *Result) {
	if len(r.Created) > 0 || len(r.Overwritten) > 0 || len(r.Updated) > 0 {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration initialized:")
	} else {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration already up to date:")
	}

	for _, f := range r.Created {
		_, _ = fmt.Fprintf(w, "  created: %s\n", f)
	}
	for _, f := range r.Skipped {
		_, _ = fmt.Fprintf(w, "  skipped: %s (already exists)\n", f)
	}
	for _, f := range r.Overwritten {
		_, _ = fmt.Fprintf(w, "  overwritten: %s\n", f)
	}
	for _, f := range r.Updated {
		_, _ = fmt.Fprintf(w, "  updated: %s (content changed)\n", f)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run /gaze for quality reports and /speckit.testreview for testability analysis.")

	// Count only user-owned skipped files for the --force hint.
	// Tool-owned reference files that are skipped (identical content)
	// do not need --force — they auto-update when content changes.
	var userSkipped int
	for _, f := range r.Skipped {
		rel := strings.TrimPrefix(f, filepath.Join(".opencode")+string(filepath.Separator))
		if !isToolOwned(rel) {
			userSkipped++
		}
	}
	if userSkipped > 0 {
		word := "file"
		if userSkipped > 1 {
			word = "files"
		}
		_, _ = fmt.Fprintf(w, "%d %s skipped (use --force to overwrite).\n", userSkipped, word)
	}
}

// assetPaths returns the relative paths of all embedded assets.
// This is used by the drift detection test to enumerate expected
// files.
func assetPaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, strings.TrimPrefix(path, "assets/"))
		return nil
	})
	return paths, err
}

// assetContent returns the raw content of an embedded asset by
// its relative path (e.g., "agents/gaze-reporter.md").
func assetContent(relPath string) ([]byte, error) {
	return assets.ReadFile(path.Join("assets", relPath))
}
