# Quickstart: Native macOS Code Signing and Notarization

**Feature**: 015-native-macos-signing
**Date**: 2026-03-02

## Prerequisites

Configure these 6 GitHub secrets before running a signed release:

| Secret Name | Purpose |
|-------------|---------|
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application certificate. |
| `MACOS_SIGN_PASSWORD` | Password for the `.p12` certificate. |
| `MACOS_SIGN_IDENTITY` | Exact Developer ID Application certificate label displayed in the temporary macOS Keychain; passed to `codesign --sign` to select the imported signing certificate. |
| `MACOS_NOTARY_KEY` | Base64-encoded App Store Connect API private key. |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID. |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect API issuer ID. |

The first five secrets came from spec 014. Add `MACOS_SIGN_IDENTITY` with the certificate label exactly as `security find-identity -v -p codesigning` reports it; the release workflow uses that label to select the imported certificate. If you have not configured the first five secrets, see `specs/014-macos-notarization/quickstart.md` for their setup instructions.

## Implementation Steps

### Step 1: Remove quill config from `.goreleaser.yaml`

Delete the entire `notarize.macos` section (lines 30-41 in the current file):

```yaml
# DELETE THIS ENTIRE BLOCK:
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
        wait: false
```

### Step 2: Update `.github/workflows/release.yml`

1. Remove the 5 `MACOS_*` env vars from the GoReleaser step (GoReleaser no longer needs them)
2. Add a `sign-macos` job after the `release` job

The `sign-macos` job structure:

```yaml
sign-macos:
  runs-on: macos-latest
  needs: release
  if: ${{ needs.release.outputs.has_signing_secrets == 'true' }}
  timeout-minutes: 30
  steps:
    - name: Import certificate into Keychain
      # Decode .p12, create temp keychain, import cert, set partition list

    - name: Prepare notary key
      # Decode .p8 to temp file

    - name: Download darwin archives
      # gh release download $TAG --pattern "gaze_*_darwin_*.tar.gz"

    - name: Sign and notarize
      # For each archive: extract, codesign, verify, zip, notarytool submit --wait, re-tar.gz

    - name: Replace release assets and update checksums
      # gh release upload --clobber for signed archives + updated checksums.txt
```

A separate `check-signing-secrets` job signals whether signing secrets are available:

```yaml
check-signing-secrets:
  needs: preflight
  runs-on: ubuntu-latest
  outputs:
    has_signing_secrets: ${{ steps.check-secrets.outputs.has_secrets }}
  steps:
    - name: Check signing secrets
      id: check-secrets
      run: |
        if [ -n "$MACOS_SIGN_P12" ] && [ -n "$MACOS_SIGN_IDENTITY" ]; then
          echo "has_secrets=true" >> "$GITHUB_OUTPUT"
        else
          echo "has_secrets=false" >> "$GITHUB_OUTPUT"
        fi
      env:
        MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}
        MACOS_SIGN_IDENTITY: ${{ secrets.MACOS_SIGN_IDENTITY }}
```

## Verification

### After Implementation (dry run)

```bash
# Validate GoReleaser config (quill section removed)
goreleaser check

# Verify snapshot build works without quill
goreleaser release --snapshot --clean
```

### First Signed Release

1. Push a test tag: `git tag v0.X.Y-rc.1 && git push origin v0.X.Y-rc.1`
2. Monitor GitHub Actions:
   - `release` job should complete on `ubuntu-latest`
   - `sign-macos` job should start on `macos-latest`
   - Watch for: "signing", "notarizing", "upload" steps
3. After `sign-macos` completes, download the darwin binary:

```bash
gh release download v0.X.Y-rc.1 --pattern "gaze_*_darwin_arm64*" --dir ./test
tar -xzf ./test/gaze_*_darwin_arm64*.tar.gz -C ./test
```

4. Verify on macOS:

```bash
# Check code signature (should show TeamIdentifier)
codesign -dv --verbose=4 ./test/gaze

# Check Gatekeeper assessment (should say "accepted")
spctl --assess --type execute --verbose=2 ./test/gaze
```

5. Verify checksums:

```bash
gh release download v0.X.Y-rc.1 --pattern "checksums.txt" --dir ./test
cd ./test && shasum -a 256 -c checksums.txt
```

### Homebrew Verification

```bash
brew install --cask gaze
gaze --version  # Should run without Gatekeeper warning
```

### Verify Graceful Degradation

On a fork without secrets configured, tag a release and verify:
- `release` job succeeds
- `sign-macos` job is skipped (not failed)
- Release contains unsigned but functional binaries
