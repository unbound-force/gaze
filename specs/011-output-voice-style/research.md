# Research: Output Voice & Style Standardization

**Branch**: `011-output-voice-style` | **Date**: 2026-03-01

## R1: Current Gaze-Reporter Prompt Structure

**Decision**: The gaze-reporter prompt (331 lines) is a self-contained markdown file that defines the agent's behavior. The voice standard is encoded primarily in the `## Output Format` section (lines 192-235) and scattered inline directives (lines 20-21, 82-88, 164-171). All voice changes can be made by rewriting these sections in-place.

**Rationale**: The prompt structure (Binary Resolution, Mode Parsing, CRAP Mode, Quality Mode, Full Mode, Output Format, Example Output, Graceful Degradation, Error Handling) is well-organized. Only the Output Format section and mode-specific inline tone directives need replacement. The structural sections (binary resolution, error handling, graceful degradation) are tone-neutral and remain unchanged.

**Alternatives considered**:
- Separate voice standard into a dedicated file and reference it from the prompt. Rejected: adds file management complexity for a single consumer. The prompt is the only place voice rules are enforced.
- Create a voice standard library in Go code. Rejected: the voice is enforced by the LLM agent, not by Go formatters. Go CLI output uses lipgloss and is out of scope.

## R2: File Modification Set

**Decision**: Six files must be modified; seven files must be deleted.

**Files to REWRITE (content changes)**:
1. `.opencode/agents/gaze-reporter.md` — active prompt, rewrite Output Format + inline directives
2. `internal/scaffold/assets/agents/gaze-reporter.md` — scaffolded copy, must remain byte-identical to active prompt

**Files to UPDATE (targeted edits)**:
3. `AGENTS.md` — replace spec 010 entry in Recent Changes (line 240) with spec 011 entry; update spec listing (line 90)

**Files to DELETE** (per FR-014, exist on `main` only):
4. `specs/010-report-voice-refinement/spec.md`
5. `specs/010-report-voice-refinement/plan.md`
6. `specs/010-report-voice-refinement/tasks.md`
7. `specs/010-report-voice-refinement/research.md`
8. `specs/010-report-voice-refinement/data-model.md`
9. `specs/010-report-voice-refinement/quickstart.md`
10. `specs/010-report-voice-refinement/checklists/requirements.md`

**Files NOT changed**:
- `.opencode/command/gaze.md` — tone-neutral command definition, no voice directives
- `internal/scaffold/assets/command/gaze.md` — same as above
- Go source files (`scaffold.go`, `scaffold_test.go`, `main_test.go`) — reference filenames only, not prompt content

**Rationale**: The scaffolded copy must always be byte-identical to the active prompt (this is the existing convention). The spec 010 files must be deleted because the user explicitly chose full deletion over archival in the clarification session.

**Branch note**: This branch diverges from `main` before spec 010 artifacts were added. The branch must be rebased onto current `main` before the spec 010 files can be deleted in a commit.

## R3: Voice Standard Design — Emoji Vocabulary

**Decision**: Define a fixed emoji vocabulary mapped to semantic roles. The vocabulary is exhaustive — no emojis outside this set appear in report output.

| Emoji | Semantic Role | Usage Context |
|-------|---------------|---------------|
| 🔍 | Report title marker | Prefixes the report title line |
| 📊 | CRAP section marker | Prefixes CRAP Summary header |
| 🧪 | Quality section marker | Prefixes Quality Summary header |
| 🏷️ | Classification section marker | Prefixes Classification Summary header |
| 🏥 | Health section marker | Prefixes Overall Health Assessment header |
| 🟢 | Good/safe severity | Grades B+ and above; Q1 quadrant; low-priority recommendations |
| 🟡 | Moderate/warning severity | Grades B through C; Q2 quadrant; medium-priority recommendations |
| 🔴 | Critical/danger severity | Grades C- and below; Q4 quadrant; high-priority recommendations |
| ⚪ | Neutral/no data | Q3 quadrant; N/A grades |
| ⚠️ | Warning callout | Advisory notices in blockquotes |

**Rationale**: A closed vocabulary ensures consistency across agent model versions and prevents emoji creep. Each emoji has exactly one semantic role.

**Alternatives considered**:
- Open vocabulary allowing any relevant emoji. Rejected: leads to inconsistency between runs and model versions.
- No emojis at all (spec 010 approach). Rejected: user explicitly requested emoji-rich output.

## R4: Voice Standard Design — Tone Rules

**Decision**: Define tone through banned anti-patterns rather than required vocabulary.

**Banned anti-patterns** (FR-004):
- Excessive exclamation marks (at most one per full report)
- Slang or meme references
- Puns on metric names
- First-person pronouns ("I", "we")

**Positive tone guidance**:
- Conversational sentence structure (contractions allowed, e.g., "they're", "don't")
- One interpretive sentence per data table, max 25 words
- Interpretive sentences state the practical takeaway, not what the data "shows"
- No pedagogical explanations of what metrics mean

**Rationale**: Anti-pattern bans are testable (grep for violations). A required word list would be fragile and constrain natural language generation unnecessarily.

## R5: Grade-to-Emoji Mapping

**Decision**: Use letter grades with deterministic emoji boundaries.

| Grade | Emoji | Severity Band |
|-------|-------|---------------|
| A     | 🟢   | Good          |
| A-    | 🟢   | Good          |
| B+    | 🟢   | Good          |
| B     | 🟡   | Moderate      |
| B-    | 🟡   | Moderate      |
| C+    | 🟡   | Moderate      |
| C     | 🟡   | Moderate      |
| C-    | 🔴   | Critical      |
| D     | 🔴   | Critical      |
| F     | 🔴   | Critical      |

**Rationale**: This matches the canonical example output (B+ = 🟢, C+ = 🟡) and was confirmed in the clarification session.

## R6: Report Structure Changes from Spec 010

**Decision**: Replace the spec 010 report structure with the canonical example structure.

| Element | Spec 010 (clinical) | Spec 011 (fun) |
|---------|---------------------|----------------|
| Title | `Gaze Health Report — <project>` | `🔍 Gaze Full Quality Report` (or mode variant) |
| Metadata | `Package: <pattern> \| Date: <date>` | `Project: <path> · Branch: <branch>` + `Gaze Version: <ver> · Go: <ver> · Date: <date>` |
| Section headers | Plain text, no decorators | Emoji-prefixed |
| Grades | Word-based (Poor/Fair/Good/Strong/Excellent) | Letter grades (A through F) with emoji indicators |
| Recommendations | No emoji prefixes | Severity emoji prefix (🔴/🟡/🟢) |
| Health assessment | Risk Matrix → Recommendations → Overall Grade → Bottom line | Summary Scorecard → Recommendations (simplified) |
| Omitted sections | Single-line note at report end | Silently omitted, no trailing note |
| Quadrant labels | Plain text (Q1 — Safe) | Emoji-prefixed (🟢 Q1 — Safe) |
| Warning notices | Not defined | Blockquote with ⚠️ prefix |

**Rationale**: The canonical example defines the target format. Each change is traceable to a specific FR in the spec.
