# Research: macOS Code Signing and Notarization

**Feature**: 014-macos-notarization
**Date**: 2026-03-01

## Decision 1: Signing and Notarization Approach

**Decision**: Use GoReleaser v2 OSS built-in `notarize.macos` configuration section.

**Rationale**: GoReleaser v2 imports quill (by Anchore) as a compiled-in Go library. This means:

- Zero external tooling to install — quill is built into the GoReleaser binary.
- Cross-platform signing works from `ubuntu-latest` — quill reimplements Mach-O code signing in pure Go, no macOS `codesign` needed.
- Available in the free/OSS tier of GoReleaser (the `notarize.macos` section; not `notarize.macos_native` which is Pro-only).
- GoReleaser's own release pipeline uses this exact approach, proving it works on Linux CI runners.

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| `gon` (mitchellh/gon) | Archived (Oct 2023) | Uses deprecated `xcrun altool` API removed from recent Xcode. Latest release v0.2.5 (March 2022). Dead project. |
| GoReleaser `signs` section + manual `notarytool` | Viable but complex | Requires macOS runner (10x more expensive), three separate steps (sign, notarize, staple), more YAML configuration to maintain. |
| GoReleaser `notarize.macos_native` | Pro-only | Requires macOS runner and GoReleaser Pro license. Only needed for `.app` bundles and `.pkg` — not applicable to bare binaries. |
| Standalone quill CLI as post-hook | Viable but unnecessary | Would require installing quill separately in CI. GoReleaser already embeds it, so there's no benefit. |

## Decision 2: CI Runner

**Decision**: Stay on `ubuntu-latest`. No macOS runner needed.

**Rationale**: The cross-platform `notarize.macos` method (via quill) runs on any OS. Since quill is a pure Go library that directly manipulates Mach-O binary format, it does not depend on any macOS system tools (`codesign`, `xcrun`, Keychain).

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| `macos-latest` runner | Viable but expensive | macOS GitHub Actions runners cost ~10x more per minute than Ubuntu. Build times are longer. Only needed for `notarize.macos_native` (Pro) or `.app` bundles. |
| Split workflow (build on Ubuntu, sign on macOS) | Over-engineered | Adds workflow complexity for no benefit since quill signs from Linux. |

## Decision 3: Notarization Wait Strategy

**Decision**: Set `wait: true` with a `timeout: 20m`.

**Rationale**: Without `wait: true`, GoReleaser submits the notarization request as fire-and-forget. The binary would be signed but Apple's notarization ticket would not be confirmed before the release artifacts are published. Setting `wait: true` ensures the release only completes after Apple confirms the notarization, guaranteeing end users see a trusted binary. The 20-minute timeout accommodates Apple's variable processing times (typically <10 minutes, occasionally longer) while staying within the global GoReleaser timeout.

**Alternatives considered**:

| Alternative | Status | Reason rejected |
|-------------|--------|-----------------|
| `wait: false` (fire-and-forget) | Risky | Binary would be published before Apple confirms notarization. First users to download could still face Gatekeeper warnings. |
| Default 10m timeout | Too tight | Apple's service occasionally takes >10 minutes. A 20m timeout provides safety margin without being excessive. |

## Decision 4: Credential Format and Secrets

**Decision**: Use 5 GitHub secrets with base64-encoded credentials.

| Secret Name | Content | Source |
|-------------|---------|--------|
| `MACOS_SIGN_P12` | Base64-encoded `.p12` certificate | Apple Developer > Certificates > Developer ID Application > export from Keychain > `base64 -w0` |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` file | Set when exporting from Keychain |
| `MACOS_NOTARY_KEY` | Base64-encoded `.p8` API key | App Store Connect > Keys > API Keys > create key > `base64 -w0` |
| `MACOS_NOTARY_KEY_ID` | Key ID string | App Store Connect > Keys list (also in filename: `AuthKey_XXXXXXXX.p8`) |
| `MACOS_NOTARY_ISSUER_ID` | Issuer UUID | App Store Connect > Keys > Issuer ID at top of page |

**Rationale**: This is the credential format used by GoReleaser's own release pipeline and documented in the GoReleaser notarization docs. Base64 encoding of binary files (.p12, .p8) is required for storage in GitHub secrets. The `isEnvSet` template function enables graceful degradation when secrets are absent.

## Decision 5: Build ID Matching

**Decision**: Omit the `ids` field from the `notarize.macos` config.

**Rationale**: When `ids` is omitted, GoReleaser defaults to matching the project name. The gaze `.goreleaser.yaml` has a single build entry with no explicit `id` field, so it defaults to the project name "gaze". The notarization config will automatically match this default ID. GoReleaser's own config omits `ids` as well.

## Decision 6: Entitlements

**Decision**: Do not include entitlements. No `entitlements` field in the config.

**Rationale**: Gaze is a CLI tool that does not require any macOS entitlements (no camera access, no network server capability, no file system sandbox exceptions, etc.). Entitlements are only needed for apps requesting specific macOS capabilities beyond basic execution.

## Risk Assessment

### Known quill Issues

| Issue | Severity | Mitigation |
|-------|----------|------------|
| [#595](https://github.com/anchore/quill/issues/595): JWT notarization errors | Medium | Reported Sep 2025. Monitor before first release with signing. If encountered, may need to update GoReleaser to a version with a patched quill dependency. |
| [#573](https://github.com/anchore/quill/issues/573): FORBIDDEN.MISSING_PROVIDER | Medium | Filed by quill maintainer (May 2025). Related to multi-team Apple accounts. Gaze likely uses a single-team account, reducing exposure. |
| [#147](https://github.com/anchore/quill/issues/147): TeamID not set during signing | Low | Open since Sep 2023. May not affect CLI binaries distributed outside the App Store. Monitor. |

### Mitigation Strategy

- Perform a dry-run release on a test tag before the first production release with signing.
- The `enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'` guard means signing failures don't block unsigned releases.
- If quill bugs surface, the fallback is to temporarily disable notarization and release unsigned while investigating.

## Exact Configuration

### `.goreleaser.yaml` Addition

```yaml
notarize:
  macos:
    - enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'
      sign:
        certificate: "{{.Env.MACOS_SIGN_P12}}"
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"
      notarize:
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"
        key: "{{.Env.MACOS_NOTARY_KEY}}"
        wait: true
        timeout: 20m
```

### `.github/workflows/release.yml` Addition

Add these environment variables to the GoReleaser step:

```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
  MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
  MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}
  MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
  MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
  MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
```
