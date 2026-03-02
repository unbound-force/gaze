# Research: Native macOS Code Signing and Notarization

**Feature**: 015-native-macos-signing
**Date**: 2026-03-02

## Decision 1: Signing Approach

**Decision**: Use Apple's native `codesign` tool on a `macos-latest` GitHub Actions runner.

**Rationale**: The cross-platform approach (quill via GoReleaser) has two unfixed bugs that prevent notarization from working:

1. quill issue #147: `TeamIdentifier` is never set in the code signature. `newCodeDirectory()` in quill's source has a `TeamOffset` field but never populates it. Apple's notary service cannot associate the binary with the developer account, causing submissions to stall at "In Progress" indefinitely.
2. quill's notarization polling is aggressive enough to exhaust Apple's App Store Connect API hourly rate limit (429 Too Many Requests).

`codesign` is Apple's native signing tool. It correctly sets TeamIdentifier from the certificate, supports hardened runtime (`--options runtime`, required for notarization since macOS 10.15), and embeds trusted timestamps (`--timestamp`).

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| quill via GoReleaser (spec 014) | Broken | TeamIdentifier not set (bug #147, open 2.5 years). Notarization polling causes rate limiting. |
| Fork quill and fix | Viable but slow | Fix is small (populate `TeamOffset`), but requires forking, building a custom GoReleaser, maintaining it. Not worth the effort when native tools are available. |
| GoReleaser `notarize.macos_native` | Pro-only | Requires GoReleaser Pro license ($). |

## Decision 2: Workflow Architecture

**Decision**: Split-runner workflow — build on `ubuntu-latest`, sign on `macos-latest`.

**Rationale**: `codesign` and `xcrun notarytool` are macOS-only tools. The build must happen on a separate runner. The split approach keeps the fast, cheap build on Ubuntu and only uses the macOS runner for the short signing step.

The architecture:
1. **Job 1 (`release`)**: GoReleaser builds all targets, creates archives, checksums, publishes GitHub Release, updates Homebrew cask. Runs on `ubuntu-latest`. Unchanged from today except removal of quill config.
2. **Job 2 (`sign-macos`)**: Runs on `macos-latest` after Job 1 completes (`needs: release`). Downloads darwin archives from the release, signs, notarizes, re-archives, replaces release assets, updates checksums.

**Cost**: The repository is public, so all GitHub Actions runners (including macOS) are free. No cost concern.

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| Full macOS runner for everything | Viable but wasteful | The entire build+release would run on macOS (~10x more expensive for private repos). No benefit since Go cross-compilation works fine on Linux. |
| Draft release + publish after signing | More complex | Homebrew cask `skip_upload: auto` skips drafts. Would need manual cask update after undrafting. More workflow complexity for marginal benefit (avoiding brief unsigned window). |

## Decision 3: Certificate Import

**Decision**: Create a temporary Keychain on the macOS runner and import the .p12 certificate into it.

**Rationale**: `codesign` reads certificates from Keychain, not from files. The standard CI pattern (documented by GitHub and Apple) is:

1. Create a temporary keychain with a random password
2. Decode the base64 .p12 secret to a file
3. Import the .p12 into the temporary keychain
4. Set the keychain partition list to allow `codesign` access without UI prompts
5. Add the keychain to the search list

The temporary keychain is destroyed when the runner VM terminates (GitHub-hosted runners are ephemeral).

**Key commands**:
```bash
KEYCHAIN_PATH=$RUNNER_TEMP/app-signing.keychain-db
KEYCHAIN_PASSWORD="$(openssl rand -base64 32)"

echo -n "$MACOS_SIGN_P12" | base64 --decode -o $RUNNER_TEMP/cert.p12
security create-keychain -p "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH
security set-keychain-settings -lut 21600 $KEYCHAIN_PATH
security unlock-keychain -p "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH
security import $RUNNER_TEMP/cert.p12 -P "$MACOS_SIGN_PASSWORD" -A -t cert -f pkcs12 -k $KEYCHAIN_PATH
security set-key-partition-list -S apple-tool:,apple: -k "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH
security list-keychain -d user -s $KEYCHAIN_PATH
```

## Decision 4: Codesign Flags

**Decision**: Use `codesign --force --timestamp --options runtime --sign "Developer ID Application: John Flowers (PGFWLVZX55)"`.

**Rationale**:
- `--force`: Re-signs even if the binary has an ad-hoc signature (Go binaries may have one)
- `--timestamp`: Embeds a trusted timestamp from Apple's TSA server (required for notarization)
- `--options runtime`: Enables Hardened Runtime (required for notarization since macOS 10.15)
- `--sign "IDENTITY"`: The full common name from the Developer ID Application certificate

No entitlements file needed — gaze is a CLI tool with no special macOS capability requirements.

## Decision 5: Notarization Submission

**Decision**: Use `xcrun notarytool submit --wait --timeout 20m` with App Store Connect API key authentication.

**Rationale**: `notarytool` is Apple's official notarization tool (replaced the deprecated `altool`). It supports `--wait` to block until Apple returns a result, with configurable `--timeout`. API key authentication (`.p8` file + key ID + issuer ID) is the recommended method for CI.

`notarytool` requires `.zip`, `.dmg`, or `.pkg` format — not `.tar.gz`. The binary must be zipped before submission using `ditto -c -k`.

**Key commands**:
```bash
echo -n "$MACOS_NOTARY_KEY" | base64 --decode -o $RUNNER_TEMP/notary_key.p8

ditto -c -k ./gaze ./gaze.zip
xcrun notarytool submit ./gaze.zip \
  --key $RUNNER_TEMP/notary_key.p8 \
  --key-id "$MACOS_NOTARY_KEY_ID" \
  --issuer "$MACOS_NOTARY_ISSUER_ID" \
  --wait --timeout 20m
```

## Decision 6: Release Asset Replacement

**Decision**: Use `gh release upload --clobber` to replace unsigned archives with signed ones, then update checksums.

**Rationale**: The `--clobber` flag replaces an existing asset with the same filename. This is simpler than delete-then-upload and is atomic from the GitHub API perspective.

Checksums must be regenerated because the archive contents changed (signed binary replaces unsigned). The approach:
1. Download existing `checksums.txt`
2. Remove darwin lines (keep linux lines intact)
3. Compute SHA256 for the new signed darwin archives
4. Append the new darwin checksums
5. Re-upload `checksums.txt` with `--clobber`

## Decision 7: Graceful Degradation

**Decision**: Use GitHub Actions `if` condition on the signing job to skip when secrets are absent.

**Rationale**: GitHub Actions supports `if: ${{ secrets.MACOS_SIGN_P12 != '' }}` (or similar patterns) to conditionally run jobs. When secrets are not configured (forks, contributors), the signing job is skipped entirely. The build job still publishes the release with unsigned binaries.

Note: GitHub Actions does not directly expose secrets in `if` conditions for security reasons. The standard pattern is to set a job output in the build job that checks for secret presence, then reference that output in the signing job's `if` condition.

## Decision 8: Stapling

**Decision**: Do not staple. Not needed and not possible for bare Mach-O binaries.

**Rationale**: `xcrun stapler staple` only works on `.app` bundles, `.dmg`, `.pkg`, and `.kext`. Bare binaries cannot be stapled. macOS Gatekeeper verifies notarization online — it contacts Apple's servers to check the notarization ticket at runtime. This is sufficient for CLI tools distributed via Homebrew or direct download.
