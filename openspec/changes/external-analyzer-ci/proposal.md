## Why

Gaze's shared CI AI-report pipeline is Go-only. The CLI has supported
external analyzers since the JSON-RPC protocol landed (issue #95),
and `gaze report` accepts `--analyzer` and `--language`, but the CI
layer hard-codes the Go path: the reusable workflow assumes a Go
coverprofile artifact and passes `--coverprofile`, and `gaze init
--ci` skips CI scaffolding entirely when no `go.mod` sits at the
repository root. Non-Go repositories cannot use the shared pipeline
at all. The first consumer is zero-dot-force/reading-stone, a
TypeScript repository (zero-dot-force/reading-stone#40), where gaze
v2.5.0 already works locally via `npx vitest run --coverage` plus
`gaze report ./src/... --analyzer gaze-analyzer-typescript`.

Related: #282 (`gaze report` should honor `--language` alone),
#306 (the original `gaze init --ci`).

## What Changes

1. **Reusable workflow**
   (`.github/workflows/reusable_gaze_ai_report.yml`): add optional
   external-analyzer inputs — `analyzer` (default `""`; empty
   selects Go-native analysis), `language`, `packages` (default
   `./...`), `setup` (optional shell run before `gaze report`),
   `node-version`/`python-version` (toolchain steps, skipped when
   empty), and `coverage-path` (analyzer-mode artifact validation).
   Every default preserves today's Go behavior. The `gaze-version`
   default is untouched (the release gate
   `validate-ai-report-default` pins it to the release tag). The
   caller-controlled `setup` input reaches shell steps only through
   `env:` bindings referenced as quoted POSIX variables (CWE-77),
   and the caller-trust model is documented in the workflow and the
   CI guide.

2. **`gaze init --ci`** (`internal/scaffold/`): when no `go.mod`
   exists, detect a project marker instead of skipping —
   `package.json` yields a Node caller that runs the TypeScript
   test suite with coverage, uploads `coverage/`, and passes
   `analyzer: gaze-analyzer-typescript` (the reading-stone wiring);
   `pyproject.toml` yields a Python caller wired to snake-eyes the
   same way. Each flavor gets its own golden file next to
   `testdata/golden-gaze-ai-report.yml`; the existing Go golden
   stays byte-identical. Repositories with no recognized marker
   keep the current warning.

3. **Fixture job**: gaze's own CI demonstrates the analyzer mode
   end to end on a TypeScript repository (reading-stone at a pinned
   ref), report-only with no secret required.

4. **Docs**: `docs/guides/ci-integration.md`, the report/protocol
   reference, and `README.md` cover the external-analyzer CI path.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-workflow`: the reusable AI-report workflow gains an
  external-analyzer mode — new inputs, a mode-dependent coverage
  contract, and injection-safe `setup` handling — with Go behavior
  unchanged.
- `ci-scaffolding`: `gaze init --ci` generalizes from go.mod-only
  to marker-based flavor detection (Go, Node, Python) with
  per-flavor golden files.

### Removed Capabilities

None.

## Impact

- `.github/workflows/reusable_gaze_ai_report.yml` — new inputs,
  mode gating, caller `setup` handling, per-mode artifact
  validation.
- `.github/workflows/test.yml` — TypeScript fixture job.
- `internal/scaffold/scaffold.go` and `internal/scaffold/ci_test.go`
  — marker detection, per-flavor templates, and tests.
- `internal/scaffold/assets/ci/` — Node and Python caller templates
  (the `ci/` subtree stays excluded from the `.opencode/` mirror
  sync; the nine-file manifest is unchanged).
- `internal/scaffold/testdata/` — new golden files beside the
  unchanged Go golden.
- `cmd/gaze/main_test.go` — summary and flag assertions extended if
  behavior requires.
- `docs/guides/ci-integration.md`, `docs/reference/cli/report.md`,
  `README.md`.
- Go callers: no behavior change — every new input defaults to Go
  mode; the dogfood job is unchanged.
- Website: user-facing change — the Website Documentation Gate
  applies (an `unbound-force/website` issue before merge).

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: PASS

The change communicates through self-describing artifacts: typed
workflow inputs with Go-preserving defaults, scaffolded workflows
with provenance headers, and golden files that pin the contract.
No runtime coupling between gaze and its consumers is added.

### II. Composability First

**Assessment**: PASS

Gaze remains independently installable and Go-only usable. The
analyzer mode is opt-in via inputs whose defaults reproduce today's
behavior; no new mandatory dependency is introduced for existing
callers.

### III. Observable Quality

**Assessment**: PASS

Report output, threshold gates, and provenance are unchanged. The
new mode's wiring is observable through workflow inputs, structural
workflow assertions, the fixture job, and machine-checked goldens.

### IV. Testability

**Assessment**: PASS

Scaffold behavior is pinned by per-flavor golden-file tests; the
reusable workflow contract is pinned by structural YAML assertions;
the TypeScript path is exercised end to end by a fixture job
against a pinned repository ref.
