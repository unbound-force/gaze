// Package apidoc provides API documentation coverage analysis by
// cross-referencing analyzer output against documentation files.
// It supports two modes: native doc_coverage (from analyzers that
// implement the optional doc_coverage protocol method) and heuristic
// fallback (scanning Markdown for backtick-quoted symbol references).
package apidoc

import "github.com/unbound-force/gaze/internal/protocol"

// AnalyzerData is the input struct that carries analyzer output
// into the coverage computation. DocCoverage is non-nil when the
// analyzer supports the native doc_coverage protocol method.
type AnalyzerData struct {
	// Functions is the list of analyzed functions from the
	// analyzer's "analyze" method.
	Functions []protocol.AnalyzedFunction `json:"functions"`

	// DocCoverage is the native documentation coverage result
	// from the analyzer's "doc_coverage" method. Nil when the
	// analyzer does not support this optional method.
	DocCoverage *protocol.DocCoverageResult `json:"doc_coverage"`

	// Language is the primary language the analyzer targets
	// (e.g., "python", "go"). Used for code block validation.
	Language string `json:"language"`
}

// APICoverageReport is the output struct containing the full
// documentation coverage analysis results, including coverage
// statistics, undocumented symbols, stale references, and code
// block issues.
type APICoverageReport struct {
	// TotalSymbols is the total number of public symbols analyzed.
	TotalSymbols int `json:"total_symbols"`

	// DocumentedSymbols is the number of symbols with documentation.
	DocumentedSymbols int `json:"documented_symbols"`

	// CoveragePercent is the documentation coverage percentage
	// (documented / total * 100). Zero when TotalSymbols is zero.
	CoveragePercent float64 `json:"coverage_percent"`

	// Source indicates how coverage was computed: "doc_coverage"
	// for native analyzer data, "heuristic" for backtick-based
	// Markdown scanning.
	Source string `json:"source"`

	// Undocumented lists the symbols that lack documentation.
	Undocumented []SymbolCoverage `json:"undocumented"`

	// StaleReferences lists Markdown references to symbols that
	// do not exist in the analyzer output.
	StaleReferences []StaleReference `json:"stale_references"`

	// CodeBlockIssues lists fenced code blocks with language tags
	// that do not match the analyzer's declared language.
	CodeBlockIssues []CodeBlockIssue `json:"code_block_issues"`
}

// SymbolCoverage represents a single undocumented symbol with its
// source location and kind.
type SymbolCoverage struct {
	// Name is the symbol name.
	Name string `json:"name"`

	// Package is the package or module path containing the symbol.
	Package string `json:"package"`

	// File is the source file path where the symbol is declared.
	File string `json:"file"`

	// Line is the line number of the symbol declaration.
	Line int `json:"line"`

	// Kind is the symbol kind (e.g., "function", "type",
	// "constant", "variable", "class", "method").
	Kind string `json:"kind"`
}

// StaleReference represents a backtick-quoted reference in a
// Markdown file that does not correspond to any known symbol
// in the analyzer output.
type StaleReference struct {
	// Symbol is the referenced symbol name found in the Markdown.
	Symbol string `json:"symbol"`

	// DocFile is the path to the Markdown file containing the
	// stale reference.
	DocFile string `json:"doc_file"`

	// DocLine is the line number of the stale reference in the
	// Markdown file.
	DocLine int `json:"doc_line"`
}

// CodeBlockIssue represents a fenced code block in a Markdown file
// whose language tag does not match the analyzer's declared language.
type CodeBlockIssue struct {
	// DocFile is the path to the Markdown file containing the
	// code block.
	DocFile string `json:"doc_file"`

	// DocLine is the line number of the opening fence.
	DocLine int `json:"doc_line"`

	// DeclaredLang is the language tag on the code fence.
	DeclaredLang string `json:"declared_lang"`

	// ExpectedLang is the language reported by the analyzer.
	ExpectedLang string `json:"expected_lang"`
}

// CoverageResult is the intermediate result returned by
// ComputeCoverage, containing coverage statistics and the list
// of undocumented symbols.
type CoverageResult struct {
	// Undocumented lists the symbols that lack documentation.
	Undocumented []SymbolCoverage `json:"undocumented"`

	// Total is the total number of public symbols analyzed.
	Total int `json:"total"`

	// Documented is the number of symbols with documentation.
	Documented int `json:"documented"`

	// Source indicates how coverage was computed: "doc_coverage"
	// or "heuristic".
	Source string `json:"source"`
}
