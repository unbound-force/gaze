# Implementation Plan: Gaze OpenCode Integration & Distribution

**Branch**: `005-gaze-opencode-integration` | **Date**: 2026-02-23 | **Spec**: `specs/005-gaze-opencode-integration/spec.md`
**Input**: Feature specification from `specs/005-gaze-opencode-integration/spec.md`

## Summary

Distribute Gaze as a first-class OpenCode integration for any Go
project. This requires three layers of work:

1. **Module path migration** — Rename `github.com/unbound-force/gaze` to
   `github.com/unbound-force/gaze` across `go.mod`, 28 source files
   (83 import lines), and documentation.
2. **Release pipeline** — GoReleaser v2 config + GitHub Actions
   workflow producing cross-platform binaries (darwin/linux, amd64/arm64)
   with version injection and Homebrew formula auto-publish to
   `unbound-force/homebrew-tap`.
3. **OpenCode integration** — New `gaze init` subcommand (with
   `internal/scaffold/` package using `embed.FS`), new `gaze-reporter`
   agent, and `/gaze` command that produces unified quality reports
   (CRAP + quality + classification).

## Technical Context

**Language/Version**: Go 1.24.2
**Primary Dependencies**: Cobra (CLI), Bubble Tea/Lipgloss (TUI),
  golang.org/x/tools (SSA), GoReleaser v2 (release), embed (stdlib)
**Storage**: Filesystem only (embedded assets via `embed.FS`)
**Testing**: Standard library `testing` package; `go test -race -count=1`
**Target Platform**: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
**Project Type**: Single Go binary CLI
**Performance Goals**: `gaze init` completes in <1s (file copy only);
  agent report time bounded by underlying gaze CLI commands
**Constraints**: No external runtime dependencies for end users;
  Homebrew install must work without Go toolchain
**Scale/Scope**: 28 Go files need import path updates; 4 embedded
  assets; 4 target platforms; 1 new internal package

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Accuracy — PASS

- The module path migration is a mechanical find-and-replace
  verified by `go build ./...` and `go test -short ./...`. No
  false positive/negative risk.
- The `gaze-reporter` agent delegates to the existing gaze CLI
  commands which are already covered by accuracy guarantees from
  Specs 001-004. The agent does not perform its own analysis.
- The `gaze init` scaffolding copies embedded files byte-for-byte,
  verified by a drift-detection Go test (FR-017).
- GoReleaser version injection is verified by SC-014 (`gaze
  --version` reports tagged version).

### II. Minimal Assumptions — PASS

- `gaze init` does not require `.gaze.yaml`, Go toolchain, or any
  project-specific configuration (FR-018).
- Homebrew installation requires only `brew` — no Go toolchain.
- The `gaze-reporter` agent checks `$PATH` first, then falls back
  to `go build`/`go install` — adapting to the user's environment
  rather than assuming a specific setup (FR-010).
- No source annotation or restructuring required in target projects.

### III. Actionable Output — PASS

- The `gaze-reporter` agent is explicitly required to produce
  actionable summaries: worst CRAP scores with function names and
  locations, specific coverage gaps, concrete recommendations
  (FR-012, FR-013, FR-014).
- `gaze init` output tells the user exactly what was created,
  skipped, or overwritten (FR-005).
- Error handling requires clear remediation suggestions (US6 AS2).

## Project Structure

### Documentation (this feature)

```text
specs/005-gaze-opencode-integration/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
# Existing structure (unchanged)
cmd/gaze/
  main.go                # Add init subcommand here
  main_test.go           # Add init command tests
  interactive.go         # Unchanged

internal/
  analysis/              # Unchanged
  classify/              # Unchanged
  config/                # Unchanged
  crap/                  # Unchanged
  docscan/               # Unchanged
  loader/                # Unchanged
  quality/               # Unchanged
  report/                # Unchanged
  taxonomy/              # Unchanged
  scaffold/              # NEW: embed + scaffold logic
    scaffold.go          # Run(), Options, Result types
    scaffold_test.go     # Tests including drift detection
    assets/              # Embedded copies of .opencode/ files
      agents/
        gaze-reporter.md
        doc-classifier.md
      command/
        gaze.md
        classify-docs.md

# Existing .opencode/ structure (add new files)
.opencode/
  agents/
    doc-classifier.md    # Existing
    gaze-reporter.md     # NEW
    reviewer-*.md        # Existing (not embedded)
  command/
    classify-docs.md     # Existing
    gaze.md              # NEW
    review-council.md    # Existing (not embedded)
    speckit.*.md         # Existing (not embedded)

# New release infrastructure
.goreleaser.yaml         # NEW: GoReleaser v2 config
.github/workflows/
  test.yml               # Existing
  mega-linter.yml        # Existing
  release.yml            # NEW: release on v* tag push
```

**Structure Decision**: Single Go binary with one new internal
package (`scaffold`). No new top-level directories. Release
infrastructure lives at repo root (`.goreleaser.yaml`) and in
`.github/workflows/`. OpenCode agent/command files live in their
existing `.opencode/` directories.

## Architecture

### Phase 0: Module Path Migration

Mechanical find-and-replace across the codebase:

```
go.mod:       module github.com/unbound-force/gaze
                → module github.com/unbound-force/gaze

28 .go files: import "github.com/unbound-force/gaze/internal/..."
                → import "github.com/unbound-force/gaze/internal/..."

README.md:    go install github.com/unbound-force/gaze/cmd/gaze@latest
                → go install github.com/unbound-force/gaze/cmd/gaze@latest

AGENTS.md:    Module: github.com/unbound-force/gaze
                → Module: github.com/unbound-force/gaze
```

Verification: `go build ./...` && `go test -race -count=1 -short ./...`

### Phase 1A: Release Pipeline

#### `.goreleaser.yaml` (GoReleaser v2 schema)

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

builds:
  - main: ./cmd/gaze
    binary: gaze
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.CommitDate}}

archives:
  - formats:
      - tar.gz
    name_template: >-
      gaze_{{ .Version }}_{{ .Os }}_{{ .Arch }}

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  use: github
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: Bug Fixes
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Documentation
      regexp: '^.*?docs(\([[:word:]]+\))??!?:.+$'
      order: 2
    - title: Others
      order: 999
  filters:
    exclude:
      - '^chore:'

homebrew_casks:
  - name: gaze
    directory: Casks
    repository:
      owner: unbound-force
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: https://github.com/unbound-force/gaze
    description: >-
      Test quality analysis via side effect detection for Go
    binaries:
      - gaze
    skip_upload: auto
    commit_msg_template: >-
      Brew cask update for {{ .ProjectName }} version {{ .Tag }}
```

Note: GoReleaser v2.10+ replaced `brews` (deprecated) with
`homebrew_casks` which generates proper Homebrew Casks. The cask
goes in `Casks/` directory (not `Formula/`).

#### `.github/workflows/release.yml`

```yaml
name: Release
on:
  push:
    tags:
      - 'v*'
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

### Phase 1B: Scaffold Package

#### Data Flow

```
gaze init [--force]
  │
  ▼
cmd/gaze/main.go (init subcommand)
  │ delegates to
  ▼
internal/scaffold.Run(Options{
  TargetDir: cwd,
  Force:     flags.force,
  Version:   version,  // from ldflags
})
  │
  ├─ Walk embed.FS assets/
  ├─ For each file:
  │    ├─ Check if target exists
  │    ├─ Skip or overwrite based on Force
  │    └─ Prepend version marker
  │
  ▼
ScaffoldResult{
  Created:     []string,
  Skipped:     []string,
  Overwritten: []string,
}
  │
  ▼
Print summary to stdout
```

#### Key Design Decisions

- **R-001: Asset storage strategy**
  - Decision: Copy `.opencode/` files into `internal/scaffold/assets/`
    and embed from there.
  - Rationale: Go's `embed.FS` cannot traverse `../..` to parent
    directories. The assets directory must be at or below the
    embedding package. A Go test verifies the copies match the
    originals (FR-017).
  - Alternative rejected: Embedding from `.opencode/` directly
    (not possible with Go embed constraints).

- **R-002: Version marker injection**
  - Decision: Prepend `<!-- scaffolded by gaze vX.Y.Z -->\n` to
    file content at scaffold time, not in the embedded source.
  - Rationale: Embedded assets must match `.opencode/` originals
    exactly (for drift detection). The version marker is added
    dynamically during `scaffold.Run()`.
  - Alternative rejected: Storing pre-marked templates (would
    break drift detection test).

- **R-003: Binary resolution in gaze-reporter agent**
  - Decision: Three-step cascade: (1) `which gaze`, (2) `go build`
    if `cmd/gaze/main.go` exists in CWD, (3) `go install`.
  - Rationale: Users who installed via Homebrew have gaze on PATH.
    Developers working in the Gaze repo can build from source.
    Fallback to `go install` covers edge cases.

### Phase 1C: OpenCode Files

#### `.opencode/agents/gaze-reporter.md`

Agent definition with YAML frontmatter:
- `description`: Comprehensive code quality reporter
- `tools`: `read: true`, `bash: true`, all others false
- Body: Mode parsing, binary resolution, command execution,
  JSON interpretation, and report formatting instructions

#### `.opencode/command/gaze.md`

Command definition with YAML frontmatter:
- `description`: Comprehensive code quality report
- `agent`: `gaze-reporter`
- Body: Passes `$ARGUMENTS` to agent; documents three usage modes

### Phase 2: Quality & Full Report (Deferred)

Quality mode (US7) and full report mode (US8) depend on Specs
001-003 being implemented. The agent markdown will include
instructions for these modes, but they will produce "unavailable"
messages until the underlying gaze commands exist.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Module path migration breaks tests | Low | High | Mechanical replacement + full test run |
| GoReleaser config errors | Medium | Medium | `goreleaser check` + snapshot test locally |
| Homebrew tap token misconfigured | Medium | Low | Clear error in workflow; documented prerequisite |
| Embedded asset drift undetected | Low | Medium | Go test compares checksums every `go test` run |
| `go install` fails from new path | Low | High | Test after pushing module path change to upstream |

## Complexity Tracking

No constitution violations to justify. All three principles pass
cleanly — the feature is distribution infrastructure that does not
alter analysis accuracy, does not add assumptions, and produces
actionable output (scaffolding summaries, quality reports).
