// Package goprovider provides Go-specific implementations of the
// provider interfaces defined in internal/crap/provider.go. These
// adapters wrap existing Go analysis tooling (gocyclo, go test
// -coverprofile, analysis.LoadAndAnalyze, classify.Classify) behind
// language-neutral interfaces, enabling the universal scoring core
// to operate without direct Go-specific imports.
package goprovider

import (
	"regexp"

	"github.com/fzipp/gocyclo"
	"github.com/unbound-force/gaze/v2/internal/crap"
)

// testFileRegexp matches Go test files by suffix. Moved from
// crap/analyze.go per design decision D9 — it is specific to Go
// complexity analysis and not needed by the universal scoring core.
var testFileRegexp = regexp.MustCompile(`_test\.go$`)

// GoComplexityProvider implements crap.ComplexityProvider by wrapping
// gocyclo.Analyze(). It converts []gocyclo.Stat to
// []crap.FunctionComplexity, removing the go/token dependency from
// the scoring pipeline.
type GoComplexityProvider struct{}

// NewComplexityProvider creates a new GoComplexityProvider.
func NewComplexityProvider() *GoComplexityProvider {
	return &GoComplexityProvider{}
}

// Analyze computes cyclomatic complexity for all functions in the
// packages matched by patterns, rooted at rootDir. Uses
// crap.ResolvePatterns to convert Go package patterns to filesystem
// paths, then delegates to gocyclo.Analyze.
func (p *GoComplexityProvider) Analyze(patterns []string, rootDir string) ([]crap.FunctionComplexity, error) {
	absPaths, err := crap.ResolvePatterns(patterns, rootDir)
	if err != nil {
		return nil, err
	}

	stats := gocyclo.Analyze(absPaths, testFileRegexp)

	// Convert gocyclo.Stat → crap.FunctionComplexity (D4).
	// Field mapping: PkgName→Package, FuncName→Function,
	// Pos.Filename→File, Pos.Line→Line, Complexity→Complexity.
	result := make([]crap.FunctionComplexity, len(stats))
	for i, stat := range stats {
		result[i] = crap.FunctionComplexity{
			Package:    stat.PkgName,
			Function:   stat.FuncName,
			File:       stat.Pos.Filename,
			Line:       stat.Pos.Line,
			Complexity: stat.Complexity,
		}
	}

	return result, nil
}
