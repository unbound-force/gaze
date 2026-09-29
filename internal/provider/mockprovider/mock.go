// Package mockprovider provides mock implementations of the provider
// interfaces defined in internal/crap/provider.go. These mocks enable
// unit testing of the universal scoring core (crap.Analyze,
// computeScores, buildSummary) with synthetic data, without requiring
// Go-specific analysis tooling.
package mockprovider

import (
	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// MockComplexityProvider implements crap.ComplexityProvider with
// configurable return values. The Analyze method returns the
// pre-configured Results and Err.
type MockComplexityProvider struct {
	// Results is the slice of FunctionComplexity values to return.
	Results []crap.FunctionComplexity

	// Err is the error to return. When non-nil, Results is ignored.
	Err error
}

// Ensure MockComplexityProvider satisfies crap.ComplexityProvider.
var _ crap.ComplexityProvider = (*MockComplexityProvider)(nil)

// Analyze returns the pre-configured Results and Err, ignoring the
// patterns and rootDir arguments.
func (m *MockComplexityProvider) Analyze(_ []string, _ string) ([]crap.FunctionComplexity, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Results, nil
}

// MockLineCoverageProvider implements crap.LineCoverageProvider with
// configurable return values. The Coverage method returns the
// pre-configured Results and Err.
type MockLineCoverageProvider struct {
	// Results is the slice of FuncCoverage values to return.
	Results []crap.FuncCoverage

	// Err is the error to return. When non-nil, Results is ignored.
	Err error
}

// Ensure MockLineCoverageProvider satisfies crap.LineCoverageProvider.
var _ crap.LineCoverageProvider = (*MockLineCoverageProvider)(nil)

// Coverage returns the pre-configured Results and Err, ignoring the
// patterns, rootDir, and coverProfile arguments.
func (m *MockLineCoverageProvider) Coverage(_ []string, _ string, _ string) ([]crap.FuncCoverage, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Results, nil
}

// MockSideEffectAnalyzer implements crap.SideEffectAnalyzer with
// configurable return values. The Analyze method returns the
// pre-configured Results and Err.
type MockSideEffectAnalyzer struct {
	// Results is the slice of AnalysisResult values to return.
	Results []taxonomy.AnalysisResult

	// Err is the error to return. When non-nil, Results is ignored.
	Err error
}

// Ensure MockSideEffectAnalyzer satisfies crap.SideEffectAnalyzer.
var _ crap.SideEffectAnalyzer = (*MockSideEffectAnalyzer)(nil)

// Analyze returns the pre-configured Results and Err, ignoring the
// pkgPath argument.
func (m *MockSideEffectAnalyzer) Analyze(_ string) ([]taxonomy.AnalysisResult, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Results, nil
}

// MockContractCoverageProvider implements
// crap.ContractCoverageProvider with configurable return values. The
// Build method returns the pre-configured LookupFunc,
// DegradedPackages, and Err.
type MockContractCoverageProvider struct {
	// LookupFunc is the contract coverage lookup function to return.
	// When nil and Err is nil, Build returns (nil, DegradedPackages, nil).
	LookupFunc func(pkg, function string) (crap.ContractCoverageInfo, bool)

	// DegradedPackages is the list of degraded package paths to return.
	DegradedPackages []string

	// Err is the error to return. When non-nil, LookupFunc and
	// DegradedPackages are ignored.
	Err error

	// BuildCalled tracks whether Build was called. Useful for
	// provider precedence tests.
	BuildCalled bool
}

// Ensure MockContractCoverageProvider satisfies
// crap.ContractCoverageProvider.
var _ crap.ContractCoverageProvider = (*MockContractCoverageProvider)(nil)

// Build returns the pre-configured LookupFunc, DegradedPackages, and
// Err, ignoring the patterns and rootDir arguments. Sets BuildCalled
// to true.
func (m *MockContractCoverageProvider) Build(_ []string, _ string) (func(pkg, function string) (crap.ContractCoverageInfo, bool), []string, error) {
	m.BuildCalled = true
	if m.Err != nil {
		return nil, nil, m.Err
	}
	return m.LookupFunc, m.DegradedPackages, nil
}
