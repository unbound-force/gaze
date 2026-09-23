# Feature Specification: Native macOS Code Signing and Notarization

**Feature Branch**: `015-native-macos-signing`  
**Created**: 2026-03-02  
**Status**: Complete  
**Supersedes**: Spec 014 (macos-notarization)  
**Input**: User description: "Replace quill-based cross-platform signing with native macOS codesign and notarytool in a split-runner CI workflow to fix TeamIdentifier bug and notarization stalling"

## Context

Spec 014 added macOS code signing and notarization to the release pipeline using GoReleaser v2's built-in cross-platform notarization (quill). In production, two critical bugs surfaced:

1. **quill does not set TeamIdentifier** in the code signature (quill issue #147, open since Sep 2023, unfixed). Apple's notary service cannot associate the binary with the developer account, causing all notarization submissions to stall at "In Progress" indefinitely.
2. **quill's notarization polling exhausts Apple's API rate limit** (429 Too Many Requests), failing the release before even reaching the second binary.

The result: binaries are signed but never notarized, and macOS Gatekeeper blocks execution with "unidentified developer" warnings.

This spec replaces the broken quill approach with Apple's native signing tools (`codesign` and `notarytool`) running on a macOS CI runner in a split-runner workflow.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trusted Homebrew Installation (Priority: P1)

A macOS user installs gaze via `brew install --cask gaze` and runs it without macOS Gatekeeper blocking execution. The binary is properly code-signed with a Developer ID Application certificate (including TeamIdentifier) and notarized by Apple, so macOS trusts it on first run.

**Why this priority**: This is the core user-facing outcome. Without working code signing and notarization, macOS users must manually override Gatekeeper security, which erodes trust and blocks adoption. Spec 014 attempted to deliver this but failed due to quill bugs.

**Independent Test**: Tag a release, verify the release pipeline signs and notarizes darwin binaries, download the binary, verify with `codesign --verify` and `spctl --assess` on macOS, then install via Homebrew and confirm no Gatekeeper warning.

**Acceptance Scenarios**:

1. **Given** a tagged release triggers the release pipeline, **When** the signing job runs on both darwin/amd64 and darwin/arm64 binaries, **Then** both binaries have a valid code signature with the correct TeamIdentifier set.
2. **Given** signed darwin binaries are submitted for notarization, **When** Apple's notary service processes them, **Then** notarization completes with status "Accepted" (not stalled at "In Progress").
3. **Given** notarized binaries are published to the GitHub Release, **When** a user installs via `brew install --cask gaze` and runs `gaze`, **Then** macOS does not display a Gatekeeper warning or block execution.
4. **Given** the signing job replaces unsigned darwin archives on the release, **When** a user downloads the tar.gz manually from GitHub Releases, **Then** the extracted binary passes `spctl --assess --type execute`.

---

### User Story 2 - Graceful Degradation Without Signing Credentials (Priority: P2)

When Apple signing secrets are not configured (e.g., fork PRs, contributors without access), the release pipeline completes successfully. The build job publishes unsigned binaries. The signing job either skips entirely or fails without affecting the published release.

**Why this priority**: The release pipeline must remain functional for all contributors. Signing is an enhancement, not a gate.

**Independent Test**: Run the release workflow without Apple signing secrets configured and verify the build job succeeds and publishes a release with working (unsigned) binaries.

**Acceptance Scenarios**:

1. **Given** the release pipeline runs without Apple signing secrets, **When** the build job executes, **Then** it completes successfully and publishes a release with unsigned darwin and linux binaries.
2. **Given** Apple signing secrets are absent, **When** the signing job evaluates its conditions, **Then** the signing job is skipped without error and does not affect the published release.

---

### User Story 3 - Checksum Integrity After Signing (Priority: P3)

After the signing job replaces unsigned darwin archives with signed ones, the published checksums file accurately reflects the signed archives. Users and automated tools that verify downloads against checksums get correct results.

**Why this priority**: The build job publishes checksums for unsigned archives. When the signing job replaces those archives, the checksums become stale. Stale checksums break `brew audit`, security-conscious users who verify downloads, and any automation that checks integrity.

**Independent Test**: After a release with signing, download the checksums file and all archives, compute SHA256 for each archive, and verify every checksum matches.

**Acceptance Scenarios**:

1. **Given** the signing job has replaced darwin archives with signed versions, **When** it updates the checksums file, **Then** the SHA256 values for darwin archives match the actual signed archives.
2. **Given** the signing job updates checksums, **When** it re-uploads the checksums file, **Then** the linux archive checksums remain unchanged from what the build job originally published.

---

### Edge Cases

- What happens if notarization for one architecture (amd64) succeeds but fails for the other (arm64)?
- What happens if the signing job fails after replacing some but not all darwin archives on the release?
- How does the pipeline handle a corrupted or expired .p12 certificate?
- What happens if the GitHub Release assets are being downloaded by a user at the exact moment the signing job is replacing them?
- What happens if Apple's notary service is down or returns errors for an extended period?
- How does the pipeline behave if the temporary Keychain creation fails on the macOS runner?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The release pipeline MUST use a separate CI job on a macOS runner to sign darwin binaries with Apple's native signing tools, producing a code signature that includes the correct TeamIdentifier.
- **FR-002**: The signing job MUST submit signed binaries to Apple's notary service and wait for notarization to complete with status "Accepted" before publishing signed archives.
- **FR-003**: The signing job MUST replace unsigned darwin archives on the GitHub Release with signed and notarized versions, preserving the original archive names and format.
- **FR-004**: The signing job MUST update the checksums file to reflect the signed darwin archives while preserving checksums for non-darwin (linux) archives.
- **FR-005**: The build job MUST continue to run on the existing CI runner environment (not macOS) for building, archiving, and publishing the initial release.
- **FR-006**: The signing job MUST be skipped without error when Apple signing secrets are not configured in the environment.
- **FR-007**: The release pipeline MUST use project secrets (stored as CI/CD secrets) for all Apple credentials; no credentials MUST be hardcoded or committed to the repository.
- **FR-008**: The existing Homebrew cask distribution MUST continue to work after this change.
- **FR-009**: The broken cross-platform signing configuration (quill-based) MUST be removed from the release pipeline to avoid confusion and unnecessary processing.
- **FR-010**: The signing job MUST set a timeout to prevent indefinite blocking if Apple's notary service is slow or unresponsive.

### Key Entities

- **Build Job**: The existing CI job that compiles binaries, creates archives, generates checksums, publishes the GitHub Release, and updates the Homebrew cask. Runs on the standard CI runner.
- **Signing Job**: A new CI job that runs on a macOS runner after the build job. Downloads unsigned darwin archives, signs binaries, submits for notarization, and replaces release assets with signed versions.
- **Temporary Keychain**: A short-lived credential store created on the macOS runner to hold the imported signing certificate. Destroyed when the runner VM terminates.
- **Signing Certificate**: A Developer ID Application certificate (.p12 format) imported into the temporary Keychain for use by the native signing tool.
- **Notarization Credentials**: An App Store Connect API key (.p8 format) with associated Key ID and Issuer ID, used to authenticate with Apple's notary service.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: macOS users can install gaze via Homebrew and run it immediately without any Gatekeeper warnings or security overrides.
- **SC-002**: The signing and notarization job completes within 30 minutes of the build job finishing.
- **SC-003**: The release pipeline succeeds without errors when Apple signing secrets are not configured, producing functional unsigned binaries.
- **SC-004**: 100% of darwin binaries (both architectures) in every release are signed with the correct TeamIdentifier and notarized when Apple credentials are available.
- **SC-005**: No changes to the existing Linux binary build or distribution process.
- **SC-006**: Published checksums match all release archives (both signed darwin and unsigned linux) after the signing job completes.
- **SC-007**: `codesign --verify` and `spctl --assess` both report the signed binary as valid and accepted on macOS.

## Assumptions

- The project maintainer has an active Apple Developer Program membership and has already configured the 5 required GitHub secrets (from spec 014): `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_ISSUER_ID`.
- macOS CI runners provide the `codesign`, `xcrun notarytool`, and `security` tools pre-installed.
- The repository is public, so macOS CI runner minutes are free.
- Apple's notary service accepts bare Mach-O binaries submitted as zip archives (it does not accept tar.gz directly).
- There is a brief window (~5-15 minutes) between the build job publishing unsigned darwin archives and the signing job replacing them with signed versions. This is acceptable because Homebrew fetches the download URL at install time, not at release time, and most users install well after the signing job completes.
- The Homebrew cask update is handled by the build job. The cask's download URLs point to the release assets, which are replaced in-place by the signing job (same filenames). No separate cask update is needed after signing.
