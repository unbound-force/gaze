# Feature Specification: Gaze OpenCode Integration & Distribution

**Feature Branch**: `005-gaze-opencode-integration`
**Created**: 2026-02-23
**Status**: Draft
**Input**: User description: "Create a distributable OpenCode
integration for Gaze that allows developers of any Go project to
install and use Gaze's quality reporting capabilities within their
OpenCode workflows via `gaze init`, a `/gaze` command, and a
`gaze-reporter` agent. Distribute the binary via Homebrew
(`unbound-force/tap/gaze`) with precompiled binaries for macOS and
Linux. Migrate the Go module path from `github.com/jflowers/gaze`
to `github.com/unbound-force/gaze`."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Module Path Migration (Priority: P0)

Migrate the Go module path from `github.com/jflowers/gaze` to
`github.com/unbound-force/gaze`. This is a one-time change that
updates `go.mod`, all internal import paths, README documentation,
and any hardcoded references throughout the codebase.

**Why this priority**: P0 because every other deliverable depends
on the canonical module path being correct. The release pipeline
builds from `unbound-force/gaze`, Homebrew formulas reference it,
`go install` must resolve from it, and the `gaze-reporter` agent's
fallback install command must use it. This change must land before
any tagged release.

**Independent Test**: Can be tested by running `go build ./...`
and `go test -short ./...` after the path change to verify the
module compiles and all imports resolve correctly.

**Acceptance Scenarios**:

1. **Given** the current module path `github.com/jflowers/gaze`,
   **When** the migration is applied, **Then** `go.mod` declares
   `module github.com/unbound-force/gaze`.
2. **Given** the migration is applied, **When** `go build ./...`
   runs, **Then** the build succeeds with zero import errors.
3. **Given** the migration is applied, **When** `go test -short
   ./...` runs, **Then** all tests pass.
4. **Given** the migration is applied, **When** the developer
   inspects any `.go` file, **Then** all import paths reference
   `github.com/unbound-force/gaze/...` instead of
   `github.com/jflowers/gaze/...`.
5. **Given** the migration is applied, **When** the developer
   inspects `README.md`, **Then** install instructions reference
   `github.com/unbound-force/gaze`.

---

### User Story 2 - Release Pipeline via GoReleaser (Priority: P1)

The project has a GoReleaser configuration and a GitHub Actions
release workflow that automates binary distribution. When a
maintainer pushes a semantic version tag (e.g., `v0.1.0`), the
pipeline builds cross-platform binaries, injects the version
string, creates a GitHub Release with archives and checksums, and
updates the Homebrew formula.

**Why this priority**: P1 because `gaze init` and Homebrew
distribution both depend on the binary being buildable and
releasable with correct version injection. Without a release
pipeline, there is no way to distribute precompiled binaries.

**Independent Test**: Can be tested by running
`goreleaser release --snapshot --clean` locally to verify the
configuration produces the expected archives without publishing.

**Acceptance Scenarios**:

1. **Given** a GoReleaser configuration exists, **When**
   `goreleaser check` runs, **Then** the configuration is valid.
2. **Given** a tag `v0.1.0` is pushed, **When** the release
   workflow runs, **Then** GoReleaser builds binaries for
   darwin/amd64, darwin/arm64, linux/amd64, and linux/arm64.
3. **Given** the release builds, **When** the binaries are
   produced, **Then** each binary reports the correct version
   via `gaze --version` (e.g., `gaze version v0.1.0`).
4. **Given** the release completes, **When** the GitHub Release
   page is inspected, **Then** it contains archives for all four
   platforms, a checksums file, and an auto-generated changelog.
5. **Given** the release completes, **When** the
   `unbound-force/homebrew-tap` repository is inspected, **Then**
   the `gaze.rb` formula has been created or updated with the
   correct version, download URLs, and SHA256 checksums.

---

### User Story 3 - Homebrew Installation (Priority: P2)

Users install gaze via `brew install unbound-force/tap/gaze` on
macOS or Linux. The Homebrew formula is auto-generated and
maintained by GoReleaser on each release. No manual formula
maintenance is required.

**Why this priority**: P2 because this is the primary distribution
channel for end users. It depends on the release pipeline (US2)
being functional.

**Independent Test**: Can be tested by tapping
`unbound-force/tap` and installing gaze after a release, then
verifying `gaze --version` and `gaze init` work correctly.

**Acceptance Scenarios**:

1. **Given** a release has been published, **When** the user runs
   `brew install unbound-force/tap/gaze`, **Then** the `gaze`
   binary is installed and available on `$PATH`.
2. **Given** the Homebrew-installed binary, **When** the user runs
   `gaze --version`, **Then** it reports the correct release
   version (not "dev").
3. **Given** the Homebrew-installed binary, **When** the user runs
   `gaze init` in a Go project, **Then** the OpenCode files are
   scaffolded with the correct version marker.
4. **Given** a new release is published, **When** the user runs
   `brew upgrade gaze`, **Then** the binary is updated to the
   latest version.
5. **Given** a macOS ARM64 (Apple Silicon) machine, **When** the
   user installs via Homebrew, **Then** a native arm64 binary is
   installed (not an x86_64 binary running under Rosetta).

---

### User Story 4 - `gaze init` Scaffolds OpenCode Files (Priority: P3)

A developer installs Gaze via Homebrew or `go install` and runs
`gaze init` in their Go project directory. Gaze creates the
`.opencode/agents/` and `.opencode/command/` directories and
writes the quality-reporting agent and command files into them.
The developer can now use `/gaze` in OpenCode without any
additional configuration.

**Why this priority**: This is the scaffolding mechanism. Without
`gaze init`, developers of other Go projects cannot use the
OpenCode integration. It must work before any agent or command
can be tested externally. It is also a key deliverable involving
Go code changes (new CLI subcommand + internal package). Depends
on US1 (module path) and US2 (release pipeline) for correct
version injection and install paths.

**Independent Test**: Can be tested by running `gaze init` in a
temporary directory with a `go.mod` file and verifying the
expected files are created with correct content.

**Acceptance Scenarios**:

1. **Given** an empty Go project directory with a `go.mod`, **When**
   the developer runs `gaze init`, **Then** Gaze creates
   `.opencode/agents/gaze-reporter.md`,
   `.opencode/agents/doc-classifier.md`,
   `.opencode/command/gaze.md`, and
   `.opencode/command/classify-docs.md` with correct content.
2. **Given** a directory where `.opencode/agents/gaze-reporter.md`
   already exists, **When** the developer runs `gaze init`,
   **Then** that file is skipped and a message reports it was
   skipped. Other missing files are still created.
3. **Given** a directory where files already exist, **When** the
   developer runs `gaze init --force`, **Then** all files are
   overwritten and a message reports each overwrite.
4. **Given** `gaze init` completes, **When** the developer inspects
   the created files, **Then** each file contains a version marker
   comment `<!-- scaffolded by gaze vX.Y.Z -->` where `X.Y.Z` is
   the installed Gaze version.
5. **Given** `gaze init` runs, **When** it completes, **Then** it
   prints a summary listing files created, files skipped, and
   files overwritten (if `--force`).
6. **Given** a directory with no `go.mod`, **When** the developer
   runs `gaze init`, **Then** Gaze prints a warning that this does
   not appear to be a Go module root but proceeds anyway (the
   files are still useful even without a `go.mod` in the CWD).

---

### User Story 5 - `/gaze` Command Routes to Reporter (Priority: P4)

A developer using OpenCode in any Go project types `/gaze` (with
optional subcommand and package arguments) and receives a
human-readable quality report. The command routes to the
`gaze-reporter` agent with the correct mode and package.

**Why this priority**: This is the user-facing entry point. It
depends on the files from US4 being present. Without the command,
the agent cannot be invoked conveniently.

**Independent Test**: Can be tested by installing the OpenCode
files (via `gaze init` or manually) and invoking `/gaze ./...`
in OpenCode, verifying the reporter agent is invoked with the
correct arguments.

**Acceptance Scenarios**:

1. **Given** `/gaze ./internal/store` is typed, **When** the
   command runs, **Then** the `gaze-reporter` agent receives
   mode=full and package=`./internal/store`.
2. **Given** `/gaze crap ./pkg/api` is typed, **When** the command
   runs, **Then** the `gaze-reporter` agent receives mode=crap
   and package=`./pkg/api`.
3. **Given** `/gaze quality ./pkg/api` is typed, **When** the
   command runs, **Then** the `gaze-reporter` agent receives
   mode=quality and package=`./pkg/api`.
4. **Given** `/gaze` is typed with no arguments, **When** the
   command runs, **Then** the agent receives mode=full and
   package=`./...` (defaults).
5. **Given** `/gaze crap` is typed with no package, **When** the
   command runs, **Then** the agent receives mode=crap and
   package=`./...`.

---

### User Story 6 - CRAP-Only Report via `gaze-reporter` (Priority: P5)

The `gaze-reporter` agent runs `gaze crap --format=json` on the
specified package, interprets the JSON output, and produces a
human-readable summary highlighting worst CRAP scores, CRAPload,
and quadrant distribution.

**Why this priority**: CRAP analysis is the most established and
independently functional part of Gaze. It works without Specs
002-003 (classification/quality) and provides immediate value.
This is the minimum viable slice of the reporter agent.

**Independent Test**: Can be tested by running `/gaze crap ./...`
on a Go project with test files and verifying the agent produces
a readable summary with correct CRAP data.

**Acceptance Scenarios**:

1. **Given** a Go project with tests, **When** the agent runs in
   CRAP mode, **Then** it executes `gaze crap --format=json` and
   produces a summary containing: total functions, CRAPload
   count, top 5 worst CRAP scores, and quadrant distribution.
2. **Given** a Go project where `gaze crap` fails (e.g., build
   errors), **When** the agent runs, **Then** it reports the
   error clearly and suggests remediation (e.g., "Fix build
   errors before running CRAP analysis").
3. **Given** no `gaze` binary is available, **When** the agent
   runs, **Then** it attempts `go install github.com/unbound-force/gaze/cmd/gaze@latest`
   or `go build -o /tmp/gaze-reporter ./cmd/gaze` (if in the
   Gaze repo) before retrying.

---

### User Story 7 - Quality-Only Report via `gaze-reporter` (Priority: P6)

The `gaze-reporter` agent runs `gaze quality --format=json` on
the specified package and produces a human-readable summary
highlighting contract coverage gaps, over-specifications, and
worst-performing tests.

**Why this priority**: Quality analysis depends on Specs 001-003
being functional, making it a secondary capability. It adds
significant value but is not the minimum viable slice.

**Independent Test**: Can be tested by running `/gaze quality
./internal/analysis` on a package with test files and verifying
the summary includes contract coverage and over-specification
data.

**Acceptance Scenarios**:

1. **Given** a package with tests, **When** the agent runs in
   quality mode, **Then** it executes `gaze quality --format=json`
   and produces a summary containing: average contract coverage,
   coverage gaps (unasserted contractual effects), over-
   specification count, and worst tests by coverage.
2. **Given** a package with no test files, **When** the agent
   runs in quality mode, **Then** it reports "no test files found"
   and suggests adding tests.
3. **Given** `gaze quality` produces warnings about mechanical-
   only classification, **When** the agent reports, **Then** it
   includes the warning and notes that `/classify-docs` can
   enhance results.

---

### User Story 8 - Full Report via `gaze-reporter` (Priority: P7)

The `gaze-reporter` agent runs all Gaze commands (crap, quality,
analyze --classify, docscan) and delegates to the `doc-classifier`
agent for enhanced classification. It produces a comprehensive
health assessment combining CRAP, quality, and classification
data.

**Why this priority**: This is the most complete but most complex
mode. It depends on all other modes working correctly and on the
`doc-classifier` agent integration. It is the ultimate goal but
not needed for initial value.

**Independent Test**: Can be tested by running `/gaze ./...` on
the Gaze project itself (dogfooding) and verifying all sections
appear in the report.

**Acceptance Scenarios**:

1. **Given** a Go project, **When** the agent runs in full mode,
   **Then** the report includes: CRAP section, Quality section,
   Classification section (contractual/ambiguous/incidental
   distribution), and Overall Health Assessment.
2. **Given** full mode, **When** the agent runs, **Then** it
   delegates to the `doc-classifier` agent for document-enhanced
   classification and incorporates the enhanced results.
3. **Given** full mode, **When** the quality pipeline fails but
   CRAP succeeds, **Then** the report includes the CRAP section
   and notes that quality data was unavailable, with the reason.
4. **Given** full mode, **When** the report is produced, **Then**
   the Overall Health Assessment identifies high-risk functions
   (high CRAP + low contract coverage) and provides prioritized
   recommendations.

---

### User Story 9 - Dogfooding in the Gaze Project (Priority: P8)

The Gaze project itself uses the same `.opencode/agents/` and
`.opencode/command/` files that `gaze init` distributes. The
embedded files in `internal/scaffold/assets/` are copies of (or
symlinks to) the files in `.opencode/`, ensuring the distributed
version is always the same as what Gaze developers use.

**Why this priority**: Dogfooding validates the integration works
in practice but does not block external distribution. It is a
quality assurance and maintenance concern.

**Independent Test**: Can be tested by running `gaze init --force`
in the Gaze project root and verifying the output files are
identical to the existing `.opencode/` files (after stripping the
version marker).

**Acceptance Scenarios**:

1. **Given** the Gaze repository, **When** a developer runs
   `/gaze ./...` in OpenCode, **Then** it produces a full quality
   report using the same agent files that `gaze init` distributes.
2. **Given** a change to `.opencode/agents/gaze-reporter.md`,
   **When** the developer forgets to update the embedded copy in
   `internal/scaffold/assets/`, **Then** a CI test or go generate
   check detects the drift and fails.
3. **Given** the Gaze build, **When** the embedded assets are
   compiled, **Then** they are identical to the `.opencode/` files
   (enforced by a test comparing checksums or content).

---

### Edge Cases

- What happens when `gaze init` is run outside a Go module
  (no `go.mod`)? Gaze MUST print a warning but proceed, since
  the user may be setting up the OpenCode files before
  initializing the module.
- What happens when the `gaze` binary is not on `$PATH` and
  the reporter agent runs? It MUST attempt to install via
  `go install github.com/unbound-force/gaze/cmd/gaze@latest`
  and report the error if installation fails.
- What happens when the target project has no tests? `gaze crap`
  still works (defaults to 0% coverage); `gaze quality` fails
  gracefully. The reporter MUST handle both cases and produce a
  partial report.
- What happens when `gaze init --force` is run and the existing
  files have local modifications? They are overwritten without
  backup. The `--force` flag name makes this explicit.
- What happens when `gaze init` is run with a development build
  (version = "dev")? The version marker MUST use "dev" as the
  version string: `<!-- scaffolded by gaze dev -->`.
- How does the reporter agent determine which gaze binary to use?
  It MUST first check if `gaze` is on `$PATH` (via `which gaze`
  or equivalent), then fall back to building from source if in a
  Gaze repo, then fall back to `go install`.
- What happens when the `doc-classifier` agent is unavailable in
  full mode? The reporter MUST skip document-enhanced
  classification and note the omission in the report.
- What happens when the `unbound-force/homebrew-tap` repository
  does not exist when a release is triggered? The GoReleaser
  Homebrew step MUST fail with a clear error. The tap repo is a
  prerequisite that must be created manually before the first
  release.
- What happens when the `HOMEBREW_TAP_GITHUB_TOKEN` secret is
  missing from the GitHub repo? The release workflow MUST fail
  with a clear error indicating the missing secret.
- What happens when a tag is pushed that does not match semantic
  versioning (e.g., `test-tag`)? The release workflow MUST only
  trigger on tags matching `v*` and ignore non-version tags.
- What happens when GoReleaser builds fail on one platform but
  succeed on others? GoReleaser MUST fail the entire release —
  partial releases are not acceptable.
- What happens when the module path migration is incomplete (some
  files still reference `github.com/jflowers/gaze`)? `go build
  ./...` MUST fail with import errors, catching the issue before
  it reaches CI.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Gaze MUST provide a `gaze init` CLI subcommand that
  creates `.opencode/agents/` and `.opencode/command/` directories
  and writes the embedded quality-reporting files into them.
- **FR-002**: The embedded files MUST include exactly:
  `gaze-reporter.md` (agent), `doc-classifier.md` (agent),
  `gaze.md` (command), `classify-docs.md` (command). No review
  council or speckit files are included.
- **FR-003**: Each scaffolded file MUST contain a version marker
  comment (`<!-- scaffolded by gaze vX.Y.Z -->`) as the first
  line, before the YAML frontmatter.
- **FR-004**: `gaze init` MUST skip files that already exist
  unless the `--force` flag is passed, in which case existing
  files are overwritten.
- **FR-005**: `gaze init` MUST print a summary of actions taken:
  files created, files skipped, and files overwritten.
- **FR-006**: The embedded assets MUST live in
  `internal/scaffold/assets/` and use Go's `embed.FS` for
  compilation into the binary.
- **FR-007**: The scaffold package (`internal/scaffold/`) MUST
  expose a `Run(opts Options) error` function that the CLI
  delegates to.
- **FR-008**: The `/gaze` command MUST route to the
  `gaze-reporter` agent and support three modes: full (default),
  crap, and quality.
- **FR-009**: The `/gaze` command MUST default the package
  argument to `./...` when not specified.
- **FR-010**: The `gaze-reporter` agent MUST build or locate the
  `gaze` binary before running commands. It MUST check `$PATH`
  first, then attempt `go build` if in a Gaze-like repo, then
  fall back to `go install`.
- **FR-011**: The `gaze-reporter` agent MUST run gaze commands
  with `--format=json` and interpret the JSON output to produce
  human-readable summaries.
- **FR-012**: In CRAP mode, the reporter MUST produce a summary
  containing: total functions analyzed, CRAPload count, top 5
  worst CRAP scores (function name, score, complexity, coverage),
  and GazeCRAP quadrant distribution (if available).
- **FR-013**: In quality mode, the reporter MUST produce a summary
  containing: average contract coverage, top coverage gaps
  (unasserted contractual effects), over-specification count, and
  worst tests by contract coverage.
- **FR-014**: In full mode, the reporter MUST run all four gaze
  commands (`crap`, `quality`, `analyze --classify`, `docscan`),
  delegate to the `doc-classifier` agent for enhanced
  classification, and produce a combined report with CRAP,
  Quality, Classification, and Overall Health sections.
- **FR-015**: The reporter MUST handle failures gracefully: if
  one command fails (e.g., quality fails due to no tests), it
  MUST still report the results from commands that succeeded.
- **FR-016**: The reporter MUST have tools configuration:
  `read: true`, `bash: true`, all others false.
- **FR-017**: A CI or build-time test MUST verify that the
  embedded assets in `internal/scaffold/assets/` are identical
  to the corresponding files in `.opencode/`, preventing drift.
- **FR-018**: `gaze init` MUST NOT require a `.gaze.yaml`
  configuration file — all gaze commands use sensible defaults.
- **FR-019**: The Go module path MUST be migrated from
  `github.com/jflowers/gaze` to `github.com/unbound-force/gaze`.
  This includes `go.mod`, all `import` statements in `.go` files,
  and all documentation references.
- **FR-020**: After the module path migration, `go build ./...`
  and `go test -short ./...` MUST pass with zero errors.
- **FR-021**: The project MUST provide a `.goreleaser.yaml`
  configuration file that builds binaries for darwin/amd64,
  darwin/arm64, linux/amd64, and linux/arm64.
- **FR-022**: GoReleaser MUST inject the version string into the
  binary via `-ldflags "-X main.version={{.Version}}"`, replacing
  the default `"dev"` value.
- **FR-023**: The project MUST provide a GitHub Actions workflow
  (`.github/workflows/release.yml`) that triggers on tags
  matching `v*` and runs GoReleaser to create a GitHub Release.
- **FR-024**: The release workflow MUST produce archives (`.tar.gz`
  for all platforms), a checksums file (`checksums.txt`), and an
  auto-generated changelog from conventional commit messages.
- **FR-025**: GoReleaser MUST publish a Homebrew formula to the
  `unbound-force/homebrew-tap` repository, creating or updating
  the `gaze.rb` formula with correct download URLs and SHA256
  checksums.
- **FR-026**: The release workflow MUST require a
  `HOMEBREW_TAP_GITHUB_TOKEN` repository secret with push access
  to `unbound-force/homebrew-tap`.
- **FR-027**: The Homebrew formula MUST install a working `gaze`
  binary that responds correctly to `gaze --version` and
  `gaze init`.
- **FR-028**: The `unbound-force/homebrew-tap` repository MUST
  be created as a prerequisite before the first release. This is
  a manual step, not automated by the release pipeline.
- **FR-029**: The `gaze-reporter` agent MUST reference
  `github.com/unbound-force/gaze` (not `github.com/jflowers/gaze`)
  in any `go install` fallback commands.

### Key Entities

- **ScaffoldOptions**: Configuration for the init command.
  Attributes: target_dir (string, default CWD), force (bool,
  default false), version (string, from build flags).
- **ScaffoldResult**: Outcome of the init command. Attributes:
  created ([]string, files created), skipped ([]string, files
  skipped because they exist), overwritten ([]string, files
  overwritten with --force).
- **AssetManifest**: The set of files embedded for distribution.
  Attributes: agents ([]AssetFile), commands ([]AssetFile).
- **AssetFile**: A single distributable file. Attributes: name
  (string), relative_path (string, e.g., "agents/gaze-reporter.md"),
  content ([]byte, from embed.FS).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: `gaze init` creates exactly 4 files in the correct
  directories (2 agents, 2 commands) when run in an empty project.
- **SC-002**: `gaze init` skips existing files and reports them in
  the summary when `--force` is not set.
- **SC-003**: `gaze init --force` overwrites all files and reports
  the overwrites.
- **SC-004**: Every scaffolded file contains the version marker
  `<!-- scaffolded by gaze vX.Y.Z -->` as the first line.
- **SC-005**: The embedded assets match the `.opencode/` source
  files exactly (verified by automated test).
- **SC-006**: `/gaze crap ./...` produces a human-readable CRAP
  summary when run in a Go project with tests.
- **SC-007**: `/gaze quality <package>` produces a human-readable
  quality summary when run on a package with tests.
- **SC-008**: `/gaze ./...` (full mode) produces a combined report
  with CRAP, Quality, Classification, and Health sections.
- **SC-009**: The reporter agent gracefully handles gaze command
  failures (reports what succeeded, explains what failed).
- **SC-010**: `gaze init` works in any Go project without
  requiring a `.gaze.yaml` file.
- **SC-011**: After module path migration, `go build ./...` and
  `go test -short ./...` pass. No `.go` file contains an import
  referencing `github.com/jflowers/gaze`.
- **SC-012**: `goreleaser check` validates the `.goreleaser.yaml`
  configuration without errors.
- **SC-013**: `goreleaser release --snapshot --clean` produces
  archives for all four target platforms (darwin/amd64,
  darwin/arm64, linux/amd64, linux/arm64) with correct version
  injection.
- **SC-014**: `gaze --version` reports the tagged version (not
  "dev") when built by GoReleaser.
- **SC-015**: `brew install unbound-force/tap/gaze` installs a
  working binary on macOS (both Intel and Apple Silicon) after
  a release is published.
- **SC-016**: The release workflow completes without errors when
  triggered by a `v*` tag push, and does not trigger on non-
  version tags.

## Dependencies

### External Dependencies

- **Spec 001**: Side effect detection — required for
  `gaze analyze` and `gaze quality`
- **Spec 002**: Contract classification — required for
  `gaze analyze --classify`
- **Spec 003**: Test quality metrics — required for `gaze quality`
- **Spec 004**: Composite metrics — required for `gaze crap`

### Infrastructure Prerequisites

- **`unbound-force/homebrew-tap` repository**: Must be created
  manually on GitHub before the first release. GoReleaser pushes
  the formula to this repo.
- **`HOMEBREW_TAP_GITHUB_TOKEN` secret**: Must be configured in
  the `unbound-force/gaze` repository settings with push access
  to `unbound-force/homebrew-tap`.
- **GoReleaser**: Must be available in the GitHub Actions runner
  (installed via `goreleaser/goreleaser-action`).

### Implementation Phases

The implementation proceeds in three phases:

1. **Phase 0** (no dependencies): Module path migration. This is
   a one-time refactor that must land before any release.
2. **Phase 1** (no spec dependencies): Release pipeline,
   Homebrew tap, `gaze init` scaffolding, `/gaze` command/agent
   markdown files. CRAP mode works independently since Spec 004
   US1 has no dependencies on other specs.
3. **Phase 2** (Specs 001-003 required): Quality mode and full
   mode become functional once the quality pipeline is available.

```
Phase 0: Module Path Migration
┌──────────────────────────────────────────┐
│ go.mod + all imports                     │
│ github.com/jflowers/gaze                 │
│   → github.com/unbound-force/gaze        │
└──────────────────┬───────────────────────┘
                   │ enables
                   ▼
Phase 1: Release Pipeline & Scaffolding
┌─────────────────────┐  ┌─────────────────────┐
│ .goreleaser.yaml    │  │ gaze init           │
│ release.yml         │  │ (internal/scaffold) │
│ (builds + publishes)│  │ (embeds + copies)   │
└──────────┬──────────┘  └──────────┬──────────┘
           │ publishes to            │ embeds
           ▼                         ▼
┌─────────────────────┐  ┌─────────────────────┐
│ Homebrew tap        │  │ .opencode/ files    │
│ unbound-force/tap   │  │ gaze-reporter.md    │
│ brew install gaze   │  │ doc-classifier.md   │
└─────────────────────┘  │ gaze.md (command)   │
                         │ classify-docs.md    │
                         └──────────┬──────────┘
                                    │ used by
                                    ▼
                         ┌─────────────────────┐
                         │ /gaze command        │
                         │ (routes to agent)    │
                         └──────────┬──────────┘
                                    │ invokes
                                    ▼
Phase 2: Full Quality Pipeline
┌─────────────────────┐
│ gaze crap           │ ← Spec 004 (Phase 1)
│ gaze quality        │ ← Specs 001-003 (Phase 2)
│ gaze analyze        │ ← Specs 001-002 (Phase 2)
│ gaze docscan        │ ← Already implemented
└─────────────────────┘
```
