// Package loader wraps go/packages to load Go packages with full
// type information for static analysis.
package loader

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"golang.org/x/tools/go/packages"
)

// LoadMode is the minimum set of flags needed for SSA-ready analysis.
const LoadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedTypesSizes

// Result holds the loaded package along with convenience accessors.
type Result struct {
	// Pkg is the loaded package.
	Pkg *packages.Package

	// Fset is the shared file set for position information.
	Fset *token.FileSet
}

// Load loads a Go package at the given import path or file pattern.
// It returns the loaded package result or an error if loading or
// type-checking fails.
func Load(pattern string) (*Result, error) {
	cfg := &packages.Config{
		Mode:  LoadMode,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, pattern)
	if err != nil {
		return nil, fmt.Errorf("loading package %q: %w", pattern, err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for pattern %q", pattern)
	}

	if len(pkgs) > 1 {
		log.Warn("pattern resolved to multiple packages, using only the first; use ResolvePackagePaths for multi-package analysis", "pattern", pattern, "count", len(pkgs))
	}

	pkg := pkgs[0]

	// Check for package-level errors (syntax, type errors, etc.).
	var errs []string
	for _, e := range pkg.Errors {
		errs = append(errs, e.Error())
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package %q has errors:\n  %s",
			pattern, strings.Join(errs, "\n  "))
	}

	return &Result{
		Pkg:  pkg,
		Fset: pkg.Fset,
	}, nil
}

// ModuleResult holds all packages loaded from a Go module.
type ModuleResult struct {
	// Packages is the list of all loaded packages in the module.
	Packages []*packages.Package

	// Fset is the shared file set for position information.
	Fset *token.FileSet
}

// LoadModule loads all packages in the Go module using the ./...
// pattern. This provides access to sibling packages for caller
// analysis. The dir parameter specifies the module root directory;
// if empty, the current directory is used.
//
// Returns a *ModuleResult containing the valid (error-free) packages
// and their shared FileSet, or an error if package loading fails or
// all packages have errors. Packages with individual errors are
// silently excluded from the result.
func LoadModule(dir string) (*ModuleResult, error) {
	cfg := &packages.Config{
		Mode:  LoadMode,
		Tests: false,
		Dir:   dir,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("loading module packages: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found in module")
	}

	// Collect only packages without errors.
	var valid []*packages.Package
	var fset *token.FileSet
	for _, pkg := range pkgs {
		if len(pkg.Errors) == 0 {
			valid = append(valid, pkg)
			if fset == nil {
				fset = pkg.Fset
			}
		}
	}

	if len(valid) == 0 {
		return nil, fmt.Errorf("all packages in module have errors")
	}

	return &ModuleResult{
		Packages: valid,
		Fset:     fset,
	}, nil
}

// FindModuleRoot walks up from startDir to find the nearest directory
// containing a go.mod file (the module root). Returns the directory
// path or an error if no go.mod is found in startDir or any parent
// directory up to the filesystem root.
func FindModuleRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found in %q or any parent directory", startDir)
		}
		dir = parent
	}
}

// ResolvePackagePaths resolves package patterns (including ./...
// wildcards) to individual fully-qualified package paths. It uses a
// lightweight NeedName load mode, deduplicates results, and filters
// out test-variant packages (those with a "_test" suffix).
func ResolvePackagePaths(patterns []string, moduleDir string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  moduleDir,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("resolving package patterns: %w", err)
	}

	pkgPaths := make([]string, 0, len(pkgs))
	seen := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || seen[pkg.PkgPath] || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}
		seen[pkg.PkgPath] = true
		pkgPaths = append(pkgPaths, pkg.PkgPath)
	}
	return pkgPaths, nil
}

// IsMainPkg checks if a package path resolves to package main.
// Uses a lightweight NeedName load mode for efficiency.
func IsMainPkg(pkgPath string) bool {
	cfg := &packages.Config{Mode: packages.NeedName}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil || len(pkgs) == 0 {
		return false
	}
	return pkgs[0].Name == "main"
}
