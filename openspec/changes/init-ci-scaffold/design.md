# Design: `gaze init --ci`

## Context

`internal/scaffold` embeds distributable OpenCode files under
`assets/` and writes them into a target project's `.opencode/`
directory with user-owned (skip-if-present) / tool-owned
(overwrite-on-diff) / `--force` semantics. The init command
(`cmd/gaze/main.go`) is a thin wiring layer: `initParams` →
`runInit` → `scaffold.Run(scaffold.Options{...})`. The build-time
version arrives via GoReleaser ldflags (`-X main.version={{.Tag}}`,
default `"dev"` in development builds).

PR #300 shipped `.github/workflows/reusable_gaze_ai_report.yml`
(first tagged in `v2.3.0`). Adopting it today means copy-pasting the
consumer example from `docs/guides/ci-integration.md` and hand-editing
the `uses:` pin, Go version, and action pins. This change adds a
`--ci` flag to `gaze init` that generates a working caller workflow
deterministically from gaze's own build knowledge.

The proposal's constitution alignment (all four gaze principles PASS)
carries into the decisions below: fail-closed version validation
(Accuracy), go.mod-derived Go version and an enforced
Go-module-at-root precondition (Minimal Assumptions), next-steps
console guidance and human-owned thresholds (Actionable Output), pure
version resolution plus golden-file and YAML-parse tests
(Testability).

## Goals / Non-Goals

### Goals

- One-command adoption of the reusable AI-report workflow with a
  correct `uses:` pin, correct action pins, and a correct Go version.
- Additive behavior: plain `gaze init` byte-identical to today.
- Report-only generated workflow: no governance gates until a human
  chooses them.
- Skip-if-present idempotency: the generated file is user-owned
  forever after first write.
- Green-to-commit output before the consumer's secret exists.

### Non-Goals

- `/gaze thresholds` slash command (phase 2, separate change).
- A `--ci-only` variant (see decision D2 — rejected).
- Integrating with the consumer's *existing* test workflow (YAML
  surgery; the self-contained starter deliberately owns its own test
  job — one extra test run is the accepted trade-off).
- Dogfooding the scaffolded caller in gaze's own CI: `test.yml`
  already calls the reusable workflow via a local
  `./.github/workflows/...` reference, which correctly exercises
  in-flight changes; a tag pin would lag.
- Auto-updating action pins or the `uses:` pin in previously
  scaffolded files (skip-if-present means the user owns the file after
  first write).
- Renaming or altering the reusable workflow or its contract.

## Decisions

### D1: Workflow file name — `gaze-ai-report.yml` (settles issue open detail #1)

The generated file is `.github/workflows/gaze-ai-report.yml`.

Rationale: the issue's own scope section names this path; it matches
the reusable workflow's file stem
(`reusable_gaze_ai_report.yml`), gaze's dogfood job name
(`gaze-ai-report` in `test.yml`), and the docs vocabulary ("AI
report"). `gaze-report.yml` was rejected because it collides mentally
with the plain `gaze report` command and its AI-less
`--format=json` path — the generated workflow is specifically the AI
report caller. Keeping the chain reusable → caller → job under one
name reduces cognitive load when a consumer debugs a failed run.

CI-010 [MUST] in the active CI convention pack states "Consumer
workflows MUST be prefixed with `ci_`" — this is a deliberate,
documented divergence: the issue itself names the path
`.github/workflows/gaze-ai-report.yml`, the file is emitted into
consumer repositories outside this repo's governance (where gaze's
convention pack has no jurisdiction), and gaze's own dogfood caller
in `test.yml` also names its `gaze-ai-report` job without the
prefix. Renaming to `ci_gaze-ai-report.yml` would break the
reusable → caller → job name chain above for no in-repo benefit.

### D2: No `--ci-only` variant (settles issue open detail #2)

Rejected. The `.opencode/` scaffolding is additive and
skip-if-present; a user who wants only the workflow receives harmless
extra files they can ignore or delete. A `--ci-only` flag doubles the
flag matrix (interactions with `--force`, with re-runs) and muddies
the additive story for a marginal use case with no evidenced demand.
If demand emerges, it can be added later as another additive flag
without breaking changes.

### D3: Action SHA pins copied verbatim from gaze's own CI (settles issue open detail #3)

The issue says "same pins the reusable workflow uses", but the
reusable workflow never uploads an artifact — it pins only checkout,
setup-go, and download-artifact. The authoritative in-repo source for
an upload-artifact SHA pin is gaze's dogfooding `test.yml`. The
template therefore pins:

- `actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683  # v4.2.2`
- `actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff  # v5.6.0`
- `actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02  # v4.6.2`

(checkout/setup-go identical to the reusable workflow; upload-artifact
identical to `test.yml`'s "Upload coverage" step.) Every pin is
copied from an existing reviewed workflow — no new pin research, no
drift between gaze's CI and what it scaffolds for consumers. The
docs' consumer example shows `actions/upload-artifact@v4` unpinned;
the rewritten install section will show scaffolded output instead, so
the docs debt resolves itself.

### D4: Version resolution — build-time version, `--gaze-version` override, `semver` validation, fail-closed before any writes

Resolution order: `Options.GazeVersion` (from `--gaze-version`) when
set, otherwise `Options.Version` (build-time). The resolved value is
validated with `golang.org/x/mod/semver`
(`semver.IsValid` + `semver.Compare(v, minWorkflowTag) >= 0`) against
the floor constant `v2.3.0`, named and commented as the first release
tag containing the reusable workflow. `golang.org/x/mod` is promoted
from indirect to direct in `go.mod` — it is already in the module
graph via `x/tools`, so this adds no new supply-chain surface. At
package scope this is `internal/scaffold`'s first `x/mod` import
edge — efferent coupling Ce 11 → 12 (ΔCe +1; the directly analogous
yaml.v3 first edge on this package measured ΔInstability +0.0076,
roughly 20× below the 0.15 entropy gate) — recorded so the
code-mode entropy review watches the Ce delta and confirms no gate
fires. A
hand-rolled regex/tuple comparison was rejected: it mishandles
pre-release tags (`v2.4.0-rc1` legitimately contains the workflow and
must compare `>= v2.3.0`), missing minor/patch components, and
multi-digit segments — precisely the edge cases a user-supplied
`--gaze-version` string will exercise.

Failure behavior: any invalid or pre-floor version — including `dev`
builds and empty strings — returns an error directing the user to
install gaze from a release tag (`>= v2.3.0`) or pass
`--gaze-version`, and **no files are written**: validation runs in
`Run` before the `.opencode/` asset walk *and before the go.mod
existence check* (D6), so a failed `--ci` invocation never leaves a
half-scaffolded tree, and the combined case — a dev build running
`--ci` in a repository without `go.mod` — fails closed on the
version error rather than taking the missing-go.mod skip path.
`--gaze-version`
without `--ci` is rejected at flag parsing (the override has no
meaning without CI scaffolding). The flag's help text states the
value must be an *existing* gaze release tag `>= v2.3.0`:
validation is offline and lexical, so it cannot prove tag existence
(the residual failure mode is recorded in Risks).

### D5: Go version — line-scan of the consumer's `go.mod` directive, validated before substitution

The `__GO_VERSION__` placeholder is substituted from the `go`
directive of `<target>/go.mod`, parsed with a minimal line scan for a
line beginning `go ` — not `modfile.Parse`. Full-file parsing would
couple CI scaffolding to unrelated `go.mod` validity (a malformed
`replace` directive should not break `gaze init`).

The directive value is **validated, never passed through verbatim**.
The accepted grammar is `major[.minor[.patch]]` with an optional
pre-release suffix (e.g. `1.25`, `1.24`, `1.25.0`, `1.25.0-rc1`) —
the version forms setup-go accepts. Validating before substitution
keeps the generated YAML well-formed and the CI failure surface
empty: the input is local (the consumer's own go.mod), but the
artifact is an executed CI workflow committed to the consumer's
repository, and an unvalidated value would either fail at run time
with an opaque setup-go error (the delayed-failure class this change
exists to eliminate) or corrupt the double-quoted
`go-version: "__GO_VERSION__"` YAML scalar (a value containing `"`
escapes the string; a raw CR from a CRLF go.mod undergoes flow
folding). A trailing CR and trailing whitespace are stripped before
matching. When the directive is absent, unreadable, or its value
does not match the grammar, the scaffold falls back to `1.25` — the
reusable workflow's own `go-version` default — with a warning. The
substituted value therefore always matches the grammar.

### D6: Missing `go.mod` — warn and skip CI scaffolding

The reusable workflow analyzes `./...` from the repository root and
requires a Go module there (documented limitation). Emitting a
workflow that cannot succeed would violate Actionable Output. When
`go.mod` is absent, `--ci` prints a warning that names the
requirement, skips the workflow write entirely, and completes the
`.opencode/` scaffolding normally (non-fatal, distinct from D4's hard
failure: a missing precondition is a skip-with-explanation, an
unresolvable pin is an error). The mutual order of the two pre-walk
checks is fixed: version validation (D4) precedes the go.mod
existence check, so the combined dev-build-without-go.mod case fails
closed on the version error with no files written — it never
scaffolds `.opencode/` and exits 0 with a skip warning.

### D7: Skip-if-present — the CI file is user-owned

The workflow file is never added to `isToolOwned` (that list governs
`.opencode/` mirrors anyway) and gets a dedicated write path separate
from `processAssetFile`: read the embedded template, substitute the
two placeholders, then apply exists/`--force` semantics equivalent to
`writeNewFile` (MkdirAll for `.github/workflows/`, 0o644 file,
0o755 dirs — matching existing conventions). Outcomes flow through
the existing `Result.Created` / `Result.Skipped` /
`Result.Overwritten` fields with display path
`.github/workflows/gaze-ai-report.yml`, so `printSummary`'s existing
loops report "created:" / "skipped: ... (already exists)" /
"overwritten:" with no new result plumbing.

### D8: `ci/` subtree excluded from the `.opencode/` mirror machinery

The template lives at `internal/scaffold/assets/ci/gaze-ai-report.yml`.
Both `Run`'s asset walk and `assetPaths()` skip entries with the `ci/`
prefix: the walk must not write `.opencode/ci/...`, and
`assetPaths()` feeding `TestEmbeddedAssetsMatchSource` must enumerate
only the nine mirrored `.opencode/` assets (the `.opencode/` side of
that comparison never contains `ci/`, so excluding it from
`assetPaths()` keeps the test green without touching the manifest
list). `TestEmbeddedPromptMatchesScaffold` (aireport) is unaffected.
The CI path reads the template directly
(`assets.ReadFile("assets/ci/gaze-ai-report.yml")`).

### D9: Template content

Structure (all elements substitutable/testable):

- Provenance header comment: `# Scaffolded by gaze init --ci` with
  the gaze version and a pointer to
  `docs/guides/ci-integration.md`. Uses `#` YAML comments — the
  `<!-- scaffolded by gaze ... -->` marker machinery is Markdown-only
  and does not apply; the file is written once and never re-processed,
  so marker removal/duplication logic is out of scope.
- `name: Gaze AI Report`; `on: push` + `pull_request` filtered to
  `branches: [main]` — assumes `main` as the default branch; the
  header comment tells consumers with other default branches to edit
  the filters (recorded assumption; the file is user-owned).
- Permissions, dual placement — one decided shape, aligned across
  spec.md, this decision, task 1.1, and the golden file: the file
  declares workflow-level `permissions: contents: read`, capping
  every job at read (hygiene default satisfying the
  pins-and-hygiene requirement), **and** the `gaze-ai-report` job
  declares its own job-level `permissions: contents: read` — the
  explicit least-privilege grant at the reusable-workflow call site
  that CI-021 [SHOULD] prefers, matching gaze's dogfood pattern of
  job-level grants and satisfying the self-contained requirement's
  scenario verbatim. The job-level block is deliberately redundant
  with the workflow-level cap: a called workflow inherits the caller
  job's permissions and can only downgrade them, never elevate, so
  the call-site grant is the precise control point.
- Caller-owned `concurrency:` group, CI-012 pattern verbatim:
  `group: ${{ github.ref }}-${{ github.workflow }}`,
  `cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}`
  (AI-report runs spend LLM budget; superseded PR runs cancel while
  main runs queue).
- `test` job: `timeout-minutes: 30` (matches the docs full example
  and the reusable workflow's report-job timeout; generous for
  unknown suite durations), checkout + setup-go with D3 pins,
  `go-version: "__GO_VERSION__"`,
  `go test -race -count=1 -short -coverprofile=coverage.out ./...`,
  upload-artifact with D3 pin (`name: coverage-out`,
  `path: coverage.out`).
- `gaze-ai-report` job: `needs: test`, job-level
  `permissions: contents: read` (see the permissions bullet above),
  `uses: unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@__GAZE_VERSION__`,
  `with: coverage-artifact: coverage-out` **and**
  `gaze-version: __GAZE_VERSION__` — the resolved, validated version
  is threaded into the reusable workflow's `gaze-version` input so
  the CI-installed binary is identical to the `uses:` pin and to the
  binary the consumer calibrates thresholds with (that input's
  default, `v2.2.0`, predates the `v2.3.0` floor; inheriting the
  default would let CI enforce locally-calibrated thresholds with an
  older binary's metrics — an unexplained-failure class and a
  Constitution III comparability break), threshold inputs as
  comment-only lines with TODO markers and no numeric values
  (uncommenting without choosing values must not be possible — there
  is deliberately nothing to blindly uncomment-and-forget), and
  `secrets: opencode-config: ${{ secrets.OPENCODE_CONFIG_CONTENT }}`.
- Substitution: two `strings.ReplaceAll` calls on the template bytes —
  a pure function, golden-file testable.

The template must pass the repo's yamllint config
(`.yamllint.yml`: line-length 400, `document-start` disabled, truthy
allows `on` as a key) because `internal/scaffold/assets/` is excluded
from MARKDOWN_MARKDOWNLINT but **not** from YAML_YAMLLINT. The
placeholders sit in scalar positions and are valid YAML.

### D10: Console next-steps CI section

Printed through the existing `opts.Stdout` writer (unit-testable
without subprocesses) when CI scaffolding was requested and not
skipped for a missing `go.mod`. Contents, per the issue:

- `gh secret set OPENCODE_CONFIG_CONTENT < opencode-config.json`
  one-liner, with a pointer to `docs/guides/ci-integration.md` for
  the config shape (provider, model, permission denials, literal key)
  — never an inline example key.
- `jq -c .` minify tip (single-line strict JSON; GitHub's multiline
  secret masking is unreliable).
- Second-secret masking pattern pointer (register the bare API key as
  a second secret referenced in an unused env var so GitHub masks
  fragment echoes).
- Paste-able threshold-guidance prompt: read the generated workflow,
  run `gaze report ./... --format=json`, propose permissive initial
  thresholds from the summary metrics, human approves final values,
  uncomment the threshold inputs.

### D11: CLI wiring

`initParams` gains `ci bool` and `gazeVersion string`;
`newInitCmd` registers `--ci` and `--gaze-version` (the latter
rejected without `--ci`); `runInit` threads both into
`scaffold.Options`. `scaffold.Options` gains `CI bool` and
`GazeVersion string` with GoDoc comments; `applyDefaults` leaves both
zero-valued semantics intact (no CI unless asked).

## Risks / Trade-offs

- **One extra test run per push** versus integrating with the
  consumer's existing test job — accepted per the issue's design
  decision 1; artifact-reuse dedup stays a docs/agent refinement.
- **Compile-time knowledge of the workflow floor**: if the reusable
  workflow file is ever moved or renamed, the floor constant, the
  template, and reality must move together. Mitigated by placing the
  constant next to the template reference and a test that asserts
  the template's `uses:` line references the expected workflow path.
- **Action pins age** in previously scaffolded files; gaze will not
  auto-update them (skip-if-present). Accepted — same trade-off as
  any scaffold; the header comment points at gaze's repo for current
  pins.
- **The `uses:` tag pin is name-addressed, not content-addressed**:
  a moved or re-pointed release tag in `unbound-force/gaze` would
  silently change the code the consumer's workflow executes, and the
  floor validation cannot detect it — it validates ordinal position,
  not the tag's current target. Trust assumption: gaze's release
  preflight creates tags via the GitHub API and release tags are
  never force-moved. Consumers who want immutable pinning can harden
  the `uses:` reference to the tag's 40-character commit SHA (noted
  in the template's provenance header).
- **Version validation is offline and lexical**: it eliminates the
  below-floor pinning class but cannot prove a tag exists. A typo
  above the floor (`v2.40.0` for `v2.4.0`) passes validation and
  fails at run time with the same opaque "workflow not found" error
  the proposal cites — which is why the `--gaze-version` help text
  states the value must be an existing release tag. Decided
  behavior: `v2.3.0-rc1`, a pre-release of the floor tag itself,
  sorts below `v2.3.0` by semver precedence and is rejected
  (pre-releases above the floor, e.g. `v2.4.0-rc1`, are accepted);
  a consumer on a `v2.3.0` pre-release must pin `v2.3.0` or newer.
- **`main`-branch assumption** in `on:` filters — mitigated by the
  header comment; the file is user-owned after first write.
- **`golang.org/x/mod` promotion** (indirect → direct) — minimal
  churn; already required transitively by `x/tools`; maintained by
  the Go team.
- **Version floor is historical, not moving**: `v2.3.0` is fixed
  forever once tagged (older tags never gain the file), so the
  constant documents a fact, not a ratchet that needs maintenance.
