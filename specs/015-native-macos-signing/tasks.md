# Tasks: Native macOS Code Signing and Notarization

**Input**: Design documents from `/specs/015-native-macos-signing/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md

**Tests**: No automated test tasks — this feature is CI workflow configuration. Verification is manual (test release + `codesign --verify` + `spctl --assess` on macOS). See quickstart.md for verification steps.

**Organization**: Tasks are grouped by user story. US1 contains the core signing job implementation. US2 adds the conditional skip logic. US3 adds checksum update. All three are implemented in `.github/workflows/release.yml`.

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

**Purpose**: No project initialization needed — this feature modifies existing files only.

*Phase 1 is empty for this feature. Proceed directly to Phase 2.*

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Remove the broken quill-based signing before adding the native replacement.

- [x] T001 [P] Remove the entire `notarize.macos` section (the `notarize:` key and all its children) from .goreleaser.yaml. The section is between the `checksum:` and `changelog:` sections. Verify with `goreleaser check` after removal.
- [x] T002 [P] Remove the 5 `MACOS_*` environment variables (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`) from the `Run GoReleaser` step's `env` block in .github/workflows/release.yml. Keep `GITHUB_TOKEN` and `HOMEBREW_TAP_GITHUB_TOKEN` unchanged.

**Checkpoint**: Quill-based signing fully removed. GoReleaser config validated. Release pipeline produces unsigned binaries only.

---

## Phase 3: User Story 1 — Trusted Homebrew Installation (Priority: P1) MVP

**Goal**: Add a `sign-macos` job to the release workflow that signs darwin binaries with `codesign` and notarizes them with `notarytool` on a macOS runner.

**Independent Test**: Tag a test release, verify `sign-macos` job runs on `macos-latest`, download darwin binary, run `codesign -dv --verbose=4` (should show `TeamIdentifier=PGFWLVZX55`), run `spctl --assess --type execute --verbose=2` (should say `accepted source=Notarized Developer ID`).

### Implementation for User Story 1

- [x] T003 [US1] Add a `sign-macos` job to .github/workflows/release.yml with `runs-on: macos-latest`, `needs: release`, and `timeout-minutes: 30`. Add a step named "Import certificate into Keychain" that: (a) decodes `MACOS_SIGN_P12` from base64 to `$RUNNER_TEMP/cert.p12`, (b) creates a temporary keychain at `$RUNNER_TEMP/app-signing.keychain-db` with a random password, (c) unlocks the keychain, (d) imports the .p12 with `security import`, (e) sets the key partition list with `security set-key-partition-list -S apple-tool:,apple:`, (f) adds the keychain to the search list. Reference the exact commands from research.md Decision 3.
- [x] T004 [US1] Add a "Prepare notary key" step to the `sign-macos` job in .github/workflows/release.yml that decodes `MACOS_NOTARY_KEY` from base64 to `$RUNNER_TEMP/notary_key.p8`.
- [x] T005 [US1] Add a "Download darwin archives" step to the `sign-macos` job in .github/workflows/release.yml that uses `gh release download "${GITHUB_REF_NAME}" --pattern "gaze_*_darwin_*.tar.gz" --dir ./artifacts`. Set `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` in the step env.
- [x] T006 [US1] Add a "Sign and notarize" step to the `sign-macos` job in .github/workflows/release.yml that loops over each archive in `./artifacts/gaze_*_darwin_*.tar.gz` and for each: (a) extracts the binary with `tar -xzf`, (b) signs with `codesign --force --timestamp --options runtime --sign "Developer ID Application: John Flowers (PGFWLVZX55)"`, (c) verifies with `codesign --verify --verbose=2`, (d) zips with `ditto -c -k` for notarytool, (e) submits with `xcrun notarytool submit --key $RUNNER_TEMP/notary_key.p8 --key-id --issuer --wait --timeout 20m`, (f) re-archives the signed binary with `tar -czf` into `./signed/` preserving the original archive name. Set `MACOS_NOTARY_KEY_ID` and `MACOS_NOTARY_ISSUER_ID` in the step env from secrets.
- [x] T007 [US1] Add a "Replace release assets" step to the `sign-macos` job in .github/workflows/release.yml that uploads each signed archive from `./signed/` using `gh release upload "${GITHUB_REF_NAME}" ./signed/*.tar.gz --clobber`. Set `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` in the step env.

**Checkpoint**: The `sign-macos` job signs and notarizes both darwin binaries and replaces the unsigned archives on the release. TeamIdentifier is correctly set by native `codesign`.

---

## Phase 4: User Story 2 — Graceful Degradation (Priority: P2)

**Goal**: Ensure the signing job is skipped cleanly when Apple secrets are not configured.

**Independent Test**: Run the release workflow in a fork or environment without `MACOS_SIGN_P12` configured. Verify the `release` job succeeds and the `sign-macos` job shows as "skipped" (not "failed").

### Implementation for User Story 2

- [x] T008 [US2] Add a "Check signing secrets" step to the `release` job in .github/workflows/release.yml with `id: check-secrets`. The step should output `has_secrets=true` to `$GITHUB_OUTPUT` if `MACOS_SIGN_P12` is non-empty, otherwise `has_secrets=false`. Add `outputs: has_signing_secrets: ${{ steps.check-secrets.outputs.has_secrets }}` to the `release` job definition.
- [x] T009 [US2] Add an `if: ${{ needs.release.outputs.has_signing_secrets == 'true' }}` condition to the `sign-macos` job in .github/workflows/release.yml so it is skipped when secrets are absent.

**Checkpoint**: The signing job is conditionally skipped based on secret presence. The release job always succeeds regardless.

---

## Phase 5: User Story 3 — Checksum Integrity (Priority: P3)

**Goal**: Update the checksums file after signing so published checksums match signed darwin archives.

**Independent Test**: After a release with signing, download `checksums.txt` and all archives, run `shasum -a 256 -c checksums.txt` and verify all lines pass.

### Implementation for User Story 3

- [x] T010 [US3] Extend the "Replace release assets" step (T007) in .github/workflows/release.yml to also: (a) download the existing `checksums.txt` from the release with `gh release download`, (b) remove darwin lines from the checksums with `grep -v darwin`, (c) compute SHA256 for each signed darwin archive in `./signed/` with `shasum -a 256`, (d) append the new darwin checksums, (e) re-upload `checksums.txt` with `gh release upload --clobber`.

**Checkpoint**: Checksums file is accurate for both signed darwin and unchanged linux archives.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and validation

- [x] T011 [P] Update AGENTS.md "Recent Changes" section to add an entry for 015-native-macos-signing documenting the switch from quill to native codesign/notarytool
- [x] T012 [P] Update AGENTS.md "Active Technologies" section to verify the 015-native-macos-signing entry was added by `update-agent-context.sh` (check for existing entry; add if missing)
- [x] T013 Run `goreleaser check` to validate .goreleaser.yaml after quill removal (dry-run verification)
- [x] T014 Run `goreleaser release --snapshot --clean` to verify the build works without the quill notarize section

**Checkpoint**: Documentation updated, YAML validated. Feature is ready for a test release.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: Empty — no setup tasks
- **Foundational (Phase 2)**: T001 and T002 modify different files and can run in parallel. MUST complete before US1 work begins.
- **User Story 1 (Phase 3)**: Depends on Phase 2 (quill removal). T003-T007 are sequential — each step builds on the previous within the same file.
- **User Story 2 (Phase 4)**: Depends on US1 (T003 creates the `sign-macos` job that T009 adds the `if` condition to). T008 modifies the `release` job; T009 modifies the `sign-macos` job — sequential.
- **User Story 3 (Phase 5)**: Depends on US1 (T007 creates the "Replace release assets" step that T010 extends).
- **Polish (Phase 6)**: T011/T012 can run in parallel and at any time. T013/T014 depend on T001 being complete.

### User Story Dependencies

- **User Story 1 (P1)**: Depends on Phase 2 (quill removal) — core implementation
- **User Story 2 (P2)**: Depends on US1 (adds condition to the job US1 creates)
- **User Story 3 (P3)**: Depends on US1 (extends the asset replacement step US1 creates)

### Parallel Opportunities

- T001 and T002 modify different files (`.goreleaser.yaml` vs `.github/workflows/release.yml`) and can run in parallel
- T011 and T012 modify the same file (`AGENTS.md`) but different sections — can run in parallel
- T013 and T014 are both validation commands and can run sequentially after Phase 2

---

## Parallel Example: Foundational Phase

```bash
# Launch both foundational tasks together (different files):
Task: "Remove notarize.macos section from .goreleaser.yaml"
Task: "Remove MACOS_* env vars from GoReleaser step in release.yml"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 2: Remove quill config (T001 + T002)
2. Complete Phase 3: Add sign-macos job (T003-T007)
3. **STOP and VALIDATE**: Push a test tag and verify signing works
4. US2 and US3 can be added after validation

### Incremental Delivery

1. T001 + T002 → Remove broken quill config
2. T003-T007 → Core signing job (US1)
3. T008 + T009 → Graceful degradation (US2)
4. T010 → Checksum update (US3)
5. T011-T014 → Documentation and validation

### Full Verification (requires Apple credentials)

1. Push a test tag (e.g., `v0.X.Y-rc.1`)
2. Monitor GitHub Actions: `release` job on ubuntu, `sign-macos` job on macos
3. Download darwin binary and verify:
   - `codesign -dv --verbose=4` → TeamIdentifier set
   - `spctl --assess --type execute --verbose=2` → accepted, Notarized Developer ID
4. Download `checksums.txt` and verify with `shasum -a 256 -c`
5. Install via `brew install --cask gaze` and confirm no Gatekeeper warning

---

## Notes

- This feature modifies 2 files: `.goreleaser.yaml` (removal only) and `.github/workflows/release.yml` (removal + addition)
- T003-T007 build up the `sign-macos` job incrementally within the same file — they are sequential, not parallel
- US2 and US3 add small modifications to the job structure created by US1
- The same 5 GitHub secrets from spec 014 are reused — no new secret configuration needed
- Full end-to-end verification requires the Apple credentials already configured from spec 014
