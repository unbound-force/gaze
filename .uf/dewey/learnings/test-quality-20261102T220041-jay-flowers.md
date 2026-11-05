---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T22:00:41Z
identity: test-quality-20261102T220041-jay-flowers
tier: draft
---

Deterministic-rendering helpers with defensive escaping branches (e.g. aireport.codeSpan's CommonMark space-padding for backtick-terminated content) need direct table-driven unit tests pinning exact output strings. Indirect coverage through template-level or golden tests with realistic fixtures leaves the defensive branches unreachable: the suite stays green while the branch is silently deletable, so a later refactor can violate the normative byte contract with no failing test. When a spec contract normatively specifies edge-case behavior (markdown-report Rule 8's wider backtick span), enumerate those edge cases as explicit table-driven cases against the helper — leading backtick, trailing backtick, internal run, all-backticks — asserting the exact emitted span, in addition to any end-to-end fixtures.
