# Data Model: Deterministic AI Report Rendering

**Feature**: `040-deterministic-ai-report-rendering`
**Package**: all new types live in `internal/aireport` unless noted.
Go types are sketches that fix names, fields, and JSON tags. Final
GoDoc is written during implementation.

---

## 1. Section selection

```go
// Section identifies one renderable report section.
type Section string

const (
    SectionCRAP           Section = "crap"
    SectionQuality        Section = "quality"
    SectionClassification Section = "classification"
    SectionHealth         Section = "health"
)

// SectionSet is an immutable, canonically ordered set of sections.
type SectionSet struct{ /* unexported bitset */ }

func ParseSections(raw []string) (SectionSet, error) // FR-026, edge "unknown section"
func AllSections() SectionSet
func (s SectionSet) Has(Section) bool
func (s SectionSet) List() []Section        // canonical order: crap, quality, classification, health
func (s SectionSet) Steps() StepSet         // R8 mapping
func CheckThresholds(s SectionSet, cfg ThresholdConfig) error // FR-028

// StepSet records which pipeline steps run.
type StepSet struct{ CRAP, Quality, Classify, Docscan bool }
```

**Validation rules**
- An empty list is an error, and so is an empty element (from
  `--sections=` or `--sections=crap,,quality`).
- Unknown names produce
  `invalid --sections value "x": valid sections are crap, quality, classification, health`.
- Duplicates collapse. Flag order never changes render order.
- Step mapping: `crap→CRAP`, `quality→Quality`,
  `classification→Classify`, `health→CRAP+Quality+Docscan`.
- Threshold → step requirements: `MaxCrapload`/`MaxGazeCrapload` →
  CRAP; `MinContractCoverage` → Quality. A violation produces
  `--min-contract-coverage requires the quality analysis, which --sections=crap does not run`.

**Effect on the pipeline**: `runProductionPipeline` receives a
`StepSet`. An unselected step is not invoked, and its payload field
and error both stay `nil`. The contract-coverage provider is built
only when CRAP runs. `checkZeroResultGate` applies only when
`StepSet.CRAP` is set (research R8).

---

## 2. Report metadata (additive JSON, FR-003 / R7)

```go
type ReportMetadata struct {
    ModulePath  string    `json:"module_path"`
    Branch      string    `json:"branch,omitempty"`
    GazeVersion string    `json:"gaze_version"`
    Date        string    `json:"date"`     // YYYY-MM-DD, UTC
    Sections    []Section `json:"sections"` // canonical order
}
```

The CLI (`runReport`) builds this value and passes it through
`RunnerOptions.Metadata`. `Run` copies it onto the payload. The
renderer reads only the payload. A clock (`reportParams.now func()
time.Time`) is injected for tests.

---

## 3. ReportPayload additions

```go
type ReportPayload struct {
    Summary    ReportSummary   `json:"-"`
    Metadata   ReportMetadata  `json:"metadata"`             // NEW
    CRAP       json.RawMessage `json:"crap"`
    Quality    json.RawMessage `json:"quality"`
    Classify   json.RawMessage `json:"classify"`
    Docscan    json.RawMessage `json:"docscan"`
    Errors     PayloadErrors   `json:"errors"`
    Assessment *Assessment     `json:"assessment"`           // NEW (FR-010)
}
```

- The new fields are additive. Existing fields are unchanged
  (spec assumption "Machine-readable output").
- `Assessment` is computed once in `Run` by
  `Assess(payload *ReportPayload) Assessment`, after analysis and
  before the format switch. It is therefore present in json, html
  (ignored by the template), markdown, and text.
- `CompactForAI` adds `metadata` and `assessment` to the compact
  payload, so the AI sees Gaze's grades instead of inventing its own.
  The addition is bounded (5 rows plus ≤ 8 recommendations) and does
  not affect the existing compact size budget test.

---

## 4. Assessment (FR-007–FR-010)

```go
// Grade is a letter grade; the closed set is A, A-, B+, B, B-, C+, C, C-, D, F.
type Grade string

// Severity is the closed severity vocabulary shared by grades and
// recommendations. Grade mapping (FR-008): A/A-/B+→low, B/B-/C+/C→moderate,
// C-/D/F→critical, unavailable→unavailable.
type Severity string

const (
    SeverityLow         Severity = "low"         // 🟢
    SeverityModerate    Severity = "moderate"    // 🟡
    SeverityCritical    Severity = "critical"    // 🔴
    SeverityUnavailable Severity = "unavailable" // ⚪
)

type Assessment struct {
    RubricVersion   string           `json:"rubric_version"` // "1"
    Scorecard       []ScorecardRow   `json:"scorecard"`      // always 5 rows, fixed order
    Recommendations []Recommendation `json:"recommendations"`// 0..8
}

type ScorecardRow struct {
    Dimension string   `json:"dimension"` // crapload | gaze_crapload | avg_line_coverage | contract_coverage | complexity
    Label     string   `json:"label"`     // "CRAPload", "GazeCRAPload", "Avg Line Coverage", "Contract Coverage", "Complexity"
    Value     *float64 `json:"value"`     // graded input (displayed one-decimal value / percent); null when N/A
    Grade     *Grade   `json:"grade"`     // null when N/A
    Severity  Severity `json:"severity"`
    Marker    string   `json:"marker"`    // 🟢 🟡 🔴 ⚪
    Detail    string   `json:"detail"`    // exactly the Details cell text rendered in markdown
}

type Recommendation struct {
    Severity    Severity `json:"severity"`
    Marker      string   `json:"marker"`
    Source      string   `json:"source"` // fix_strategy | ssa_degraded | skipped_tests | ambiguous_classification
    Text        string   `json:"text"`   // exactly the list-item text rendered in markdown (without "N. <marker> ")
    Function    string   `json:"function,omitempty"`
    File        string   `json:"file,omitempty"`
    Line        int      `json:"line,omitempty"`
    FixStrategy string   `json:"fix_strategy,omitempty"`
}
```

**Invariants (round-trip seam, FR-035 / SC-005)**
- `Scorecard[i].Marker + " " + Grade` (or `⚪ N/A`) and `Detail` are
  the same strings the markdown scorecard row prints for
  `Label`.
- `Recommendations[i]` renders as
  `"{i+1}. {Marker} {Text}"`.
- The order is fixed: scorecard rows follow the dimension order above.
  Recommendations follow `recommended_actions` order (never
  re-ranked, FR-006), and the condition-derived items come after them
  in the order `ssa_degraded`, `skipped_tests`,
  `ambiguous_classification`.

The rubric tables, N/A rules, and recommendation templates are in
[contracts/grade-rubric.md](contracts/grade-rubric.md).

---

## 5. AI answer and outcome (FR-012, FR-018–FR-021)

```go
// AIAnswer is the only judgment shape accepted from any adapter.
type AIAnswer struct {
    Summary         string   `json:"summary"`
    Recommendations []string `json:"recommendations"`
}

// AnswerRejection is returned by ParseAIAnswer; it never embeds the raw output.
type AnswerRejection struct {
    Rule   string // stable rule ID, see contracts/ai-answer.md
    Detail string // short, bounded (≤ 200 bytes), no raw-output echo
}
func (e *AnswerRejection) Error() string // "AI answer rejected (rule <Rule>): <Detail>"

func ParseAIAnswer(raw string) (AIAnswer, error) // pure; error is *AnswerRejection

// AIOutcome tells the renderer what to put in the AI slots.
type AIOutcome struct {
    Answer       *AIAnswer // non-nil when validated
    FailureNote  string    // non-empty when the adapter failed or the answer was rejected
}
```

**State transitions (text format)**

```text
adapter.Format ──err──────────────► Outcome{FailureNote: "AI narrative unavailable: <short adapter error>"}
      │ok                                        │
      ▼                                          ▼
ParseAIAnswer ──*AnswerRejection──► Outcome{FailureNote: "AI narrative rejected: <rule detail>"}
      │ok                                        │
      ▼                                          ▼
Outcome{Answer}            RenderMarkdown(payload, sections, outcome)
                                     │
                          stdout + step summary (always)
                                     │
                 return errors.Join(aiErr, thresholdErr)  // aiErr non-nil ⇒ exit ≠ 0
```

In markdown format the outcome is the zero value: no slots, and no
adapter is contacted.

---

## 6. Rendering entry point

```go
// RenderMarkdown renders the full report deterministically from payload
// alone (plus the AI outcome for the two slots). Pure: no I/O, no clock.
func RenderMarkdown(p *ReportPayload, outcome AIOutcome) (string, error)
```

The sections come from `p.Metadata.Sections`. If that list is empty
(older payloads or tests), all sections render. The renderer decodes
`p.CRAP` / `p.Quality` / `p.Docscan` into private view structs: a
`crap.Report`, the existing `qualityOutput`, and
`apidoc.DocscanEnvelope`. It uses `p.Summary` for counts. Each section
renderer is a separate function writing to a `*strings.Builder`
(CC ≤ 10; research R12).

The layout is in [contracts/markdown-report.md](contracts/markdown-report.md).

---

## 7. OpenCode event stream (internal to the adapter)

```go
type openCodeEvent struct {
    Type  string          `json:"type"`  // step_start | text | tool_use | step_finish | reasoning | error
    Part  *openCodePart   `json:"part,omitempty"`
    Error json.RawMessage `json:"error,omitempty"`
}

type openCodePart struct {
    ID        string `json:"id"`
    MessageID string `json:"messageID"`
    Type      string `json:"type"`      // text | tool | step-start | step-finish
    Text      string `json:"text,omitempty"`
    Synthetic bool   `json:"synthetic,omitempty"`
    Ignored   bool   `json:"ignored,omitempty"`
    Tool      string `json:"tool,omitempty"`
    State     *struct{ Status string `json:"status"` } `json:"state,omitempty"`
    Reason    string `json:"reason,omitempty"`
}

// extractFinalText implements research R1. Pure function over stdout bytes.
func extractFinalText(stdout []byte) (string, error)
```

---

## 8. RunnerOptions / reportParams changes

| Struct | Change |
|--------|--------|
| `RunnerOptions` | `+ Sections SectionSet`, `+ Metadata ReportMetadata`. `- SystemPrompt` is removed; the runner always uses the embedded report-mode prompt. `Format` accepts `"markdown"`. |
| `reportParams` (cmd/gaze) | `+ sections []string`, `+ now func() time.Time`, `+ analyzeFunc` (a test seam forwarded to `RunnerOptions.AnalyzeFunc` for CLI-level golden tests), `+ gazeVersion string` |
| `validateRunnerOpts` | Formats: text, json, html, markdown. An adapter is required only for text. |
| `cliutil.ValidateFormat` call | `ValidateFormat(p.format, "html", "markdown")` |

---

## 9. Removed entities

| Removed | Replaced by |
|---------|-------------|
| `internal/aireport/assets/agents/gaze-reporter.md` | `internal/aireport/assets/report-mode-prompt.md` (not scaffolded) |
| `LoadPrompt`, `stripFrontmatter`, `defaultPromptRaw` | `ReportModePrompt()` (embedded string accessor) |
| `TestEmbeddedPromptMatchesScaffold`, `TestLoadPrompt_*`, `TestStripFrontmatter*` | `TestReportModePrompt_*` (content contract tests) |
| AGENTS.md File Sync Invariants row 2 | — |

---

## 10. Scaffold ownership

`isToolOwned("agents/gaze-reporter.md") == true` (FR-034). The
manifest stays at 9 files. Contents change for `agents/gaze-reporter.md`
(rewrite), `references/example-report.md` (real rendered example),
and `commands/gaze.md` (description wording only). Each of these keeps
its `.opencode/` mirror byte-identical (`TestEmbeddedAssetsMatchSource`).
