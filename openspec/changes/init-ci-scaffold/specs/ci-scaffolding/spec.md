# CI Scaffolding Requirements

## ADDED Requirements

### Requirement: CI scaffolding MUST be opt-in and additive

`gaze init` MUST accept a `--ci` boolean flag. Without `--ci`, the
command MUST behave identically to the current implementation: same
files written, same skip/overwrite semantics, no CI workflow emitted.
With `--ci`, the command MUST additionally write
`.github/workflows/gaze-ai-report.yml` into the target root while the
existing `.opencode/` scaffolding proceeds unchanged.

#### Scenario: Plain init is unchanged

- **GIVEN** a target directory with no prior gaze scaffolding
- **WHEN** `gaze init` runs without `--ci`
- **THEN** the `.opencode/` assets are scaffolded exactly as before and
  no `.github/workflows/gaze-ai-report.yml` file is created

#### Scenario: Opt-in emits the workflow

- **GIVEN** a target directory with a `go.mod` at its root
- **WHEN** `gaze init --ci` runs
- **THEN** the `.opencode/` assets are scaffolded and
  `.github/workflows/gaze-ai-report.yml` is created

### Requirement: Generated workflow MUST be self-contained

The generated `.github/workflows/gaze-ai-report.yml` MUST own a
self-sufficient test job that produces and uploads the coverage
artifact, and a `gaze-ai-report` job with `needs: test` and
`permissions: contents: read` that calls the reusable AI-report
workflow at the resolved version pin with the coverage artifact and
the `opencode-config` secret. The generated file MUST NOT require
edits to any other workflow in the consumer repository.

#### Scenario: Test job produces the coverage artifact

- **GIVEN** the generated workflow file
- **WHEN** its test job is inspected
- **THEN** it checks out the repository, sets up Go, runs
  `go test -race -count=1 -short -coverprofile=coverage.out ./...`,
  and uploads `coverage.out` as an artifact named `coverage-out`

#### Scenario: Report job calls the reusable workflow

- **GIVEN** the generated workflow file
- **WHEN** its `gaze-ai-report` job is inspected
- **THEN** it declares `needs: test` and `permissions: contents: read`,
  calls `unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml`
  at the resolved pin, passes `coverage-artifact: coverage-out` and
  `gaze-version` at the same resolved pin, and forwards
  `opencode-config: ${{ secrets.OPENCODE_CONFIG_CONTENT }}`

#### Scenario: Generated workflow runs end to end unmodified

- **GIVEN** a Go-module-at-root repository where
  `.github/workflows/gaze-ai-report.yml` was scaffolded by
  `gaze init --ci` and the `OPENCODE_CONFIG_CONTENT` secret is set
- **WHEN** the workflow triggers on push or pull request
- **THEN** the test job runs the test suite with coverage, uploads the
  artifact, and the `gaze-ai-report` job consumes it and produces the
  AI quality report

### Requirement: Generated workflow MUST carry gaze's own CI pins and hygiene defaults

Actions in the test job MUST carry SHA pins identical to those used
in gaze's own CI (checkout, setup-go, upload-artifact), each with a
version annotation comment. The generated file MUST include a
workflow-level `permissions: contents: read` grant, a caller-owned
workflow-level `concurrency:` group following the CI-012 pattern, and
a provenance header comment identifying the scaffold as
gaze-generated.

#### Scenario: Actions are SHA-pinned

- **GIVEN** the generated workflow file
- **WHEN** its `uses:` references are inspected
- **THEN** checkout, setup-go, and upload-artifact steps carry the same
  SHA pins gaze's own CI uses, each with a version annotation comment

#### Scenario: Concurrency and provenance are present

- **GIVEN** the generated workflow file
- **WHEN** its top-level keys and leading comments are inspected
- **THEN** a workflow-level `permissions: contents: read` grant, a
  CI-012 concurrency group, and a gaze provenance header comment are
  all present

### Requirement: Generated workflow MUST be report-only by default

The generated workflow MUST NOT set live values for the
`max-crapload`, `max-gaze-crapload`, or `min-contract-coverage`
inputs. The threshold inputs MUST be present in the file as commented
lines with a TODO marker explaining how to enable them, so the
workflow exits 0 (report-only) until a human removes the comments and
chooses values.

#### Scenario: No thresholds until a human adds them

- **GIVEN** the generated workflow file
- **WHEN** its `with:` block for the reusable workflow is inspected
- **THEN** only `coverage-artifact` and `gaze-version` are set to live
  values and the three threshold inputs appear only as commented
  lines with TODO
  guidance

#### Scenario: Green before the secret exists

- **GIVEN** a repository with the scaffolded workflow committed but no
  `OPENCODE_CONFIG_CONTENT` secret configured
- **WHEN** the workflow runs
- **THEN** the run succeeds: the reusable workflow's secret-absent path
  skips the report steps with a notice while validating the coverage
  artifact handoff

### Requirement: Version pin MUST be validated against the minimum workflow-shipping release

The `uses:` version pin MUST be resolved from the build-time gaze
version and MUST be a valid semantic version `>= v2.3.0`, the first
release tag containing the reusable workflow. The `--gaze-version`
flag MUST accept an override and MUST apply the identical validation.

#### Scenario: Release build emits its own tag

- **GIVEN** a gaze binary built with `-X main.version=v2.4.0`
- **WHEN** `gaze init --ci` runs in a Go-module-at-root directory
- **THEN** the generated `uses:` reference ends in `@v2.4.0`

#### Scenario: Override is validated the same way

- **GIVEN** a gaze binary of any version
- **WHEN** `gaze init --ci --gaze-version=<value>` runs with `<value>`
  a valid semver `>= v2.3.0`
- **THEN** the generated `uses:` reference ends in `@<value>`

### Requirement: Unresolvable version pins MUST fail closed

Development builds (version `dev` or otherwise non-semver), pre-floor
versions, and invalid `--gaze-version` values MUST cause
`gaze init --ci` to fail with an error directing the user to install
gaze from a release tag or pass `--gaze-version`. On validation
failure, no workflow file MUST be written.

#### Scenario: Dev build fails closed

- **GIVEN** a gaze binary whose version is `dev`
- **WHEN** `gaze init --ci` runs without `--gaze-version`
- **THEN** the command fails with guidance to install gaze from a
  release tag or pass `--gaze-version`, and no workflow file is written

#### Scenario: Pre-floor version is rejected

- **GIVEN** a gaze binary whose version is `v2.2.0` (or any valid
  semver below `v2.3.0`)
- **WHEN** `gaze init --ci` runs
- **THEN** the command fails explaining that the reusable workflow
  first ships in `v2.3.0`, and no workflow file is written

#### Scenario: Invalid override is rejected

- **GIVEN** a gaze binary of any version
- **WHEN** `gaze init --ci --gaze-version=<value>` runs with `<value>`
  invalid semver or below `v2.3.0`
- **THEN** the command fails and no workflow file is written

### Requirement: Go toolchain version MUST come from the consumer's go.mod

The generated workflow's setup-go `go-version` MUST be read from the
`go` directive of the consumer's `go.mod` at the target root. Only a
directive value matching the accepted grammar — `major[.minor[.patch]]`
with optional pre-release suffix, the forms setup-go accepts — MAY be
substituted; a value outside the grammar MUST NOT be emitted. When the
directive is absent, unreadable, or outside the grammar, the scaffold
MUST fall back to the reusable workflow's default Go version (`1.25`)
and warn.

#### Scenario: go.mod directive is substituted

- **GIVEN** a target directory whose `go.mod` declares `go 1.24`
- **WHEN** `gaze init --ci` runs
- **THEN** the generated workflow's setup-go step sets
  `go-version: "1.24"`

#### Scenario: Unreadable directive falls back with a warning

- **GIVEN** a target directory whose `go.mod` has no parseable `go`
  directive
- **WHEN** `gaze init --ci` runs
- **THEN** the generated workflow uses `1.25` and the command prints a
  warning about the fallback

#### Scenario: Malformed directive value falls back with a warning

- **GIVEN** a target directory whose `go.mod` carries a `go` directive
  whose value does not match the accepted grammar (e.g. `go banana`)
- **WHEN** `gaze init --ci` runs
- **THEN** the generated workflow uses `1.25` and the command prints a
  warning about the fallback

### Requirement: CI scaffolding MUST require a Go module at the repository root

Because the reusable workflow analyzes `./...` from the repository
root, `gaze init --ci` MUST only emit the workflow when a `go.mod`
exists at the target root. When it is absent, the command MUST print a
warning, skip CI scaffolding, and still complete the `.opencode/`
scaffolding.

#### Scenario: No go.mod skips CI scaffolding

- **GIVEN** a target directory without a `go.mod`
- **WHEN** `gaze init --ci` runs
- **THEN** a warning explains that CI scaffolding requires a Go module
  at the repository root, no workflow file is written, and the
  `.opencode/` scaffolding completes normally

### Requirement: CI scaffold MUST be skip-if-present

The generated workflow file MUST be user-owned: when it already exists
and `--force` is not set, a rerun MUST leave it byte-identical and
report it as skipped. The file MUST NOT be added to the tool-owned
overwrite-on-diff list, because users edit threshold values in it.
With `--force`, the file MAY be overwritten.

#### Scenario: Rerun does not clobber user edits

- **GIVEN** a target directory where the workflow was previously
  scaffolded and a human then uncommented threshold inputs
- **WHEN** `gaze init --ci` runs again without `--force`
- **THEN** the file is unchanged and reported as skipped (exists)

#### Scenario: Force overwrites

- **GIVEN** the same directory
- **WHEN** `gaze init --ci --force` runs
- **THEN** the file is regenerated from the template and reported as
  overwritten

### Requirement: The ci/ asset subtree MUST be excluded from the .opencode/ mirror sync

The workflow template MUST live at
`internal/scaffold/assets/ci/gaze-ai-report.yml` with
`__GAZE_VERSION__` and `__GO_VERSION__` placeholders. The `ci/`
subtree MUST be excluded from the scaffold's `.opencode/` asset walk
and from the embedded-asset drift test (`assetPaths()` /
`TestEmbeddedAssetsMatchSource`), because no `.opencode/` mirror
exists for it.

#### Scenario: Plain asset walk never touches .opencode/ci/

- **GIVEN** the embedded asset tree containing `assets/ci/`
- **WHEN** `gaze init` walks the embedded assets
- **THEN** no file is written under `.opencode/ci/` and the drift test
  enumerates only the nine mirrored `.opencode/` assets

### Requirement: init --ci MUST print CI next-steps guidance

When `--ci` scaffolding succeeds (or is skipped as existing), the
command summary MUST include a CI section covering four items: the
`OPENCODE_CONFIG_CONTENT` secret-setup one-liner, a single-line JSON
minify tip, the second-secret masking pattern pointer, and a
paste-able threshold-guidance prompt. When CI scaffolding is skipped
for a missing `go.mod`, the section MUST NOT be printed.

#### Scenario: Next steps print on success

- **GIVEN** a successful `gaze init --ci` run
- **WHEN** the summary output is inspected
- **THEN** it contains the `gh secret set OPENCODE_CONFIG_CONTENT`
  one-liner, the `jq -c` minify tip, the masking pointer, and the
  threshold-guidance prompt (read the generated workflow, run
  `gaze report ./... --format=json`, propose permissive initial
  thresholds from the summary metrics, human approves final values,
  uncomment the threshold inputs)

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
