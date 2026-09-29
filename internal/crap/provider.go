// provider.go defines language-neutral provider interfaces for the
// CRAP scoring pipeline. These interfaces decouple language-specific
// data acquisition (complexity analysis, coverage profiling, side
// effect detection, contract coverage) from the universal scoring
// engine (Formula, ClassifyQuadrant, computeScores, buildSummary).
//
// Go-specific implementations live in internal/provider/goprovider/.
// This file intentionally imports only standard library types and
// internal/taxonomy — no go/ast, go/types, go/packages, go/ssa,
// or github.com/fzipp/gocyclo.
//
// Design decision D1: Interfaces are defined here (not in a separate
// internal/provider/ package) to avoid import cycles — the interfaces
// reference types (ContractCoverageInfo, FuncCoverage) that live in
// the crap package.
package crap

import "github.com/unbound-force/gaze/v2/internal/taxonomy"

// FunctionComplexity is a language-neutral representation of
// per-function cyclomatic complexity. It replaces gocyclo.Stat in
// the scoring pipeline to remove the dependency on go/token.
//
// Design decision D4: The conversion from gocyclo.Stat to
// FunctionComplexity happens inside GoComplexityProvider.Analyze().
type FunctionComplexity struct {
	// Package is the package name (e.g., "crap").
	Package string `json:"package"`

	// Function is the function or method name (e.g., "Analyze" or
	// "(*Store).Save").
	Function string `json:"function"`

	// File is the absolute filesystem path to the source file.
	File string `json:"file"`

	// Line is the line number of the function declaration.
	Line int `json:"line"`

	// Complexity is the cyclomatic complexity value.
	Complexity int `json:"complexity"`
}

// ComplexityProvider computes per-function cyclomatic complexity for
// the given package patterns. Implementations wrap language-specific
// complexity analyzers (e.g., gocyclo for Go).
type ComplexityProvider interface {
	// Analyze computes cyclomatic complexity for all functions in
	// the packages matched by patterns, rooted at rootDir.
	// Returns a slice of FunctionComplexity or an error.
	Analyze(patterns []string, rootDir string) ([]FunctionComplexity, error)
}

// LineCoverageProvider produces per-function line coverage data.
// Implementations wrap language-specific coverage tools (e.g.,
// go test -coverprofile for Go).
type LineCoverageProvider interface {
	// Coverage returns per-function line coverage for the packages
	// matched by patterns, rooted at rootDir. When coverProfile is
	// non-empty, the provider uses the pre-generated profile
	// instead of generating coverage data internally.
	Coverage(patterns []string, rootDir string, coverProfile string) ([]FuncCoverage, error)
}

// SideEffectAnalyzer detects and classifies side effects for
// functions in a single package. Implementations handle both
// detection and classification internally — the returned
// AnalysisResult entries have Classification already attached.
//
// Note: SideEffectAnalyzer is NOT a field on crap.Options. It is
// consumed only by ContractCoverageProvider implementations as a
// composition dependency (see design decision D5).
type SideEffectAnalyzer interface {
	// Analyze detects and classifies side effects for all functions
	// in the given package path. Returns classified results or an
	// error.
	Analyze(pkgPath string) ([]taxonomy.AnalysisResult, error)
}

// ContractCoverageProvider builds a contract coverage lookup
// function by orchestrating the quality pipeline (side effect
// analysis, classification, test loading, assertion mapping).
type ContractCoverageProvider interface {
	// Build runs the quality pipeline for the given package
	// patterns and returns:
	//   - A lookup function that returns ContractCoverageInfo for
	//     a given (pkg, function) pair, or (zero, false) if no
	//     quality data exists.
	//   - A list of package paths where SSA construction failed
	//     (degraded packages).
	//   - An error if the pipeline fails entirely.
	Build(patterns []string, rootDir string) (func(pkg, function string) (ContractCoverageInfo, bool), []string, error)
}
