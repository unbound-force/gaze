# Quickstart: macOS Code Signing and Notarization

**Feature**: 014-macos-notarization
**Date**: 2026-03-01

## Prerequisites

Before implementing this feature, the project maintainer must complete these manual steps:

### 1. Apple Developer Program

- Enroll at [developer.apple.com](https://developer.apple.com/programs/) ($99/year)
- Note your **Team ID** (shown in developer account settings)

### 2. Create Developer ID Application Certificate

1. Go to Apple Developer > Certificates, Identifiers & Profiles > Certificates
2. Click "+" to create a new certificate
3. Select **Developer ID Application**
4. Follow the CSR generation steps (Keychain Access > Certificate Assistant > Request a Certificate)
5. Download and install the certificate into your Keychain
6. Export as `.p12`:
   - Open Keychain Access
   - Find the "Developer ID Application: [Your Name]" certificate
   - Right-click > Export Items
   - Choose `.p12` format
   - Set a strong password (you'll need this as `MACOS_SIGN_PASSWORD`)
7. Base64 encode: `base64 -w0 < DeveloperIDApplication.p12 | pbcopy`

### 3. Create App Store Connect API Key

1. Go to [appstoreconnect.apple.com](https://appstoreconnect.apple.com) > Users and Access > Integrations > App Store Connect API
2. Click "+" to generate a new key
3. Name: `gaze-notarization` (or similar)
4. Access: **Developer** role (minimum needed for notarization)
5. Download the `.p8` key file (you can only download it once)
6. Note the **Key ID** (shown in the keys list, also in the filename: `AuthKey_XXXXXXXX.p8`)
7. Note the **Issuer ID** (shown at the top of the API keys page)
8. Base64 encode: `base64 -w0 < AuthKey_XXXXXXXX.p8 | pbcopy`

### 4. Configure GitHub Secrets

Go to GitHub repository > Settings > Secrets and variables > Actions > New repository secret.

Add these 5 secrets:

| Secret Name | Value |
|-------------|-------|
| `MACOS_SIGN_P12` | Base64-encoded `.p12` certificate (from step 2) |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` file (from step 2) |
| `MACOS_NOTARY_KEY` | Base64-encoded `.p8` API key (from step 3) |
| `MACOS_NOTARY_KEY_ID` | Key ID string (from step 3) |
| `MACOS_NOTARY_ISSUER_ID` | Issuer UUID (from step 3) |

## Implementation Steps

### Step 1: Update `.goreleaser.yaml`

Add the `notarize` section after the existing `checksum` section:

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

### Step 2: Update `.github/workflows/release.yml`

Add the 5 Apple credential environment variables to the GoReleaser step:

```yaml
- name: Run GoReleaser
  uses: goreleaser/goreleaser-action@ec59f474b9834571250b370d4735c50f8e2d1e29  # v7.0.0
  with:
    distribution: goreleaser
    version: '~> v2'
    args: release --clean
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
    MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
    MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}
    MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}
    MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}
    MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
```

## Verification

### Local Dry-Run (without secrets)

```bash
goreleaser release --snapshot --clean
```

This verifies the YAML syntax is valid and that notarization is gracefully skipped when secrets are absent.

### First Signed Release

1. Push a test tag: `git tag v0.X.Y-rc.1 && git push origin v0.X.Y-rc.1`
2. Monitor the GitHub Actions release workflow
3. Verify the workflow log shows signing and notarization steps completing
4. Download the darwin binary from the GitHub Release
5. On a macOS machine, verify the signature:

```bash
# Check code signature
codesign --verify --deep --strict --verbose=2 gaze

# Check Gatekeeper assessment
spctl --assess --type execute --verbose gaze

# Check notarization status
xcrun stapler validate gaze  # Note: may not apply to bare binaries
```

### Homebrew Verification

```bash
brew install --cask gaze
gaze --version  # Should run without Gatekeeper warning
```
