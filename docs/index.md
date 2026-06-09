# Gaze Documentation

Gaze is a static analysis tool for Go that detects observable side effects in functions and computes CRAP (Change Risk Anti-Patterns) scores by combining cyclomatic complexity with test coverage. It helps you find functions that are complex and under-tested — the riskiest code to change.

These docs cover everything from first-time setup to language porting contracts. Pick the section that matches what you need.

---

## Where to Start

| If you want to… | Start here |
|-----------------|------------|
| **Install and try Gaze** | [Getting Started](getting-started/installation.md) — install, run your first analysis, interpret the output |
| **Understand the model** | [Concepts](concepts/side-effects.md) — side effects, classification, scoring, quality metrics, and the analysis pipeline |
| **Look up a command** | [CLI Reference](reference/cli/analyze.md) — flags, defaults, and output formats for every subcommand |
| **Integrate with CI or AI** | [Guides](guides/ci-integration.md) — GitHub Actions workflows, AI-powered reports, score improvement strategies |
| **Contribute to Gaze** | [Architecture](architecture/overview.md) — package structure, data flow, coding conventions, extension points |
| **Port Gaze to another language** | [Porting](porting/contracts.md) — language-agnostic contracts, capability requirements, taxonomy reference |

---

## Table of Contents

### Getting Started

- [Installation](getting-started/installation.md) — Homebrew, `go install`, build from source, platform notes
- [Why Line Coverage Isn't Enough](getting-started/concepts.md) — Why line coverage is insufficient and what contract-level analysis adds
- [Quickstart](getting-started/quickstart.md) — Guided walkthrough from install to meaningful output in under 10 minutes

### Concepts

- [Side Effects](concepts/side-effects.md) — All 37 effect types across 5 tiers (P0–P4) with definitions and detection status
- [Classification](concepts/classification.md) — Signal analyzers, confidence scoring, tier-based boosts, and classification labels
- [Scoring](concepts/scoring.md) — CRAP formula, GazeCRAP formula, four quadrants, fix strategies, CRAPload and GazeCRAPload
- [Quality Assessment](concepts/quality.md) — Test-target pairing, assertion mapping, contract coverage, and over-specification
- [Analysis Pipeline](concepts/analysis-pipeline.md) — How AST and SSA analysis work together to detect side effects

### Reference

- **CLI Commands**
  - [`gaze analyze`](reference/cli/analyze.md) — Side effect detection
  - [`gaze crap`](reference/cli/crap.md) — CRAP score analysis
  - [`gaze quality`](reference/cli/quality.md) — Test quality assessment
  - [`gaze report`](reference/cli/report.md) — AI-powered quality reports
  - [`gaze self-check`](reference/cli/self-check.md) — Self-analysis
  - [`gaze docscan`](reference/cli/docscan.md) — Documentation scanner
  - [`gaze schema`](reference/cli/schema.md) — JSON Schema output
  - [`gaze init`](reference/cli/init.md) — OpenCode integration setup
- [Configuration Reference](reference/configuration.md) — `.gaze.yaml` keys, types, defaults, and CLI flag interaction
- [JSON Schema Reference](reference/json-schemas.md) — Schema references and annotated example output for JSON-format commands
- [Glossary](reference/glossary.md) — Canonical definitions for all domain-specific terms

### Guides

- [CI Integration](guides/ci-integration.md) — GitHub Actions workflow, coverage profile reuse, threshold enforcement, baseline comparison
- [AI Reports](guides/ai-reports.md) — Adapter setup for Claude, Gemini, Ollama, and OpenCode
- [OpenCode Integration](guides/opencode-integration.md) — `gaze init`, scaffolded files, and the `/gaze` command
- [Improving Scores](guides/improving-scores.md) — Fix strategies with before/after examples: decompose, add tests, add assertions

### Architecture

- [Architecture Overview](architecture/overview.md) — Package dependency graph, data flow, and the role of each internal package
- [Contributing Guide](architecture/contributing.md) — Dev environment, build/test commands, coding conventions, spec-first workflow
- [Extending Gaze](architecture/extending.md) — Adding side effect types, classification signals, output formats, and AI adapters

### Porting to Other Languages

- [Behavioral Contracts](porting/contracts.md) — Language-agnostic contracts a port must honor
- [Porting Requirements](porting/requirements.md) — Required vs optional capabilities for a conforming port
- [Taxonomy Reference](porting/taxonomy-reference.md) — All 37 effect types with tier assignments and scoring formulas
