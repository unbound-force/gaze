---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-15T01:12:12Z
identity: test-quality-20261115T011212-jay-flowers
tier: draft
---

When every attacker-influenced value funnels through a fixed set of escape sinks, pin each sink with a byte-exact fixture matrix (sink × input family, exact expected output bytes per TC-009) plus render-level wiring tests for one representative position per sink. The matrix is total by construction (the choke-point argument enumerates every value path), and the wiring tests catch call-site regressions the matrix cannot see. Gaze's markdown-autolink-entity-bypass change is the reference shape: TestNeutralizeEntityEncodedAutolinks drives 25 hostile fixtures (every trigger character × decimal/hex/named/hybrid encoding families) through all three sinks with exact want strings, while TestRenderMarkdown_EntityEncodedAutolinksDefanged verifies the metadata header, callout, gap-cell, and AI-slot wiring end-to-end.
