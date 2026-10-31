---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-10-30T22:01:34Z
identity: test-quality-20261030T220134-jay-flowers
tier: draft
---

When unit-testing a hybrid classifier whose rules can independently fire (e.g. gaze's redactEnvValues: sensitivity by name marker OR by value-length threshold), isolate each rule by neutralizing the other in every case: name-marker cases use values below the length threshold, and threshold cases use names carrying no marker. Otherwise a case can pass via the unintended rule and mask a regression in the intended one — a broken KEY-marker matcher would still pass a KEY test whose 40-byte value independently trips the length threshold. In gaze's redact_test.go (internal/aireport), all four marker cases use sub-threshold values (configJSON at 20 bytes, token fixtures at 16-19 bytes) and both threshold cases use the markerless CI_RUNNER_ID name, with boundary values derived from the live constants (redactValueLengthThreshold, redactMinReplacementLength) so constant changes re-derive the test geometry rather than silently invalidating the rule isolation. Complements the existing self-verifying-straddle and env-planting learnings for the same suite.
