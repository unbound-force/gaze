# Design: Markdown Autolink Entity Bypass

## Context

The deterministic markdown renderer (`internal/aireport/render_markdown.go`,
spec 040) publishes to GitHub Step Summaries, which cmark-gfm renders.
Every attacker-influenced value flows through one of four sinks:

| Sink | Path | Used for |
|------|------|----------|
| `sanitizeHeaderValue` | `headerEscaper.Replace(neutralizeAutolinks(s))` | module path, branch, gaze version, date |
| `sanitizeProse` | `proseEscaper.Replace(neutralizeAutolinks(s))` | AI summary, AI failure notes, callout details, SSA package lists, recommendation text |
| `escapeCell` | `escapeCellMarkup(neutralizeAutolinks(s))` | plain table cells (locations, gap types/descriptions, scorecard labels/details, fix strategies) |
| `tableName` | `escapeCellMarkup(codeSpan(s))` | function and test names, inside code spans |

`neutralizeAutolinks` defangs only *literal* trigger bytes: `://` →
`[:]//`, dotted-email `@` → `[@]` (via `autolinkEmailRe`), and
case-insensitive `www.` → `www[.]`. It short-circuits — returns the
input unchanged — when none of `://`, `@`, or `www.` is literally
present.

**The bypass.** cmark-gfm's inline scanner resolves backslash escapes
and HTML character references into text-node content *while building*
the inline tree; the autolink extension (`linkify`) then
post-processes the completed text nodes, matching `www.`, scheme-URL,
and email patterns against the **decoded** text. So a value like
`&#119;ww.evil.example` — which contains only two literal `w` bytes
and therefore trips none of `neutralizeAutolinks`' literal checks —
decodes to the text `www.evil.example`, which the extension linkifies.
Verified live on the PR #314 branch after commit f764b07 (issue
#321). Every encoding family participates: decimal (`&#64;`),
hexadecimal with either `x` case and any digit case (`&#x3A;`), and
the HTML5 named references that decode to trigger characters
(`&colon;`, `&period;`, `&sol;`, `&commat;`; there is no named
reference for `w`). None of `headerEscaper`, `proseEscaper`, or
`escapeCellMarkup` currently escapes `&`.

**Scope boundary.** Entity-decoded characters do *not* participate in
delimiter processing — links, emphasis, and raw HTML are built only
from literal punctuation bytes scanned by the inline parser, so
`&#91;` (decodes to `[`) cannot open a link and `&lt;` cannot open a
tag. The existing bracket/angle-bracket escaping remains correct for
those surfaces. The autolink extension is the one post-hoc consumer of
decoded text, which is why the entity bypass is specifically an
autolink bypass. Escaping `&` still hardens all surfaces at once.

**Rendering semantics relied on.** CommonMark backslash escapes cover
ASCII punctuation, which includes `&` (U+0026): `\&` renders as a
literal `&` and can never begin a character reference. Inside code
spans, neither backslash escapes nor character references are
interpreted — content is verbatim — and the autolink extension never
fires there.

## Goals / Non-Goals

### Goals

- Close the entity-reconstruction class for every non-code-span sink,
  for every encoding family, unconditionally.
- Keep the spec 040 golden files byte-identical (epic #337 rule 3);
  rendered output for benign input unchanged.
- Pin every encoding family of every trigger character with
  byte-level fixtures so no encoder round-trip reopens the surface
  (epic #337 rule 2).

### Non-Goals

- Not switching to HTML-entity escaping (`&` → `&amp;`) — see D1.
- Not making `neutralizeAutolinks` entity-aware — see D3.
- Not changing the JSON payload, JSON Schema, CLI surface, AI answer
  validator (`answer.go`), or any analyzer/adapter code — the
  renderer is the choke point.
- Not auditing non-markdown output formats (JSON, terminal text) —
  they are not cmark-gfm surfaces.

## Decisions

### D1: Backslash-escape `\&` rather than entity-encode or defang selectively

Three candidates:

1. **`\&` (chosen).** Consistent with the existing backslash-based
   escapers (`\[`, `\<`, `\|`). CommonMark guarantees a backslash
   escape cannot open a reference, so *one* rule closes *every*
   encoding family, known and future. Raw markdown stays greppable
   and the escape is stylistically uniform.
2. **`&amp;` (rejected).** Also safe, and it would preserve the old
   rendered text for values containing valid references (`a&amp;b`
   would still display `a&b`). Rejected because it embeds
   HTML-flavored encoding in a markdown document for no functional
   gain, diverges from the established backslash style of all three
   escapers, and still displays as `&amp;` if the raw markdown is
   viewed unrendered (terminal `cat`, code review) — no better than
   `\&` there.
3. **Targeted entity defanging inside `neutralizeAutolinks`
   (rejected).** Would require enumerating every reference that
   decodes to a trigger character (and staying current with the
   HTML5 named list), reworking the fast path, and would still leave
   `&` unescaped for markup surfaces. A blocklist fights an
   open-ended encoding space; a single unconditional escape does not.

The issue's suggested fix names this option first ("escape `&`
outside code spans — a backslash-escaped `\&` cannot open an entity
reference").

### D2: Escape at the three non-code-span sinks; split the shared cell escaper

Add the `"&", "\\&"` pair to `headerEscaper` and `proseEscaper`
(neither ever wraps content in code spans, so the pair is safe there
by construction). For table cells, `escapeCellMarkup` is shared
between `escapeCell` (plain cells — needs the `&` escape) and
`tableName` (code-span content — must NOT escape `&`, or the
backslash would display). The plain-cell path therefore gets its own
escaping layer that includes `&`, while the code-span path keeps the
markup-only escaper. One concrete shape (implementation may choose
another that preserves the invariant):

```go
// escapeCell: plain cells escape & alongside the markup bytes
func escapeCell(s string) string {
    return escapeCellMarkup(escapeAmp(neutralizeAutolinks(s)))
}
// tableName: code-span content stays verbatim for &
func tableName(s string) string {
    return escapeCellMarkup(codeSpan(s))
}
```

Placement after `neutralizeAutolinks` matches the layering model:
trigger defanging is grammar-aware neutralization, markup escaping is
byte-level — `&` belongs to the byte layer. Order is immaterial
today (neutralization neither emits nor consumes `&`), but the
layering keeps the reasoning local. GoDoc on the affected helpers
must be updated to name `&` among the escaped bytes (DR-001).

Constitution note (Observable Quality): the escape lives at the same
choke points the spec 040 renderer already funnels every value
through, so the hardening is provably total by construction — every
attacker-influenced byte path is enumerated above — and is verified
by fixtures at both unit and render level.

### D3: `neutralizeAutolinks` stays byte-trigger-only

Literal triggers still need defanging (`www.evil.example` as literal
bytes has no `&` to escape), so the function keeps its behavior and
its fast path unchanged. The `&` escape must NOT move into
`neutralizeAutolinks` precisely because of that fast path: an input
like `&#119;ww.evil.example` contains no literal trigger, so the
function returns early — the escape would never run. The escapers run
unconditionally, which is what makes the fix total.

### D4: Verbatim rendering for entity-bearing values is the accepted trade-off

Pre-change, a value containing a valid reference was silently decoded
by cmark-gfm on display: a branch literally named `a&amp;b` rendered
as `a&b`. Post-change it renders verbatim as `a&amp;b`. This is a
rendered-output change for exactly the class of values that forms the
injection vector; benign values (no `&`, or `&` that cannot continue
a reference, like `R&D`) render identically because `\&` displays as
`&`. Verbatim is also the more accurate semantic for identifiers —
the code-span path already displays function names verbatim for the
same reason — and the golden fixtures contain no `&`, so the spec 040
byte-level contract is untouched. Recorded here as the deliberate
justification for the benign-output requirement's scope.

### D5: Test strategy — byte-level fixtures per sink × encoding family

- **Unit level**: table-driven cases per sink
  (`sanitizeHeaderValue`, `sanitizeProse`, `escapeCell`) crossing each
  trigger character (`w`, `.`, `@`, `:`, `/`) with each encoding
  family (decimal, hex with `x`/`X` and mixed digit case, named,
  hybrid-with-literal). Assert exact escaped bytes (e.g.
  `\&#119;ww.evil.example`), not just absence of the bare reference —
  TC-009.
- **Render level**: hostile branch name, module path, gap-description
  cell, and callout detail through `RenderMarkdown`, asserting no
  unescaped `&` outside code spans in the emitted report (a scanner
  over the output, excluding code-span content, is the strongest
  form).
- **Code-span immunity**: a function name containing `&#64;` renders
  verbatim inside its backtick span.
- **Benign parity**: `R&D` → `R\&D`; golden suite runs without
  regeneration.
- Coverage: the touched helpers are pure string functions exercised
  by the new table tests; no external services (Principle IV). Slow
  paths do not exist here, so no `testing.Short` guards are needed.

Constitution note (Testability): fixtures assert observable output
bytes of the pure render pipeline — no implementation details, no
subprocesses.

## Risks / Trade-offs

- **Raw markdown shows `\&`.** Cosmetic; only in unrendered views of
  the markdown output, same as the existing `\[`, `\<`, `\|` escapes.
- **Entity-bearing values render verbatim** (D4). Deliberate; the
  affected class is the attack vector, and identifiers rendering
  verbatim is the accuracy-preserving behavior.
- **New named references in future HTML revisions.** A non-issue by
  construction: the escape is not a blocklist; any reference needing
  a literal `&` to form is dead on arrival.
- **Escape-order regressions in `strings.NewReplacer` pair lists.**
  `headerEscaper` already pins pair ordering in
  `TestSanitizeHeaderValue_Pairs` (`\r\n` before `\r`/`\n`); the `&`
  pair interacts with no other pair (`&` is not a prefix or suffix of
  any existing pattern), and the pinning test gains an ampersand case
  so reordering is caught.

## Open Questions

None material. One minor implementation choice is left open in D2
(exact shape of the plain-cell split — dedicated helper versus a
merged replacer); both preserve the invariant, and the tasks accept
either.

## Constitution Alignment

The proposal's Constitution Alignment (all four principles PASS)
carries over: Accuracy — values render verbatim and the
neutralization invariant is regression-pinned; Minimal Assumptions —
unconditional choke-point hardening with no configuration;
Actionable Output — deterministic byte-comparable output and untouched
JSON contract; Testability — pure-function byte-level fixtures at
unit and render level, coverage strategy specified in D5.
