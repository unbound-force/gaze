// Package loader wraps go/packages to load Go packages with full
// type information for static analysis.
package loader

import (
	"fmt"
	"go/token"
	"strings"

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
