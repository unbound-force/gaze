// Package scaffold embeds distributable OpenCode agent and command
// files and writes them to a target project directory.
package scaffold

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets/*
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
// If a file already exists and opts.Force is false, the file is
// skipped. If opts.Force is true, the file is overwritten.
//
// Run returns a Result summarizing what was created, skipped, or
// overwritten.
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
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		fmt.Fprintln(opts.Stdout, "Warning: no go.mod found in current directory.")
		fmt.Fprintln(opts.Stdout, "Gaze works best in a Go module root.")
		fmt.Fprintln(opts.Stdout)
	}

	result := &Result{}
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

		// Check if the file already exists.
		_, statErr := os.Stat(outPath)
		exists := statErr == nil

		if exists && !opts.Force {
			result.Skipped = append(result.Skipped, filepath.Join(".opencode", relPath))
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
			return fmt.Errorf("creating %s: %w", filepath.Join(".opencode", relPath), err)
		}

		if exists {
			result.Overwritten = append(result.Overwritten, filepath.Join(".opencode", relPath))
		} else {
			result.Created = append(result.Created, filepath.Join(".opencode", relPath))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Print summary.
	printSummary(opts.Stdout, result, opts.Force)

	return result, nil
}

// printSummary writes a human-readable summary of the scaffold
// operation to w.
func printSummary(w io.Writer, r *Result, force bool) {
	fmt.Fprintln(w, "Gaze OpenCode integration initialized:")

	for _, f := range r.Created {
		fmt.Fprintf(w, "  created: %s\n", f)
	}
	for _, f := range r.Skipped {
		fmt.Fprintf(w, "  skipped: %s (already exists)\n", f)
	}
	for _, f := range r.Overwritten {
		fmt.Fprintf(w, "  overwritten: %s\n", f)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run /gaze in OpenCode to generate quality reports.")

	if len(r.Skipped) > 0 {
		fmt.Fprintf(w, "%d file(s) skipped (use --force to overwrite).\n", len(r.Skipped))
	}
}

// AssetPaths returns the relative paths of all embedded assets.
// This is used by the drift detection test to enumerate expected
// files.
func AssetPaths() ([]string, error) {
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

// AssetContent returns the raw content of an embedded asset by
// its relative path (e.g., "agents/gaze-reporter.md").
func AssetContent(relPath string) ([]byte, error) {
	return assets.ReadFile("assets/" + relPath)
}
