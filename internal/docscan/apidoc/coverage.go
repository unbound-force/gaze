package apidoc

import (
	"fmt"
	"strings"

	"github.com/unbound-force/gaze/internal/docscan"
)

// ComputeCoverage computes documentation coverage from analyzer output.
// When data.DocCoverage is non-nil (native path), it iterates the
// SymbolDocStatus entries. When nil (heuristic path), it scans Markdown
// content for backtick-quoted function name references.
func ComputeCoverage(data *AnalyzerData, docs []docscan.DocumentFile) (*CoverageResult, error) {
	if data == nil {
		return nil, fmt.Errorf("data must not be nil")
	}

	if data.DocCoverage != nil {
		return computeNative(data)
	}

	return computeHeuristic(data, docs)
}

// computeNative computes coverage from the analyzer's native
// doc_coverage response by iterating SymbolDocStatus entries.
func computeNative(data *AnalyzerData) (*CoverageResult, error) {
	result := &CoverageResult{
		Source: "doc_coverage",
	}

	for _, sym := range data.DocCoverage.Symbols {
		result.Total++
		if sym.Documented {
			result.Documented++
		} else {
			result.Undocumented = append(result.Undocumented, SymbolCoverage{
				Name:    sym.Name,
				Package: sym.Package,
				File:    sym.File,
				Line:    sym.Line,
				Kind:    sym.Kind,
			})
		}
	}

	return result, nil
}

// computeHeuristic derives documentation coverage by scanning
// Markdown content for backtick-quoted function name references.
func computeHeuristic(data *AnalyzerData, docs []docscan.DocumentFile) (*CoverageResult, error) {
	result := &CoverageResult{
		Source: "heuristic",
		Total:  len(data.Functions),
	}

	if len(data.Functions) == 0 {
		return result, nil
	}

	// Collect all backtick-quoted strings from all doc files.
	quotedNames := extractBacktickQuoted(docs)

	for _, fn := range data.Functions {
		if isDocumented(fn.Name, fn.Package, quotedNames) {
			result.Documented++
		} else {
			result.Undocumented = append(result.Undocumented, SymbolCoverage{
				Name:    fn.Name,
				Package: fn.Package,
				File:    fn.File,
				Line:    fn.Line,
				Kind:    "function",
			})
		}
	}

	return result, nil
}

// extractBacktickQuoted extracts all backtick-quoted strings from
// the content of the provided documentation files. Returns a set
// of unique quoted strings for O(1) lookup.
//
// Content inside fenced code blocks (triple-backtick regions) is
// skipped to avoid false matches from code examples. This mirrors
// the approach in ValidateReferences.
func extractBacktickQuoted(docs []docscan.DocumentFile) map[string]bool {
	quoted := make(map[string]bool)
	for _, doc := range docs {
		lines := strings.Split(doc.Content, "\n")
		inCodeBlock := false

		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Track fenced code block boundaries.
			if strings.HasPrefix(trimmed, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}

			if inCodeBlock {
				continue
			}

			// Use the shared backtickRe from validation.go to
			// find single-backtick-quoted content on this line.
			matches := backtickRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				quoted[m[1]] = true
			}
		}
	}
	return quoted
}

// isDocumented checks whether a function name appears in the set
// of backtick-quoted strings from documentation. Matches both
// unqualified ("ProcessData") and qualified ("pkg.ProcessData")
// forms. Matching is case-sensitive and exact — no partial matches.
func isDocumented(name, pkg string, quotedNames map[string]bool) bool {
	// Check unqualified name: `ProcessData`
	if quotedNames[name] {
		return true
	}

	// Check qualified name: `pkg.ProcessData`
	// Use the last segment of the package path for qualification.
	if pkg != "" {
		lastDot := strings.LastIndex(pkg, "/")
		shortPkg := pkg
		if lastDot >= 0 {
			shortPkg = pkg[lastDot+1:]
		}
		qualified := shortPkg + "." + name
		if quotedNames[qualified] {
			return true
		}
	}

	return false
}
