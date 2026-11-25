## ADDED Requirements

### Requirement: Tool-owned `/gaze` command file

`commands/gaze.md` MUST be classified as tool-owned by `gaze init` and
use overwrite-on-diff semantics: when the file exists and its content
differs from the embedded asset (rendered with the current version
marker), `gaze init` MUST replace it without `--force` and report it as
`updated` in the summary. The embedded asset content and the nine-file
byte-identical mirror manifest MUST NOT change.

#### Scenario: Stale copy is updated

- **GIVEN** a project whose `commands/gaze.md` was scaffolded by an
  older gaze and differs from the current embedded asset
- **WHEN** `gaze init` runs without `--force`
- **THEN** the file MUST be replaced with the embedded asset content
  plus exactly one current-version marker
- **AND** the summary MUST report it under `updated:`

#### Scenario: Identical copy is skipped

- **GIVEN** a project whose `commands/gaze.md` is byte-identical to the
  embedded asset rendered with the current version marker
- **WHEN** `gaze init` runs without `--force`
- **THEN** the file MUST be skipped and appear under `skipped:`
- **AND** it MUST NOT be counted by the `--force` hint (no user-owned
  `.opencode/` file was skipped)

#### Scenario: First-run creation is unchanged

- **GIVEN** a project with no `commands/gaze.md`
- **WHEN** `gaze init` runs
- **THEN** the file MUST be created and reported under `created:` with
  exactly one current-version marker

#### Scenario: Duplicate version markers are healed

- **GIVEN** a project whose `commands/gaze.md` contains two or more
  `<!-- scaffolded by gaze ... -->` marker lines
- **WHEN** `gaze init` updates the file
- **THEN** the result MUST contain exactly one marker, carrying the
  current gaze version

#### Scenario: Customized copy is overwritten

- **GIVEN** a project whose `commands/gaze.md` carries local
  customizations that differ from the embedded asset
- **WHEN** `gaze init` runs without `--force`
- **THEN** the file MUST be overwritten with the embedded asset content
  (the documented preservation path is the escape hatch below; the
  prior version remains recoverable from version control)

### Requirement: Custom command escape hatch

`gaze init` MUST NOT read, write, or remove any `.opencode/commands/`
path outside the nine embedded assets — including under `--force`. Any
such path (conventionally `commands/gaze-local.md`) is a user-owned
escape hatch for a customized `/gaze` command, and OpenCode exposes it
as a slash command by filename with no gaze involvement.

#### Scenario: Escape-hatch file untouched without `--force`

- **GIVEN** a project with a customized `.opencode/commands/gaze-local.md`
- **WHEN** `gaze init` runs without `--force`
- **THEN** the file's content MUST be byte-identical before and after
- **AND** it MUST NOT appear in any summary list (`created`, `skipped`,
  `overwritten`, `updated`, `removed`)

#### Scenario: Escape-hatch file untouched under `--force`

- **GIVEN** a project with a customized `.opencode/commands/gaze-local.md`
- **WHEN** `gaze init --force` runs
- **THEN** the file's content MUST be byte-identical before and after
- **AND** it MUST NOT appear in any summary list

### Requirement: Escape-hatch guidance on update

When `gaze init` updates `commands/gaze.md`, the summary MUST print a
one-line tip naming `commands/gaze-local.md` as the path that preserves
a customized command. The tip MUST NOT print when the file is created,
skipped as identical, or overwritten via `--force`.

#### Scenario: Tip printed on update

- **GIVEN** a project whose `commands/gaze.md` differs from the
  embedded asset
- **WHEN** `gaze init` runs without `--force` and updates the file
- **THEN** the summary output MUST contain a tip line that names
  `commands/gaze-local.md`

#### Scenario: No tip on identical skip or creation

- **GIVEN** a project whose `commands/gaze.md` is identical to the
  embedded asset, or a fresh project receiving the file for the first
  time
- **WHEN** `gaze init` runs without `--force`
- **THEN** the summary output MUST NOT contain the escape-hatch tip
