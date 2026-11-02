# gaze init

Scaffold OpenCode agent and command files into the current directory. After running this command, you can use the `/gaze` command in OpenCode to generate AI-powered quality reports.

## Synopsis

```bash
gaze init [flags]
```

## Arguments

This command takes no positional arguments. It always operates on the current working directory.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | `bool` | `false` | Overwrite existing files. Without this flag, existing user-owned files are skipped. |
| `--ci` | `bool` | `false` | Also scaffold `.github/workflows/gaze-ai-report.yml`, a starter caller for gaze's reusable AI-report workflow. Requires a `go.mod` at the repository root; skipped with a warning otherwise. |
| `--gaze-version` | `string` | `""` | Gaze release tag (`>= v2.3.0`) pinning the scaffolded CI workflow's `uses:` reference. Requires `--ci`. |

## Scaffolded Files

The `init` command creates 9 files with the following directory structure:

```text
.opencode/
├── agents/
│   ├── gaze-reporter.md          # AI agent prompt for quality reporting (tool-owned)
│   ├── gaze-test-generator.md    # Test generation agent
│   └── divisor-gaze-testing.md   # Test-quality reviewer persona for review councils
├── commands/
│   ├── gaze.md                   # /gaze command definition (user-owned)
│   ├── gaze-fix.md               # /gaze fix command definition
│   └── speckit.testreview.md     # /speckit.testreview command definition
├── dcp.jsonc                     # DCP context-preservation configuration
└── references/
    ├── example-report.md         # Canonical example output for the reporter
    └── doc-scoring-model.md      # Document-enhanced classification scoring model
```

### File Ownership Model

Gaze uses a mixed ownership model for scaffolded files:

- **User-owned files** (`commands/gaze.md`): Created once, never overwritten on subsequent `gaze init` runs (unless `--force` is used). Users can customize this file.
- **Tool-owned files** (everything under `references/`, plus `agents/gaze-reporter.md`, `agents/gaze-test-generator.md`, `agents/divisor-gaze-testing.md`, `commands/gaze-fix.md`, `commands/speckit.testreview.md`, and `dcp.jsonc`): Updated automatically on `gaze init` if the content has changed, and reported as `updated`. These files are maintained by Gaze and should not be manually edited — local customizations to `agents/gaze-reporter.md` are overwritten.

### Divisor Review-Config Registration

When run in a project that already uses unbound-force tooling (a `.uf/` directory exists), `gaze init` also registers the `divisor-gaze-testing` persona in the existing review configuration: it appends the reviewer entry to `.uf/reviewer-capabilities.yaml` and swaps the generic `divisor-testing` advisor for `divisor-gaze-testing` in the `code`, `spec`, `triage`, and `feedback` advisor lists of `.uf/review-matrix.yaml` (a matrix missing any of the four lists is skipped with a warning). The merge is idempotent, comment-preserving, and never creates `.uf` files — projects without a `.uf/` directory get a notice instead.

`gaze init` also removes a legacy `.opencode/agents/reviewer-testing.md` when its content carries the gaze version marker (`<!-- scaffolded by gaze ... -->`) — the file was superseded by `agents/divisor-gaze-testing.md`. Marker-less (user-customized) copies are left untouched.

## CI Workflow Scaffolding

The `--ci` flag additionally scaffolds `.github/workflows/gaze-ai-report.yml`: a self-contained starter workflow for gaze's [reusable AI-report workflow](https://github.com/unbound-force/gaze/blob/main/.github/workflows/reusable_gaze_ai_report.yml). Plain `gaze init` is unchanged — the flag is purely additive.

The generated workflow owns both halves of the pipeline:

- A `test` job that checks out the code, sets up Go, runs `go test -race -count=1 -short -coverprofile=coverage.out ./...`, and uploads the profile as the `coverage-out` artifact.
- A `gaze-ai-report` job that calls `unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@<version>` with the coverage artifact and the `OPENCODE_CONFIG_CONTENT` secret.

Behavior details:

- **Report-only by default.** The threshold inputs (`max-crapload`, `max-gaze-crapload`, `min-contract-coverage`) are emitted commented out; the workflow exits 0 until a human chooses values and uncomments them. Governance gates belong to the repository owner, not the scaffolder.
- **User-owned.** The workflow is never overwritten by later `gaze init --ci` runs (skip-if-present); only `--force` regenerates it, so threshold edits survive re-scaffolding.
- **Version pinning.** The `uses:` reference is pinned to the gaze release tag resolved from the running binary's build-time version and validated as `>= v2.3.0` — the first release containing the reusable workflow. Dev builds (and any version below the floor) fail closed with an error instead of writing an unusable pin. `--gaze-version` overrides the pin with the same validation; it is rejected without `--ci`, since it affects only the CI scaffolding.
- **Go version from `go.mod`.** The workflow's `setup-go` version is read from the module's `go` directive and validated before substitution — never passed through verbatim. When the directive is absent, unreadable, or malformed, the workflow falls back to the reusable workflow's default (`1.25`) with a warning.
- **Go module required at the repository root.** The reusable workflow analyzes `./...` from the repository root, so `--ci` requires a `go.mod` there. When it is absent, CI scaffolding is skipped with a distinct, non-fatal warning and the `.opencode/` scaffolding proceeds unchanged.

After a successful `--ci` run, the console prints a **CI next steps** section: the `gh secret set OPENCODE_CONFIG_CONTENT` one-liner (with a `jq -c .` minify tip — GitHub's multiline secret masking is unreliable), the second-secret pointer for masking API-key fragments echoed into logs, and a paste-able prompt for choosing initial threshold values from a baseline report.

## Configuration Interaction

This command does not read `.gaze.yaml`.

## Examples

### Initialize OpenCode integration

```bash
cd /path/to/your/go/project
gaze init
```

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

### Re-initialize (update tool-owned files)

```bash
gaze init
```

On subsequent runs, user-owned files are skipped and tool-owned files are updated if their content has changed. In projects with a `.uf/` directory, the persona registration is idempotent (already-registered files are not rewritten), and a gaze-marked legacy `.opencode/agents/reviewer-testing.md` orphan is removed with a `removed:` summary line.

### Force overwrite all files

```bash
gaze init --force
```

Overwrites all files, including user-customized agent prompts and commands.

### Scaffold the CI workflow

```bash
cd /path/to/your/go/project
gaze init --ci
```

In addition to the nine `.opencode/` files, this writes `.github/workflows/gaze-ai-report.yml` and prints the CI next-steps guidance:

```text
Gaze OpenCode integration initialized:
  created: .opencode/agents/divisor-gaze-testing.md
  ...
  created: .github/workflows/gaze-ai-report.yml

Run /gaze for quality reports and /speckit.testreview for testability analysis.

CI next steps for .github/workflows/gaze-ai-report.yml:

1. Store your opencode configuration as the OPENCODE_CONFIG_CONTENT secret:

     gh secret set OPENCODE_CONFIG_CONTENT < opencode-config.json
...
```

A dev build (or any version below `v2.3.0`) fails closed instead of writing an unusable pin:

```text
Error: gaze version "dev" cannot pin the CI workflow: the reusable AI-report workflow first ships in v2.3.0; install gaze from a release tag (>= v2.3.0) or pass --gaze-version
```

Pass `--gaze-version v2.4.0` to pin an explicit release tag.

## See Also

- [OpenCode Integration](../../guides/opencode-integration.md) — how to use `/gaze` after initialization
- [CI Integration](../../guides/ci-integration.md) — the reusable AI-report workflow the scaffolded caller invokes
- [`gaze report`](report.md) — the CLI equivalent of the `/gaze` OpenCode command
