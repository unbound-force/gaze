# Tasks: Gaze OpenCode Integration & Distribution

**Input**: Design documents from `specs/005-gaze-opencode-integration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: No new project structure needed — this feature adds to an existing Go project. Setup consists of preparing version infrastructure.

- [x] T001 Add `commit` and `date` build variables alongside existing `version` in cmd/gaze/main.go (change `var version = "dev"` to `var (version = "dev"; commit = "none"; date = "unknown")` per research.md R-005)
- [x] T002 Update `gaze --version` output in cmd/gaze/main.go to include commit and date (e.g., `gaze version v0.1.0 (commit abc1234, built 2026-02-23)`)

---

## Phase 2: User Story 1 — Module Path Migration (Priority: P0) 🎯 MVP

**Goal**: Migrate Go module path from `github.com/unbound-force/gaze` to `github.com/unbound-force/gaze` so all downstream work uses the canonical path.

**Independent Test**: `go build ./...` and `go test -race -count=1 -short ./...` pass with zero errors. No `.go` file contains `github.com/unbound-force/gaze`.

### Implementation for User Story 1

- [x] T003 [US1] Run `go mod edit -module github.com/unbound-force/gaze` to update go.mod
- [x] T004 [US1] Replace module path in all .go files under cmd/ and internal/ (28 source files, 83 import lines) using sed per research.md R-001
- [x] T005 [US1] Replace module path in testdata .go files: internal/quality/testdata/src/multilib/multilib_test.go and internal/classify/testdata/src/callers/callers.go
- [x] T006 [US1] Replace module path in string literals: JSON Schema `$id` URLs in internal/report/schema.go and hardcoded paths in internal/loader/loader_test.go
- [x] T007 [P] [US1] Replace module path in README.md (install instructions, module reference)
- [x] T008 [P] [US1] Replace module path in AGENTS.md (module path reference)
- [x] T009 [P] [US1] Replace module path in all spec files under specs/ (5 spec directories)
- [x] T010 [US1] Run `go mod tidy` to regenerate go.sum
- [x] T011 [US1] Verify build succeeds: `go build ./...`
- [x] T012 [US1] Verify tests pass: `go test -race -count=1 -short ./...`
- [x] T013 [US1] Verify no remaining references: grep confirmed zero results

**Checkpoint**: Module path migration complete. All builds and tests pass with new path. SC-011 verified.

---

## Phase 3: User Story 2 — Release Pipeline via GoReleaser (Priority: P1)

**Goal**: Create GoReleaser v2 configuration and GitHub Actions release workflow so tagged pushes produce cross-platform binaries and auto-update the Homebrew tap.

**Independent Test**: `goreleaser check` validates config. `goreleaser release --snapshot --clean` produces archives for all 4 platforms.

### Implementation for User Story 2

- [x] T014 [P] [US2] Create .goreleaser.yaml with GoReleaser v2 schema per plan.md (builds for darwin/linux x amd64/arm64, CGO_ENABLED=0, ldflags with version/commit/date, homebrew_casks for unbound-force/homebrew-tap, changelog grouping by conventional commits)
- [x] T015 [P] [US2] Create .github/workflows/release.yml triggered on v* tags per plan.md (checkout with fetch-depth 0, setup-go from go.mod, goreleaser-action@v7 with GITHUB_TOKEN and HOMEBREW_TAP_GITHUB_TOKEN)
- [x] T016 [US2] Validate GoReleaser config: run `goreleaser check` and fix any errors

**Checkpoint**: Release pipeline configured. SC-012 verified. SC-013 can be verified locally with `goreleaser release --snapshot --clean`.

---

## Phase 4: User Story 3 — Homebrew Installation (Priority: P2)

**Goal**: Ensure `brew install unbound-force/tap/gaze` works after a release is published.

**Independent Test**: After a release, run `brew install unbound-force/tap/gaze && gaze --version && gaze init`.

**Note**: This story is primarily validated by US2's release pipeline — the `homebrew_casks` config in `.goreleaser.yaml` handles formula generation. The only prerequisite is the tap repo.

### Implementation for User Story 3

- [x] T017 [US3] Document the manual prerequisite steps in quickstart.md: (1) Create `unbound-force/homebrew-tap` repo on GitHub, (2) Create PAT with repo scope, (3) Add as HOMEBREW_TAP_GITHUB_TOKEN secret
- [x] T018 [US3] Verify Homebrew tap repo exists at github.com/unbound-force/homebrew-tap (manual check, report if missing)

**Checkpoint**: Homebrew distribution prerequisites documented and verified. SC-015 verified after first release.

---

## Phase 5: User Story 4 — `gaze init` Scaffolds OpenCode Files (Priority: P3)

**Goal**: Implement `gaze init` subcommand that copies embedded OpenCode agent and command files into a target project's `.opencode/` directory.

**Independent Test**: Run `gaze init` in a temp directory with a go.mod and verify 4 files created with correct content and version markers.

### Implementation for User Story 4

- [x] T019 [P] [US4] Create .opencode/agents/gaze-reporter.md with YAML frontmatter (description, tools: read+bash only) and agent body (mode parsing, binary resolution, command execution, JSON interpretation, report formatting per plan.md Phase 1C)
- [x] T020 [P] [US4] Create .opencode/command/gaze.md with YAML frontmatter (description, agent: gaze-reporter) and command body (passes $ARGUMENTS to agent, documents three usage modes: full, crap, quality)
- [x] T021 [US4] Create internal/scaffold/ directory structure: scaffold.go, assets/agents/, assets/command/
- [x] T022 [P] [US4] Copy .opencode/agents/gaze-reporter.md to internal/scaffold/assets/agents/gaze-reporter.md
- [x] T023 [P] [US4] Copy .opencode/agents/doc-classifier.md to internal/scaffold/assets/agents/doc-classifier.md
- [x] T024 [P] [US4] Copy .opencode/command/gaze.md to internal/scaffold/assets/command/gaze.md
- [x] T025 [P] [US4] Copy .opencode/command/classify-docs.md to internal/scaffold/assets/command/classify-docs.md
- [x] T026 [US4] Implement scaffold.go: Options and Result types, embed.FS directive for assets/*, Run() function that walks embedded FS, creates directories, checks file existence, prepends version marker, writes files, and returns Result (per contracts/scaffold-api.md and data-model.md)
- [x] T027 [US4] Implement summary output in scaffold.go: print created/skipped/overwritten files and hint message to Stdout (per contracts/cli-init.md output format)
- [x] T028 [US4] Add go.mod warning detection in scaffold.go: check if go.mod exists in TargetDir, print warning if absent (per US4-AS6)
- [x] T029 [US4] Add `init` subcommand to cmd/gaze/main.go: Cobra command with --force flag, delegates to scaffold.Run() (per contracts/scaffold-api.md CLI Integration section)
- [x] T030 [US4] Write scaffold_test.go: TestRun_CreatesFiles (SC-001), TestRun_SkipsExisting (SC-002), TestRun_ForceOverwrites (SC-003), TestRun_VersionMarker (SC-004), TestRun_NoGoMod_PrintsWarning (US4-AS6)
- [x] T031 [US4] Write TestEmbeddedAssetsMatchSource in scaffold_test.go: drift detection test comparing internal/scaffold/assets/ against .opencode/ source files (FR-017, SC-005)
- [x] T032 [US4] Verify `go build ./...` and `go test -race -count=1 -short ./...` pass with scaffold package

**Checkpoint**: `gaze init` works end-to-end. SC-001 through SC-005, SC-010 verified.

---

## Phase 6: User Story 5 — `/gaze` Command Routes to Reporter (Priority: P4)

**Goal**: The `/gaze` OpenCode command routes to the `gaze-reporter` agent with the correct mode and package arguments.

**Independent Test**: Install OpenCode files via `gaze init`, invoke `/gaze ./...` in OpenCode, verify agent receives correct arguments.

**Note**: The agent and command markdown files were created in Phase 5 (T019, T020). This phase validates routing behavior.

### Implementation for User Story 5

- [ ] T033 [US5] Verify .opencode/command/gaze.md correctly passes $ARGUMENTS to gaze-reporter agent and documents the three usage modes (full, crap, quality) with default package ./...
- [ ] T034 [US5] Verify .opencode/agents/gaze-reporter.md contains correct mode parsing logic: first argument `crap` or `quality` selects mode, otherwise full; remaining arguments are the package pattern

**Checkpoint**: `/gaze` command routes correctly. Manual testing in OpenCode validates US5 acceptance scenarios.

---

## Phase 7: User Story 6 — CRAP-Only Report via `gaze-reporter` (Priority: P5)

**Goal**: The `gaze-reporter` agent produces a human-readable CRAP score summary when invoked with `/gaze crap <package>`.

**Independent Test**: Run `/gaze crap ./...` on the Gaze project itself and verify the summary contains total functions, CRAPload, top 5 worst scores, and quadrant distribution.

### Implementation for User Story 6

- [ ] T035 [US6] Verify gaze-reporter.md contains CRAP mode instructions: run `gaze crap --format=json`, interpret JSON output, produce summary per FR-012 (total functions, CRAPload count, top 5 worst CRAP scores with function/file/line, GazeCRAP quadrant distribution)
- [ ] T036 [US6] Verify gaze-reporter.md contains binary resolution logic: (1) check $PATH via `which gaze`, (2) try `go build` if cmd/gaze/main.go exists, (3) fall back to `go install github.com/unbound-force/gaze/cmd/gaze@latest` per FR-010 and FR-029
- [ ] T037 [US6] Verify gaze-reporter.md contains error handling: report gaze command failures clearly with remediation suggestions per FR-015
- [ ] T038 [US6] Dogfood test: run `/gaze crap ./...` in OpenCode on the Gaze project and verify output matches SC-006

**Checkpoint**: CRAP-only reporting works. SC-006 and SC-009 verified.

---

## Phase 8: User Story 7 — Quality-Only Report (Priority: P6)

**Goal**: The `gaze-reporter` agent produces a human-readable quality metrics summary when invoked with `/gaze quality <package>`.

**Independent Test**: Run `/gaze quality ./internal/analysis` and verify summary contains contract coverage, gaps, over-specifications.

**Note**: Quality mode depends on Specs 001-003 being implemented. The agent instructions are written now but will produce "unavailable" messages until those specs are complete.

### Implementation for User Story 7

- [ ] T039 [US7] Verify gaze-reporter.md contains quality mode instructions: run `gaze quality --format=json`, interpret JSON output, produce summary per FR-013 (avg contract coverage, coverage gaps, over-specification count, worst tests)
- [ ] T040 [US7] Verify gaze-reporter.md handles quality unavailable gracefully: report that quality analysis requires Specs 001-003 or that no test files were found per US7-AS2

**Checkpoint**: Quality mode instructions in place. SC-007 verified when Specs 001-003 complete.

---

## Phase 9: User Story 8 — Full Report via `gaze-reporter` (Priority: P7)

**Goal**: The `gaze-reporter` agent runs all gaze commands, delegates to doc-classifier, and produces a comprehensive health assessment.

**Independent Test**: Run `/gaze ./...` on the Gaze project and verify all sections appear (CRAP, Quality, Classification, Health Assessment).

### Implementation for User Story 8

- [ ] T041 [US8] Verify gaze-reporter.md contains full mode instructions: run crap, quality, analyze --classify, and docscan; delegate to doc-classifier agent; produce combined report per FR-014
- [ ] T042 [US8] Verify gaze-reporter.md contains graceful degradation: if quality fails but crap succeeds, include crap section and note quality unavailable per US8-AS3
- [ ] T043 [US8] Verify gaze-reporter.md contains Overall Health Assessment section: cross-reference high CRAP + low contract coverage to identify high-risk functions, produce prioritized recommendations per US8-AS4

**Checkpoint**: Full report instructions in place. SC-008 verified when all underlying commands functional.

---

## Phase 10: User Story 9 — Dogfooding in the Gaze Project (Priority: P8)

**Goal**: The Gaze project uses the same `.opencode/` files that `gaze init` distributes, with automated drift detection.

**Independent Test**: Run `go test ./internal/scaffold/...` and verify drift detection test passes. Run `gaze init --force` in repo root and verify output matches `.opencode/` files (minus version marker).

### Implementation for User Story 9

- [ ] T044 [US9] Run `/gaze ./...` in OpenCode on the Gaze project as a dogfooding validation — verify all available sections produce output
- [ ] T045 [US9] Verify TestEmbeddedAssetsMatchSource (T031) catches drift: temporarily modify an asset file, run `go test ./internal/scaffold/...`, confirm test fails, then revert

**Checkpoint**: Dogfooding validated. SC-005 verified by drift detection test.

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and final validation

- [ ] T046 [P] Update README.md with `gaze init` usage, Homebrew install instructions (`brew install unbound-force/tap/gaze`), and `/gaze` command documentation
- [ ] T047 [P] Update AGENTS.md with new internal/scaffold/ package description, `gaze init` command, and release pipeline documentation
- [ ] T048 [P] Add .gitattributes entry to force LF for embedded assets: `internal/scaffold/assets/** text eol=lf` per research.md R-003
- [ ] T049 Run full test suite: `go test -race -count=1 -short ./...` and verify all tests pass including new scaffold tests
- [ ] T050 Run quickstart.md validation: verify documented commands work (`go build`, `go test`, `gaze init`, `goreleaser check`)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — can start immediately
- **Phase 2 (US1 — Module Path)**: Depends on Phase 1 — BLOCKS all subsequent phases
- **Phase 3 (US2 — Release Pipeline)**: Depends on Phase 2 (needs correct module path)
- **Phase 4 (US3 — Homebrew)**: Depends on Phase 3 (tap config in .goreleaser.yaml)
- **Phase 5 (US4 — gaze init)**: Depends on Phase 2 (needs correct module path); can run in parallel with Phase 3
- **Phase 6 (US5 — /gaze command)**: Depends on Phase 5 (needs agent/command files)
- **Phase 7 (US6 — CRAP report)**: Depends on Phase 6 (needs command routing)
- **Phase 8 (US7 — Quality report)**: Depends on Phase 6; deferred until Specs 001-003 complete
- **Phase 9 (US8 — Full report)**: Depends on Phases 7 and 8
- **Phase 10 (US9 — Dogfooding)**: Depends on Phase 5 (needs scaffold tests)
- **Phase 11 (Polish)**: Depends on all desired phases being complete

### User Story Dependencies

```text
US1 (Module Path) ──┬──> US2 (Release Pipeline) ──> US3 (Homebrew)
                    │
                    └──> US4 (gaze init) ──> US5 (/gaze cmd) ──> US6 (CRAP)
                                │                                    │
                                │                               US7 (Quality)
                                │                                    │
                                └──> US9 (Dogfooding)           US8 (Full)
```

### Parallel Opportunities

**After US1 completes** (Phase 2), these can run in parallel:
- US2 (Release Pipeline) — Phase 3
- US4 (gaze init) — Phase 5

**Within Phase 5** (US4), these can run in parallel:
- T019 and T020 (create agent and command markdown files)
- T022, T023, T024, T025 (copy files to assets/)

**Within Phase 2** (US1), these can run in parallel:
- T007, T008, T009 (markdown file updates)

---

## Parallel Example: User Story 4

```bash
# Create OpenCode files in parallel:
Task: "Create .opencode/agents/gaze-reporter.md"
Task: "Create .opencode/command/gaze.md"

# Copy to assets/ in parallel (after OpenCode files exist):
Task: "Copy gaze-reporter.md to internal/scaffold/assets/agents/"
Task: "Copy doc-classifier.md to internal/scaffold/assets/agents/"
Task: "Copy gaze.md to internal/scaffold/assets/command/"
Task: "Copy classify-docs.md to internal/scaffold/assets/command/"
```

---

## Implementation Strategy

### MVP First (User Stories 1-2 Only)

1. Complete Phase 1: Setup (version variables)
2. Complete Phase 2: US1 — Module Path Migration
3. Complete Phase 3: US2 — Release Pipeline
4. **STOP and VALIDATE**: `goreleaser check` passes, snapshot build produces binaries
5. Ready to tag `v0.1.0` and publish first release

### Incremental Delivery

1. US1 (Module Path) → Foundation for everything
2. US2 (Release Pipeline) → Can publish binaries
3. US4 (gaze init) → Other projects can install integration
4. US5+US6 (/gaze + CRAP) → Users get value from /gaze crap
5. US3 (Homebrew) → Verified after first release
6. US7+US8 (Quality + Full) → Deferred to Specs 001-003
7. US9 (Dogfooding) → Continuous validation

### Suggested MVP Scope

**US1 + US2 + US4**: Module path migration, release pipeline, and `gaze init`. This gives users a way to install the binary (via `go install` or Homebrew after first release), scaffold the OpenCode files, and start using `/gaze crap`. Total: ~32 tasks.

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US7 and US8 are intentionally lightweight (verify agent markdown) because their full functionality depends on Specs 001-003
- The gaze-reporter agent is a markdown file (not Go code) — its "implementation" is writing the correct instructions
- Commit after each phase checkpoint
- Stop at any checkpoint to validate story independently
