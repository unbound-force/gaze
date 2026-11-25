---
tag: ci-reliability
author: jay-flowers
category: gotcha
created_at: 2026-11-12T20:05:29Z
identity: ci-reliability-20261112T200529-jay-flowers
tier: draft
---

A scheduled canary that (a) deliberately runs with cold caches, (b) has zero retries on its network-dependent steps, and (c) reports failures only through the default scheduled-workflow email is engineered for alert fatigue: its false-positive rate is structurally higher than the consumer path it mirrors, and the sole recipient will start ignoring the signal before it ever catches real breakage. Mitigation that preserves fidelity and gate integrity: bounded retries (3 attempts, short backoff) on the network steps only, leaving local verification unretried; the final attempt stays fatal with the existing ::error:: message, so persistent breakage still surfaces (satisfies the MUST-fail spec contract and the AGENTS.md rule against retries masking broken install paths). Prefer an inline bash retry helper over a retry action to avoid new SHA-pinned action surface.
