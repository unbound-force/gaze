---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-11-01T16:43:48Z
identity: structural-entropy-20261101T164348-jay-flowers
tier: draft
---

Spot-checking `vibe-check analyze --output` graph JSONs with the ripgrep-backed grep tool fails with "Ripgrep JSON record exceeded 65536 bytes": the schemaVersion-1.3 graphs are pretty-printed overall (511 lines for gaze's 22 packages) but contain at least one record longer than ripgrep's 65536-byte line limit, so every grep against the file errors regardless of pattern. Use the read tool with a small line limit for spot checks (top-of-file fields — schemaVersion, language, and per-module ca/ce/instability/abstractness/distance/lcom/zone — sit on short lines), and treat `vibe-check diff` as the authoritative consumer and partial-build/unreliability detector: exit-0 analyze runs plus fully populated non-zeroed package metrics plus a diff reporting "Verdict: APPROVE, Reasons: (none)" is sufficient evidence of a reliable measurement without grepping the raw graphs.
