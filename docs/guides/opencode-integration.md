# OpenCode Integration

Gaze integrates with [OpenCode](https://opencode.ai) to provide AI-assisted quality reporting directly from your editor. The `gaze init` command scaffolds agent definitions, command files, and reference materials into your project so that OpenCode can run Gaze analysis and produce human-readable reports.

## What `gaze init` Does

The [`gaze init`](../reference/cli/init.md) command writes OpenCode configuration files into the `.opencode/` directory of your project:

```bash
gaze init
```

Output:

```text
Notice: no .uf directory found; skipping Divisor review-config registration.
Gaze never creates .uf files; unbound-force projects register the persona automatically.

Gaze OpenCode integration initialized:
  created: .opencode/agents/divisor-gaze-testing.md
  created: .opencode/agents/gaze-reporter.md
  created: .opencode/agents/gaze-test-generator.md
  created: .opencode/commands/gaze-fix.md
  created: .opencode/commands/gaze.md
  created: .opencode/commands/speckit.testreview.md
  created: .opencode/dcp.jsonc
  created: .opencode/references/doc-scoring-model.md
  created: .opencode/references/example-report.md

Run /gaze for quality reports and /speckit.testreview for testability analysis.
```

Use `--force` to overwrite all existing files:

```bash
gaze init --force
```

## Scaffolded Files

The scaffold creates 9 files: three subdirectories plus the DCP configuration at the `.opencode/` root:

### Agents

| File | Description | Ownership |
|------|-------------|-----------|
| `agents/gaze-reporter.md` | Quality report agent — shows the Gaze-rendered markdown report verbatim and adds judgment (narrative plus prioritized recommendations) | Tool-owned |
| `agents/gaze-test-generator.md` | Test generation agent — generates Go test functions from Gaze quality data (gap hints, gaps, fix strategies) | Tool-owned |
| `agents/divisor-gaze-testing.md` | Testing persona for review councils — universal gaze test-quality rubric with conditional Go-native and external-analyzer sections | Tool-owned |

### Commands

| File | Description | Ownership |
|------|-------------|-----------|
| `commands/gaze.md` | `/gaze` command — runs `gaze report --format=markdown` and delegates to the gaze-reporter agent, which shows the rendered report and adds judgment | User-owned |
| `commands/gaze-fix.md` | `/gaze fix` command — batch remediation using the gaze-test-generator agent | Tool-owned |
| `commands/speckit.testreview.md` | `/speckit.testreview` command — read-only spec testability analysis | Tool-owned |

### Configuration

| File | Description | Ownership |
|------|-------------|-----------|
| `dcp.jsonc` | DCP context-preservation configuration for OpenCode | Tool-owned |

### References

| File | Description | Ownership |
|------|-------------|-----------|
| `references/example-report.md` | Canonical example of the expected report output format | Tool-owned |
| `references/doc-scoring-model.md` | Document-enhanced classification scoring model for the gaze-reporter agent | Tool-owned |

## Tool-Owned vs User-Owned Files

Gaze distinguishes between two ownership models for scaffolded files:

**User-owned files** (`commands/gaze.md`) are written once and never overwritten on subsequent `gaze init` runs. You can customize this file — add project-specific instructions or change how the command invokes Gaze. Your changes are preserved.

**Tool-owned files** (everything under `references/`, plus `agents/gaze-reporter.md`, `agents/gaze-test-generator.md`, `agents/divisor-gaze-testing.md`, `commands/gaze-fix.md`, `commands/speckit.testreview.md`, and `dcp.jsonc`) use overwrite-on-diff behavior. When you run `gaze init`, these files are compared against the version embedded in the Gaze binary. If the content differs (e.g., because you upgraded Gaze), the file is updated and reported. If the content is identical, the file is skipped. This keeps reference materials and tool-managed agents current without requiring `--force`. Local customizations to these files are overwritten — keep project-specific instructions in your own agent files instead.

The `--force` flag overrides both behaviors and overwrites all files unconditionally.

## Divisor Review-Config Registration

When `gaze init` runs in a project that already uses unbound-force tooling (a `.uf/` directory exists), it registers the `divisor-gaze-testing` persona in the existing review configuration:

- `.uf/reviewer-capabilities.yaml` — appends the `{agent: divisor-gaze-testing, capability: review, scopes: [test-quality]}` entry
- `.uf/review-matrix.yaml` — swaps the generic `divisor-testing` advisor for `divisor-gaze-testing` in the `code`, `spec`, `triage`, and `feedback` advisor lists (a matrix missing any of the four lists is skipped with a warning)

The merge is idempotent and comment-preserving: files are rewritten only when the registration actually changes them, and operational comments (including the `# scaffolded by uf` marker) survive verbatim. Malformed or missing expected keys warn and are skipped — `gaze init` never fails on them.

Projects without a `.uf/` directory get a notice instead, and **no `.uf` files are ever created** — a minimal capabilities file missing unbound-force's canonical reviewers would be rejected by uf tooling.

### Orphan Cleanup

`gaze init` also removes a legacy `.opencode/agents/reviewer-testing.md` when its content carries the `<!-- scaffolded by gaze ... -->` version marker — that file was superseded by `agents/divisor-gaze-testing.md`. The removal is reported in the init summary:

```text
  removed: .opencode/agents/reviewer-testing.md (superseded by agents/divisor-gaze-testing.md)
```

Copies without the marker (user-authored or heavily customized) are left untouched — preserve any customizations manually before upgrading if you want to keep them.

## Using the `/gaze` Command

After running `gaze init`, the `/gaze` command is available in OpenCode:

```text
/gaze ./...                     # Full report: CRAP + quality + classification
/gaze crap ./internal/store     # CRAP scores only
/gaze quality ./pkg/api         # Test quality metrics only
```

The `/gaze` command:

1. Runs `gaze report --format=markdown` (adding `--sections=crap` or `--sections=quality` for the focused modes)
2. Shows the Gaze-rendered markdown report verbatim
3. Adds an `## Assessment` section with judgment: a short narrative and up to five prioritized recommendations

The report includes:

- Module health summary with letter grades computed by Gaze's rubric
- [CRAP](../reference/glossary.md) and [GazeCRAP](../reference/glossary.md) score analysis
- [Quadrant](../reference/glossary.md) distribution (Q1 Safe through Q4 Dangerous)
- Worst offenders with [fix strategy](../reference/glossary.md) labels
- [Contract coverage](../reference/glossary.md) gaps with specific unasserted effects
- Actionable recommendations prioritized by impact, followed by the agent's assessment

## What the Gaze-Reporter Agent Does

The gaze-reporter agent is an OpenCode agent definition that shows Gaze's rendered report and adds judgment. It:

- Resolves the Gaze binary (local build, then `PATH`, then `go install`)
- Parses the mode (`crap`, `quality`, or full) and detects external analyzers from `.gaze.yaml` or project markers
- Runs `gaze report --format=markdown` and shows the output verbatim — it never re-renders tables, re-grades metrics, or re-ranks the deterministic recommendations
- Adds one `## Assessment` section: a short narrative and up to five prioritized recommendations that cite the report's metrics
- In quality mode, inspects unmapped assertions and reports likely semantic mappings as suggestions at confidence 50, without changing any rendered coverage number
- In full mode, may consult `references/doc-scoring-model.md` for document-informed ambiguity notes that never change the rendered counts

All metrics, grades, and deterministic recommendations are computed by Gaze. The agent adds interpretation and prioritization only.

## What the Gaze-Test-Generator Agent Does

The gaze-test-generator agent generates Go test functions based on Gaze quality data. It reads:

- **GapHints**: Specific unasserted side effects for each function
- **Gaps**: Functions with zero contract coverage
- **FixStrategy**: The recommended remediation action (`add_tests`, `add_assertions`, `decompose`, `decompose_and_test`)

The agent supports five actions:

| Action | Description |
|--------|-------------|
| `add_tests` | Generate new test functions for functions with zero coverage |
| `add_assertions` | Add assertions to existing tests that execute code but don't verify behavior |
| `add_docs` | Add GoDoc comments to shift classification confidence |
| `decompose_and_test` | Split complex functions and generate tests for the extracted pieces |
| `decompose` | Reduce cyclomatic complexity by extracting helper functions |

Use the `/gaze fix` command to trigger batch remediation:

```text
/gaze fix ./internal/store
```

## Using `/speckit.testreview`

The `/speckit.testreview` command performs read-only testability analysis on spec artifacts. It examines `spec.md`, `plan.md`, and `tasks.md` for:

- Testability gaps in acceptance criteria
- Missing coverage strategy in the plan
- Tasks that lack test hooks or verification steps

This command does not modify any files — it produces a report with recommendations.

## Upgrading

When you upgrade Gaze (e.g., via `brew upgrade gaze`), run `gaze init` again to update tool-owned files:

```bash
gaze init
```

Tool-owned files with changed content are automatically updated, including `agents/gaze-reporter.md` — local edits to that agent are overwritten. User-owned files are skipped, so your customizations to `commands/gaze.md` are preserved. The output shows what was updated:

```text
Gaze OpenCode integration already up to date:
  updated: .opencode/agents/gaze-reporter.md (content changed)
  updated: .opencode/references/example-report.md (content changed)
  skipped: .opencode/commands/gaze.md (already exists)
```

## Version Marker

Every scaffolded file includes a version marker comment:

```html
<!-- scaffolded by gaze v1.2.3 -->
```

This marker is inserted after the YAML frontmatter (if present) or appended to the end of the file. It identifies which version of Gaze produced the file, which is useful for debugging when tool-owned files are not updating as expected.
