---
tag: aireport
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:17:27Z
identity: aireport-20261102T211727-jay-flowers
tier: draft
---

Diagnostic clauses in parsing contracts are the clauses that silently go unimplemented. Gaze's spec 040 opencode-adapter contract (specs/040-deterministic-ai-report-rendering/contracts/opencode-adapter.md) says non-JSON stream lines are "skipped and counted", but parseEvents (internal/aireport/opencode_events.go) shipped with a bare `continue` — no counter, no surfaced signal — and the suite stayed green because the fixtures covered only all-valid streams and the all-invalid not_json.txt. A missing diagnostic has no failing test unless one is written for it: lock every contract clause that mandates observability (counts, surfaced degradation warnings) with a mixed valid+invalid input fixture plus an assertion on the exact surfaced value, since happy-path and total-failure tests both pass without it.
