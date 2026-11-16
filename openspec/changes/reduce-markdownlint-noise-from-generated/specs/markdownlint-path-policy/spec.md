## ADDED Requirements

### Requirement: Markdown Path Classification

The repository's Markdown surface MUST be classified into exactly four ownership buckets: user-owned, generated, tool-owned (including vendored), and historical/structured. Any tracked `*.md` path that matches no documented exclusion MUST be treated as user-owned and linted.

#### Scenario: Unmatched path defaults to user-owned

- **GIVEN** a newly added `NOTES.md` at the repository root that matches no exclusion fragment
- **WHEN** the markdownlint scope is computed
- **THEN** the file MUST be linted

#### Scenario: Classification is exhaustive over tracked Markdown

- **GIVEN** the set of tracked `*.md` files at the time of this change
- **WHEN** each path is evaluated against the exclusion policy
- **THEN** every path MUST resolve to exactly one bucket
- **AND** the bucket-to-path mapping MUST be recorded in the policy documentation

### Requirement: Non-User-Owned Paths Are Excluded from Markdownlint

Generated, tool-owned, vendored, and historical/structured Markdown paths MUST be excluded from markdownlint, and the exclusion MUST be declared in `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` — never in the global `FILTER_REGEX_EXCLUDE`. The excluded set MUST cover every tracked Markdown tree that is not user-owned; the scenarios below enumerate the required coverage.

#### Scenario: Vendored schema templates are not linted

- **GIVEN** `openspec/schemas/unbound-force/templates/proposal.md` containing MD041 and MD033 violations
- **WHEN** MegaLinter computes the markdownlint scope
- **THEN** the file MUST NOT appear in the analyzed file set

#### Scenario: Dewey learning records are not linted

- **GIVEN** a regenerated `.uf/dewey/learnings/` Markdown record
- **WHEN** MegaLinter computes the markdownlint scope
- **THEN** the record MUST NOT appear in the analyzed file set

#### Scenario: Archived and active OpenSpec changes share one rule

- **GIVEN** `openspec/changes/<name>/proposal.md` (active) and `openspec/changes/archive/<date>-<name>/proposal.md` (archived)
- **WHEN** the exclusion is evaluated
- **THEN** both MUST be excluded by the same `openspec/changes/` fragment
- **AND** no per-file or per-change ad-hoc exclusion entries MAY exist for OpenSpec artifacts

#### Scenario: Tool-scaffolded and mirror trees are not linted

- **GIVEN** a Markdown file under `.opencode/`, under `.specify/`, under `internal/scaffold/assets/`, or under `internal/aireport/assets/`
- **WHEN** MegaLinter computes the markdownlint scope
- **THEN** the file MUST NOT appear in the analyzed file set

#### Scenario: Structured spec trees are not linted

- **GIVEN** a Markdown file under `specs/` (Speckit point-in-time artifacts) or under `openspec/specs/` (synced main specs)
- **WHEN** MegaLinter computes the markdownlint scope
- **THEN** the file MUST NOT appear in the analyzed file set

### Requirement: User-Owned Markdown Remains Linted

`README.md`, `AGENTS.md`, `CLAUDE.md`, and `docs/**/*.md` MUST remain in the markdownlint scope, and every markdownlint rule active before this change MUST remain active for those files, except for narrowly scoped, documented rule adjustments.

#### Scenario: A docs violation is reported

- **GIVEN** `docs/guides/example.md` containing a fenced code block with no language (MD040)
- **WHEN** MegaLinter runs markdownlint
- **THEN** the file MUST be in the analyzed set and the violation MUST be reported

#### Scenario: Root documentation stays in scope

- **GIVEN** `README.md`, `AGENTS.md`, and `CLAUDE.md` at the repository root
- **WHEN** MegaLinter computes the markdownlint scope
- **THEN** all three files MUST be included

### Requirement: Every Exclusion Documented

Each exclusion fragment MUST carry its ownership bucket and rationale as comments in `.mega-linter.yml` adjacent to the exclusion regex, and the policy MUST be summarized in `AGENTS.md`. A new generated or tool-owned Markdown path MUST receive a documented exclusion decision (or an explicit decision to keep it linted) before the change introducing it merges.

#### Scenario: Rationale lives with the configuration

- **GIVEN** the `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` line in `.mega-linter.yml`
- **WHEN** a reviewer reads the comment block directly above it
- **THEN** every regex fragment MUST map to a named bucket and a reason why hand-fixing is not durable or the content is not author-owned

#### Scenario: AGENTS.md carries the convention

- **GIVEN** the MegaLinter section of `AGENTS.md`
- **WHEN** a contributor adds a new tool that writes Markdown into the repository
- **THEN** the documented convention MUST instruct them to classify the path and record an exclusion decision before merge

### Requirement: Mirror Pair Integrity

Byte-identical mirror trees (`internal/scaffold/assets/` and `internal/aireport/assets/` versus their `.opencode/` sources, guarded by `TestEmbeddedAssetsMatchSource`) MUST be linted on both sides or neither — never exactly one side. Markdownlint fix passes MUST NOT be executed against only one side of a mirror pair.

#### Scenario: Mirror byte-identity survives lint fixes

- **GIVEN** this change applies markdownlint fixes to user-owned files
- **WHEN** the fix pass runs
- **THEN** no file under `internal/scaffold/assets/` or `internal/aireport/assets/` MAY be modified
- **AND** `TestEmbeddedAssetsMatchSource` MUST pass
