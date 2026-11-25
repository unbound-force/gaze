## ADDED Requirements

### Requirement: Language-only external dispatch on gaze report

`gaze report` MUST select the external analyzer pipeline when `--language` is set without `--analyzer`. The dispatch condition MUST be identical to the one `gaze crap`, `gaze quality`, and `gaze docscan` already use: `--analyzer` set, `--language` set, or both.

#### Scenario: Language-only outside a Go module

- **GIVEN** a working directory that is not inside a Go module
- **WHEN** `gaze report --language python ./src` runs
- **THEN** the external analyzer pipeline MUST be selected
- **AND** the run MUST NOT fail with a Go module-root error

#### Scenario: Language-only inside a Go module

- **GIVEN** a working directory inside a Go module
- **WHEN** `gaze report --language python ./src` runs
- **THEN** the external analyzer pipeline MUST be selected
- **AND** Go-native analysis MUST NOT run for the project

#### Scenario: Analyzer flag alone is unchanged

- **GIVEN** any working directory
- **WHEN** `gaze report --analyzer snake-eyes ./src` runs
- **THEN** the external analyzer pipeline MUST be selected exactly as before this change

### Requirement: External module-dir resolution

When either `--analyzer` or `--language` is set, `gaze report` MUST resolve its analysis root to the working directory and MUST NOT require a Go module root — the analyzed project may not be a Go module.

#### Scenario: No go.mod with language set

- **GIVEN** a working directory with no `go.mod` at or above it
- **WHEN** `gaze report --language python ./src` runs and discovery resolves an analyzer
- **THEN** module-dir resolution MUST succeed using the working directory
- **AND** the run MUST proceed to analyzer discovery without a module-root failure

### Requirement: Language-specific discovery failure diagnostic

When `--language` is set and three-tier discovery (CLI flag → `.gaze.yaml` config → `gaze-analyzer-<language>` PATH convention) finds no analyzer, `gaze report` MUST fail with the language-specific error naming the requested language. The failure MUST NOT surface a Go module-root error, which would misidentify the problem.

#### Scenario: Undiscoverable language

- **GIVEN** no `--analyzer` flag, no `.gaze.yaml` analyzer entry, and no `gaze-analyzer-<lang>` binary on PATH for the requested language
- **WHEN** `gaze report --language <lang> .` runs
- **THEN** the error MUST contain `no analyzer found for language`
- **AND** the error MUST NOT contain `finding module root`

### Requirement: Go-native default preserved

When neither `--analyzer` nor `--language` is set, `gaze report` MUST behave exactly as before this change: the Go module root remains required, and the Go-native pipeline runs.

#### Scenario: No flags outside a Go module

- **GIVEN** a working directory that is not inside a Go module
- **WHEN** `gaze report` runs with neither `--analyzer` nor `--language`
- **THEN** the run MUST fail with the `finding module root` error, as before this change

#### Scenario: No flags inside a Go module

- **GIVEN** a working directory inside a Go module
- **WHEN** `gaze report` runs with neither `--analyzer` nor `--language`
- **THEN** the Go-native pipeline MUST run, unchanged by this change

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
