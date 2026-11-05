---
tag: spec-contract-fidelity
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:17:17Z
identity: spec-contract-fidelity-20261102T211717-jay-flowers
tier: draft
---

In spec-driven work, an extraction contract clause like 'lines that fail to unmarshal are skipped and counted' can be implemented as just the skip: a bare `continue` satisfies the skip while silently dropping the counting obligation. Gaze's opencode adapter (PR #314, internal/aireport/opencode_events.go) shipped parseEvents with a silent `continue` on json.Unmarshal failure — a mostly-corrupt `--format json` stream could yield a validated answer with zero degradation signal, hiding pinned-version drift (opencode-ai 1.19.0) and silently dropping corrupted error/tool_use events. When a PR introduces both a contract and its implementation, check every observability verb in the contract (counted, logged, surfaced, warned) against the code — these clauses are the easiest to drop silently because the happy path never exercises them.
