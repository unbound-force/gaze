# Tasks: macOS Code Signing and Notarization

**Input**: Design documents from `/specs/014-macos-notarization/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, quickstart.md

**Tests**: No automated test tasks — this feature is pure YAML configuration. Verification is manual (dry-run + `spctl --assess` on macOS). See quickstart.md for verification steps.

**Organization**: Tasks are grouped by user story. US1 contains the core implementation; US2 and US3 are addressed by specific configuration choices within the same files.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

```text
.goreleaser.yaml                     # GoReleaser configuration (repository root)
.github/workflows/release.yml       # GitHub Actions release workflow
```

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: No project initialization needed — this feature modifies existing files only. No new directories, no new dependencies, no new Go code.

*Phase 1 is empty for this feature. Proceed directly to Phase 2.*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The maintainer must configure Apple credentials as GitHub secrets before signing can be exercised. This is a manual prerequisite, not an automatable task.

- [x] T001 Document the 5 required GitHub secrets in the repository wiki or README, referencing specs/014-macos-notarization/quickstart.md for step-by-step credential setup instructions

**Checkpoint**: GitHub secrets documentation is available for the maintainer to follow when ready.

---

## Phase 3: User Story 1 — Trusted Installation via Homebrew (Priority: P1) MVP

**Goal**: Add code signing and notarization to the release pipeline so darwin binaries are trusted by macOS Gatekeeper.

**Independent Test**: Tag a release, verify the pipeline signs and notarizes darwin binaries, install via Homebrew on macOS, confirm no Gatekeeper warning.

### Implementation for User Story 1

- [x] T002 [P] [US1] Add `notarize.macos` section to .goreleaser.yaml after the `checksum` section. Include `sign.certificate` and `sign.password` fields referencing `MACOS_SIGN_P12` and `MACOS_SIGN_PASSWORD` env vars via GoReleaser template syntax. Include `notarize.issuer_id`, `notarize.key_id`, and `notarize.key` fields referencing `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, and `MACOS_NOTARY_KEY` env vars. Use exact YAML from research.md "Exact Configuration" section.
- [x] T003 [P] [US1] Add 5 Apple credential environment variables (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`) to the `Run GoReleaser` step's `env` block in .github/workflows/release.yml, mapping each to its corresponding `${{ secrets.* }}` value. Preserve existing `GITHUB_TOKEN` and `HOMEBREW_TAP_GITHUB_TOKEN` env vars.

**Checkpoint**: Both configuration files updated. The release pipeline will sign and notarize darwin binaries when Apple secrets are configured.

---

## Phase 4: User Story 2 — Graceful Degradation Without Signing Credentials (Priority: P2)

**Goal**: Ensure the release pipeline succeeds without errors when Apple signing secrets are not configured.

**Independent Test**: Run `goreleaser release --snapshot --clean` locally without setting Apple env vars; verify it completes without error.

### Implementation for User Story 2

- [x] T004 [US2] Verify the `enabled` field in the `notarize.macos` section of .goreleaser.yaml uses the template guard `'{{ isEnvSet "MACOS_SIGN_P12" }}'` (this should already be set by T002). Confirm by reading the file and validating the exact template string. If not present, add it.

**Checkpoint**: The `isEnvSet` guard ensures signing is skipped when secrets are absent. No additional configuration needed — graceful degradation is built into the GoReleaser template syntax chosen in T002.

---

## Phase 5: User Story 3 — Notarization Timeout Handling (Priority: P3)

**Goal**: Prevent the release pipeline from blocking indefinitely on Apple's notarization service.

**Independent Test**: Verify the notarize config includes `wait: true` and `timeout: 20m`; confirm the GitHub Actions workflow global timeout accommodates the notarization timeout.

### Implementation for User Story 3

- [x] T005 [US3] Verify the `notarize` subsection in .goreleaser.yaml includes `wait: true` and `timeout: 20m` (this should already be set by T002). Confirm by reading the file. If not present, add both fields.
- [x] T006 [US3] Verify the release job in .github/workflows/release.yml has a `timeout-minutes` value that exceeds the notarization timeout (20m) plus build time. If no job-level timeout exists, add `timeout-minutes: 45` to the `release` job to provide adequate headroom.

**Checkpoint**: Notarization will timeout after 20 minutes, and the overall job timeout prevents indefinite CI billing.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and dry-run validation

- [x] T007 [P] Update AGENTS.md "Active Technologies" section to include the GoReleaser v2 notarization technology entry if not already present (check for existing `014-macos-notarization` entry added by `update-agent-context.sh`)
- [x] T008 [P] Update README.md to document that macOS binaries are code-signed and notarized, under the installation or distribution section (if such a section exists)
- [x] T009 Run `goreleaser release --snapshot --clean` locally to validate YAML syntax and confirm no errors when secrets are absent (dry-run verification of US2)
- [x] T010 Run `goreleaser check` to validate .goreleaser.yaml schema compliance

**Checkpoint**: Documentation updated, YAML validated. Feature is ready for the maintainer to configure Apple secrets and perform a test release.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Empty — no setup tasks
- **Foundational (Phase 2)**: T001 is documentation-only; does not block implementation
- **User Story 1 (Phase 3)**: No dependencies — can start immediately. T002 and T003 modify different files and can run in parallel.
- **User Story 2 (Phase 4)**: T004 depends on T002 (verifies the guard set in T002)
- **User Story 3 (Phase 5)**: T005 depends on T002 (verifies timeout set in T002); T006 modifies `.github/workflows/release.yml` and depends on T003
- **Polish (Phase 6)**: T007/T008 can run in parallel and at any time. T009/T010 depend on T002 and T003 being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Independent — core implementation
- **User Story 2 (P2)**: Depends on US1 (verifies configuration set in US1)
- **User Story 3 (P3)**: Depends on US1 (verifies configuration set in US1)

### Parallel Opportunities

- T002 and T003 modify different files (`.goreleaser.yaml` vs `.github/workflows/release.yml`) and can run in parallel
- T007 and T008 modify different files and can run in parallel
- T004, T005 are verification tasks that can run after T002 completes

---

## Parallel Example: User Story 1

```bash
# Launch both US1 tasks together (different files):
Task: "Add notarize.macos section to .goreleaser.yaml"
Task: "Add 5 Apple credential env vars to .github/workflows/release.yml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete T002 and T003 in parallel (Phase 3: US1)
2. **STOP and VALIDATE**: Run `goreleaser check` and `goreleaser release --snapshot --clean`
3. US2 and US3 are inherently satisfied by the configuration choices in US1

### Incremental Delivery

1. T002 + T003 → Core signing and notarization config (US1) 
2. T004 → Verify graceful degradation (US2)
3. T005 + T006 → Verify timeout handling (US3)
4. T007-T010 → Documentation and validation polish

### Full Verification (requires Apple credentials)

1. Maintainer configures 5 GitHub secrets per quickstart.md
2. Push a test tag (e.g., `v0.X.Y-rc.1`)
3. Monitor GitHub Actions release workflow
4. Download darwin binary and verify with `codesign --verify` and `spctl --assess` on macOS
5. Install via `brew install --cask gaze` and confirm no Gatekeeper warning

---

## Notes

- This feature is pure YAML configuration — no Go code changes
- T002 and T003 contain the entire implementation; T004-T006 are verification tasks
- US2 and US3 are satisfied by configuration choices made in US1 (the `enabled` guard and `timeout` field)
- The maintainer must manually configure Apple credentials (GitHub secrets) before signing works in production
- Dry-run validation (T009, T010) can be performed without Apple credentials
- Full end-to-end verification requires Apple Developer Program membership and configured secrets
