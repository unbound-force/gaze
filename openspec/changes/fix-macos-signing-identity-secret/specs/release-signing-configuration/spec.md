## ADDED Requirements

### Requirement: Complete macOS signing-secret gate

The release workflow MUST consider macOS signing credentials available only
when `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_SIGN_IDENTITY`,
`MACOS_NOTARY_KEY`, `MACOS_NOTARY_KEY_ID`, and `MACOS_NOTARY_ISSUER_ID` are
non-empty GitHub Actions secrets. The workflow MUST route releases with an
incomplete signing secret set through the existing unsigned-cask path and MUST
NOT run the `sign-macos` job.

#### Scenario: Complete signing configuration runs macOS signing
- **GIVEN** the repository provides all six non-empty signing secrets
- **WHEN** a release reaches the signing stage
- **THEN** the signing-secret check reports signing credentials available and
  the `sign-macos` job is eligible to run

#### Scenario: Missing signing identity preserves unsigned release
- **GIVEN** the repository provides the other five signing secrets but does
  not provide `MACOS_SIGN_IDENTITY`
- **WHEN** a release reaches the signing stage
- **THEN** the signing-secret check reports signing credentials unavailable,
  the `sign-macos` job is skipped, and the unsigned-cask job remains eligible

#### Scenario: Another missing signing secret preserves unsigned release
- **GIVEN** the repository does not provide one of the other five signing
  secrets
- **WHEN** a release reaches the signing stage
- **THEN** the signing-secret check reports signing credentials unavailable,
  the `sign-macos` job is skipped, and the unsigned-cask job remains eligible

### Requirement: Secret-backed signing identity

The `sign-macos` job MUST provide `MACOS_SIGN_IDENTITY` to `codesign` from
`secrets.MACOS_SIGN_IDENTITY`. The workflow MUST NOT contain a developer-
specific fallback identity or source the identity from a GitHub Actions
variable.

#### Scenario: Signing uses the repository secret
- **GIVEN** the macOS signing job is eligible to run
- **WHEN** it invokes `codesign`
- **THEN** the command receives the identity through the
  `MACOS_SIGN_IDENTITY` environment variable mapped from
  `secrets.MACOS_SIGN_IDENTITY`

### Requirement: macOS signing maintainer prerequisites

The macOS-signing documentation MUST list `MACOS_SIGN_IDENTITY` as the sixth
required GitHub secret alongside the P12 certificate, certificate password,
and three notary credentials. It MUST explain that its value is the current
Apple Developer ID Application certificate label.

#### Scenario: Maintainer provisions signing secrets
- **GIVEN** a maintainer prepares repository secrets for macOS signing
- **WHEN** they follow the documented prerequisites
- **THEN** they can identify all six required secret names and the expected
  purpose of `MACOS_SIGN_IDENTITY`
