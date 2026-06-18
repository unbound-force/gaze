# AGENTS.md

## Project Overview

Gaze is a static analysis tool for Go that detects observable side effects in functions and computes CRAP (Change Risk Anti-Patterns) scores by combining cyclomatic complexity with test coverage. It helps developers find functions that are complex and under-tested — the riskiest code to change.

- **Language**: Go 1.24+
- **Module**: `github.com/unbound-force/gaze`
- **License**: Apache 2.0

## Core Mission

- **Strategic Architecture**: Engineers shift from manual coding to directing an "infinite supply of junior developers" (AI agents).
- **Outcome Orientation**: Focus on conveying business value and user intent rather than low-level technical sub-tasks.
- **Intent-to-Context**: Treat specs and rules as the medium through which human intent is manifested into code.

## Behavioral Constraints

- **Zero-Waste Mandate**: No orphaned code, unused dependencies, or "Feature Zombie" bloat.
- **Neighborhood Rule**: Changes must be audited for negative impacts on adjacent modules or the wider ecosystem.
- **Intent Drift Detection**: Evaluation must detect when the implementation drifts away from the original human-written "Statement of Intent."
- **Automated Governance**: Primary feedback is provided via automated constraints, reserving human energy for high-level security and logic.

### Gatekeeping Value Protection

Agents MUST NOT modify values that serve as quality or
governance gates to make an implementation pass. The
following categories are protected:

1. **Coverage thresholds and CRAP scores** — minimum
   coverage percentages, CRAP score limits, coverage
   ratchets
2. **Severity definitions and auto-fix policies** —
   CRITICAL/HIGH/MEDIUM/LOW boundaries, auto-fix
   eligibility rules
3. **Convention pack rule classifications** —
   MUST/SHOULD/MAY designations on convention pack rules
   (downgrading MUST to SHOULD is prohibited)
4. **CI flags and linter configuration** — `-race`,
   `-count=1`, `govulncheck`, `golangci-lint` rules,
   pinned action SHAs
5. **Agent temperature and tool-access settings** —
   frontmatter `temperature`, `tools.write`, `tools.edit`,
   `tools.bash` restrictions
6. **Constitution MUST rules** — any MUST rule in
   `.specify/memory/constitution.md` or hero constitutions
7. **Review iteration limits and worker concurrency** —
   max review iterations, max concurrent Swarm workers,
   retry limits
8. **Workflow gate markers** — `<!-- spec-review: passed
   -->`, task completion checkboxes used as gates, phase
   checkpoint requirements

**What to do instead**: When an implementation cannot
meet a gate, the agent MUST stop, report which gate is
blocking and why, and let the human decide whether to
adjust the gate or rework the implementation. Modifying
a gate without explicit human authorization is a
constitution violation (CRITICAL severity).

### Workflow Phase Boundaries

Agents MUST NOT cross workflow phase boundaries:

- **Specify/Clarify/Plan/Tasks/Analyze/Checklist** phases:
  spec artifacts ONLY (`specs/NNN-*/` directory). No
  source code, test, agent, command, or config changes.
- **Implement** phase: source code changes allowed,
  guided by spec artifacts.
- **Review** phase: findings and minor fixes only. No new
  features.

A phase boundary violation is treated as a process error.
The agent MUST stop and report the violation rather than
proceeding with out-of-phase changes.

## Technical Guardrails

- **WORM Persistence**: Use Write-Once-Read-Many patterns where data integrity is paramount.
- **CI Parity Gate**: Before marking any implementation task complete or declaring a PR ready, agents MUST replicate the CI checks locally. Read `.github/workflows/` to identify the exact commands CI runs, then execute those same commands. Any failure is a blocking error — a task is not complete until all CI-equivalent checks pass locally. Do not rely on a memorized list of commands; always derive them from the workflow files, which are the source of truth.

## Council Governance Protocol

- **The Architect**: Must verify that "Intent Driving Implementation" is maintained.
- **The Adversary**: Acts as the primary "Automated Governance" gate for security.
- **The Guard**: Detects "Intent Drift" to ensure the business value remains intact.
- **The Tester**: Must verify that test quality, coverage strategy, and testability are maintained.

**Rule**: A Pull Request is only "Ready for Human" once the `/review-council` command returns an **APPROVE** status from all four reviewers.

### Review Council as PR Prerequisite

Before submitting a pull request, agents **must** run `/review-council` and resolve all REQUEST CHANGES findings until all four reviewers return APPROVE. There must be **minimal to no code changes** between the council's APPROVE verdict and the PR submission — the council reviews the final code, not a draft that changes afterward.

Workflow:

1. Complete all implementation tasks
2. Run CI checks locally (build, test, vet)
3. Run `/review-council` — fix any findings, re-run until APPROVE
4. Commit, push, and submit PR immediately after council APPROVE
5. Do NOT make further code changes between APPROVE and PR submission

Exempt from council review:

- Constitution amendments (governance documents, not code)
- Documentation-only changes (README, AGENTS.md, spec artifacts)
- Emergency hotfixes (must be retroactively reviewed)

## Spec-First Development (Mandatory)

All changes that modify production code, test code, agent prompts, embedded assets, or CI configuration **must** be preceded by a spec workflow. The constitution (`.specify/memory/constitution.md`) is the highest-authority document in this project — all work must align with it.

Two spec workflows are available:

| Workflow | Location | Best For |
|----------|----------|----------|
| **Speckit** | `specs/NNN-name/` | Numbered feature specs with the full pipeline (specify → clarify → plan → tasks → implement) |
| **OpenSpec** | `openspec/changes/name/` | Targeted changes with lightweight artifacts (proposal → design → specs → tasks) via `/opsx-propose` and `/opsx-apply` |

**What requires a spec** (no exceptions without explicit user override):

- New features or capabilities
- Refactoring that changes function signatures, extracts helpers, or moves code between packages
- Test additions or assertion strengthening across multiple functions
- Agent prompt changes (embedded assets under `internal/scaffold/assets/`)
- CI workflow modifications
- Data model changes (new struct fields, JSON schema updates)

**What is exempt** (may be done directly):

- Constitution amendments (governed by the constitution's own Governance section)
- Typo corrections, comment-only changes, single-line formatting fixes
- Emergency hotfixes for critical production bugs (must be retroactively documented)

When an agent is unsure whether a change is trivial, it **must** ask the user rather than proceeding without a spec. The cost of an unnecessary spec is minutes; the cost of an unplanned change is rework, drift, and broken CI.

### Pipeline

The workflow is a strict, sequential pipeline. Each stage has a corresponding `/speckit.*` command:

```text
constitution → specify → clarify → plan → tasks → analyze → checklist → implement
```

| Command | Purpose |
|---------|---------|
| `/speckit.constitution` | Create or update the project constitution |
| `/speckit.specify` | Create a feature specification from a description |
| `/speckit.clarify` | Reduce ambiguity in the spec before planning |
| `/speckit.plan` | Generate the technical implementation plan |
| `/speckit.tasks` | Generate actionable, dependency-ordered task list |
| `/speckit.analyze` | Non-destructive cross-artifact consistency analysis |
| `/speckit.checklist` | Generate requirement quality validation checklists |
| `/speckit.implement` | Execute the implementation plan task by task |
| `/speckit.taskstoissues` | Convert tasks.md into GitHub Issues |
| `/speckit.testreview` | Analyze spec artifacts for testability gaps (read-only) |

### Ordering Constraints

1. Constitution must exist before specs.
2. Spec must exist before plan.
3. Plan must exist before tasks.
4. Tasks must exist before implementation and analysis.
5. Clarify should run before plan (skipping increases rework risk).
6. Analyze should run after tasks but before implementation.
7. All checklists must pass before implementation (or user must explicitly override).

### Spec Organization

Specs are numbered with 3-digit zero-padded prefixes and stored under `specs/`:

```text
.specify/
  memory/
    constitution.md              # Governance document (highest authority)
  templates/                     # Templates for all artifact types
  scripts/bash/                  # Automation scripts
specs/
  001-side-effect-detection/     # spec.md, plan.md, tasks.md
  002-contract-classification/   # spec.md, plan.md, tasks.md
  003-test-quality-metrics/      # spec.md, plan.md, clarify.md, tasks.md
  004-composite-metrics/         # spec.md, plan.md, tasks.md (retroactive)
  005-gaze-opencode-integration/ # spec.md, plan.md, tasks.md, research.md
  006-agent-quality-report-enhancements/ # spec.md, plan.md, tasks.md, report.md (pre-dates research.md convention; report.md serves as research artifact)
  007-assertion-mapping-depth/   # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md
  008-contract-coverage-gaps/    # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md
  009-crapload-reduction/        # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md
  # 010-report-voice-refinement: deleted — superseded by 011-output-voice-style before implementation began
  011-output-voice-style/        # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  012-consolidate-classify-docs/ # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md
  # 013: reserved/unused — number was skipped; 014 follows directly
  014-macos-notarization/        # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/ (SUPERSEDED by 015)
  015-native-macos-signing/      # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  016-agent-context-reduction/   # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  017-testing-persona/           # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  018-ci-report/                 # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  019-opencode-adapter/          # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
  020-report-coverprofile/       # spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md, checklists/
```

OpenSpec changes use kebab-case names and are archived after merge under `openspec/changes/archive/`:

```text
openspec/
  changes/
    report-actionability/          # proposal.md, design.md, specs/, tasks.md (active)
    reporter-fix-strategy-awareness/ # proposal.md, design.md, specs/, tasks.md (active)
    archive/
      2026-03-12-fix-xtools-go125-panic/
      2026-03-13-assess-graceful-degradation/
      2026-03-14-adapter-format-decomposition/
      2026-03-14-crapload-analyze-decomposition/
      2026-03-14-pipeline-step-testing/
      2026-03-14-ssa-goroutine-panic/
      2026-03-15-q4-complexity-reduction/
  specs/                           # Main specs (synced from delta specs)
```

Branch names follow the same numbering pattern for Speckit (e.g., `001-side-effect-detection`) or kebab-case for OpenSpec (e.g., `assess-graceful-degradation`).

### Task Completion Bookkeeping

When a task from `tasks.md` is completed during implementation, its checkbox **must** be updated from `- [ ]` to `- [x]` immediately. Do not defer this — mark tasks complete as they are finished, not in a batch after all work is done. This keeps the task list an accurate, real-time view of progress and prevents drift between the codebase and the plan.

### Documentation Validation Gate

Before marking any task complete, you **must** validate whether the change requires documentation updates. Check and update as needed:

- `README.md` — new/changed commands, flags, output formats, or architecture
- `AGENTS.md` — new conventions, packages, patterns, or workflow changes
- GoDoc comments — new or modified exported functions, types, and packages
- Spec artifacts under `specs/` — if the change affects planned behavior

A task is not complete until its documentation impact has been assessed and any necessary updates have been made. Skipping this step causes documentation drift, which compounds over time and erodes project accuracy.

### Spec Commit Gate

All spec artifacts (`spec.md`, `plan.md`, `tasks.md`, and any other files under `specs/`) **must** be committed and pushed before implementation begins. This ensures the planning record is preserved in version control before code changes start, and provides a clean baseline to diff against if implementation drifts from the plan. Run `/speckit.implement` only after the spec commit is on the remote.

### Constitution Check

A mandatory gate at the planning phase. The constitution's four core principles — Accuracy, Minimal Assumptions, Actionable Output, and Testability — must each receive a PASS before proceeding. Constitution violations are automatically CRITICAL severity and non-negotiable.

### Website Documentation Gate

When a change affects user-facing behavior, hero
capabilities, CLI commands, or workflows, a GitHub issue
**MUST** be created in the `unbound-force/website`
repository to track required documentation or website
updates. The issue must be created before the
implementing PR is merged.

```bash
gh issue create --repo unbound-force/website \
  --title "docs: <brief description of what changed>" \
  --body "<what changed, why it matters, which pages
          need updating>"
```

**Exempt changes** (no website issue needed):
- Internal refactoring with no user-facing behavior
  change
- Test-only changes
- CI/CD pipeline changes
- Spec artifacts (specs are internal planning documents)

**Examples requiring a website issue**:
- New CLI command or flag added
- Hero capabilities changed (new agent, removed feature)
- Installation steps changed (`uf setup` flow)
- New convention pack added
- Breaking changes to any user-facing workflow

## Build & Test Commands

```bash
# Build
go build ./cmd/gaze

# Run unit + integration tests (use -short to skip e2e)
go test -race -count=1 -short ./...

# Run e2e tests only (self-check: spawns go test -coverprofile)
go test -race -count=1 -run 'TestRunSelfCheck' -timeout 30m ./cmd/gaze/...

# Run all tests (no -short, requires ~15min)
go test -race -count=1 ./...

# Lint
golangci-lint run
```

Always run tests with `-race -count=1`. CI enforces this.

### Test Suites

Tests are organized into two CI suites that run in parallel:

| Suite | Command | Timeout | What it runs |
|-------|---------|---------|-------------|
| Unit + Integration | `go test -race -count=1 -short ./...` | 10m (default) | All tests except those guarded by `testing.Short()` |
| E2E | `go test -race -count=1 -run TestRunSelfCheck -timeout 30m ./cmd/gaze/...` | 20m | Self-check tests that spawn `go test -coverprofile` on the full module |

Use `testing.Short()` to guard tests that spawn external `go test` processes or analyze the entire module. These are too slow for the standard CI timeout.

## Architecture

Single binary CLI with layered internal packages:

```text
cmd/gaze/              CLI layer (Cobra commands, Bubble Tea TUI)
internal/
  analysis/            Core side effect detection engine (AST + SSA)
  taxonomy/            Domain types: SideEffect, AnalysisResult, Tier, etc.
  classify/            Contractual classification engine
  config/              Configuration file handling (.gaze.yaml)
  loader/              Go package loading (go/packages wrapper)
  report/              Output formatters (JSON, text, HTML stub)
  crap/                CRAP score computation and reporting
  quality/             Test quality assessment (contract coverage)
  docscan/             Documentation file scanner
  scaffold/            OpenCode file scaffolding (embed.FS)
  aireport/            AI-powered CI quality report pipeline (gaze report)
```

All business logic lives under `internal/` and cannot be imported externally.

### Key Patterns

- **AST + SSA dual analysis**: Returns, sentinels, and P1/P2 effects use Go AST. Mutation tracking uses SSA via `golang.org/x/tools`.
- **Testable CLI pattern**: Commands delegate to `runXxx(params)` functions. Params structs include `io.Writer` for stdout/stderr, enabling unit testing without subprocess execution.
- **Options structs**: Configurable behavior uses options/params structs rather than long parameter lists.
- **Tiered effect taxonomy**: Side effects are organized into priority tiers P0-P4.

## Coding Conventions

- **Formatting**: `gofmt` and `goimports` (enforced by golangci-lint).
- **Naming**: Standard Go conventions. PascalCase for exported, camelCase for unexported.
- **Comments**: GoDoc-style comments on all exported functions and types. Package-level doc comments on every package.
- **Error handling**: Return `error` values. Wrap with `fmt.Errorf("context: %w", err)`.
- **Import grouping**: Standard library, then third-party, then internal packages (separated by blank lines).
- **No global state**: The logger is the only package-level variable. Prefer functional style.
- **Constants**: Use string-typed constants for enumerations (`SideEffectType`, `Tier`, `Quadrant`).
- **JSON tags**: Required on all struct fields intended for serialization.

## Knowledge Retrieval

Agents SHOULD prefer Dewey MCP tools over grep/glob/read
for cross-repo context, design decisions, and
architectural patterns. Dewey provides semantic search
across all indexed Markdown files, specs, and web
documentation — returning ranked results with provenance
metadata that grep cannot match.

### Tool Selection Matrix

| Query Intent | Dewey Tool | When to Use |
|-------------|-----------|-------------|
| Conceptual understanding | `dewey_semantic_search` | "How does X work?" |
| Keyword lookup | `dewey_search` | Known terms, FR numbers |
| Read specific page | `dewey_get_page` | Known document path |
| Relationship discovery | `dewey_find_connections` | "How are X and Y related?" |
| Similar documents | `dewey_similar` | "Find specs like this one" |
| Tag-based discovery | `dewey_find_by_tag` | "All pages tagged #decision" |
| Property queries | `dewey_query_properties` | "All specs with status: draft" |
| Filtered semantic | `dewey_semantic_search_filtered` | Semantic search within source type |
| Graph navigation | `dewey_traverse` | Dependency chain walking |

### When to Fall Back to grep/glob/read

Use direct file operations instead of Dewey when:
- **Dewey is unavailable** — MCP tools return errors or
  are not configured
- **Exact string matching is needed** — searching for a
  specific error message, variable name, or code pattern
- **Specific file path is known** — reading a file you
  already know the path to (use Read directly)
- **Binary/non-Markdown content** — Dewey indexes
  Markdown; use grep for Go source, JSON, YAML, etc.

### Graceful Degradation (3-Tier Pattern)

**Tier 3 (Full Dewey)** — semantic + structured search:
- `dewey_semantic_search` — natural language queries
- `dewey_search` — keyword queries
- `dewey_get_page`, `dewey_find_connections`,
  `dewey_traverse` — structured navigation
- `dewey_find_by_tag`, `dewey_query_properties` —
  metadata queries

**Tier 2 (Graph-only, no embedding model)** — structured
search only:
- `dewey_search` — keyword queries (no embeddings needed)
- `dewey_get_page`, `dewey_traverse`,
  `dewey_find_connections` — graph navigation
- `dewey_find_by_tag`, `dewey_query_properties` —
  metadata queries
- Semantic search unavailable — use exact keyword matches

**Tier 1 (No Dewey)** — direct file access:
- Use Read tool for direct file access
- Use Grep for keyword search across the codebase
- Use Glob for file pattern matching

## Testing Conventions

- **Framework**: Standard library `testing` package only. No testify, gomega, or other external assertion libraries.
- **Assertions**: Use `t.Errorf` / `t.Fatalf` directly. No assertion helpers from third-party packages.
- **Test naming**: `TestXxx_Description` (e.g., `TestReturns_PureFunction`, `TestFormula_ZeroCoverage`).
- **Test files**: `*_test.go` alongside source in the same directory. Both internal and external package test styles are used.
- **Test fixtures**: Real Go packages in `testdata/src/` directories, loaded via `go/packages`.
- **Benchmarks**: Separate `bench_test.go` files with `BenchmarkXxx` functions.
- **Acceptance tests**: Named after spec success criteria (e.g., `TestSC001_ComprehensiveDetection`, `TestSC004_SingleFunctionPerformance`).
- **JSON Schema validation**: Tests validate JSON output against the embedded JSON Schema (Draft 2020-12).
- **Output width**: Report output is verified to fit within 80-column terminals.

## Core Principles

These principles (from the project constitution) guide all development:

1. **Accuracy**: Gaze MUST correctly identify all observable side effects. False positives erode trust and MUST be treated as bugs. False negatives MUST be tracked, measured, and driven toward zero. Accuracy claims MUST be backed by automated regression tests.
2. **Minimal Assumptions**: Gaze MUST operate with the fewest possible assumptions about the host project's language, test framework, or coding style. No source annotation or restructuring required. When assumptions are unavoidable, they MUST be explicit and enforced.
3. **Actionable Output**: Every piece of output MUST guide the user toward a concrete improvement. Reports MUST identify specific test, target, and unasserted change. Output formats MUST support human-readable and machine-readable (JSON). Metrics MUST be comparable across runs.
4. **Testability**: Every function Gaze analyzes, and every function within Gaze itself, MUST be testable in isolation. Test contracts MUST verify observable side effects, not implementation details. Coverage strategy MUST be specified in plans for new code. Coverage ratchets MUST be enforced; regression MUST be treated as test failure.

## Git & Workflow

- **Commit format**: Conventional Commits — `type: description` (e.g., `feat:`, `fix:`, `docs:`, `chore:`, `refactor:`).
- **Branching**: Feature branches required. No direct commits to `main` except trivial doc fixes.
- **Code review**: Required before merge.
- **Semantic versioning**: For releases.

## CI/CD

Three GitHub Actions workflows:

1. **Test** (`.github/workflows/test.yml`): Build + test with `-race -count=1` on push/PR to `main`.
2. **MegaLinter** (`.github/workflows/mega-linter.yml`): Runs golangci-lint, revive, markdownlint, yamllint, and gitleaks on push/PR to `main`. Auto-commits lint fixes to PR branches.
3. **Release** (`.github/workflows/release.yml`): Triggered via `workflow_dispatch` with a `tag` input (e.g., `v0.15.0`). A `preflight` job validates tag format, uniqueness, semver ordering, verifies CI checks passed on HEAD via the GitHub Checks API, confirms unreleased commits exist, then creates and pushes the annotated tag. The `release` job runs GoReleaser v2 to build cross-platform binaries (darwin/linux x amd64/arm64), create GitHub Releases, and update the Homebrew cask in `unbound-force/homebrew-tap`. To release: use the GitHub Actions "Run workflow" button or `gh workflow run release.yml -f tag=v1.2.3`.

## Linting

golangci-lint v2 is configured in `.golangci.yml` with these linters enabled:

- errcheck, govet, staticcheck, ineffassign, unused, misspell

Formatters: gofmt, goimports.

## Active Technologies
- N/A — Markdown files only (no Go code changes) + None — plain Markdown, no static site generator, no build step (037-project-documentation)
- Filesystem only — `docs/` directory at repository root (037-project-documentation)

- Go 1.24+ (no Go code changes; prompt is markdown) + OpenCode agent runtime (renders markdown prompt), embed.FS (scaffolds prompt copy) (011-output-voice-style)
- Filesystem only (markdown files) (011-output-voice-style)
- Go 1.24+ (scaffold Go code); Markdown (agent/command prompts) + `embed.FS` (Go standard library), OpenCode agent runtime (012-consolidate-classify-docs)
- Filesystem only (embedded assets via `embed.FS`, `.opencode/` directory) (012-consolidate-classify-docs)
- Go 1.24+ (no Go code changes; YAML/workflow configuration only) + GoReleaser v2 (OSS), quill (embedded in GoReleaser as a Go library) (014-macos-notarization)
- Go 1.24+ (no Go code changes; YAML/workflow configuration only) + GitHub Actions, `codesign` (macOS native), `xcrun notarytool` (macOS native), `security` (macOS Keychain), `gh` CLI (GitHub) (015-native-macos-signing)
- Go 1.24+ (scaffold Go code changes); Markdown (agent prompt and reference files) + `embed.FS` (Go standard library), OpenCode agent runtime (016-agent-context-reduction)
- Go 1.24+ (scaffold Go code changes); Markdown (agent/command prompts) + `embed.FS` (Go standard library), OpenCode agent runtime (017-testing-persona)
- Go 1.24+ + Cobra (CLI), `exec.Command` (claude/gemini subprocess), `net/http` (ollama HTTP API), `embed.FS` (embedded default prompt), existing internal packages (`crap`, `quality`, `analysis`, `classify`, `docscan`, `loader`, `taxonomy`) (018-ci-report)
- N/A — ephemeral pipeline only; no persistent state introduced (018-ci-report)
- Filesystem only — temp files for system prompt delivery (removed after subprocess exits) (018-ci-report)
- Go 1.24+ + `os/exec` (subprocess), `os` (temp dir), `path/filepath` (agent file path), `strings` (output trimming), `bytes` (stderr buffer) — all standard library; no new external dependencies (019-opencode-adapter)
- N/A — ephemeral temp dir only; cleaned up via `defer os.RemoveAll` (019-opencode-adapter)
- Go 1.25.0 (module minimum; `go.mod` directive) + `golang.org/x/tools@v0.43.0` (SSA builder), `charmbracelet/log` (available but not currently used for runtime logging — project uses `fmt.Fprintf(stderr, ...)`) (021-ssa-panic-recovery)
- N/A (no persistence changes) (021-ssa-panic-recovery)
- Go 1.25+ + `golang.org/x/tools` (SSA builder), Cobra (CLI) (022-report-gazecrap-pipeline)
- Go 1.25+ (module minimum per go.mod directive) + `go/ast`, `go/types`, `golang.org/x/tools/go/ssa`, `golang.org/x/tools/go/packages` (all existing; no new dependencies) (034-container-assertion-mapping)
- N/A — no persistence changes (034-container-assertion-mapping)
- Go 1.25+ (module minimum per go.mod directive) + `go/ast`, `go/types`, `golang.org/x/tools/go/packages` (all existing; no new dependencies) (035-fix-ambiguous-classification)

- Go 1.24+ + `golang.org/x/tools` (go/packages, go/ssa), Cobra (CLI), Bubble Tea/Lipgloss (TUI)
- Filesystem only (embedded assets via `embed.FS`)
- GoReleaser v2 (release pipeline, Homebrew cask publishing)

## Recent Changes

- baseline-comparison: Added `--baseline` flag to `gaze crap`. Auto-detects `.gaze/baseline.json` for per-function CRAP/GazeCRAP regression detection. New `BaselineConfig` in `.gaze.yaml` for epsilon (0.5) and new-function threshold (30). `ComparisonResult`, `FunctionDelta`, `ComparisonSummary` types in `internal/crap/crap.go`. Comparison logic in `internal/crap/compare.go`, report output in `internal/crap/compare_report.go`. Exit code 1 on regression. Closes constitutional mandate (Principle III: cross-run comparability).
- agent-quality-workflow: Added `RecommendedAction` type and `recommended_actions` sorted list to `crap.Summary` (`internal/crap/crap.go`) — prioritized by fix strategy then CRAP descending, top 20. Added `AssertionCount int` to `taxonomy.QualityReport` (`internal/taxonomy/types.go`) — total detected assertion sites per test-target pair, populated from `len(DetectAssertions(...))` in `quality.Assess`. Updated Quality JSON Schema with `assertion_count` field. Agents can now distinguish "no tests" from "tests without assertions" and work in optimal remediation order. Partially closes #48.
- quality-include-unexported: Added `--include-unexported` flag to `gaze quality` for parity with `analyze`. Auto-detects `package main` and includes unexported functions automatically (a `main` package has no exported API by definition). Auto-detect threaded through `runQuality` (CLI), `runQualityForPackage` (report pipeline), and `analyzePackageCoverage` (contract coverage builder). Added `mainpkg` test fixture. Closes #70.
- p0-contractual-default: Added `tierBoost()` function to `internal/classify/score.go` — P0 effects (ReturnValue, ErrorReturn, SentinelError, ReceiverMutation, PointerArgMutation) receive +25 confidence boost (base 75), P1 effects +10 (base 60), P2-P4 unchanged at 50. Expanded `accumulateSignals` and `ComputeScore` signatures to accept `effectType`. P0 effects now trend toward contractual by default, making contract coverage meaningful out-of-the-box without `.gaze.yaml` or GoDoc tuning. Closes #71.
- gaze-test-generation: Added `gaze-test-generator` agent (`.opencode/agents/gaze-test-generator.md`) for generating Go test functions from gaze quality data (GapHints, Gaps, FixStrategy). Five actions: `add_tests`, `add_assertions`, `add_docs`, `decompose_and_test`, `decompose`. Added `/gaze fix` command (`.opencode/commands/gaze-fix.md`) for batch remediation. Added per-task test generation hooks to `/opsx-apply` and `/speckit.implement` (mandatory by default, configurable to advisory via `.gaze.yaml`). Scaffold updated: `gaze init` now produces 8 files (up from 6).
- report-actionability: Added `FixStrategy` field to `crap.Score` and `FixStrategyCounts` to `crap.Summary` (`internal/crap/crap.go`). Each CRAPload function now carries a deterministic remediation label: `decompose` (complexity >= threshold, coverage can't help), `add_tests` (zero coverage), `add_assertions` (Q3 — has line coverage but lacks contract assertions), `decompose_and_test` (both needed). Strategy computed in `assignFixStrategy` (`internal/crap/analyze.go`), accumulated in `buildSummary`. Text report shows "Remediation Breakdown" section and `[strategy]` labels on worst offenders. JSON output includes new fields automatically. Addresses #42, #44, #48.
- adapter-stderr-diagnostics: Changed `runSubprocess` return from `([]byte, error)` to `([]byte, []byte, error)` in `internal/aireport/adapter.go` — now returns stderr alongside stdout on success. Added `formatStderrSuffix` helper. All three subprocess adapters (OpenCode, Claude, Gemini) include truncated stderr in empty-output error messages (FR-009/FR-016). Surfaces actual error causes (e.g., "Invalid API key") instead of generic "returned empty output."
- reporter-fix-strategy-awareness: Added "Scoring Consistency Rules" section to gaze-reporter agent prompt (`.opencode/agents/gaze-reporter.md`). CRAPload threshold read from `summary.crap_threshold`, contract coverage uses module-wide average, quadrant descriptions use "contract coverage" terminology. Added "Metric Definitions" subsection distinguishing CRAPload, GazeCRAPload, and quadrant counts (GazeCRAPload is NOT the Q4 count).
- ci-gaze-report-v2: Split gaze report CI step into PR-safe threshold check (`--format=json`, no secrets) and push-only AI report (`--ai=opencode`, with `OPENCODE_API_KEY`). Added `-coverprofile=coverage.out` to test step. Threshold: `--max-crapload=35 --max-gaze-crapload=5 --min-contract-coverage=8`. Job timeout increased to 30min.
- 022-report-gazecrap-pipeline: Extracted `buildContractCoverageFunc` from `cmd/gaze/main.go` to `internal/crap/contract.go` as `BuildContractCoverageFunc`. Wired it into `runProductionPipeline` (`internal/aireport/runner.go`) before the CRAP step so that `gaze report` produces GazeCRAP scores, quadrant distribution (`quadrant_counts`), and `gaze_crapload` in its JSON output. Expanded `runCRAPStep` and `pipelineStepFuncs` signatures to accept a `ContractCoverageFunc` parameter. The `--max-gaze-crapload` threshold now evaluates against actual computed GazeCRAPload (previously always passed with value 0). Added `TestSC002_GazeCRAPloadMatchBetweenCrapAndReport` (exact-match test between `gaze crap` and `gaze report`), `TestSC004_PayloadContainsQuadrantCounts` (FakeAdapter payload assertion), and `TestRunProductionPipeline_GazeCRAPloadFlowsThroughPipeline` (pipeline data flow test).
- target-warning-guidance: Updated "multiple target functions detected" warning in `InferTargets` (`internal/quality/pairing.go`) to reference README "How Target Inference Works" section, giving users guidance on resolving ambiguous target detection. Closes #61.
- ssa-diagnostics-in-report: Added `SSADegradedPackages []string` to `crap.Summary` (`internal/crap/crap.go`) and `crap.Options` (`internal/crap/analyze.go`) for per-package SSA failure tracking in CRAP output. Changed `analyzePackageCoverage` to return degraded package path. Changed `buildContractCoverageFunc` to collect degraded packages and pass them through to `crap.Options`. CRAP text report now shows SSA diagnostics section listing failed packages. Closes #62.
- coverage-reason-guidance: Added `ContractCoverageReason` and `EffectConfidenceRange` fields to `crap.Score` (`internal/crap/crap.go`). When contract coverage is 0% because all effects are classified "ambiguous", the reason field explains why and the confidence range shows how close effects are to the contractual threshold. Changed `ContractCoverageFunc` to return `ContractCoverageInfo` struct instead of bare `float64`, carrying the reason and confidence range alongside the percentage. Text report worst-offenders section now displays the reason (e.g., "all effects ambiguous, confidence 78-79"). Closes #60.
- ai-assertion-mapping: Added `AIMapperFunc` callback to `quality.Options` and `AIMapperContext` struct (`internal/quality/ai_mapper.go`) for AI-assisted assertion mapping. When all mechanical passes (direct identity, indirect root, helper bridge, inline call) fail to map an assertion, an optional AI model evaluates the semantic relationship between the assertion and the target function's side effects. AI mappings are assigned confidence 50 (lower than all mechanical passes). Added `tryAIMapping` function, `extractExprSource`/`extractFuncSource` helpers, `BuildAIMapperPrompt`/`ParseAIMapperResponse` public helpers, and 7 tests. `MapAssertionsToEffects` now accepts an optional variadic `AIMapperFunc` parameter (backward-compatible). Updated gaze-reporter agent prompt with "Unmapped Assertion Evaluation" section — the agent reads source files for unmapped assertions and evaluates semantic relationships during `/gaze` command execution. Partially addresses #6.
- ast-ssa-helper-bridge: Added `CallerArgs []ast.Expr` to `AssertionSite` (`internal/quality/assertion.go`) and `buildHelperBridge` to `mapping.go` for bridging helper function parameters back to caller argument variables. When a test calls `assertEqual(t, got, 12)`, the assertion inside the helper now resolves `got` (helper param) → `got` (test variable) → side effect ID. Added `matchInlineCall` for matching assertions that call the target function inline without assignment (e.g., `if c.Value() != 5`). Mapping accuracy improved from 78.8% (52/66) to 86.4% (57/66). Ratchet floor raised from 76.0% to 85.0%. Partially addresses #6.
- report-classification-breakdown: Added `classify.CountLabels` helper (`internal/classify/classify.go`) to count side effects by classification label. Added `classifyStepResult` struct to `runClassifyStep` (`internal/aireport/runner_steps.go`) so classification counts flow through the pipeline alongside the raw JSON. Added `Contractual`, `Ambiguous`, `Incidental` fields to `ReportSummary` (`internal/aireport/payload.go`). Updated `pipelineStepFuncs.classifyStep` type to return `*classifyStepResult`. Closes #42.
- ssa-diagnostics-in-report: Added `SSADegradedPackages []string` to `taxonomy.PackageSummary` (`internal/taxonomy/types.go`) for per-package SSA failure tracking. Added `SSADegraded` and `SSADegradedPackages` to `ReportSummary` (`internal/aireport/payload.go`) so degradation is visible at the report payload level. Changed `runQualityForPackage` to return degraded package path (string) instead of boolean. Quality text report now shows SSA diagnostics section listing failed packages. Updated `QualitySchema` with `ssa_degraded_packages` array field. Closes #46.
- report-actionability: Added `FixStrategy` field to `crap.Score` and `FixStrategyCounts` to `crap.Summary` (`internal/crap/crap.go`). Each CRAPload function now carries a deterministic remediation label: `decompose` (complexity >= threshold, coverage can't help), `add_tests` (zero coverage), `add_assertions` (Q3 — has line coverage but lacks contract assertions), `decompose_and_test` (both needed). Strategy computed in `assignFixStrategy` (`internal/crap/analyze.go`), accumulated in `buildSummary`. Text report shows "Remediation Breakdown" section and `[strategy]` labels on worst offenders. JSON output includes new fields automatically. Addresses #42, #44, #48.
- q4-complexity-reduction: Decomposed all 5 Q4 Dangerous functions to reduce cyclomatic complexity. `scaffold.Run` 22→11 (extracted `applyDefaults`, `processAssetFile`, `handleToolOwnedFile`, `writeNewFile`). `crap.WriteText` 18→2 (extracted `writeScoreTable`, `writeSummarySection`, `writeQuadrantSection`, `writeWorstSection`). `classify.ComputeScore` 16→9 (extracted `accumulateSignals`, `classifyLabel`). `aireport.Run` 16→5 (extracted `validateRunnerOpts`, `runJSONPath`, `runTextPath`). `(*OllamaAdapter).Format` 16→12 (extracted `resolveOllamaHost`). No API surface changes. All existing tests pass without modification.
- pipeline-step-testing: Added `pipelineStepFuncs` dependency injection to `runProductionPipeline` (`internal/aireport/runner.go`) enabling unit testing of the four-step orchestration logic without running real analysis. Added 8 unit tests in `pipeline_internal_test.go` covering: all steps succeed, individual step failures (CRAP/quality/classify/docscan), multiple simultaneous failures, empty patterns validation, and summary field propagation. CRAP score for `runProductionPipeline` reduced from 42 (0% coverage) to well below threshold. No API surface changes — `RunnerOptions` and `Run()` behavior identical.
- adapter-format-decomposition: Extracted shared `runSubprocess` helper (`internal/aireport/adapter.go`) from the three subprocess AI adapter Format methods. `ClaudeAdapter.Format` complexity reduced from 11 to 6, `GeminiAdapter.Format` from 12 to 7, `OpenCodeAdapter.Format` from 12 to 7. The helper handles binary lookup, subprocess pipe setup with stdout/stderr capture, output size limiting, and error formatting. Added 5 unit tests for `runSubprocess` in `subprocess_test.go`. No API surface changes — `AIAdapter` interface, output formats, and CLI behavior identical. CRAP scores for all three adapters reduced from 104-127 to well below threshold.
- crapload-analyze-decomposition: Extracted `computeScores` from `crap.Analyze` (`internal/crap/analyze.go`) to reduce cyclomatic complexity from 15 to 8. The score computation loop (join complexity with coverage, filter generated/test files, compute CRAP and GazeCRAP, classify quadrants) is now a separate function directly testable with synthetic inputs. Added 7 unit tests in `internal/crap/analyze_internal_test.go` covering all branches: basic CRAP, test file skip, generated file skip/include, zero coverage, GazeCRAP with/without callback. No API surface changes — `Analyze` signature and behavior are identical. GazeCRAP score for `crap.Analyze` reduced from 43.1 (Q4 Dangerous) to a lower quadrant.
- 033-ssa-goroutine-panic: Added `ssa.BuildSerially` to the SSA builder mode flags in both `BuildSSA` (`internal/analysis/mutation.go`) and `BuildTestSSA` (`internal/quality/pairing.go`). The `recover()` guards from spec 021 did not catch panics because `ssa.Program.Build()` spawns child goroutines by default and Go's `recover()` is goroutine-scoped. `BuildSerially` forces `Build()` to run all SSA construction on the calling goroutine, making the existing `safeSSABuild` recovery effective. Corrected the synchronous assumption in `specs/021-ssa-panic-recovery/research.md` R2. Closes #33.
- assess-graceful-degradation: Refactored `quality.Assess` (`internal/quality/quality.go`) to degrade gracefully when `BuildTestSSA` fails instead of returning a hard error. When SSA construction fails (e.g., upstream `x/tools` panic under Go 1.25), `Assess` now returns partial results with AST-only data (test function enumeration, assertion detection confidence) and zero-valued contract coverage/over-specification. Added `SSADegraded bool` field to `taxonomy.PackageSummary` (`internal/taxonomy/types.go`) as a machine-readable indicator of partial results. Updated `QualitySchema` JSON Schema (`internal/report/schema.go`) with optional `ssa_degraded` boolean. Added `BuildSSAFunc` injection point to `quality.Options` for testing. Updated `analyzePackageCoverage` caller (`cmd/gaze/main.go`) with warning log for degraded results. All three callers (`gaze quality`, `gaze report`, `gaze crap`) now receive degraded results instead of nil/error. Closes #30.
- 021-ssa-panic-recovery: Added `recover()` guards around `prog.Build()` in `analysis.BuildSSA` (`internal/analysis/mutation.go`) and `quality.BuildTestSSA` (`internal/quality/pairing.go`) to prevent SSA builder panics from crashing `gaze report`, `gaze quality`, and `gaze analyze` under Go 1.25. Panics from upstream `x/tools` SSA type substitution bugs (e.g., `go-json-experiment/json` generic variadic parameters) are caught and converted to graceful nil/error returns. Warning-level log identifies the skipped package; debug-level log captures the raw panic value. New `safeSSABuild` helper isolates the `recover()` pattern for testability. Bumped `golang.org/x/tools` to `v0.43.0` and `go` directive to `1.25.0`. Added Go 1.24+1.25 CI matrix to `.github/workflows/test.yml`.
- 020-report-coverprofile: Added `--coverprofile` flag to `gaze report` so users can pass a pre-generated Go coverage profile and skip the internal `go test` run (eliminates the CI double-test-run). Pre-flight path validation in `runReport` (`os.Stat` + `IsDir`) produces a hard exit for invalid paths (FR-004, FR-005). Profile is threaded through `reportParams.coverProfile` → `RunnerOptions.CoverProfile` → `runCRAPStep` → `crap.Options.CoverProfile`. Modified: `cmd/gaze/main.go`, `internal/aireport/runner.go`, `internal/aireport/runner_steps.go`. Added: `internal/aireport/testdata/sample.coverprofile` fixture. README updated with CI integration example.
- 019-opencode-adapter: Added `opencode` as a fourth AI adapter for `gaze report --ai=opencode`. System prompt delivered as `.opencode/agents/gaze-reporter.md` in a temp dir via `--dir` flag (Gemini-style). Payload via stdin, plain-text stdout (`--format default`). Implements `AdapterValidator` (`ValidateBinary` / `exec.LookPath`). Optional `--model` flag passed as `-m`. Updated `validAdapters` allowlist, `NewAdapter` factory, `--ai` help text, error messages, usage examples, and `TestSC006_CrossAdapterStructure`. New files: `adapter_opencode.go`, `adapter_opencode_test.go` (10 tests), `testdata/fake_opencode/main.go`.
- 018-ci-report: Added `gaze report` subcommand with AI CLI adapter integration. Orchestrates four analysis operations (CRAP, quality, classification, docscan), pipes combined JSON payload to `claude` (exec+temp file), `gemini` (exec+GEMINI.md temp dir), or `ollama` (net/http POST /api/generate). Appends formatted markdown to `$GITHUB_STEP_SUMMARY`. Optional threshold flags (`--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`) enforce CI quality gates. New `internal/aireport` package (~10 files). `O_NOFOLLOW` symlink protection on Step Summary write. `*int` + `cmd.Flags().Changed()` pattern for zero-as-live-threshold semantics.
- 017-testing-persona: Added The Tester (reviewer-testing agent) as 4th review council member for test quality and testability auditing. Added `/speckit.testreview` command for read-only spec testability analysis. Amended constitution with Principle IV: Testability (v1.0.0 → v1.1.0). Scaffold expanded from 4 to 7 files with mixed ownership model — `isToolOwned` now uses explicit file list (prefix for `references/`, exact match for `command/speckit.testreview.md` and `command/review-council.md`). Review council scaffolded as tool-owned for deployment via `gaze init`.
- 016-agent-context-reduction: Reduced gaze-reporter agent prompt from 17,775 to 13,050 bytes (26.6% reduction) by externalizing canonical example output and document-enhanced classification scoring model into `.opencode/references/` files loaded on demand via Read tool. Added scaffold overwrite-on-diff behavior for tool-owned reference files (`references/` directory) while preserving skip-if-present for user-owned files (`agents/`, `command/`). Scaffold now manages 4 files (up from 2). Added `Updated` field to scaffold `Result` struct and `isToolOwned` helper. Quadrant labels deduplicated to 2 locations (Quick Reference Example + Emoji Vocabulary table).
- 015-native-macos-signing: Replaced broken quill-based cross-platform signing with native `codesign`/`notarytool` on `macos-latest` runner. Removed `notarize.macos` from `.goreleaser.yaml`. Added `sign-macos` job to release workflow (Keychain import, codesign with hardened runtime, notarytool submit --wait, asset replacement with --clobber, checksum update). Conditional on `MACOS_SIGN_P12` secret via job output gate.
- 014-macos-notarization: Added macOS code signing and notarization to GoReleaser release pipeline via built-in `notarize.macos` (quill), conditional on `MACOS_SIGN_P12` secret presence, 20m notarization timeout, 45m job timeout, no runner change (stays ubuntu-latest) (SUPERSEDED by 015)
- 012-consolidate-classify-docs: Removed /classify-docs command and doc-classifier agent, inlined document-signal scoring model into gaze-reporter, added emoji formatting override block and sandwich prompt structure, reduced scaffold from 4 to 2 files
- 011-output-voice-style: Rewrote gaze-reporter agent prompt for fun, emoji-rich output — emoji section markers (🔍📊🧪🏷️🏥), colored circle severity indicators (🟢🟡🔴⚪), letter grades with emoji, severity-prefixed recommendations, tone anti-pattern bans, canonical example output
- 009-crapload-reduction: CRAPload reduction — contract-level tests for `docscan.Filter` and `LoadModule`, dependency injection for `runCrap`/`runSelfCheck`, decomposition of `buildContractCoverageFunc` into `resolvePackagePaths`/`analyzePackageCoverage`, and decomposition of `AnalyzeP1Effects`/`AnalyzeP2Effects` into per-node-type handler functions
- 008-contract-coverage-gaps: Contract coverage gap remediation — direct unit tests for 8 functions with zero contract coverage across `internal/classify/`, `internal/analysis/`, and `cmd/gaze/` (test-only, no production code changes)
- 007-assertion-mapping-depth: Assertion mapping depth improvements — resolveExprRoot (selector/index/builtin unwinding), two-pass matching (direct 75/indirect 65), helper return value tracing (depth-1 SSA verification). Mapping accuracy improved from 73.8% to 78.8% (ratchet floor 76.0%)
- 006-agent-quality-report-enhancements: Unmapped assertion reasons, gap hints, discarded return details, ambiguous effects expansion in quality reports
- 005-gaze-opencode-integration: Added `gaze init` subcommand (internal/scaffold), GoReleaser v2 release pipeline, Homebrew cask distribution, OpenCode agent (gaze-reporter) and command (/gaze) files
