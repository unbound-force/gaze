# Delta Spec: API Documentation Coverage (`internal/docscan/apidoc`)

## ADDED Requirements

### Requirement: `APICoverageReport` Type

The `apidoc` package MUST define an `APICoverageReport` struct with the following fields:
- `TotalSymbols int` (JSON: `"total_symbols"`) — total public symbols analyzed
- `DocumentedSymbols int` (JSON: `"documented_symbols"`) — symbols with documentation
- `CoveragePercent float64` (JSON: `"coverage_percent"`) — (documented / total) × 100
- `Source string` (JSON: `"source"`) — `"doc_coverage"` (native) or `"heuristic"` (fallback)
- `Undocumented []SymbolCoverage` (JSON: `"undocumented"`) — symbols lacking documentation
- `StaleReferences []StaleReference` (JSON: `"stale_references"`) — Markdown references to missing symbols
- `CodeBlockIssues []CodeBlockIssue` (JSON: `"code_block_issues"`) — language tag mismatches

#### Scenario: Report from native `doc_coverage`
- **GIVEN** an analyzer that supports `doc_coverage` and reports 40 total symbols with 35 documented
- **WHEN** `apidoc.Analyze` is called with the analyzer's `DocCoverageResult`
- **THEN** the report MUST have `TotalSymbols: 40`, `DocumentedSymbols: 35`, `CoveragePercent: 87.5`, `Source: "doc_coverage"`, and `Undocumented` containing 5 entries

#### Scenario: Report from heuristic fallback
- **GIVEN** an analyzer that does NOT support `doc_coverage` but reports 20 functions via `analyze`
- **WHEN** `apidoc.Analyze` is called with `[]AnalyzedFunction` and Markdown files referencing 15 function names
- **THEN** the report MUST have `TotalSymbols: 20`, `DocumentedSymbols: 15`, `CoveragePercent: 75.0`, `Source: "heuristic"`, and `Undocumented` containing 5 entries

### Requirement: Heuristic Matching Rules

When `doc_coverage` is unavailable, the heuristic fallback MUST use these matching rules:

1. **Backtick-only matching**: Only backtick-quoted content (`` `SymbolName` ``) in Markdown is considered a potential symbol reference. Bare prose mentions (e.g., "ProcessData handles data") do NOT count as documentation.
2. **Exact match**: Matching is case-sensitive and exact. `` `ProcessData` `` matches `ProcessData` but NOT `processData`, `processdata`, or `Process`.
3. **Qualified names**: Both unqualified (`` `ProcessData` ``) and qualified (`` `pkg.ProcessData` ``) forms match the symbol `ProcessData`. When the analyzer reports a function named `ProcessData` in package `pkg`, all of the following Markdown references count as a documentation match: `` `ProcessData` ``, `` `pkg.ProcessData` ``.
4. **No partial matching**: `` `Process` `` does NOT match `ProcessData`. `` `ProcessDataHandler` `` does NOT match `ProcessData`.
5. **Short names accepted**: Common short function names like `New`, `Run`, `Get`, `Set`, `Read`, `Write` are matched using the same exact rules. False positives from these appearing in prose context are accepted trade-offs of the heuristic.
6. **Scope**: The heuristic covers only functions-with-side-effects (those present in `AnalyzedFunction` output), not all public symbols. This is a known limitation — the heuristic is a lower bound of documentation coverage. Native `doc_coverage` data covers all public symbols.

#### Scenario: Case-sensitive exact match
- **GIVEN** a README.md containing `` `processData` `` (lowercase) and an analyzer reporting a function named `ProcessData`
- **WHEN** heuristic coverage is computed
- **THEN** `ProcessData` MUST be reported as undocumented (no case-insensitive matching)

#### Scenario: Qualified name match
- **GIVEN** a README.md containing `` `pkg.ProcessData` `` and an analyzer reporting `ProcessData` in package `pkg`
- **WHEN** heuristic coverage is computed
- **THEN** `ProcessData` MUST be counted as documented

#### Scenario: Partial match rejected
- **GIVEN** a README.md containing `` `Process` `` and an analyzer reporting `ProcessData`
- **WHEN** heuristic coverage is computed
- **THEN** `ProcessData` MUST be reported as undocumented (no partial matching)

#### Scenario: Prose mention not counted
- **GIVEN** a README.md containing `ProcessData handles incoming data` (no backticks) and an analyzer reporting `ProcessData`
- **WHEN** heuristic coverage is computed
- **THEN** `ProcessData` MUST be reported as undocumented (prose mentions without backticks are ignored)

### Requirement: `SymbolCoverage` Type

The `SymbolCoverage` struct MUST include:
- `Name string` (JSON: `"name"`) — symbol name
- `Package string` (JSON: `"package"`) — package/module path
- `File string` (JSON: `"file"`) — source file path
- `Line int` (JSON: `"line"`) — declaration line number
- `Kind string` (JSON: `"kind"`) — symbol kind

### Requirement: `StaleReference` Type

The `StaleReference` struct MUST include:
- `Symbol string` (JSON: `"symbol"`) — the referenced symbol name
- `DocFile string` (JSON: `"doc_file"`) — Markdown file containing the reference
- `DocLine int` (JSON: `"doc_line"`) — line number of the reference in the Markdown file

#### Scenario: Renamed function detected in README
- **GIVEN** a README.md containing `` `OldFunctionName` `` at line 15
- **AND** the analyzer's output contains no function named `OldFunctionName`
- **WHEN** `ValidateReferences` is called
- **THEN** the result MUST include a `StaleReference` with `Symbol: "OldFunctionName"`, `DocFile: "README.md"`, `DocLine: 15`

#### Scenario: Valid reference not flagged
- **GIVEN** a README.md containing `` `ProcessData` `` at line 10
- **AND** the analyzer's output contains a function named `ProcessData`
- **WHEN** `ValidateReferences` is called
- **THEN** the result MUST NOT include a `StaleReference` for `ProcessData`

### Requirement: `CodeBlockIssue` Type

The `CodeBlockIssue` struct MUST include:
- `DocFile string` (JSON: `"doc_file"`) — Markdown file containing the code block
- `DocLine int` (JSON: `"doc_line"`) — line number of the opening fence
- `DeclaredLang string` (JSON: `"declared_lang"`) — language tag on the code fence
- `ExpectedLang string` (JSON: `"expected_lang"`) — language reported by the analyzer

#### Scenario: Wrong language tag
- **GIVEN** a docs/tutorial.md containing ` ```python ` at line 20
- **AND** the analyzer reports language `"go"`
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** the result MUST include a `CodeBlockIssue` with `DeclaredLang: "python"`, `ExpectedLang: "go"`

#### Scenario: Untagged code block ignored
- **GIVEN** a README.md containing ` ``` ` (no language tag) at line 5
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** the code block MUST NOT produce a `CodeBlockIssue`

#### Scenario: Matching language tag
- **GIVEN** a docs/api.md containing ` ```go ` at line 10
- **AND** the analyzer reports language `"go"`
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** the code block MUST NOT produce a `CodeBlockIssue`

### Requirement: `AnalyzerData` Input Type

The `apidoc` package MUST define an `AnalyzerData` struct for passing analyzer output to the coverage computation:
- `Functions []protocol.AnalyzedFunction` — from `analyze` method
- `DocCoverage *protocol.DocCoverageResult` — from `doc_coverage` method (nil when unsupported)
- `Language string` — analyzer's declared language (for code block validation)

### Requirement: `Analyze` Function

The `apidoc` package MUST export an `Analyze` function:
```go
func Analyze(docs []docscan.DocumentFile, data *AnalyzerData) (*APICoverageReport, error)
```

- When `data` is nil, the function MUST return `nil, nil` (no-op)
- When `data.DocCoverage` is non-nil, the function MUST use native doc coverage data and set `Source: "doc_coverage"`
- When `data.DocCoverage` is nil, the function MUST derive coverage heuristically from `data.Functions` and set `Source: "heuristic"`
- The function MUST call `ValidateReferences` and `ValidateCodeBlocks` regardless of the coverage source
- The function MUST build the `symbolNames` map for `ValidateReferences` as follows: when using native `doc_coverage`, the map MUST contain each `SymbolDocStatus.Name`; when using heuristic fallback, the map MUST contain each `AnalyzedFunction.Name`. In both cases, the map MUST also contain qualified forms (`Package.Name`) for each symbol to enable qualified-name matching in stale reference detection

#### Scenario: Nil data returns nil
- **GIVEN** a slice of `DocumentFile` entries
- **WHEN** `Analyze` is called with `data = nil`
- **THEN** the function MUST return `(nil, nil)`

### Requirement: `CoverageResult` Type

The `apidoc` package MUST define a `CoverageResult` struct:
- `Undocumented []SymbolCoverage` (JSON: `"undocumented"`) — symbols lacking documentation
- `Total int` (JSON: `"total"`) — total public symbols analyzed
- `Documented int` (JSON: `"documented"`) — symbols with documentation
- `Source string` (JSON: `"source"`) — `"doc_coverage"` or `"heuristic"`

### Requirement: `ComputeCoverage` Function

The `apidoc` package MUST export a `ComputeCoverage` function:
```go
func ComputeCoverage(data *AnalyzerData, docs []docscan.DocumentFile) (*CoverageResult, error)
```

- When `data.DocCoverage` is non-nil, iterate `Symbols` and collect entries where `Documented == false`
- When `data.DocCoverage` is nil, extract function names from `data.Functions`, scan Markdown content for backtick-quoted references, and report unmatched functions as undocumented
- When `Total == 0`, `CoveragePercent` in the parent report MUST be `0.0` (not NaN or division-by-zero)

#### Scenario: Zero symbols
- **GIVEN** an analyzer that reports 0 public symbols (empty `Functions` slice, nil `DocCoverage`)
- **WHEN** `ComputeCoverage` is called
- **THEN** `Total` MUST be 0, `Documented` MUST be 0, `Undocumented` MUST be empty, and no error MUST be returned

### Requirement: `ValidateReferences` Function

The `apidoc` package MUST export a `ValidateReferences` function:
```go
func ValidateReferences(docs []docscan.DocumentFile, symbolNames map[string]bool) []StaleReference
```

- MUST scan each `DocumentFile.Content` for backtick-quoted symbol names (`` `SymbolName` ``)
- MUST report any backtick-quoted name that is NOT in `symbolNames` as a stale reference
- MUST include the file path and line number of each stale reference
- SHOULD ignore backtick content matching any of these patterns: starts with `-` or `--` (CLI flags), contains `/` (file paths/URLs), starts with `$` (environment variables), is a Go keyword or built-in (`nil`, `true`, `false`, `error`, `string`, `int`, `bool`, `any`, `func`, `if`, `for`, `return`, `defer`, `go`), or is all-lowercase-with-hyphens (likely a command name)

#### Scenario: Empty symbol set
- **GIVEN** an empty `symbolNames` map and Markdown files containing backtick-quoted names
- **WHEN** `ValidateReferences` is called
- **THEN** all backtick-quoted names (except those matching ignore patterns) MUST be reported as stale references

#### Scenario: Empty documents
- **GIVEN** a non-empty `symbolNames` map and an empty `docs` slice
- **WHEN** `ValidateReferences` is called
- **THEN** the result MUST be an empty slice (no stale references)

#### Scenario: Non-symbol backtick content ignored
- **GIVEN** a README.md containing `` `--verbose` ``, `` `/path/to/file` ``, `` `$HOME` ``, and `` `nil` ``
- **AND** none of these are in the `symbolNames` map
- **WHEN** `ValidateReferences` is called
- **THEN** none of these MUST be reported as stale references

### Requirement: `ValidateCodeBlocks` Function

The `apidoc` package MUST export a `ValidateCodeBlocks` function:
```go
func ValidateCodeBlocks(docs []docscan.DocumentFile, expectedLang string) []CodeBlockIssue
```

- MUST find all fenced code blocks (` ```lang `) in each document
- MUST compare the declared language tag against `expectedLang`
- MUST ignore code blocks with no language tag
- MUST ignore language tags in the following exhaustive set (exported as `GenericLanguageTags`): `text`, `plaintext`, `console`, `shell`, `bash`, `sh`, `zsh`, `json`, `yaml`, `yml`, `toml`, `xml`, `html`, `css`, `sql`, `diff`, `ini`, `csv`, `makefile`, `dockerfile`, `markdown`, `md`, `output`, `log`

#### Scenario: Generic language tag not flagged
- **GIVEN** a README.md containing ` ```json ` code blocks
- **AND** the analyzer reports language `"python"`
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** the `json` code block MUST NOT produce a `CodeBlockIssue`

#### Scenario: Multiple code blocks in one file
- **GIVEN** a docs/tutorial.md containing ` ```go ` at line 5, ` ```python ` at line 20, ` ``` ` at line 35, and ` ```json ` at line 50
- **AND** the analyzer reports language `"go"`
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** the result MUST contain exactly 1 `CodeBlockIssue` for the `python` block at line 20 (go block matches, untagged block ignored, json is generic)

#### Scenario: Empty expected language
- **GIVEN** an empty `expectedLang` string
- **WHEN** `ValidateCodeBlocks` is called
- **THEN** no code blocks MUST be flagged (validation is skipped)

## MODIFIED Requirements

_(None.)_

## REMOVED Requirements

_(None.)_
