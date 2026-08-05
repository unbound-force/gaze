## Why

The `gaze-test-generator` agent prompt at
`internal/scaffold/assets/agents/gaze-test-generator.md` (line 275)
contains advisory prose: "ALWAYS verify generated code compiles
before reporting success." This is a T3 weakness — the required
verification step is inline text only, not enforced as a concrete
tool call. Under context compression or fast-path reasoning, an
agent can skip compilation checking and write tests that do not
compile.

This issue was filed as
[gaze#204](https://github.com/unbound-force/gaze/issues/204) and a
fix was applied in
[unbound-force/unbound-force#404](https://github.com/unbound-force/unbound-force/pull/404),
but that PR modified the copy in the **unbound-force** repo, not the
gaze repo. The gaze repo's scaffold copy (which `gaze init` ships to
users) and its active `.opencode/agents/` copy still contain the old
advisory text.

This change ports the fix to the gaze repo — the correct location
for the canonical scaffold source.

Fixes: gaze#204

## What Changes

Replace the advisory compile verification prose with a concrete
3-step pre-write compile gate protocol in the `gaze-test-generator`
agent prompt. The agent will be instructed to run `go build` before
any Write or Edit tool call and halt if compilation fails.

Additionally, update the Output Format section to distinguish
per-file pre-write gate verification from the final batch integrity
check.

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `gaze-test-generator`: The "Important Constraints" section
  replaces advisory "ALWAYS verify" prose with a numbered 3-step
  compile gate protocol using MUST NOT halt language. The "Output
  Format" section clarifies that individual files are verified by
  the pre-write gate and the final `go build` is a batch integrity
  check.

### Removed Capabilities
- None

## Impact

- **Files**: `internal/scaffold/assets/agents/gaze-test-generator.md`
  (scaffold canonical copy),
  `.opencode/agents/gaze-test-generator.md` (active runtime copy —
  currently identical to scaffold)
- **Behavior**: The agent will now be instructed to run `go build`
  before writing files and to halt (not write) if compilation fails.
  This may cause the agent to report more errors instead of silently
  writing broken tests.
- **Users**: Projects that have previously run `gaze init` will
  receive the update automatically on the next `gaze init` run.
  The `gaze-test-generator.md` file is classified as tool-owned
  (`isToolOwned` in `internal/scaffold/scaffold.go`), which uses
  overwrite-on-diff behavior — the scaffold automatically replaces
  the active copy when content differs from the embedded version.

## Constitution Alignment

Assessed against the Unbound Force org constitution.

### I. Autonomous Collaboration

**Assessment**: N/A

This change modifies an agent's internal instruction set. It does
not affect artifact-based communication between heroes or
inter-hero protocols. The gaze-test-generator continues to consume
gaze quality JSON artifacts and produce test files independently.

### II. Composability First

**Assessment**: PASS

The compile gate uses `go build ./...`, universally available in
any Go project. No new dependencies are introduced. The
gaze-test-generator remains independently usable on any Go project.

### III. Observable Quality

**Assessment**: PASS

The compile gate adds a concrete verification step that produces
observable, machine-parseable output (go build exit code and error
output). This strengthens quality observability — compilation
status was previously unverified advisory text.

### IV. Testability

**Assessment**: PASS

The change enforces verification of observable side effects (does
the generated code compile?) before writing. This directly aligns
with the testability principle's requirement that "test contracts
MUST verify observable side effects."
