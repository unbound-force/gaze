# CI Scaffolding Requirements — Project Flavor Detection

<!--
  Requirement numbering allocation (meta-prompt MP-014):
  - This change allocates CS-FR-100 .. CS-FR-104 for
    ci-scaffolding.
  - CS-FR-105 and upward are reserved for future ci-scaffolding
    changes.
  - CS-FR-001 .. CS-FR-099 are held for retrospective numbering of
    the unnumbered requirements already carried by the active
    `init-ci-scaffold` delta, which predates this convention; ids
    here start above that band to avoid collisions. This change
    supersedes that delta's go.mod-only requirement; see design.md
    decision D7.
  - Sibling capability: ci-workflow requirements (CW-FR-*) live in
    specs/ci-workflow/spec.md within this change.
-->

## ADDED Requirements

### Requirement: CI scaffolding MUST select the caller flavor by project marker

CS-FR-100

When `gaze init --ci` runs, the scaffold MUST select the caller
workflow flavor by repository marker at the target root, in
precedence order: `go.mod` selects Go; otherwise `package.json`
selects Node; otherwise `pyproject.toml` selects Python. With no
recognized marker, the command MUST print a warning naming the
markers, skip CI scaffolding, and still complete the `.opencode/`
scaffolding. The Go flavor output MUST remain byte-identical to
the pre-change output.

#### Scenario: package.json without go.mod selects the Node flavor

- **GIVEN** a target directory with a `package.json` and no
  `go.mod`
- **WHEN** `gaze init --ci` runs
- **THEN** a Node caller workflow is written and the `.opencode/`
  scaffolding completes normally

#### Scenario: go.mod wins over package.json

- **GIVEN** a target directory with both `go.mod` and
  `package.json`
- **WHEN** `gaze init --ci` runs
- **THEN** the Go caller workflow is written, byte-identical to
  the pre-change output

#### Scenario: No recognized marker keeps the skip-with-warning path

- **GIVEN** a target directory with none of the three markers
- **WHEN** `gaze init --ci` runs
- **THEN** a warning names the recognized markers, no workflow
  file is written, and the `.opencode/` scaffolding completes
  normally

### Requirement: The Node caller MUST wire the reading-stone analyzer

CS-FR-101

The Node caller workflow MUST set up Node with a SHA-pinned
`setup-node` step, run the TypeScript test suite with coverage,
upload the `coverage/` directory as the coverage artifact, and call
the reusable workflow with `analyzer: gaze-analyzer-typescript` and
the analyzer's package pattern. The threshold inputs MUST remain
commented out with TODO guidance, keeping the scaffolded workflow
report-only by default.

#### Scenario: Node caller generates and uploads coverage

- **GIVEN** the Node caller workflow generated in a repository
  with `package.json` and no `go.mod`
- **WHEN** its test job is inspected
- **THEN** it runs `npx vitest run --coverage` and uploads the
  `coverage/` directory as the coverage artifact

#### Scenario: Node caller passes the analyzer through

- **GIVEN** the same generated workflow
- **WHEN** its `gaze-ai-report` job is inspected
- **THEN** it calls the reusable workflow with
  `analyzer: gaze-analyzer-typescript` and `packages: ./src/...`,
  and the threshold inputs appear only as commented TODO lines

### Requirement: The Python caller MUST wire the snake-eyes analyzer

CS-FR-102

The Python caller workflow MUST set up Python with a SHA-pinned
`setup-python` step, run the project's test suite with coverage,
upload the coverage output the analyzer reads, and call the
reusable workflow with `analyzer: snake-eyes` and the package
pattern from the README's Python examples (`./src`). The threshold
inputs MUST remain commented out with TODO guidance, keeping the
scaffolded workflow report-only by default.

#### Scenario: Python caller passes the analyzer through

- **GIVEN** the Python caller workflow generated in a repository
  with `pyproject.toml` and neither `go.mod` nor `package.json`
- **WHEN** its `gaze-ai-report` job is inspected
- **THEN** it calls the reusable workflow with
  `analyzer: snake-eyes` and the threshold inputs appear only as
  commented TODO lines

### Requirement: Each flavor MUST be pinned by its own golden file

CS-FR-103

The Go, Node, and Python caller outputs MUST each be pinned by a
golden file under `internal/scaffold/testdata/` — the existing
`golden-gaze-ai-report.yml` for Go, byte-identical to before this
change, and new Node and Python goldens beside it. Per-flavor
templates MUST live under `internal/scaffold/assets/ci/`, and the
`ci/` subtree MUST remain excluded from the `.opencode/` mirror
sync so the nine-file embedded-asset manifest is unchanged.

#### Scenario: Go golden is unchanged

- **GIVEN** the repository before and after this change
- **WHEN** the Go-flavor golden files are compared
- **THEN** they are byte-identical

#### Scenario: Node and Python goldens pin their flavors

- **GIVEN** `gaze init --ci` run in `package.json`-only and
  `pyproject.toml`-only fixture directories
- **WHEN** the generated workflows are compared against their
  goldens
- **THEN** each output matches its flavor's golden file exactly

### Requirement: CI next-steps guidance MUST print for every flavor

CS-FR-104

When `--ci` scaffolding succeeds for any flavor, the command
summary MUST include the CI next-steps section — the
`OPENCODE_CONFIG_CONTENT` secret-setup one-liner, the JSON minify
tip, the second-secret masking pointer, and the
threshold-guidance prompt — the same four items the Go flavor
prints. When scaffolding is skipped for an unrecognized marker,
the section MUST NOT be printed.

#### Scenario: Node flavor prints the same next steps

- **GIVEN** a successful `gaze init --ci` run in a
  `package.json`-only directory
- **WHEN** the summary output is inspected
- **THEN** it contains the same four CI next-steps items the Go
  flavor prints

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
