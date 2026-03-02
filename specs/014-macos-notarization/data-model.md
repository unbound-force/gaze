# Data Model: macOS Code Signing and Notarization

**Feature**: 014-macos-notarization
**Date**: 2026-03-01

## Overview

This feature has no application-level data model (no new Go types, no database entities). The "data model" consists of configuration entities in YAML files and CI/CD secrets. This document defines the structure and relationships of those entities.

## Configuration Entities

### GoReleaser Notarize Configuration

**Location**: `.goreleaser.yaml` > `notarize.macos[]`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `enabled` | template string | No | `"false"` | GoReleaser template expression. Must evaluate to `"true"` or `"false"`. Uses `isEnvSet` for conditional activation. |
| `ids` | string[] | No | Project name | Build IDs to match for signing. Omitted to use project default. |
| `sign.certificate` | template string | Yes (when enabled) | — | Base64-encoded `.p12` certificate or file path. |
| `sign.password` | template string | Yes (when enabled) | — | Password for the `.p12` certificate. |
| `notarize.issuer_id` | template string | Yes (when enabled) | — | App Store Connect API issuer UUID. |
| `notarize.key_id` | template string | Yes (when enabled) | — | App Store Connect API key ID. |
| `notarize.key` | template string | Yes (when enabled) | — | Base64-encoded `.p8` API key or file path. |
| `notarize.wait` | boolean | No | `false` | Whether to block until Apple confirms notarization. |
| `notarize.timeout` | duration | No | `10m` | Maximum time to wait for notarization response. |

**Relationships**:
- `ids` references the `id` field of entries in `builds[]`. When omitted, defaults to the project name.
- `sign.certificate` and `sign.password` reference CI/CD secrets via GoReleaser template syntax (`{{.Env.VAR}}`).
- `notarize.*` fields reference CI/CD secrets via the same template syntax.

### CI/CD Secrets

**Location**: GitHub repository settings > Secrets and variables > Actions

| Secret Name | Content Type | Encoding | Source |
|-------------|-------------|----------|--------|
| `MACOS_SIGN_P12` | Binary (.p12 file) | Base64 | `base64 -w0 < DeveloperIDApplication.p12` |
| `MACOS_SIGN_PASSWORD` | Plain text | None | Password set during Keychain export |
| `MACOS_NOTARY_KEY` | Binary (.p8 file) | Base64 | `base64 -w0 < AuthKey_XXXXXXXX.p8` |
| `MACOS_NOTARY_KEY_ID` | Plain text | None | Key ID from App Store Connect (e.g., `XS319FABCD`) |
| `MACOS_NOTARY_ISSUER_ID` | UUID | None | Issuer ID from App Store Connect Keys page |

**Relationships**:
- Secrets are mapped to environment variables in `.github/workflows/release.yml`.
- GoReleaser reads these environment variables via `{{.Env.VAR}}` template syntax.
- The `isEnvSet "MACOS_SIGN_P12"` guard checks for the presence of the signing certificate to enable/disable the entire notarization flow.

### Workflow Environment Variables

**Location**: `.github/workflows/release.yml` > `jobs.release.steps[Run GoReleaser].env`

| Env Var | Source | Existing/New |
|---------|--------|-------------|
| `GITHUB_TOKEN` | `${{ secrets.GITHUB_TOKEN }}` | Existing |
| `HOMEBREW_TAP_GITHUB_TOKEN` | `${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}` | Existing |
| `MACOS_SIGN_P12` | `${{ secrets.MACOS_SIGN_P12 }}` | **New** |
| `MACOS_SIGN_PASSWORD` | `${{ secrets.MACOS_SIGN_PASSWORD }}` | **New** |
| `MACOS_NOTARY_KEY` | `${{ secrets.MACOS_NOTARY_KEY }}` | **New** |
| `MACOS_NOTARY_KEY_ID` | `${{ secrets.MACOS_NOTARY_KEY_ID }}` | **New** |
| `MACOS_NOTARY_ISSUER_ID` | `${{ secrets.MACOS_NOTARY_ISSUER_ID }}` | **New** |

## State Transitions

### Notarization Lifecycle

```text
[Build Complete] → [Signing] → [Signed] → [Notarization Submitted] → [Waiting] → [Notarized] → [Published]
                                                                          ↓
                                                                    [Timeout/Error] → [Release Fails]
```

When secrets are absent:

```text
[Build Complete] → [Signing Skipped] → [Published (unsigned)]
```

## Validation Rules

- `MACOS_SIGN_P12` must be a valid base64-encoded PKCS#12 file containing a Developer ID Application certificate and private key.
- `MACOS_SIGN_PASSWORD` must correctly decrypt the `.p12` file.
- `MACOS_NOTARY_KEY` must be a valid base64-encoded `.p8` private key issued by App Store Connect.
- `MACOS_NOTARY_KEY_ID` must match the key ID associated with the `.p8` key.
- `MACOS_NOTARY_ISSUER_ID` must be a valid UUID matching the Apple Developer team's issuer.
- The `.p12` certificate must not be expired or revoked.
- The App Store Connect API key must have active status.
