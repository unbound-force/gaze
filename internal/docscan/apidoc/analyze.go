package apidoc

import (
	"strings"

	"github.com/unbound-force/gaze/internal/docscan"
)

// Analyze orchestrates the three API documentation analysis sub-functions:
// ComputeCoverage, ValidateReferences, and ValidateCodeBlocks. It assembles
// their results into a single APICoverageReport.
//
// When data is nil, Analyze returns (nil, nil) — a no-op indicating that
// no analyzer data is available for documentation analysis.
func Analyze(docs []docscan.DocumentFile, data *AnalyzerData) (*APICoverageReport, error) {
	if data == nil {
		return nil, nil
	}

	coverage, err := ComputeCoverage(data, docs)
	if err != nil {
		return nil, err
	}

	symbolNames := buildSymbolNames(data)

	staleRefs := ValidateReferences(docs, symbolNames)
	codeBlockIssues := ValidateCodeBlocks(docs, data.Language)

	// Compute coverage percentage, guarding against division by zero.
	var coveragePct float64
	if coverage.Total > 0 {
		coveragePct = float64(coverage.Documented) / float64(coverage.Total) * 100.0
	}

	return &APICoverageReport{
		TotalSymbols:      coverage.Total,
		DocumentedSymbols: coverage.Documented,
		CoveragePercent:   coveragePct,
		Source:            coverage.Source,
		Undocumented:      coverage.Undocumented,
		StaleReferences:   staleRefs,
		CodeBlockIssues:   codeBlockIssues,
	}, nil
}

// buildSymbolNames constructs the symbol name lookup set used by
// ValidateReferences. Each symbol is added in both unqualified
// ("ProcessData") and qualified ("pkg.ProcessData") forms so that
// documentation references using either style are recognized.
//
// Native path (DocCoverage != nil): names come from SymbolDocStatus entries.
// Heuristic path: names come from AnalyzedFunction entries, with the
// qualified form using the last segment of the package path
// (e.g., "math_utils" from "some/path/math_utils").
func buildSymbolNames(data *AnalyzerData) map[string]bool {
	names := make(map[string]bool)

	if data.DocCoverage != nil {
		for _, sym := range data.DocCoverage.Symbols {
			names[sym.Name] = true
			if sym.Package != "" {
				names[sym.Package+"."+sym.Name] = true
			}
		}
		return names
	}

	// Heuristic path: use Functions list.
	for _, fn := range data.Functions {
		names[fn.Name] = true
		if fn.Package != "" {
			shortPkg := fn.Package
			if idx := strings.LastIndex(fn.Package, "/"); idx >= 0 {
				shortPkg = fn.Package[idx+1:]
			}
			names[shortPkg+"."+fn.Name] = true
		}
	}

	return names
}
