---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-10T01:28:53Z
identity: test-quality-20261110T012853-jay-flowers
tier: draft
---

To unit-test both directions of a subprocess-spawning guard hermetically, clear PATH (t.Setenv("PATH", "")) so any accidental spawn fails instantly with a distinguishable exec error instead of running the real toolchain. The refusal direction asserts the sentinel error — an accidental spawn produces a different error, so errors.Is fails loudly — and the proceed direction asserts a spawn failure rather than the sentinel. This converts a potentially slow or recursive subprocess into fast, assertable unit tests with no real execution and no network/toolchain dependency.
