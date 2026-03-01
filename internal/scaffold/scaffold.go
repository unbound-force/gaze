// Package scaffold embeds distributable OpenCode agent and command
// files and writes them to a target project directory.
package scaffold

import (
	"bufio"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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
	Created []string `json:"created,omitempty"`

	// Updated lists files that existed with an outdated version
	// marker and were replaced with current content.
	Updated []string `json:"updated,omitempty"`

	// UpToDate lists files that existed with a current version
	// marker and were left untouched.
	UpToDate []string `json:"up_to_date,omitempty"`

	// Overwritten lists files that existed and were unconditionally
	// replaced (Force was true).
	Overwritten []string `json:"overwritten,omitempty"`

	// UpdatedFrom maps each updated file's relative path to the
	// previous version string extracted from its on-disk marker.
	// Empty string indicates the old file had no recognizable marker.
	UpdatedFrom map[string]string `json:"updated_from,omitempty"`
}

// markerPrefix is the fixed prefix of the version marker comment.
const markerPrefix = "<!-- scaffolded by gaze "

// markerSuffix is the fixed suffix of the version marker comment.
const markerSuffix = " -->"

// extractVersion reads the first line of the file at path and
// extracts the version string from the version marker comment.
// It returns the version string (e.g., "v1.0.0", "dev") or an
// empty string if the file is empty, unreadable, or does not
// contain a recognizable version marker on its first line.
func extractVersion(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "" // empty file or read error
	}
	line := scanner.Text()

	if !strings.HasPrefix(line, markerPrefix) || !strings.HasSuffix(line, markerSuffix) {
		return ""
	}

	version := strings.TrimPrefix(line, markerPrefix)
	version = strings.TrimSuffix(version, markerSuffix)
	return version
}

// versionMarker returns the version marker comment to prepend to
// each scaffolded file.
func versionMarker(version string) string {
	if version == "" {
		version = "dev"
	}
	return fmt.Sprintf("<!-- scaffolded by gaze %s -->\n", version)
}

// Run scaffolds OpenCode agent and command files into the target
// directory. It creates .opencode/agents/ and .opencode/command/
// subdirectories and writes the embedded quality-reporting files.
//
// Each file is prepended with a version marker comment:
//
//	<!-- scaffolded by gaze vX.Y.Z -->
//
// Run uses version-aware update logic to determine each file's
// disposition:
//
//   - If a file does not exist, it is created.
//   - If a file exists and opts.Force is true, it is overwritten
//     unconditionally.
//   - If a file exists and the running version is "dev", it is
//     always updated (dev builds refresh all files).
//   - If a file exists and its version marker matches the running
//     version, it is left untouched (up to date).
//   - If a file exists and its version marker differs (or is
//     missing/unparseable), it is updated with current content.
//
// Run returns a Result summarizing what was created, updated,
// left up to date, or overwritten.
func Run(opts Options) (*Result, error) {
	if opts.TargetDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		opts.TargetDir = cwd
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}

	// Check for go.mod and warn if absent.
	goModPath := filepath.Join(opts.TargetDir, "go.mod")
	if _, err := os.Stat(goModPath); errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintln(opts.Stdout, "Warning: no go.mod found in current directory.")
		_, _ = fmt.Fprintln(opts.Stdout, "Gaze works best in a Go module root.")
		_, _ = fmt.Fprintln(opts.Stdout)
	}

	result := &Result{
		UpdatedFrom: make(map[string]string),
	}
	marker := versionMarker(opts.Version)

	// Walk the embedded assets directory and write each file.
	err := fs.WalkDir(assets, "assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// Strip the "assets/" prefix to get the relative path
		// under .opencode/.
		relPath := strings.TrimPrefix(path, "assets/")
		outPath := filepath.Join(opts.TargetDir, ".opencode", relPath)
		displayPath := filepath.Join(".opencode", relPath)

		// Check if the file already exists. Return an error for
		// stat failures other than "not exist" (e.g., permission
		// denied) rather than silently treating them as absent.
		_, statErr := os.Stat(outPath)
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return fmt.Errorf("checking %s: %w", displayPath, statErr)
		}
		exists := statErr == nil

		if exists && !opts.Force {
			// Version-aware update logic: compare the on-disk
			// version marker against the running version.
			oldVersion := extractVersion(outPath)

			// Dev builds always update all files (FR-008).
			// Otherwise, update only if the version differs.
			if opts.Version != "dev" && oldVersion == opts.Version {
				result.UpToDate = append(result.UpToDate, displayPath)
				return nil
			}

			// File is outdated (or marker is missing/unparseable).
			// Read embedded content and overwrite.
			content, readErr := assets.ReadFile(path)
			if readErr != nil {
				return fmt.Errorf("reading embedded asset %s: %w", path, readErr)
			}

			dir := filepath.Dir(outPath)
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return fmt.Errorf("creating directory %s: %w", dir, mkErr)
			}

			out := append([]byte(marker), content...)
			if writeErr := os.WriteFile(outPath, out, 0o644); writeErr != nil {
				return fmt.Errorf("writing %s: %w", displayPath, writeErr)
			}

			result.Updated = append(result.Updated, displayPath)
			result.UpdatedFrom[displayPath] = oldVersion
			return nil
		}

		// Read the embedded file content.
		content, err := assets.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading embedded asset %s: %w", path, err)
		}

		// Create parent directories.
		dir := filepath.Dir(outPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}

		// Prepend version marker and write.
		out := append([]byte(marker), content...)
		if err := os.WriteFile(outPath, out, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", displayPath, err)
		}

		if exists {
			result.Overwritten = append(result.Overwritten, displayPath)
		} else {
			result.Created = append(result.Created, displayPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Print summary.
	printSummary(opts.Stdout, result, opts.Version)

	return result, nil
}

// printSummary writes a human-readable summary of the scaffold
// operation to w. It lists each file with its disposition and
// prints a summary count footer.
func printSummary(w io.Writer, r *Result, version string) {
	// Header: "initialized" if any files were changed, "already up
	// to date" if nothing was modified.
	if len(r.Created) > 0 || len(r.Updated) > 0 || len(r.Overwritten) > 0 {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration initialized:")
	} else {
		_, _ = fmt.Fprintln(w, "Gaze OpenCode integration already up to date:")
	}

	// Per-file listing.
	for _, f := range r.Created {
		_, _ = fmt.Fprintf(w, "  created:     %s\n", f)
	}
	for _, f := range r.Updated {
		oldVer := r.UpdatedFrom[f]
		if oldVer == "" {
			oldVer = "(unknown)"
		}
		_, _ = fmt.Fprintf(w, "  updated:     %s (%s -> %s)\n", f, oldVer, version)
	}
	for _, f := range r.UpToDate {
		_, _ = fmt.Fprintf(w, "  up to date:  %s\n", f)
	}
	for _, f := range r.Overwritten {
		_, _ = fmt.Fprintf(w, "  overwritten: %s\n", f)
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Run /gaze in OpenCode to generate quality reports.")

	// Summary count footer: show non-zero categories.
	var parts []string
	if n := len(r.Created); n > 0 {
		parts = append(parts, fmt.Sprintf("%d created", n))
	}
	if n := len(r.Updated); n > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", n))
	}
	if n := len(r.UpToDate); n > 0 {
		parts = append(parts, fmt.Sprintf("%d up to date", n))
	}
	if n := len(r.Overwritten); n > 0 {
		parts = append(parts, fmt.Sprintf("%d overwritten", n))
	}
	if len(parts) > 0 {
		_, _ = fmt.Fprintf(w, "%s.\n", strings.Join(parts, ", "))
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
	return assets.ReadFile("assets/" + relPath)
}
