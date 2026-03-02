# Data Model: Native macOS Code Signing and Notarization

**Feature**: 015-native-macos-signing
**Date**: 2026-03-02

## Overview

This feature has no application-level data model. The "data model" consists of CI workflow jobs, their dependencies, and the secrets they consume. This document defines the workflow architecture and entity relationships.

## Workflow Architecture

### Job Dependency Graph

```text
[tag push v*]
    │
    ▼
[release] (ubuntu-latest)
    │  GoReleaser: build, archive, checksum, publish release, update Homebrew cask
    │  Outputs: tag name, release URL
    │
    ▼
[sign-macos] (macos-latest, needs: release, conditional on secrets)
    │  1. Import .p12 cert into temp Keychain
    │  2. Decode .p8 notary key to temp file
    │  3. Download darwin archives from release
    │  4. For each darwin archive:
    │     ├── Extract binary from tar.gz
    │     ├── codesign --force --timestamp --options runtime --sign "..."
    │     ├── Verify: codesign --verify
    │     ├── Zip binary (ditto -c -k) for notarytool
    │     ├── notarytool submit --wait --timeout 20m
    │     └── Re-tar.gz signed binary
    │  5. Replace darwin archives on release (gh release upload --clobber)
    │  6. Update checksums.txt (remove darwin lines, add new, re-upload)
    │
    ▼
[Release complete with signed darwin binaries]
```

### When Secrets Are Absent

```text
[tag push v*]
    │
    ▼
[release] (ubuntu-latest)
    │  GoReleaser: build, archive, checksum, publish release, update Homebrew cask
    │
    ▼
[sign-macos] — SKIPPED (condition not met)
    │
    ▼
[Release complete with unsigned darwin binaries]
```

## CI/CD Secrets

**Location**: GitHub repository settings > Secrets and variables > Actions

| Secret Name | Used By | Content Type | Encoding |
|-------------|---------|-------------|----------|
| `MACOS_SIGN_P12` | sign-macos (Keychain import) | Binary (.p12 file) | Base64 |
| `MACOS_SIGN_PASSWORD` | sign-macos (Keychain import) | Plain text | None |
| `MACOS_NOTARY_KEY` | sign-macos (notarytool) | Binary (.p8 file) | Base64 |
| `MACOS_NOTARY_KEY_ID` | sign-macos (notarytool) | Plain text | None |
| `MACOS_NOTARY_ISSUER_ID` | sign-macos (notarytool) | UUID | None |
| `GITHUB_TOKEN` | release + sign-macos | Auto-generated | N/A |
| `HOMEBREW_TAP_GITHUB_TOKEN` | release (GoReleaser) | PAT | N/A |

**Changes from spec 014**: The 5 `MACOS_*` secrets are no longer passed to the GoReleaser step. They are only used by the `sign-macos` job. `GITHUB_TOKEN` is used by both jobs (build publishes release, sign-macos downloads/uploads assets).

## Temporary Artifacts

Created and destroyed within the `sign-macos` job:

| Artifact | Path | Lifecycle |
|----------|------|-----------|
| Decoded .p12 certificate | `$RUNNER_TEMP/cert.p12` | Created at step 1, destroyed with VM |
| Temporary Keychain | `$RUNNER_TEMP/app-signing.keychain-db` | Created at step 1, destroyed with VM |
| Decoded .p8 notary key | `$RUNNER_TEMP/notary_key.p8` | Created at step 2, destroyed with VM |
| Downloaded archives | `./artifacts/*.tar.gz` | Created at step 3, consumed at step 4 |
| Signed archives | `./signed/*.tar.gz` | Created at step 4, uploaded at step 5 |
| Zip for notarytool | `$workdir/gaze.zip` | Created and consumed within step 4 loop |

## Release Asset Lifecycle

### Darwin Archives

```text
[Build Job creates] → gaze_X.Y.Z_darwin_amd64.tar.gz (unsigned)
                    → gaze_X.Y.Z_darwin_arm64.tar.gz (unsigned)

[Sign Job replaces] → gaze_X.Y.Z_darwin_amd64.tar.gz (signed + notarized)
                    → gaze_X.Y.Z_darwin_arm64.tar.gz (signed + notarized)
```

### Checksums

```text
[Build Job creates] → checksums.txt (SHA256 for all archives, unsigned darwin)

[Sign Job updates]  → checksums.txt (linux checksums preserved, darwin checksums updated)
```

### Linux Archives (unchanged)

```text
[Build Job creates] → gaze_X.Y.Z_linux_amd64.tar.gz
                    → gaze_X.Y.Z_linux_arm64.tar.gz
```
