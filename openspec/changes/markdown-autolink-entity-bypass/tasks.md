# Tasks: Markdown Autolink Entity Bypass

## 1. Escaper changes (internal/aireport/render_markdown.go)

- [x] 1.1 Add the `"&", "\\&"` pair to `headerEscaper` (design D2)
  and update its GoDoc to name the ampersand among the escaped bytes:
  a backslash-escaped `&` cannot open an HTML character reference, so
  cmark-gfm cannot decode entity-encoded autolink triggers after
  byte-level defanging. Extend `TestSanitizeHeaderValue_Pairs` with
  ampersand cases, including one proving the pair does not disturb
  the existing pair ordering (`a&amp;b` → `a\\&amp;b`).
- [x] 1.2 Add the `"&", "\\&"` pair to `proseEscaper` (design D2)
  and update its GoDoc the same way. Extend
  `TestSanitizeProse_Markup` with ampersand cases (benign `R&D` →
  `R\&D`, hostile `user&#64;host.example` → `user\&#64;host.example`).
- [x] 1.3 Split the plain-cell path so `escapeCell` escapes `&` while
  `tableName`'s code-span content keeps `&` verbatim (design D2; e.g.
  an `escapeAmp` layer applied in `escapeCell` only, or a dedicated
  plain-cell replacer — either shape is acceptable). Update the
  GoDoc on `escapeCell` and `escapeCellMarkup` to state the split:
  code spans never interpret references, so escaping `&` there would
  display the backslash. Extend `TestEscapeCell_Markup` with
  ampersand cases and extend the code-span immunity assertions in
  `TestRenderMarkdown_AutolinkTriggersDefanged` with a function name
  containing `&#64;` that must render verbatim inside its backtick
  span.

## 2. Byte-level entity fixtures (epic #337 rule 2)

- [x] 2.1 Add a table-driven unit test (e.g.
  `TestNeutralizeEntityEncodedAutolinks`) crossing every
  non-code-span sink (`sanitizeHeaderValue`, `sanitizeProse`,
  `escapeCell`) with every trigger character (`w`, `.`, `@`, `:`,
  `/`) and every encoding family from the delta spec, enumerated as
  the complete cell set: decimal numeric for all five triggers
  (`&#119;`, `&#46;`, `&#64;`, `&#58;`, `&#47;`); hexadecimal numeric
  for all five triggers, covering lowercase `x`, uppercase `X`, and
  mixed-case digit forms (`&#x77;`/`&#X77;`, `&#x2E;`/`&#X2e;`,
  `&#x40;`/`&#X40;`, `&#x3A;`/`&#X3A;`/`&#x3a;`, `&#x2F;`/`&#X2f;`);
  named references for `.`, `@`, `:`, `/` (`&period;`, `&commat;`,
  `&colon;`, `&sol;`) — the `w` cell is structurally impossible in
  this family because no HTML5 named reference encodes `w`; and
  hybrid literal-plus-entity strings
  (`&#119;ww&period;evil&period;example`,
  `user&commat;host&period;example`,
  `https&colon;&sol;&sol;evil.example`, `www&#46;evil&#46;example`,
  plus one literal-plus-entity interplay case such as
  `www.evil&#46;example`, whose literal `www.` is defanged before the
  `&` escape). Assert exact escaped output bytes per case (TC-009),
  e.g. `sanitizeHeaderValue("&#119;ww.evil.example")` contains
  `\&#119;ww.evil.example` and no bare `&#119;`. Drive the matrix from
  a fixture table so the sweep is enumerable, not hand-asserted.
- [x] 2.2 Add render-level hostile-value tests to
  `TestRenderMarkdown_AutolinkTriggersDefanged` (or a sibling): craft
  payloads whose metadata branch, module path, coverage-gap
  description, and quality-step error string each carry
  representative encodings from every family, render the full report,
  and assert the emitted bytes contain the escaped forms. Two renders
  are needed: a failed quality step renders the error-string callout
  and suppresses the gaps table, so a second render with quality data
  present covers the coverage-gap description cell. Include a
  scanner assertion that no `&` byte outside code-span content is
  unescaped anywhere in the report. The scanner rule is pinned: a
  backtick toggle over the emitted bytes, paired with hostile
  payloads that contain no backtick bytes, so no attacker byte
  shifts the toggle; the remaining soundness premises — the fixture
  data contains no `&` byte, and the one backtick-bearing fixture
  value embeds an even number of backticks so the toggle
  resynchronizes after its cell — are asserted as in-test
  preconditions rather than assumed. Backtick-bearing hostile values
  are covered by task 2.1's byte-exact sink assertions rather than
  the scanner.

## 3. Contract integrity (epic #337 rule 3)

- [x] 3.1 Run the golden test suite without `-update` and confirm
  every golden file under `internal/aireport/testdata/golden/`
  matches byte-for-byte (no fixture contains `&`); add an explicit
  benign-parity test pinning `R&D` → `R\&D` (renders as `R&D`) and
  the verbatim-entity case `a&amp;b` → `a\&amp;b` (design D4) so the
  trade-off is regression-pinned as intended behavior.
- [x] 3.2 [P] Confirm the literal-trigger defanging behavior is
  unchanged: the existing assertions in
  `TestRenderMarkdown_AutolinkTriggersDefanged`
  (`www\[.\]evil.example`, `user\[@\]host.example`,
  `https\[:\]//evil.example`, dotless `user@localhost` pass-through)
  still pass unmodified, and `neutralizeAutolinks` itself is
  untouched (design D3).

## 4. Verification and gates

- [x] 4.1 [P] Documentation validation gate: check `README.md`,
  `docs/reference/cli/report.md`, and `docs/guides/` for any
  enumeration of the renderer's escaped characters (none exists
  today); update only if found. Record the website-gate assessment:
  exempt — security hardening with no user-facing behavior change for
  benign input (no command, flag, workflow, or installation change).
- [x] 4.2 Run CI-parity checks locally per AGENTS.md: `go build
  ./cmd/gaze`, `go test -race -count=1 -short ./...`,
  `golangci-lint run`. Any failure is a blocking error.
- [x] 4.3 Verify constitution alignment (Principle I accuracy —
  hostile values render verbatim; Principle IV testability — pure
  byte-level fixtures): the tests from tasks 2.1–2.2 and 3.1 must be
  present and passing; no gate values were modified to make them pass.
- [x] 4.4 `openspec validate markdown-autolink-entity-bypass`
  passes with no issues other than the expected advisory for the
  `originating_issue` metadata key (required by /uf.finale for
  PR-body threading).
