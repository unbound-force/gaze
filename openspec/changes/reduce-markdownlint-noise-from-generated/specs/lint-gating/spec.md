## ADDED Requirements

### Requirement: Markdownlint Gate Is Blocking

`.mega-linter.yml` MUST set `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS: false` so that markdownlint violations fail the MegaLinter check instead of being downgraded to non-blocking warnings by the formatter default of the pinned MegaLinter version. The gate MUST NOT rely on `MARKDOWN_MARKDOWNLINT_DISABLE_ERRORS_IF_LESS_THAN` count thresholds to tolerate existing violations.

#### Scenario: A new violation fails the check

- **GIVEN** a pull request that introduces a non-auto-fixable markdownlint violation (for example MD040) in a covered user-owned file
- **WHEN** the MegaLinter check runs on that pull request
- **THEN** the check MUST fail

#### Scenario: Warning-only reporting is eliminated

- **GIVEN** a full-codebase MegaLinter run on `main` after this change
- **WHEN** the markdownlint linter completes
- **THEN** the log MUST NOT report markdownlint errors as "non blocking"
- **AND** a run with zero markdownlint errors MUST succeed

### Requirement: Zero-Error Baseline for User-Owned Scope

At the completion of this change, the user-owned Markdown scope (`README.md`, `AGENTS.md`, `CLAUDE.md`, `docs/**`) MUST produce zero markdownlint errors. Existing violations MUST be fixed rather than tolerated by a threshold, so that any subsequent error is by construction a newly introduced violation.

#### Scenario: The full user-owned scope is clean

- **GIVEN** the repository at the completion of this change
- **WHEN** markdownlint runs over the complete user-owned file set
- **THEN** it MUST report zero errors and exit successfully

#### Scenario: Auto-fixable debt does not linger invisibly

- **GIVEN** formatting violations that markdownlint can auto-fix (for example MD032 blank lines)
- **WHEN** this change's scoped fix pass runs over user-owned files
- **THEN** those violations MUST be resolved in the committed tree rather than repaired only ephemerally by CI's `--fix` pass

### Requirement: CI Output Distinguishes Ignored Paths from Genuine Failures

Excluded paths MUST NOT appear in markdownlint's analyzed file set, and genuine markdownlint failures MUST surface as check failures rather than warnings, so that a failing Markdown result unambiguously indicates an author-owned regression and a passing result unambiguously indicates a clean user-owned scope.

#### Scenario: The analyzed file count reflects the classified scope

- **GIVEN** the repository after this change
- **WHEN** a full-codebase MegaLinter run reports the markdownlint analyzed-file count
- **THEN** the count MUST equal the user-owned scope and MUST NOT include any path from the generated, tool-owned, or historical buckets

#### Scenario: A genuine failure is an error, not a warning

- **GIVEN** a markdownlint violation in a user-owned file
- **WHEN** MegaLinter processes the linter result
- **THEN** the MegaLinter check MUST conclude as failed
- **AND** the markdownlint entry in the summary MUST NOT be presented with the warning indicator

### Requirement: Gate Narrowing Must Not Weaken User-Owned Coverage

Exclusion changes MUST be scoped to `MARKDOWN_MARKDOWNLINT_FILTER_REGEX_EXCLUDE` and MUST NOT be routed through the global `FILTER_REGEX_EXCLUDE`, because the global variable also governs yamllint and gitleaks coverage. The change MUST NOT disable markdownlint for user-owned paths and MUST NOT disable markdownlint rules wholesale; any rule configuration adjustment MUST be narrowly scoped and its rationale documented in `.mega-linter.yml` comments and `AGENTS.md`.

#### Scenario: The global exclusion is untouched

- **GIVEN** `.mega-linter.yml` before and after this change
- **WHEN** the `FILTER_REGEX_EXCLUDE` values are compared
- **THEN** they MUST be identical

#### Scenario: Rule adjustments are scoped and documented

- **GIVEN** the `MD024` `siblings_only` adjustment made by this change
- **WHEN** a reviewer inspects `.markdownlint.json` and the documentation
- **THEN** no rule MAY be set to `false` that was enabled before this change
- **AND** the rationale for the `MD024` adjustment MUST be documented outside the JSON config, which cannot carry comments
