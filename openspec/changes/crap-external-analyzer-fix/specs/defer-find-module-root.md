## ADDED Requirements

### Requirement: Deferred Module Root Resolution for Crap Command

`newCrapCmd`'s `RunE` closure MUST NOT call `loader.FindModuleRoot` unconditionally. The `moduleDir` field in `crapParams` MUST be populated with the current working directory (`os.Getwd()`). Module root resolution MUST be deferred to the Go-native path inside `runCrap`.

#### Scenario: External analyzer on non-Go project

- **GIVEN** `gaze crap --analyzer snake-eyes --language python .` is invoked from a directory without `go.mod`
- **WHEN** `newCrapCmd`'s `RunE` closure executes
- **THEN** execution MUST NOT fail with `"no go.mod found"` and MUST reach `runCrapWithExternalAnalyzer`

#### Scenario: Go-native analysis continues to resolve module root

- **GIVEN** `gaze crap ./...` is invoked from a Go module directory without `--analyzer`
- **WHEN** `runCrap` executes the Go-native path
- **THEN** `loader.FindModuleRoot` MUST be called and MUST resolve the module root before downstream analysis

#### Scenario: FindModuleRoot failure in Go-native path

- **GIVEN** `gaze crap ./...` is invoked from a directory without `go.mod` and without `--analyzer`
- **WHEN** `runCrap` calls `loader.FindModuleRoot`
- **THEN** the error MUST be returned with wrapping format `"finding module root: %w"`

## MODIFIED Requirements

### Requirement: crapParams.moduleDir Semantics

Previously: `crapParams.moduleDir` always contained the resolved Go module root (output of `loader.FindModuleRoot`).

After this change: `crapParams.moduleDir` contains the current working directory (output of `os.Getwd()`). For the Go-native path, `runCrap` resolves it to the module root before use. For the external analyzer path, it is used as-is (the working directory is the correct project root for non-Go projects).

### Requirement: runCrap Go-native Path Pre-condition

`runCrap` MUST call `loader.FindModuleRoot(p.moduleDir)` and replace `p.moduleDir` with the result before any downstream use (`crap.Analyze`, `resolveBaselineAndCompare`, provider construction) in the Go-native path. This call MUST occur after the `--analyzer` dispatch and before the contract coverage provider setup.

## REMOVED Requirements

None.
