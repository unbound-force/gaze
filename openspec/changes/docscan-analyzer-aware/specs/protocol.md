# Delta Spec: Protocol Extension — `doc_coverage` Method

## ADDED Requirements

### Requirement: `doc_coverage` Optional Protocol Method

The external analyzer protocol MUST support an optional `doc_coverage` method that reports documentation status for public symbols. The method MUST be announced via `Capabilities.DocCoverage` (boolean) in the `initialize` response. When `DocCoverage` is `false`, the method MUST NOT be called.

#### Scenario: Analyzer supports `doc_coverage`
- **GIVEN** an external analyzer that returns `{"doc_coverage": true}` in its `Capabilities` during the initialize handshake
- **WHEN** gaze calls the `doc_coverage` method with `{"root_path": "/project", "patterns": ["./..."]}`
- **THEN** the analyzer MUST return a `DocCoverageResult` containing a `symbols` array where each entry has `name`, `package`, `file`, `line`, `kind`, and `documented` fields

#### Scenario: Analyzer does not support `doc_coverage`
- **GIVEN** an external analyzer that returns `{"doc_coverage": false}` in its `Capabilities`
- **WHEN** gaze processes the analyzer's capabilities
- **THEN** gaze MUST NOT call the `doc_coverage` method and SHOULD fall back to heuristic documentation coverage

### Requirement: `DocCoverage` Capability Flag

The `Capabilities` struct in `internal/protocol/types.go` MUST include a `DocCoverage bool` field with JSON tag `"doc_coverage"`.

#### Scenario: Capability round-trip serialization
- **GIVEN** a `Capabilities` struct with `DocCoverage: true`
- **WHEN** the struct is marshaled to JSON and unmarshaled back
- **THEN** the `DocCoverage` field MUST round-trip correctly

### Requirement: `doc_coverage` Method Constant

A `MethodDocCoverage` constant with value `"doc_coverage"` MUST be added to the protocol method constants in `internal/protocol/types.go`.

#### Scenario: Method constant usage
- **GIVEN** a caller constructing a JSON-RPC request for documentation coverage
- **WHEN** the caller uses `protocol.MethodDocCoverage` as the method name
- **THEN** the JSON-RPC request MUST have `"method": "doc_coverage"`

### Requirement: `DocCoverageParams` Type

The `DocCoverageParams` struct MUST include:
- `RootPath string` (JSON: `"root_path"`) — project root directory
- `Patterns []string` (JSON: `"patterns"`) — package/module patterns to analyze

#### Scenario: Params construction
- **GIVEN** a project at `/home/user/project` with pattern `./...`
- **WHEN** `DocCoverageParams` is constructed and marshaled
- **THEN** the JSON MUST be `{"root_path": "/home/user/project", "patterns": ["./..."]}`

### Requirement: `DocCoverageResult` Type

The `DocCoverageResult` struct MUST include:
- `Symbols []SymbolDocStatus` (JSON: `"symbols"`) — documentation status for each public symbol

### Requirement: `SymbolDocStatus` Type

The `SymbolDocStatus` struct MUST include:
- `Name string` (JSON: `"name"`) — fully qualified symbol name
- `Package string` (JSON: `"package"`) — package/module path
- `File string` (JSON: `"file"`) — source file path
- `Line int` (JSON: `"line"`) — declaration line number
- `Kind string` (JSON: `"kind"`) — symbol kind: `"function"`, `"type"`, `"constant"`, `"variable"`, `"class"`, or `"method"`
- `Documented bool` (JSON: `"documented"`) — whether the symbol has a docstring/doc comment
- `DocSnippet string` (JSON: `"doc_snippet,omitempty"`) — first line of documentation (MAY be empty)

#### Scenario: Undocumented function
- **GIVEN** a public function `ProcessData` in `pkg/processor` at line 42 with no docstring
- **WHEN** the analyzer returns the symbol status
- **THEN** the `SymbolDocStatus` MUST have `name: "ProcessData"`, `kind: "function"`, `documented: false`, and `doc_snippet: ""`

#### Scenario: Documented class
- **GIVEN** a public class `DataStore` in `pkg/storage` with docstring "Persistent key-value store."
- **WHEN** the analyzer returns the symbol status
- **THEN** the `SymbolDocStatus` MUST have `kind: "class"`, `documented: true`, and `doc_snippet: "Persistent key-value store."`

## MODIFIED Requirements

_(None — all protocol changes are additive.)_

## REMOVED Requirements

_(None.)_
