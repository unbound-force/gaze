// Package protocol implements a JSON-RPC 2.0 client for communicating
// with external analyzer binaries over stdin/stdout. The protocol
// defines 10 methods for gaze ↔ analyzer communication: 5 required
// (initialize, analyze, complexity, coverage, shutdown) and 5 optional
// (discover, test_mapping, classify_signals, analyze/stream, doc_coverage).
//
// Design decision D1: JSON-RPC 2.0 over stdin/stdout, consistent
// with the LSP transport model. No HTTP, no gRPC — subprocess
// stdin/stdout is simpler and works in sandboxed environments.
//
// Design decision D8: Standard JSON-RPC 2.0 message format with
// request/response/error types.
package protocol

import (
	"encoding/json"
	"fmt"
)

// Protocol method constants for the 10 methods defined in the protocol.
const (
	// Required methods — analyzer must implement all of these.
	MethodInitialize = "initialize"
	MethodAnalyze    = "analyze"
	MethodComplexity = "complexity"
	MethodCoverage   = "coverage"
	MethodShutdown   = "shutdown"

	// Optional methods — declared via capabilities in initialize response.
	MethodDiscover        = "discover"
	MethodTestMapping     = "test_mapping"
	MethodClassifySignals = "classify_signals"
	MethodAnalyzeStream   = "analyze/stream"
	MethodDocCoverage     = "doc_coverage"
)

// ProtocolVersion is the current protocol version. Included in the
// initialize handshake for compatibility checking.
const ProtocolVersion = "1.1.0"

// JSON-RPC 2.0 version string.
const jsonRPCVersion = "2.0"

// Request is a JSON-RPC 2.0 request message sent from gaze to the
// analyzer subprocess via stdin.
type Request struct {
	// JSONRPC is always "2.0".
	JSONRPC string `json:"jsonrpc"`

	// ID is a unique request identifier for response matching.
	ID int64 `json:"id"`

	// Method is the protocol method name.
	Method string `json:"method"`

	// Params is the method-specific parameters. Encoded as a JSON
	// object (map or struct). Nil for methods with no parameters
	// (e.g., shutdown).
	Params any `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response message read from the
// analyzer subprocess's stdout.
type Response struct {
	// JSONRPC is always "2.0".
	JSONRPC string `json:"jsonrpc"`

	// ID matches the request ID this response corresponds to.
	ID int64 `json:"id"`

	// Result is the method-specific result payload. Nil when Error
	// is non-nil. Stored as raw JSON for deferred unmarshaling
	// into method-specific result types.
	Result json.RawMessage `json:"result,omitempty"`

	// Error is the JSON-RPC error object. Nil on success.
	Error *Error `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object returned in error responses.
type Error struct {
	// Code is a numeric error code. Standard JSON-RPC codes:
	//   -32700  Parse error
	//   -32600  Invalid request
	//   -32601  Method not found
	//   -32602  Invalid params
	//   -32603  Internal error
	Code int `json:"code"`

	// Message is a short human-readable error description.
	Message string `json:"message"`

	// Data is optional additional error data.
	Data any `json:"data,omitempty"`
}

// Error implements the error interface for JSON-RPC errors.
func (e *Error) Error() string {
	if e.Data != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Data)
	}
	return e.Message
}

// --- Initialize method types ---

// InitializeParams is the params object for the "initialize" method.
type InitializeParams struct {
	// RootPath is the absolute path to the project root directory.
	RootPath string `json:"root_path"`

	// Config is optional analyzer-specific configuration from
	// .gaze.yaml. Nil when no config is provided.
	Config map[string]any `json:"config,omitempty"`
}

// InitializeResult is the result object for the "initialize" method.
// It declares the analyzer's capabilities and identity.
//
// Design decision D3: Capability negotiation via initialize response.
type InitializeResult struct {
	// Capabilities declares which optional methods the analyzer
	// supports.
	Capabilities Capabilities `json:"capabilities"`

	// ProtocolVersion is the protocol version the analyzer
	// implements (e.g., "1.0.0").
	ProtocolVersion string `json:"protocol_version"`

	// AnalyzerName is the human-readable analyzer name (e.g.,
	// "snake-eyes").
	AnalyzerName string `json:"analyzer_name"`

	// Language is the primary language the analyzer targets
	// (e.g., "python", "rust").
	Language string `json:"language"`

	// LanguageVersion is the runtime/compiler version for the
	// target language (e.g., "3.12.0" for Python, "1.25.0" for Go).
	LanguageVersion string `json:"language_version,omitempty"`
}

// Capabilities declares which optional protocol methods the analyzer
// supports. Required methods (initialize, analyze, complexity,
// coverage, shutdown) are always assumed to be supported.
type Capabilities struct {
	// Discover indicates whether the analyzer supports the
	// "discover" method for finding source and test files.
	// Design decision D11: discover is optional.
	Discover bool `json:"discover"`

	// TestMapping indicates whether the analyzer supports the
	// "test_mapping" method for mapping test assertions to side
	// effects. When false, GazeCRAP is unavailable.
	TestMapping bool `json:"test_mapping"`

	// ClassifySignals indicates whether the analyzer supports the
	// "classify_signals" method for providing raw classification
	// signals. When false, gaze uses the pre-classified effects
	// from the analyze response.
	ClassifySignals bool `json:"classify_signals"`

	// Streaming indicates whether the analyzer supports the
	// "analyze/stream" method for incremental JSONL analysis
	// results. When true, gaze calls analyze/stream instead of
	// analyze. When false, the batch analyze method is used.
	Streaming bool `json:"streaming"`

	// DocCoverage indicates whether the analyzer supports the
	// "doc_coverage" method for reporting documentation status
	// of public symbols. When false, gaze uses heuristic
	// documentation coverage.
	DocCoverage bool `json:"doc_coverage"`
}

// --- Analyze method types ---

// AnalyzeParams is the params object for the "analyze" method.
type AnalyzeParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/file patterns to analyze
	// (e.g., ["./..."]).
	Patterns []string `json:"patterns"`
}

// AnalyzeResult is the result object for the "analyze" method.
type AnalyzeResult struct {
	// Functions is the list of analyzed functions with their
	// detected side effects.
	Functions []AnalyzedFunction `json:"functions"`
}

// AnalyzedFunction represents a single function's analysis results
// from an external analyzer.
type AnalyzedFunction struct {
	// Name is the function or method name.
	Name string `json:"name"`

	// Package is the package/module path.
	Package string `json:"package"`

	// File is the source file path (relative to root_path or
	// absolute).
	File string `json:"file"`

	// Line is the line number of the function declaration.
	Line int `json:"line"`

	// SideEffects is the list of detected side effects.
	SideEffects []AnalyzedSideEffect `json:"side_effects"`
}

// AnalyzedSideEffect represents a single side effect detected by
// an external analyzer. The Type field uses gaze's taxonomy
// constants (e.g., "ReturnValue", "ErrorReturn",
// "ReceiverMutation").
//
// Design decision D9: External analyzers map their language concepts
// to gaze's existing taxonomy types.
type AnalyzedSideEffect struct {
	// Type is the side effect type from gaze's taxonomy.
	Type string `json:"type"`

	// Description is a human-readable explanation.
	Description string `json:"description"`

	// Location is the source position (file:line:col or similar).
	Location string `json:"location,omitempty"`

	// Target is the affected entity (variable name, field, etc.).
	Target string `json:"target,omitempty"`

	// Classification is the pre-computed classification. Nil when
	// the analyzer does not classify effects (classification is
	// then done by gaze via classify_signals or defaults).
	Classification *AnalyzedClassification `json:"classification,omitempty"`

	// Detail is optional language-specific metadata provided by
	// the external analyzer. It is opaque to gaze's scoring and
	// classification logic — passed through to taxonomy.SideEffect
	// and ultimately to JSON output and AI report pipelines.
	Detail map[string]any `json:"detail,omitempty"`
}

// AnalyzedClassification is the classification data attached to a
// side effect by the external analyzer.
type AnalyzedClassification struct {
	// Label is "contractual", "incidental", or "ambiguous".
	Label string `json:"label"`

	// Confidence is the classification confidence (0-100).
	Confidence int `json:"confidence"`
}

// --- Complexity method types ---

// ComplexityParams is the params object for the "complexity" method.
type ComplexityParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/file patterns.
	Patterns []string `json:"patterns"`
}

// ComplexityResult is the result object for the "complexity" method.
type ComplexityResult struct {
	// Functions is the list of functions with their cyclomatic
	// complexity.
	Functions []FunctionComplexityData `json:"functions"`
}

// FunctionComplexityData represents per-function cyclomatic
// complexity from an external analyzer.
type FunctionComplexityData struct {
	// Name is the function or method name.
	Name string `json:"name"`

	// Package is the package/module path.
	Package string `json:"package"`

	// File is the source file path.
	File string `json:"file"`

	// Line is the line number of the function declaration.
	Line int `json:"line"`

	// Complexity is the cyclomatic complexity value.
	Complexity int `json:"complexity"`
}

// --- Coverage method types ---

// CoverageParams is the params object for the "coverage" method.
type CoverageParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/file patterns.
	Patterns []string `json:"patterns"`
}

// CoverageResult is the result object for the "coverage" method.
type CoverageResult struct {
	// Functions is the list of functions with their coverage data.
	Functions []FunctionCoverageData `json:"functions"`
}

// FunctionCoverageData represents per-function line coverage from
// an external analyzer.
type FunctionCoverageData struct {
	// File is the source file path.
	File string `json:"file"`

	// Function is the function or method name.
	Function string `json:"function"`

	// StartLine is the function declaration start line.
	StartLine int `json:"start_line"`

	// EndLine is the function body end line.
	EndLine int `json:"end_line"`

	// CoveredStmts is the number of statements covered by tests.
	CoveredStmts int64 `json:"covered_stmts"`

	// TotalStmts is the total number of statements in the function.
	TotalStmts int64 `json:"total_stmts"`

	// Percentage is the coverage percentage (0-100).
	Percentage float64 `json:"percentage"`
}

// --- Discover method types (optional, D11) ---

// DiscoverParams is the params object for the "discover" method.
type DiscoverParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`
}

// DiscoverResult is the result object for the "discover" method.
type DiscoverResult struct {
	// SourceFiles is the list of source file paths.
	SourceFiles []string `json:"source_files"`

	// TestFiles is the list of test file paths.
	TestFiles []string `json:"test_files"`

	// Framework is the test framework name (e.g., "pytest",
	// "unittest").
	Framework string `json:"framework"`
}

// --- TestMapping method types (optional) ---

// TestMappingParams is the params object for the "test_mapping"
// method.
type TestMappingParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/file patterns.
	Patterns []string `json:"patterns"`
}

// TestMappingResult is the result object for the "test_mapping"
// method.
type TestMappingResult struct {
	// Mappings is the list of assertion-to-effect mappings.
	Mappings []AssertionMappingData `json:"mappings"`
}

// AssertionMappingData represents a single test assertion mapped to
// a side effect by the external analyzer.
type AssertionMappingData struct {
	// TestFunction is the test function name.
	TestFunction string `json:"test_function"`

	// TestFile is the test file path.
	TestFile string `json:"test_file"`

	// AssertionLocation is the source position of the assertion
	// (file:line).
	AssertionLocation string `json:"assertion_location"`

	// AssertionType is the kind of assertion (e.g., "equality",
	// "error_check").
	AssertionType string `json:"assertion_type"`

	// TargetFunction is the function under test.
	TargetFunction string `json:"target_function"`

	// TargetPackage is the package of the function under test.
	TargetPackage string `json:"target_package"`

	// SideEffectType is the type of side effect being asserted on.
	SideEffectType string `json:"side_effect_type"`

	// Confidence is the mapping confidence (0-100).
	Confidence int `json:"confidence"`
}

// --- ClassifySignals method types (optional) ---

// ClassifySignalsParams is the params object for the
// "classify_signals" method.
type ClassifySignalsParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/file patterns.
	Patterns []string `json:"patterns"`
}

// ClassifySignalsResult is the result object for the
// "classify_signals" method.
type ClassifySignalsResult struct {
	// Signals is the list of classification signals.
	Signals []ClassifySignalData `json:"signals"`
}

// ClassifySignalData represents a classification signal from an
// external analyzer. These signals are fed into gaze's scoring
// engine (ComputeScore) for classification.
type ClassifySignalData struct {
	// Function is the function name the signal applies to.
	Function string `json:"function"`

	// Package is the package path.
	Package string `json:"package"`

	// SideEffectType is the type of side effect this signal
	// relates to.
	SideEffectType string `json:"side_effect_type"`

	// Source identifies the signal type (e.g., "docstring",
	// "type_annotation", "decorator").
	Source string `json:"source"`

	// Weight is the numeric contribution to the confidence score.
	Weight int `json:"weight"`

	// Reasoning explains why this signal was applied.
	Reasoning string `json:"reasoning,omitempty"`
}

// --- DocCoverage method types (optional) ---

// DocCoverageParams is the params object for the "doc_coverage" method.
type DocCoverageParams struct {
	// RootPath is the project root directory.
	RootPath string `json:"root_path"`

	// Patterns is the list of package/module patterns to analyze.
	Patterns []string `json:"patterns"`
}

// DocCoverageResult is the result object for the "doc_coverage" method.
type DocCoverageResult struct {
	// Symbols is the documentation status for each public symbol.
	Symbols []SymbolDocStatus `json:"symbols"`
}

// SymbolDocStatus represents the documentation status of a single
// public symbol as reported by an external analyzer.
type SymbolDocStatus struct {
	// Name is the fully qualified symbol name.
	Name string `json:"name"`

	// Package is the package/module path.
	Package string `json:"package"`

	// File is the source file path.
	File string `json:"file"`

	// Line is the declaration line number.
	Line int `json:"line"`

	// Kind is the symbol kind: "function", "type", "constant",
	// "variable", "class", or "method".
	Kind string `json:"kind"`

	// Documented indicates whether the symbol has a docstring
	// or doc comment.
	Documented bool `json:"documented"`

	// DocSnippet is the first line of documentation. Empty when
	// the symbol is undocumented.
	DocSnippet string `json:"doc_snippet,omitempty"`
}
