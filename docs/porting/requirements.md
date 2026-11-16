# Porting Requirements

This document lists every capability in the Gaze tool, labeled as **REQUIRED** or **OPTIONAL**. A conforming port must implement all REQUIRED capabilities. OPTIONAL capabilities add value but are not necessary for a minimal viable port.

For behavioral contracts that each capability must honor, see [contracts.md](contracts.md). For the canonical effect type and formula reference tables, see [taxonomy-reference.md](taxonomy-reference.md).

---

## REQUIRED Capabilities

These capabilities form the core of Gaze. Without them, the tool cannot produce its primary output (CRAP scores with contract-aware risk assessment).

### R1: Side Effect Detection

**What it does**: Analyzes source code to detect observable side effects in functions. Uses the language's AST (Abstract Syntax Tree) for return values, error returns, and sentinel errors. Uses a deeper analysis pass (SSA in Go, equivalent in other languages) for mutation tracking (receiver mutations, pointer argument mutations, slice/map mutations, global mutations).

**Contracts**: EC-001 through EC-005 (effect taxonomy, tier membership, identity, structure, language adaptation).

**Detection tiers**:

| Tier | Detection Requirement |
|------|----------------------|
| P0 (6 types) | MUST detect with zero false negatives, zero false positives |
| P1 (11 types) | MUST detect; false positives acceptable if documented |
| P2 (16 types) | SHOULD detect; partial detection acceptable |
| P3 (9 types) | MAY detect; mark as "Defined — detection not yet implemented" if skipped |
| P4 (6 types) | MAY detect; these are exotic and language-specific |

**Suggested test approach**: Create test fixtures — small source files with known side effects — and verify that the detector finds exactly the expected effects. Test each P0 type individually. Test that pure functions (no side effects) produce an empty result.

**Key design decisions**:

- The reference implementation uses a dual-pass architecture: AST for structural effects (returns, sentinels) and SSA for data-flow effects (mutations). A port may use a single pass if the language's analysis tooling supports it.
- Generated files (e.g., files with `// Code generated` headers) SHOULD be excluded by default.
- Test files MUST be excluded from analysis.

---

### R2: Classification

**What it does**: Determines whether each detected side effect is **contractual** (part of the function's intended API), **incidental** (an implementation detail), or **ambiguous** (insufficient evidence). Uses five mechanical signal analyzers that examine the code's structure, not its runtime behavior.

**Contracts**: CC-001 through CC-006 (confidence formula, clamping, thresholds, contradiction detection, five signal categories, signal recording).

**The five signal analyzers**:

| # | Signal | What It Examines | Max Weight |
|---|--------|-----------------|------------|
| 1 | Interface Satisfaction | Does the function implement an interface/trait/protocol? | +30 |
| 2 | API Visibility | Is the function/type/return type public? | +20 |
| 3 | Caller Dependency | How many modules call this function? | +15 |
| 4 | Naming Convention | Does the function name follow contractual/incidental patterns? | +10 (or +30 for sentinels) |
| 5 | Documentation | Does the doc comment declare behavioral intent? | +15 |

**Suggested test approach**: Create functions with controlled signal profiles (e.g., an exported method implementing an interface with GoDoc declaring "returns") and verify the confidence score and label. Test the contradiction penalty by providing both positive and negative signals. Test tier boosts by classifying the same signal set against P0, P1, and P2 effect types.

**Key design decisions**:

- Thresholds are configurable via a configuration file (`.gaze.yaml` in Go). Default: contractual >= 80, incidental < 50.
- The tier boost ensures P0 effects (return values, errors, mutations) trend toward contractual by default, making contract coverage meaningful out-of-the-box without configuration tuning.

---

### R3: CRAP Scoring

**What it does**: Computes CRAP (Change Risk Anti-Patterns) scores by combining cyclomatic complexity with test coverage. Identifies the riskiest functions to change — those that are both complex and under-tested.

**Contracts**: SC-001 through SC-003 (CRAP formula, GazeCRAP formula, CRAPload/GazeCRAPload).

**Inputs required**:

- **Cyclomatic complexity** per function (integer >= 1). A port must either compute this or accept it from an external tool.
- **Line coverage** per function (percentage, 0–100). A port must either generate a coverage profile by running tests or accept a pre-generated one.
- **Contract coverage** per function (percentage, 0–100) — only needed for GazeCRAP. Requires R1 (detection), R2 (classification), and R5 (quality assessment) to be implemented.

**Suggested test approach**: Verify the CRAP formula with known inputs:

| Complexity | Coverage | Expected CRAP |
|-----------|----------|---------------|
| 1 | 100% | 1.0 |
| 1 | 0% | 2.0 |
| 10 | 0% | 110.0 |
| 10 | 100% | 10.0 |
| 5 | 50% | 8.125 |

Test CRAPload counting with a mix of above/below threshold functions.

---

### R4: Quadrant Classification and Fix Strategies

**What it does**: Classifies each function into one of four risk quadrants (Q1–Q4) based on CRAP and GazeCRAP scores, and assigns a remediation strategy to each function in the CRAPload.

**Contracts**: SC-004 through SC-006 (quadrant classification, fix strategy assignment, recommended actions ordering).

**Suggested test approach**: Verify the quadrant truth table with all four combinations of high/low CRAP × high/low GazeCRAP. Verify fix strategy assignment rules in priority order. Verify that recommended actions are sorted by strategy priority then CRAP descending.

---

### R5: Output Formatting

**What it does**: Produces analysis results in machine-readable (JSON) and human-readable (text) formats.

**Contracts**: OC-001 through OC-003 (dual format, JSON field names, nullable fields).

**Suggested test approach**: Verify JSON output against a schema. Verify that optional fields (GazeCRAP, contract coverage, quadrant) are null/absent when not computed, not zero.

---

## OPTIONAL Capabilities

These capabilities enhance Gaze but are not required for a conforming minimal port. A port SHOULD document which optional capabilities it supports.

### O1: Quality Assessment (Test-Target Pairing)

**What it does**: Pairs test functions with the production functions they test (target inference), detects assertions in test code, maps assertions to detected side effects, and computes **contract coverage** — the percentage of contractual side effects that tests actually assert on.

**Why it matters**: Contract coverage is the input to GazeCRAP. Without quality assessment, a port can compute CRAP (using line coverage) but not GazeCRAP (which requires contract coverage). The quadrant classification and `add_assertions` fix strategy also depend on contract coverage.

**Key sub-capabilities**:

| Sub-capability | Description |
|---------------|-------------|
| Target inference | Heuristically pair `TestFoo` with `Foo` based on naming, call graph, and package structure |
| Assertion detection | Identify assertion sites in test code (equality checks, error checks, nil checks, etc.) |
| Assertion mapping | Link each assertion to the side effect it verifies, using multiple passes: direct identity match, indirect root resolution, helper function bridging, inline call matching |
| Contract coverage | Ratio of contractual effects with at least one mapped assertion |
| Over-specification | Ratio of assertions that verify incidental effects (refactoring fragility indicator) |

**Contracts honored**: The quality assessment pipeline feeds into SC-002 (GazeCRAP formula) and SC-004 (quadrant classification). Its internal behavior is not covered by the porting contracts — a port MAY use any mapping strategy that produces accurate contract coverage percentages.

**Suggested test approach**: Create test fixtures with known test-target pairs and verify that contract coverage matches expected values. Test with zero assertions, partial assertions, and full assertions.

---

### O2: AI-Powered Reports

**What it does**: Pipes the combined analysis output (CRAP scores, quality metrics, classification breakdown, documentation scan) to an AI model (Claude, Gemini, Ollama, OpenCode) for narrative interpretation. The AI produces a human-readable report with severity assessments, recommendations, and risk summaries.

**Why it matters**: AI reports translate raw metrics into actionable prose. They are particularly useful in CI pipelines where the report is appended to a GitHub Step Summary or similar artifact.

**Architecture**: The reference implementation uses an adapter pattern — each AI backend implements a common interface (`Format(systemPrompt, payload) → string`). The adapter handles binary lookup, subprocess management, and output capture.

**Contracts honored**: None specific. AI reports consume the output of R1–R5 but do not define behavioral contracts of their own. A port MAY use any AI integration approach.

---

### O3: Document Scanning

**What it does**: Scans project documentation files (README, API docs, architecture docs) for behavioral declarations that contribute to classification signals. Extends the documentation signal (Signal 5 in the classification contract) beyond function-level doc comments to project-wide documentation.

**Why it matters**: Some functions' contractual nature is documented in README files or architecture docs rather than in code comments. Document scanning captures these signals.

**Contracts honored**: Feeds into CC-005 (Signal 5: Documentation). The scanning mechanism is not specified — a port MAY use any approach to extract behavioral keywords from documentation files.

---

### O4: Interactive TUI

**What it does**: Provides a terminal user interface for browsing analysis results interactively, with filtering, sorting, and drill-down into individual functions.

**Why it matters**: Useful for local development workflows where a developer wants to explore results without piping JSON through `jq`.

**Contracts honored**: None. The TUI is a presentation layer with no behavioral contracts.

---

### O5: CI Threshold Enforcement

**What it does**: Accepts threshold flags (`--max-crapload`, `--max-gaze-crapload`, `--min-contract-coverage`) and exits with a non-zero status code when thresholds are violated. Enables CI pipelines to fail builds when code quality degrades.

**Why it matters**: Without threshold enforcement, Gaze is informational only. With it, Gaze becomes a quality gate.

**Contracts honored**: SC-003 (CRAPload/GazeCRAPload definitions). The threshold comparison is straightforward: if `crapload > max_crapload`, exit non-zero.

---

### O6: Coverage Profile Reuse

**What it does**: Accepts a pre-generated coverage profile (`--coverprofile` flag) instead of running tests internally. Avoids the "double test run" problem in CI where tests run once for results and again inside Gaze for coverage.

**Why it matters**: In CI pipelines, test runs can take minutes. Running tests twice (once for CI, once for Gaze) wastes time and compute. Coverage profile reuse eliminates the second run.

**Contracts honored**: None specific. The coverage profile format is language-dependent.

---

### O7: Configuration File

**What it does**: Loads classification thresholds, document scan settings, and other options from a configuration file (`.gaze.yaml` in the reference implementation).

**Why it matters**: Allows teams to customize thresholds and exclusion patterns without command-line flags.

**Key configurable values**:

| Setting | Default | Description |
|---------|---------|-------------|
| `classification.thresholds.contractual` | 80 | Minimum confidence for contractual label |
| `classification.thresholds.incidental` | 50 | Upper bound for incidental label |
| `classification.doc_scan.exclude` | vendor, node_modules, .git, testdata, etc. | Glob patterns to exclude from doc scanning |
| `classification.doc_scan.timeout` | 30s | Maximum duration for document scanning |

---

## Capability Dependency Graph

```text
R1 (Detection) ──→ R2 (Classification) ──→ O1 (Quality) ──→ R3 (CRAP + GazeCRAP)
                                                              ↓
                                                         R4 (Quadrants + Fix Strategies)
                                                              ↓
                                                         R5 (Output)
                                                              ↓
                                                    O2 (AI Reports)  O4 (TUI)  O5 (CI Thresholds)
```

- R3 can compute basic CRAP without R2 or O1 (using line coverage only)
- R3 needs R1 + R2 + O1 to compute GazeCRAP (using contract coverage)
- R4 needs GazeCRAP, so it transitively needs R1 + R2 + O1
- O3 (Document Scanning) feeds into R2 (Classification) as an additional signal source

---

## Minimum Viable Port

A minimum viable port implements **R1 + R3 + R5** (detection, CRAP scoring with line coverage, JSON/text output). This produces CRAP scores and CRAPload — useful but without contract-level analysis.

A **recommended port** adds **R2 + R4 + O1** (classification, quadrants, quality assessment). This enables GazeCRAP, quadrant classification, and fix strategies — the full Gaze value proposition.
