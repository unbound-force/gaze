# Quickstart: Native macOS Code Signing and Notarization

**Feature**: 015-native-macos-signing
**Date**: 2026-03-02

## Prerequisites

The 5 GitHub secrets from spec 014 must already be configured:

| Secret Name | Already configured? |
|-------------|-------------------|
| `MACOS_SIGN_P12` | Yes (from spec 014) |
| `MACOS_SIGN_PASSWORD` | Yes (from spec 014) |
| `MACOS_NOTARY_KEY` | Yes (from spec 014) |
| `MACOS_NOTARY_KEY_ID` | Yes (from spec 014) |
| `MACOS_NOTARY_ISSUER_ID` | Yes (from spec 014) |

No new secrets are needed. If you haven't configured these yet, see `specs/014-macos-notarization/quickstart.md` for setup instructions.

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

The `release` job needs an output to signal whether signing secrets are available:

```yaml
release:
  outputs:
    has_signing_secrets: ${{ steps.check-secrets.outputs.has_secrets }}
  steps:
    - name: Check signing secrets
      id: check-secrets
      run: |
        if [ -n "${{ secrets.MACOS_SIGN_P12 }}" ]; then
          echo "has_secrets=true" >> "$GITHUB_OUTPUT"
        else
          echo "has_secrets=false" >> "$GITHUB_OUTPUT"
        fi
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
