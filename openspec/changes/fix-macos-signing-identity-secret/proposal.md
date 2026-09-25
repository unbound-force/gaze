## Why

The macOS signing job falls back to a specific Apple Developer ID identity in
the release workflow. This couples releases to one developer identity and
does not match the workflow's existing secret-based configuration for signing
material. Repository administrators need to rotate the identity without a
source change while preserving the existing unsigned-release behavior when
signing credentials are unavailable.

## What Changes

- Replace the `vars.MACOS_SIGN_IDENTITY` expression and hardcoded fallback in
  the `sign-macos` job with `secrets.MACOS_SIGN_IDENTITY`.
- Extend the signing-secret availability check to require all six secrets used
  by the signing job so a partially configured secret set takes the existing
  unsigned-cask path.
- Document the required repository secret in the macOS signing maintainer
  guidance.
- Add a workflow-focused regression check that verifies the signing job uses
  the secret and that the unsigned fallback remains guarded by complete
  signing-secret availability.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `release-signing-configuration`: Configure the macOS signing identity as a
  required GitHub Actions secret and treat incomplete signing configuration as
  unavailable signing credentials.

### Removed Capabilities
- None.

## Impact

- `.github/workflows/release.yml` changes the signing-identity source and the
  secret-readiness condition.
- The readiness condition now validates the existing certificate password and
  notary credentials together with the P12 certificate and signing identity.
- macOS-signing maintainer documentation gains `MACOS_SIGN_IDENTITY` as a
  required secret.
- CI validation gains a focused regression check for the workflow contract.
- A repository administrator must provision `MACOS_SIGN_IDENTITY` before the
  signed release path can run; releases without the complete secret set remain
  unsigned by design.

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

The workflow continues to obtain signing configuration through GitHub Actions
inputs rather than coupling the release process to a developer-specific value
in source. The secret name makes the required operator action explicit.

### II. Composability First

**Assessment**: PASS

The change adds no runtime dependency and preserves both release modes:
repositories with a complete signing-secret set sign macOS artifacts, while
repositories without it publish unsigned artifacts.

### III. Observable Quality

**Assessment**: PASS

The workflow's branch condition remains observable in the release job graph,
and the focused regression check will verify the configured secret contract
without exposing its value.

### IV. Testability

**Assessment**: PASS

The implementation plan includes an isolated static workflow regression test
for complete and incomplete signing configuration. It will assert observable
workflow expressions and branch behavior without requiring GitHub secrets or
Apple signing services.

### Gaze Constitution Alignment

The repository's Gaze Constitution also governs this implementation.

#### I. Accuracy

**Assessment**: PASS

The release gate accurately selects the signed macOS path only when all six
secrets consumed by the signing job are present. Extending the existing check
to the certificate password and notary credentials is a minor defensive scope
expansion that prevents predictable late signing failures.

#### II. Minimal Assumptions

**Assessment**: PASS

The workflow validates all six required values using GitHub Actions secrets.
It introduces no developer-specific identity or external runtime dependency.

#### III. Actionable Output

**Assessment**: PASS

When any required signing secret is unavailable, the existing unsigned-cask
path remains eligible and the signing job is skipped.

#### IV. Testability

**Assessment**: PASS

The workflow contract is verified by an isolated static test that reads only
repository configuration and checks the six-secret gate, direct identity
mapping, and signed-versus-unsigned job routing without live credentials or
Apple services.
