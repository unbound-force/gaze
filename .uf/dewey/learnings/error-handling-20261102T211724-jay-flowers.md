---
tag: error-handling
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:17:24Z
identity: error-handling-20261102T211724-jay-flowers
tier: draft
---

Tolerant parsing without a skip counter converts silent corruption into a validated result. In gaze's opencode event extraction (spec 040, internal/aireport/opencode_events.go), parseEvents skips non-JSON lines per its contract, but the contract's "and counted" clause was never implemented: a mostly-corrupt stream yielding one decodable answer passes the fail-closed validator with zero degradation signal, and the all-junk path errors with "no JSON events" without saying how many lines were junk. Pattern: when a parser is deliberately tolerant of malformed input, ship the skip counter and its surfacing in the same increment as the tolerance itself — count at the skip site, then surface on both the zero-result failure (error suffix) and the success path (stderr diagnostic at the adapter I/O seam). Where sentinel errors are matched with errors.Is (adapter_opencode.go matches errNoFinalText), append counts via %w wrapping or at the adapter seam so sentinel comparison keeps working.
