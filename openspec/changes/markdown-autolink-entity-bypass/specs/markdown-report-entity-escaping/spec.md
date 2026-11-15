# Markdown Report Entity Escaping Requirements

## ADDED Requirements

### Requirement: Ampersands MUST be escaped outside code spans

The deterministic markdown renderer MUST backslash-escape every `&`
byte as `\&` in attacker-influenced values emitted outside code
spans: a backslash-escaped ampersand cannot open an HTML character
reference in cmark-gfm, so no reference at those positions can
decode into text after rendering. Code-span content is governed by
the verbatim requirement below.

#### Scenario: Every non-code-span position is covered

- **GIVEN** hostile values placed in each attacker-influenced
  position — the metadata header values (module path, branch, gaze
  version, date), the prose positions (AI summary, AI failure notes,
  step-unavailable callout details, SSA-degraded package lists,
  recommendation text), and the plain table cells (fix-strategy
  labels, locations, gap types, gap descriptions, gap locations,
  scorecard labels and details)
- **WHEN** the markdown report is rendered
- **THEN** no `&` byte outside code-span content is emitted without
  a preceding backslash

#### Scenario: Entity-encoded www trigger in a branch name

- **GIVEN** a report payload whose metadata branch is
  `&#119;ww.evil.example`
- **WHEN** the markdown report is rendered
- **THEN** the emitted header bytes contain `\&#119;ww.evil.example`
- **AND** the emitted header does not contain any `&` byte that is
  not backslash-escaped

#### Scenario: Entity-encoded email trigger in a callout detail

- **GIVEN** a report payload whose quality step failed with an error
  string containing `mail user&#64;host.example now`
- **WHEN** the markdown report renders the step-unavailable callout
- **THEN** the emitted bytes contain `user\&#64;host.example`

#### Scenario: Entity-encoded scheme trigger in a plain table cell

- **GIVEN** a report payload whose coverage-gap description contains
  `see https&colon;//evil.example`
- **WHEN** the markdown report renders the gaps table
- **THEN** the emitted cell bytes contain `https\&colon;//evil.example`
  (the literal `://` defanging is unchanged and orthogonal)

#### Scenario: Benign ampersand is escaped and still renders as itself

- **GIVEN** a report payload whose metadata module path contains
  `R&D`
- **WHEN** the markdown report is rendered
- **THEN** the emitted header bytes contain `R\&D`, which cmark-gfm
  renders as the visible text `R&D`

### Requirement: Entity-encoded autolink triggers MUST NOT survive rendering

For every HTML character reference encoding of every autolink
trigger character, the emitted report bytes MUST NOT contain a
decodable reference at a non-code-span position where cmark-gfm
could resolve it into text that the GFM autolink extension
linkifies. Byte-level fixtures MUST cover the decimal numeric,
hexadecimal numeric (either `x` case, any digit case),
named-reference, and hybrid families; the universal `&` escape is
what closes variant encodings within those families.

#### Scenario: Decimal numeric encoding of the www trigger

- **GIVEN** a hostile value containing `&#119;ww.evil.example`
- **WHEN** it passes through any non-code-span sink
  (`sanitizeHeaderValue`, `sanitizeProse`, `escapeCell`)
- **THEN** the sink output contains `\&#119;ww.evil.example` and no
  bare `&#119;`

#### Scenario: Hexadecimal encoding with uppercase marker

- **GIVEN** a hostile value containing `&#X3A;//evil.example`
- **WHEN** it passes through any non-code-span sink
- **THEN** the sink output contains `\&#X3A;//evil.example` and no
  bare `&#X3A;`

#### Scenario: Named reference encoding of the email trigger

- **GIVEN** a hostile value containing `user&commat;host.example`
- **WHEN** it passes through any non-code-span sink
- **THEN** the sink output contains `user\&commat;host.example` and
  no bare `&commat;`

#### Scenario: Hybrid literal-and-entity encoding of the www trigger

- **GIVEN** a hostile value containing `www&#46;evil&#46;example`
- **WHEN** it passes through any non-code-span sink
- **THEN** the sink output contains `www\&#46;evil\&#46;example` and
  no bare `&#46;`

#### Scenario: Render-level hostile branch and cell

- **GIVEN** a report payload whose metadata branch is
  `&#119;ww.evil.example` and whose coverage-gap description contains
  `user&#64;host.example`
- **WHEN** the full markdown report is rendered
- **THEN** the emitted bytes contain no `&` byte that is not
  backslash-escaped outside code spans

### Requirement: Code-span content MUST keep ampersands verbatim

Function and test names render inside code spans. Entities are never
interpreted inside code spans and the autolink extension never fires
there, so `&` inside code-span content MUST NOT be escaped — a
backslash there would display literally.

#### Scenario: Function name containing an entity reference

- **GIVEN** a report payload whose worst-CRAP function name contains
  `parse&#64;Input`
- **WHEN** the worst-CRAP table is rendered
- **THEN** the emitted bytes contain the code span
  `` `parse&#64;Input` `` with the `&` verbatim (no `\&` inside the
  backtick span)

### Requirement: Rendered output for benign input MUST NOT change

The escaping change MUST NOT alter the rendered output for values
without a decodable character reference, and the golden files under
`internal/aireport/testdata/golden/` — the normative byte-level
contract of `specs/040-deterministic-ai-report-rendering/contracts/markdown-report.md`
— MUST remain byte-identical.

#### Scenario: Ampersand that forms no valid reference

- **GIVEN** a value containing `R&D` (the `&` is followed by a
  character that cannot continue a reference)
- **WHEN** it passes through any non-code-span sink
- **THEN** the sink output contains `R\&D`, which cmark-gfm renders
  as the same visible text `R&D` the unescaped byte produced before
  this change

#### Scenario: Golden files unchanged

- **GIVEN** the existing golden payloads and AI outcomes
- **WHEN** the renderer test suite runs
- **THEN** every golden file matches byte-for-byte without
  regeneration (no fixture contains `&`)

#### Scenario: Value containing a valid reference renders verbatim

- **GIVEN** a metadata branch literally named `a&amp;b`
- **WHEN** the markdown report is rendered
- **THEN** the emitted header bytes contain `a\&amp;b`, which
  cmark-gfm renders as the visible text `a&amp;b` (the pre-change
  renderer silently displayed `a&b` — decoding a reference inside an
  identifier; see design D4)
