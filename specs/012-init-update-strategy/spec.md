# Feature Specification: Init Update Strategy

**Feature Branch**: `012-init-update-strategy`  
**Created**: 2026-03-01  
**Status**: Draft  
**Input**: User description: "please help decide where addressing updates to an existing deployment with `gaze init`. Currently it doesn't overwrite existing files that are out of date."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Seamless Upgrade After Gaze Update (Priority: P1)

A developer upgrades Gaze to a new version. The new version ships with improved agent prompts and command definitions in its embedded assets. The developer runs `gaze init` and expects it to detect that the existing scaffolded files are from an older version and update them automatically — without requiring `--force` and without losing track of what changed.

**Why this priority**: This is the core problem. Today, running `gaze init` after a Gaze upgrade silently skips all existing files, leaving the project with stale agent prompts and commands. The only workaround is `--force`, which overwrites everything unconditionally with no indication of what actually changed. Users need a safe, informative upgrade path that is the default behavior.

**Independent Test**: Can be fully tested by scaffolding files tagged with version "v1.0.0", then running `gaze init` with version "v2.0.0" and verifying that outdated files are replaced with new content and the output clearly distinguishes "updated" files from other states.

**Acceptance Scenarios**:

1. **Given** a project with files scaffolded by gaze v1.0.0, **When** the user runs `gaze init` with gaze v2.0.0 (which has updated embedded assets), **Then** the outdated files are replaced with the new versions and the output labels each as "updated."
2. **Given** a project with files scaffolded by the same version of gaze currently running, **When** the user runs `gaze init`, **Then** no files are modified and the output says "already up to date."
3. **Given** a project with files scaffolded by gaze v1.0.0, **When** the user runs `gaze init` with gaze v2.0.0, **Then** the updated files contain the new version marker reflecting v2.0.0.

---

### User Story 2 - Clear Reporting of File Dispositions (Priority: P2)

A developer or team lead wants to understand what `gaze init` did after an upgrade. They need clear output distinguishing between files that were newly created (didn't exist before), files that were updated (existed but were outdated), files that were already current (no action needed), and files that were forcefully overwritten.

**Why this priority**: Transparency builds trust. Without clear reporting, developers cannot tell whether `gaze init` actually changed anything or what the impact was. This is especially important in team environments where scaffolded files are committed to version control.

**Independent Test**: Can be fully tested by setting up a mixed scenario — some files missing, some outdated, some current — and verifying the output summary correctly categorizes each file.

**Acceptance Scenarios**:

1. **Given** a project where one file is missing, one is outdated, and two are current, **When** the user runs `gaze init`, **Then** the output shows the missing file as "created," the outdated file as "updated," and the current files as "up to date" or omits them from the listing.
2. **Given** any `gaze init` run that updates at least one file, **When** the operation completes, **Then** the output for each updated file includes the old version and the new version (e.g., "v1.0.0 -> v2.0.0").
3. **Given** any `gaze init` run, **When** the operation completes, **Then** a summary line shows counts for each category (e.g., "1 created, 1 updated, 2 up to date").

---

### User Story 3 - Force as an Escape Hatch (Priority: P3)

A developer wants to reset all scaffolded files to the canonical versions shipped with their current Gaze binary, regardless of version markers or local edits. They use `--force` to unconditionally overwrite everything.

**Why this priority**: `--force` already exists and works. This story ensures the new update logic does not break or remove the existing force behavior. It remains the escape hatch for when smart detection is insufficient (e.g., corrupted files, manually edited files the user wants to reset).

**Independent Test**: Can be fully tested by scaffolding files, manually editing one, then running `gaze init --force` and verifying all files are replaced regardless of version or content.

**Acceptance Scenarios**:

1. **Given** a project with files scaffolded by the current version of gaze (already up to date), **When** the user runs `gaze init --force`, **Then** all files are overwritten and reported as "overwritten."
2. **Given** a project with a manually edited scaffolded file, **When** the user runs `gaze init --force`, **Then** the edited file is replaced with the canonical embedded version.

---

### Edge Cases

- What happens when a scaffolded file exists but has no version marker (e.g., created manually or by a pre-marker version of gaze)? The file is treated as outdated and updated, since its provenance cannot be confirmed.
- What happens when a scaffolded file has a version marker that cannot be parsed (e.g., corrupted or unexpected format)? The file is treated as outdated and updated.
- What happens when the running gaze binary has version "dev" (development build)? Dev builds always update all files, regardless of the on-disk version marker. This ensures developers working on gaze itself always get the latest embedded assets without needing `--force`.
- What happens when a new file is added to the embedded assets in a new gaze version but the project was scaffolded by an older version? The new file is created normally (it does not exist on disk, so the existing "created" path handles it).
- What happens when a file is removed from embedded assets in a new gaze version? Files on disk that are no longer in embedded assets are left untouched. Gaze does not delete files it no longer ships.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST compare the version marker in each existing scaffolded file against the version of the currently running gaze binary to determine whether the file is outdated.
- **FR-002**: When a scaffolded file is outdated (version marker differs from current version), `gaze init` MUST update the file with the current embedded content and new version marker, without requiring `--force`.
- **FR-003**: When a scaffolded file is current (version marker matches current version), `gaze init` MUST leave the file untouched.
- **FR-004**: The `--force` flag MUST continue to unconditionally overwrite all scaffolded files, regardless of version comparison.
- **FR-005**: The operation result MUST distinguish between four file states: created (new file), updated (existed but outdated), up to date (existed and current), and overwritten (existed and force was used).
- **FR-006**: Files without a recognizable version marker MUST be treated as outdated and updated.
- **FR-007**: When a file is updated, the output MUST include the previous version and the new version for that file (e.g., "v1.0.0 -> v2.0.0").
- **FR-008**: When the running gaze binary has version "dev", `gaze init` MUST always update all existing scaffolded files, regardless of the on-disk version marker.

### Key Entities

- **Version Marker**: The comment line prepended to every scaffolded file that records which version of gaze produced it. Serves as the provenance record and staleness indicator.
- **Scaffolded File**: A file written by `gaze init` into the `.opencode/` directory tree. Each has a version marker and content derived from embedded assets.
- **Scaffold Result**: The outcome of a `gaze init` run, categorizing each file by its disposition (created, updated, up to date, overwritten).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Running `gaze init` after a version upgrade updates all outdated files without requiring `--force`, with zero additional user intervention beyond running the command.
- **SC-002**: Running `gaze init` when all files are current produces no file modifications and reports "already up to date."
- **SC-003**: The output of every `gaze init` run categorizes each file into one of the defined states (created, updated, up to date, overwritten), with no ambiguous or unlabeled files.
- **SC-004**: Existing `--force` behavior is fully preserved — all files are overwritten unconditionally when the flag is set.
- **SC-005**: Files with missing or unparseable version markers are updated rather than silently skipped.
- **SC-006**: 100% of existing scaffold tests continue to pass after the change, confirming backward compatibility.

## Assumptions

- Version comparison is based on string equality of the version marker, not semantic version ordering. A file is "current" if and only if its marker version exactly matches the running binary's version. This avoids the complexity of semver parsing and handles non-semver version strings gracefully.
- The version marker format is stable and will not change between versions. If the format were to change in the future, a separate migration path would be needed.
- Users do not intentionally edit scaffolded files for customization purposes. If they do, `--force` is the appropriate mechanism to reset them. The update strategy does not attempt to merge or preserve local modifications.
- Content comparison (byte-level diff between embedded asset and on-disk file) is not used for staleness detection. Version marker comparison is sufficient and avoids the overhead of reading and comparing full file contents.
