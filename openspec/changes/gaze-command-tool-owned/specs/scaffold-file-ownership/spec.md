## ADDED Requirements

### Requirement: `commands/gaze.md` is tool-owned

`gaze init` MUST classify the scaffolded `.opencode/commands/gaze.md`
as a tool-owned file. On re-runs without `--force`, an existing copy
whose content differs from the embedded asset MUST be overwritten with
the embedded content and reported in the summary as updated; an
identical copy MUST be skipped. `--force` MUST overwrite it
unconditionally, like every other scaffolded file. The classification
lives in `isToolOwned` and MUST NOT depend on file content, marker
state, or the presence of local edits.

#### Scenario: Stale command updated on plain re-run

- **GIVEN** a project scaffolded by an older gaze release whose
  `.opencode/commands/gaze.md` differs from the embedded asset of the
  current gaze binary
- **WHEN** `gaze init` runs without `--force`
- **THEN** `.opencode/commands/gaze.md` MUST be overwritten with the
  embedded content
- **AND** the summary MUST report it as
  `updated: .opencode/commands/gaze.md (content changed)`

#### Scenario: Identical command skipped

- **GIVEN** a project whose `.opencode/commands/gaze.md` is
  byte-identical to the embedded asset with the current version marker
  inserted
- **WHEN** `gaze init` runs without `--force`
- **THEN** the file MUST NOT be rewritten
- **AND** the summary MUST report it as skipped

#### Scenario: Local customization overwritten

- **GIVEN** a `.opencode/commands/gaze.md` containing local edits
- **WHEN** `gaze init` runs without `--force`
- **THEN** the local edits MUST be overwritten, because the file is
  tool-owned
- **AND** the on-disk content MUST equal the embedded asset with the
  current version marker inserted

#### Scenario: Separately-named commands untouched

- **GIVEN** a `.opencode/commands/` directory containing command files
  gaze does not ship (for example `gaze-local.md`)
- **WHEN** `gaze init` runs with or without `--force`
- **THEN** those files MUST NOT be read, modified, or removed

### Requirement: Single current version marker after rewrite

Whenever `gaze init` rewrites `commands/gaze.md` — through the
tool-owned update path or through `--force` — the result MUST contain
exactly one `<!-- scaffolded by gaze <version> -->` marker, carrying
the version of the rewriting run and placed after the YAML frontmatter.
This MUST also hold when the pre-existing file carries multiple
markers (the `dev` plus `v1.8.0` duplicates observed in issue #353),
because marker deduplication runs on every rewrite.

#### Scenario: Duplicate markers collapsed

- **GIVEN** a `.opencode/commands/gaze.md` carrying two scaffold
  markers and content older than the embedded asset
- **WHEN** `gaze init` runs without `--force` with version `v2.5.0`
- **THEN** the rewritten file MUST contain exactly one marker:
  `<!-- scaffolded by gaze v2.5.0 -->`
- **AND** the rewritten content MUST match the embedded asset with that
  marker inserted

#### Scenario: Marker-only drift still rewrites

- **GIVEN** a `.opencode/commands/gaze.md` whose content matches the
  embedded asset but whose marker carries a different version
- **WHEN** `gaze init` runs without `--force`**
- **THEN** the file MUST be rewritten so the marker carries the current
  version, and the summary MUST report it as updated

### Requirement: User-owned skip hint fires only for user-owned files

The scaffold summary's `N file(s) skipped (use --force to overwrite)`
hint MUST fire exactly when at least one skipped file is user-owned.
Because no `.opencode/` asset is user-owned after this change, a plain
`gaze init` re-run with unchanged content MUST NOT print the hint. The
CI workflow written by `gaze init --ci` remains user-owned, so a
`--ci` re-run with an existing workflow MUST still print the hint.

#### Scenario: Plain identical re-run without hint

- **GIVEN** a project scaffolded by the current gaze binary with no
  local modifications
- **WHEN** `gaze init` runs without `--force`
- **THEN** the summary MUST NOT contain `use --force to overwrite`

#### Scenario: CI workflow skip keeps the hint

- **GIVEN** a project with an existing
  `.github/workflows/gaze-ai-report.yml`
- **WHEN** `gaze init --ci` runs without `--force`
- **THEN** the summary MUST contain
  `1 file skipped (use --force to overwrite)`

### Requirement: Scaffold invariants preserved

This change alters ownership classification only. The 9-file embedded
asset manifest MUST be unchanged, and `.opencode/commands/gaze.md`
MUST remain byte-identical to
`internal/scaffold/assets/commands/gaze.md`.

#### Scenario: Asset mirror remains byte-identical

- **GIVEN** the gaze repository after this change
- **WHEN** `TestEmbeddedAssetsMatchSource` runs
- **THEN** it MUST pass with no changes to either copy of
  `commands/gaze.md`

#### Scenario: Manifest unchanged

- **WHEN** the embedded asset paths are enumerated
- **THEN** exactly 9 files MUST ship, identical to the pre-change
  manifest

### Requirement: Documentation reflects tool ownership

User-facing documentation MUST describe `commands/gaze.md` as
tool-owned, MUST NOT advise customizing it, and MUST direct users who
need a custom `/gaze` command to a separately-named command file that
gaze never touches.

#### Scenario: Ownership documentation updated

- **GIVEN** the documentation rendered after this change
- **WHEN** a reader consults the file-ownership guidance in
  `docs/guides/opencode-integration.md` or `docs/reference/cli/init.md`
- **THEN** `commands/gaze.md` MUST be documented as tool-owned with
  overwrite-on-diff behavior
- **AND** the separately-named-command escape hatch MUST be documented
  as the supported way to customize `/gaze`
