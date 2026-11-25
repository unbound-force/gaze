---
tag: test-infrastructure
author: jay-flowers
category: pattern
created_at: 2026-11-02T19:40:26Z
identity: test-infrastructure-20261102T194026-jay-flowers
tier: draft
---

Hermetic regression testing for AI-dependent report pipelines (gaze spec 040, internal/aireport): (1) a ~150-line fake CLI binary (testdata/fake_opencode) validates the full invocation contract — subcommand order, --agent/--format values, agent-file frontmatter (mode: primary, deny-all permissions), 0600 file mode — and exits 2 for invocation errors so tests can distinguish contract violations from simulated CLI failures; (2) recorded JSONL event streams selected via FAKE_OPENCODE_STREAM replay real adapter output shapes, including incident replays (narration-instead-of-answer, non-JSON output, over-long summaries, predominantly-Chinese answers); (3) FAKE_OPENCODE_CAPTURE_AGENT/_ARGS env hooks let tests assert exactly which prompt and argv reached the subprocess — sentinel-based: a hostile string planted in the project's real interactive prompt must NOT appear in the captured report-mode prompt; (4) deterministic goldens stay honest through layered drift guards: golden name table ↔ on-disk orphan check ↔ consuming case table, a compile-time struct mirror (var _ = ReportSummary(mirror{})) for fixture-only summary blocks, a cross-package test that greps the duplicated goldenDir constant declaration, a doc-drift test pinning the published rubric table to the code's bound arrays, and a docs example pinned byte-for-byte to the renderer golden; (5) CLI-level golden tests reuse the package's testdata directory instead of duplicating fixtures. Result: end-to-end coverage of the AI adapter path with zero network access and no real model.
