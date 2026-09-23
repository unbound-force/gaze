## Context

The `sign-macos` job currently maps `MACOS_SIGN_IDENTITY` from a GitHub
Actions variable with a hardcoded Apple Developer ID fallback. The preceding
secret-availability job checks only for the P12 certificate. Consequently, a
repository can enter the signed path with no configured secret-backed
identity, and the workflow remains tied to the fallback's developer identity.

The proposal's constitution alignment is PASS for both the OpenSpec
organization principles and the Gaze Constitution. The design preserves
autonomous configuration through GitHub Actions secrets, keeps unsigned
releases independently usable, exposes the selected path in the workflow job
graph, and uses isolated static validation rather than Apple services.

## Goals / Non-Goals

### Goals
- Source the macOS signing identity exclusively from
  `secrets.MACOS_SIGN_IDENTITY`.
- Treat the P12 certificate and signing identity as one complete signing
  configuration at the existing release branch point.
- Preserve unsigned releases when either required signing secret is absent.
- Make the sixth required secret clear in maintainer-facing documentation.
- Verify the workflow contract without exposing secret values or invoking
  `codesign` or `notarytool`.

### Non-Goals
- Change the certificate, notary credentials, keychain commands, notarization,
  process, cask publication, or release permissions.
- Add a secret value, rotate an Apple certificate, or modify repository
  settings; an authorized repository administrator performs provisioning.
- Change the behavior of releases that already have a complete signing-secret
  configuration.

## Decisions

### Require the identity in the existing readiness check

The `check-signing-secrets` job will map both required secret names into its
environment and report signing credentials available only when both values are
non-empty. This keeps the existing release branch decision in one place and
avoids a late `codesign` failure for an incomplete identity configuration.

### Remove the variable and hardcoded fallback

The `sign-macos` step will map `MACOS_SIGN_IDENTITY` directly from
`secrets.MACOS_SIGN_IDENTITY`. Removing both `vars.MACOS_SIGN_IDENTITY` and
the fallback makes the repository-secret contract unambiguous and avoids
coupling releases to a named developer identity.

### Validate statically and in CI

Implementation will add an isolated regression check that reads the workflow
as configuration and asserts the two-secret condition, direct identity mapping,
and absence of the variable/fallback expression. It will not load or print
secret values. The existing CI YAML linting and the project's mandated local
CI-parity commands remain the syntax and integration gate. A release run with
configured repository secrets is the operational acceptance check.

### Documentation location

The implementation will update the README macOS code-signing maintainer
section, including its link from superseded spec 014 to the native macOS-
signing quickstart in spec 015, which lists the five current secrets.
Historical spec 014 records remain out of scope.

## Risks / Trade-offs

- Repositories with a P12 secret but no identity secret will publish unsigned
  artifacts after this change rather than silently use the historical
  fallback. This is intentional and requires administrator provisioning
  before the next signed release.
- GitHub masks secrets and does not expose their values to local tests, so the
  automated regression check validates configuration structure rather than a
  real signing operation.
- Adding a focused configuration test increases maintenance slightly but
  prevents a future fallback or partial-secret gate regression.
