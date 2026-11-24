# Markdown Autolink Entity Bypass

## Why

`neutralizeAutolinks` (`internal/aireport/render_markdown.go`) defangs
only *literal* autolink trigger bytes in tool-rendered markdown:
`://` becomes `[:]//`, a dotted email's `@` becomes `[@]`, and a
case-insensitive `www.` becomes `www[.]`. cmark-gfm — the renderer
behind GitHub Step Summaries, where `gaze report` publishes its
deterministic markdown — resolves HTML character references
(`&#119;`, `&#64;`, `&#46;`, `&#58;`, `&#x3a;`, `&colon;`,
`&period;`, …) into text nodes during inline parsing, *before* the
autolink extension post-processes the inline tree. An
attacker-controlled value — a branch named `&#119;ww.evil.example` or
`user&#64;host.example` — therefore reconstructs its trigger after
byte-level defanging and still linkifies, despite neutralization.

Verified live on the PR #314 branch after commit f764b07 pinned
literal-trigger defanging: values echoed into the deterministic
report (branch names, header metadata, table cells) can gain autolink
treatment in GitHub Step Summaries. Neither `headerEscaper` nor
`proseEscaper` nor `escapeCellMarkup` currently escapes `&`, so every
non-code-span sink is exposed (issue #321).

This change is Tier 1 must-ship of the v2.4.0 release plan (#337):
the spec 040 deterministic-rendering surface ships with v2.4.0, and
the fix should go out with it. It lands in `internal/aireport`, so it
is parallel-safe with #330 (`internal/crap`).

## What Changes

### Ampersand escaping at the non-code-span sinks

Every attacker-influenced value the deterministic markdown renderer
emits outside a code span gains `&` → `\&` escaping:

- `headerEscaper` — metadata header values (module path, branch,
  gaze version, date)
- `proseEscaper` — AI summary slot, AI failure notes,
  step-unavailable callout details, SSA-degraded package lists,
  recommendation text
- the plain table-cell path (`escapeCell`) — split from the shared
  `escapeCellMarkup` helper, which `tableName` also uses for
  code-span content

A backslash-escaped `\&` cannot open a character reference in
CommonMark, so no encoding — decimal numeric, hexadecimal numeric
(either `x` case, any digit case), or named — can decode into an
autolink trigger after defanging. This closes the entire entity class
unconditionally, regardless of how the trigger characters are
encoded. Placing the escape in the always-run escapers (rather than
inside `neutralizeAutolinks`, whose fast path returns early when no
literal trigger is present) is what makes it unconditional.

### Code spans stay verbatim

Function and test names render inside code spans (`tableName` →
`codeSpan`). Entities are never interpreted inside code spans and the
autolink extension never fires there, so `&` inside code-span content
stays verbatim — a naive escape there would display the backslash.

### `neutralizeAutolinks` unchanged

Literal-trigger defanging remains exactly as today: a literal `www.`,
dotted email, or `://` still defangs. The `&` escape is additive and
closes the entity-reconstruction path on top.

### Byte-level negative fixtures

Table-driven and render-level tests pin every entity encoding of
every trigger character (epic #337 rule 2): decimal numeric
(`&#119;`, `&#64;`, `&#46;`, `&#58;`, `&#47;`), hexadecimal numeric
(`&#x77;`, `&#X3A;`, mixed digit case), the named references that
decode to trigger characters (`&colon;`, `&period;`, `&sol;`,
`&commat;`), and hybrid strings mixing encodings with literal bytes —
so no encoder round-trip reopens the surface.

### Benign output unchanged

The golden files under `internal/aireport/testdata/golden/` contain
no `&` byte, so the spec 040 byte-level contract
(`specs/040-deterministic-ai-report-rendering/contracts/markdown-report.md`)
is untouched. For benign inputs — values without a decodable
character reference — rendered output is identical: `\&` renders as
`&`. Values that do contain a valid character reference now render
verbatim (design D4 documents the trade-off); that class is precisely
the injection vector.

## Capabilities

### New Capabilities

- `markdown-report-entity-escaping`: the deterministic markdown
  report escapes `&` outside code spans so HTML character references
  cannot reconstruct autolink triggers after byte-level defanging

### Modified Capabilities

- None

### Removed Capabilities

- None

## Impact

- **Code**: `internal/aireport/render_markdown.go` (escaper
  definitions, plain-cell/code-span split, GoDoc updates); no other
  package.
- **Tests**: `internal/aireport/render_markdown_test.go` —
  pair-pinning, markup, autolink-defanging, and hostile-data tests
  extended; new entity-fixture tests added.
- **Contracts**: spec 040 golden files remain byte-identical (no `&`
  in any fixture); JSON payload, JSON Schema, CLI surface, and CI
  workflows unchanged.
- **Docs**: no current document enumerates the renderer's escaped
  characters; the documentation validation gate is recorded as a task
  with the expected outcome.
- **Website gate**: exempt — security hardening with no user-facing
  behavior change for benign input (no command, flag, workflow, or
  installation change).
- **Release**: Tier 1 of the v2.4.0 plan (#337).

## Constitution Alignment

### I. Accuracy

**Assessment**: PASS

Hostile or entity-encoded values now render verbatim instead of being
silently decoded by the host markdown renderer: a branch literally
named `a&amp;b` displays as `a&amp;b`, exactly the bytes that exist.
The neutralization invariant — attacker-chosen values cannot gain
autolink treatment in the report — is enforced and regression-pinned
by automated byte-level tests, per the mandate that accuracy
violations MUST be treated as bugs.

### II. Minimal Assumptions

**Assessment**: PASS

No new assumptions about the host project: no configuration, no
annotation, no opt-in. The escape applies unconditionally at the
renderer's choke points and changes nothing for already-supported
environments — rendered output for benign input is identical.

### III. Actionable Output

**Assessment**: PASS

The report remains deterministic and byte-comparable across runs, and
the machine-readable JSON contract is untouched. The change removes a
class of link injection from the human-readable surface without
altering any metric, grade, or recommendation.

### IV. Testability

**Assessment**: PASS

The escapers are pure functions over strings; every requirement is
verified by byte-level table-driven unit tests and render-level tests
against the existing payload fixtures, with no external services. The
coverage strategy (unit + render-level + golden byte-identity) is
specified in the delta spec and tasks.
