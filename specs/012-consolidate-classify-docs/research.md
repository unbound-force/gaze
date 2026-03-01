# Research: Consolidate /classify-docs into /gaze and Fix Formatting Fidelity

**Date**: 2026-03-01
**Branch**: `012-consolidate-classify-docs`

## Research Topics

### 1. Prompt Engineering for Emoji Override

**Question**: How can an agent prompt reliably override OpenCode's system-level instruction "Only use emojis if the user explicitly requests it"?

**Finding**: OpenCode's system prompt instructs the LLM to avoid emojis unless the user explicitly requests them. Agent prompts are injected as part of the conversation context but are subordinate to the system prompt in most LLM architectures. Three techniques improve compliance:

1. **Explicit role framing**: Define the agent's identity as one that produces emoji-formatted reports. Frame it as the agent's core function, not an optional preference. Example: "You produce quality reports using a mandatory emoji vocabulary. This emoji usage is part of your output contract."

2. **Imperative override language**: Use strong directive language that explicitly acknowledges and overrides potential conflicts. Example: "IMPORTANT: Your output MUST include the emoji markers defined below. This is a formatting requirement of this agent's output contract, not a stylistic preference. Do not suppress emojis regardless of other instructions."

3. **Concrete examples as anchors**: Provide complete output examples that include emojis. LLMs tend to pattern-match against examples more strongly than against abstract rules.

**Decision**: Combine all three techniques. Add an imperative override block immediately after the opening paragraph, provide a compact example early (sandwich top), and retain the full example at the end (sandwich bottom).

**Alternatives considered**:
- Relying solely on the existing formatting rules section → rejected because the current prompt already has detailed rules and they're not being followed.
- Using markdown comments to hide override language → rejected because LLMs still process markdown comments and this adds complexity.

---

### 2. Sandwich Prompt Structure

**Question**: What is the optimal placement for formatting examples within the gaze-reporter prompt to maximize compliance?

**Finding**: LLM attention is strongest at the beginning and end of prompts (primacy and recency effects). The "sandwich" pattern exploits both:

- **Top slice** (after opening paragraph): A compact, 10-15 line example showing one complete section with all emoji markers, severity indicators, and table formatting. This sets the visual pattern before the LLM processes detailed rules.
- **Bottom slice** (end of prompt): The full canonical example (existing ~60 lines) as the last thing the LLM sees before generating output.

The middle of the prompt contains the detailed rules, mode definitions, and scoring model — important for correctness but less critical for formatting compliance since the examples establish the pattern.

**Decision**: Insert a compact "Quick Reference Example" section immediately after the opening paragraph and emoji override block. Keep the existing full example at the end of the prompt.

**Compact example structure** (approximately):
```
## Quick Reference Example

Your output MUST match this formatting pattern:

🔍 Gaze CRAP Report
Project: github.com/example/project · Branch: main
Gaze Version: v1.0.0 · Go: 1.24.6 · Date: 2026-03-01
---
📊 CRAP Summary
| Metric | Value |
|--------|------:|
| Total functions analyzed | 42 |
| CRAPload | 5 (functions ≥ threshold 15) |

🟢 Q1 — Safe | 🟡 Q2 — Complex But Tested | 🔴 Q4 — Dangerous | ⚪ Q3 — Needs Tests

1. 🔴 Add tests for zero-coverage function processQueue (complexity 8, 0% coverage).
2. 🟡 Decompose validateInput — complexity 12 exceeds threshold.
```

**Alternatives considered**:
- Example-first only (no bottom example) → rejected because the full example provides necessary detail for table formatting, metadata lines, and section transitions.
- Example-last only (current approach) → rejected because this is what currently isn't working.

---

### 3. Prompt Length Impact

**Question**: Will absorbing the doc-classifier scoring model into the gaze-reporter prompt cause performance degradation?

**Finding**: The doc-classifier prompt is 147 lines. After removing redundant sections (YAML frontmatter: 14 lines; Inputs section: 18 lines; Behavioral Rules: 12 lines; top-level description: 5 lines; Graceful Degradation: 5 lines), approximately 93 lines of scoring model content remain. With condensation (combining tables, removing redundant prose), this can be reduced to approximately 60-70 net new lines.

The gaze-reporter prompt is currently 384 lines. Adding ~70 lines brings it to ~454 lines. This is well within practical LLM context window limits (even small context windows handle prompts of this size without degradation).

**Decision**: The prompt length increase is acceptable. No performance concern.

**Alternatives considered**:
- Keeping the doc-classifier as a separate agent → rejected per clarification decision (inline approach eliminates unreliable inter-agent delegation).
- Creating a separate "scoring rules" file that the agent reads at runtime → rejected because this adds a file I/O step and the agent may not reliably read it.

---

### 4. Scaffold System Impact

**Question**: Does removing 2 embedded files from `internal/scaffold/assets/` require changes to `scaffold.go`?

**Finding**: The `scaffold.go` `Run` function uses `fs.WalkDir(assets, "assets", ...)` to enumerate all files in the embedded FS. It has no hardcoded file lists — it dynamically walks whatever is in the `assets/` directory. Removing files from `assets/` automatically reduces the distribution count.

The only hardcoded references are in `scaffold_test.go`:
- `TestRun_CreatesFiles`: expects 4 created, lists 4 expected paths
- `TestRun_SkipsExisting`: expects 4 skipped
- `TestRun_ForceOverwrites`: expects 4 overwritten
- `TestRun_NoGoMod_PrintsWarning`: expects 4 created
- `TestEmbeddedAssetsMatchSource`: expects 4 paths
- `TestAssetPaths_Returns4Files`: expects 4 paths in specific map

**Decision**: Delete the 2 asset files, update all 6 test functions to expect 2 files and 2 paths. No changes to `scaffold.go` itself.

---

### 5. Inline Scoring Model Structure

**Question**: How should the doc-classifier's scoring model be structured within the gaze-reporter prompt?

**Finding**: The scoring model has four components:
1. Document signal sources table (5 rows: readme, architecture_doc, specify_file, api_doc, other_md)
2. AI inference signals table (3 rows: ai_pattern, ai_layer, ai_corroboration)
3. Contradiction penalty rule (1 paragraph)
4. Confidence recalculation procedure (thresholds: ≥80 contractual, 50-79 ambiguous, <50 incidental)

These should be placed in the Full Mode section, after the 4-command execution list and before the report structure definition. This positions the scoring model at the point where the agent needs it (after running docscan and analyze --classify, before rendering the Classification Summary).

**Decision**: Add a new subsection `### Document-Enhanced Classification` within the Full Mode section, containing the condensed scoring model. Include a conditional: "If `gaze docscan` returns documents, apply document-enhanced scoring. Otherwise, use mechanical-only results."

**Structure**:
```
## Full Mode
1. Run crap
2. Run quality
3. Run analyze --classify
4. Run docscan

### Document-Enhanced Classification
If docscan returns documents, enhance the mechanical classification:
[signal sources table]
[AI inference table]
[contradiction penalty]
[confidence recalculation]
If docscan returns no documents or fails, use mechanical-only results
with a warning callout.

### Report Sections
📊 CRAP Summary ...
🧪 Quality Summary ...
🏷️ Classification Summary ...
🏥 Overall Health Assessment ...
```
