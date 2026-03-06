# Feature Specification: macOS Code Signing and Notarization

**Feature Branch**: `014-macos-notarization`  
**Created**: 2026-03-01  
**Status**: Superseded (replaced by Spec 015 — native macOS signing)  
**Input**: User description: "macOS code signing and notarization for Homebrew-distributed binary via GoReleaser v2 built-in notarization using quill"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Trusted Installation via Homebrew (Priority: P1)

A macOS user installs gaze via `brew install --cask gaze` and runs it without macOS Gatekeeper blocking execution. Today, Gatekeeper blocks the binary because it is unsigned and unnotarized. After this feature, the binary is signed with a Developer ID Application certificate and notarized by Apple, so macOS trusts it on first run.

**Why this priority**: This is the entire purpose of the feature. Without code signing and notarization, macOS users cannot run gaze without manually overriding Gatekeeper security settings, which is a significant friction point that erodes trust and blocks adoption.

**Independent Test**: Can be fully tested by tagging a release, verifying the release pipeline produces signed and notarized darwin binaries, installing via Homebrew on a clean macOS machine, and confirming gaze launches without a Gatekeeper warning.

**Acceptance Scenarios**:

1. **Given** a tagged release triggers the release pipeline, **When** the pipeline builds darwin/amd64 and darwin/arm64 binaries, **Then** both binaries are code-signed with a valid Developer ID Application certificate.
2. **Given** signed darwin binaries are produced, **When** the pipeline submits them for notarization, **Then** Apple's notary service accepts them and returns a notarization ticket.
3. **Given** a notarized binary is installed via `brew install --cask gaze` on macOS, **When** the user runs `gaze` for the first time, **Then** macOS does not display a Gatekeeper warning or block execution.

---

### User Story 2 - Graceful Degradation Without Signing Credentials (Priority: P2)

When the release pipeline runs in an environment where Apple signing secrets are not configured (e.g., fork PRs, local testing, or contributors without access to the project's Apple credentials), the build completes successfully and produces unsigned binaries rather than failing.

**Why this priority**: The release pipeline must remain functional for all contributors. Signing is an enhancement to the release process, not a hard gate. Breaking the build for contributors without Apple credentials would harm the development workflow.

**Independent Test**: Can be fully tested by running GoReleaser in a CI environment (or locally) without the Apple signing secrets configured, and verifying the build succeeds and produces working (unsigned) binaries.

**Acceptance Scenarios**:

1. **Given** the release pipeline runs without Apple signing secrets configured, **When** GoReleaser executes, **Then** the build completes successfully and produces unsigned darwin binaries.
2. **Given** the release pipeline runs without Apple signing secrets configured, **When** GoReleaser reaches the notarization step, **Then** signing and notarization are skipped without error.

---

### User Story 3 - Notarization Timeout Handling (Priority: P3)

Apple's notarization service has variable processing times. The release pipeline must handle scenarios where notarization takes longer than expected without leaving the release in an inconsistent state.

**Why this priority**: Notarization processing times are unpredictable. Without proper timeout handling, long-running notarization could block releases indefinitely or cause partial releases with a mix of signed and unsigned binaries.

**Independent Test**: Can be tested by configuring a timeout for the notarization step and verifying the pipeline behavior when the timeout is reached (either by simulating a timeout or observing a real slow notarization).

**Acceptance Scenarios**:

1. **Given** the notarization step is configured with a timeout, **When** Apple's notary service does not respond within the timeout period, **Then** the release pipeline reports a clear error indicating notarization timed out.
2. **Given** the notarization step times out, **When** the release pipeline finishes, **Then** the release does not publish partially signed artifacts (either all darwin binaries are signed and notarized, or none are).

---

### Edge Cases

- What happens when the Apple Developer certificate expires mid-release cycle?
- How does the pipeline behave if Apple's notary service is temporarily unavailable?
- What happens if notarization succeeds for one architecture (amd64) but fails for the other (arm64)?
- What happens if the signing certificate is revoked by Apple?
- How does the pipeline handle a corrupted or invalid .p12 certificate secret?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The release pipeline MUST code-sign all darwin (macOS) binaries with a Developer ID Application certificate before publishing them.
- **FR-002**: The release pipeline MUST submit signed darwin binaries to Apple's notary service and wait for notarization to complete.
- **FR-003**: The release pipeline MUST skip signing and notarization gracefully when Apple signing secrets are not present in the environment, producing unsigned binaries without failing the build.
- **FR-004**: The release pipeline MUST continue to run on the existing CI runner environment without requiring a macOS-specific runner.
- **FR-005**: The release pipeline MUST apply signing and notarization only to darwin binaries; linux binaries MUST NOT be affected.
- **FR-006**: The release pipeline MUST configure a timeout for the notarization step to prevent indefinite blocking.
- **FR-007**: The release pipeline MUST use project secrets (stored as CI/CD secrets) for all Apple credentials; no credentials MUST be hardcoded or committed to the repository.
- **FR-008**: The existing Homebrew cask distribution MUST continue to work after this change, with the cask installing the signed and notarized binary.

### Key Entities

- **Signing Certificate**: A Developer ID Application certificate (.p12 format) issued by Apple, used to code-sign macOS binaries. Protected by a password.
- **Notarization Credentials**: An App Store Connect API key (.p8 format) with associated Key ID and Issuer ID, used to authenticate with Apple's notary service.
- **CI/CD Secrets**: Five secrets stored in the CI/CD platform that provide the signing certificate, its password, the notarization API key, Key ID, and Issuer ID.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: macOS users can install gaze via Homebrew and run it immediately without any Gatekeeper warnings or security overrides.
- **SC-002**: The release pipeline completes signing and notarization within 30 minutes of the build step finishing.
- **SC-003**: The release pipeline succeeds without errors when Apple signing secrets are not configured, producing functional unsigned binaries.
- **SC-004**: 100% of darwin binaries (both architectures) in every release are signed and notarized when Apple credentials are available.
- **SC-005**: No changes to the existing Linux binary build or distribution process.

## Assumptions

- The project maintainer has or will obtain an Apple Developer Program membership ($99/year).
- The project maintainer will generate and configure the required Apple credentials (Developer ID Application certificate, App Store Connect API key) and store them as CI/CD secrets before this feature can be exercised in production.
- GoReleaser v2's built-in notarization (via quill) supports cross-platform signing from a Linux CI runner without requiring a macOS runner.
- Apple's notarization service accepts binaries built with `CGO_ENABLED=0` cross-compilation.
- Homebrew cask distribution does not require stapling (zip archives cannot be stapled, and Homebrew handles notarization verification via the bundle ID).
