---
tag: test-coverage
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:22:51Z
identity: test-coverage-20261102T222251-jay-flowers
tier: draft
---

When verifying a "no test covers X" claim in a Go table-driven suite, audit test-helper defaults and adjacent exact-count assertions before accepting it. In gaze PR #314 (feedback item on internal/aireport/assess.go), the triage's "verified" fact — "NO test exercises classification-section-present + classify-error-nil + all-tallies-zero" — was wrong twice over: assessPayload's variadic `sections ...Section` parameter defaults to ALL sections (including SectionClassification) when omitted, so every test passing a bare ReportSummary{} already ran zero-tally classification-present payloads; and TestAssess_FixStrategyTemplates' assertion `len(a.Recommendations) == 1` incidentally pinned the zero-tally no-recommendation boundary, so the feared "regression would pass" was false. The real gap was elsewhere (the nonzero tie arm, ambiguous == decided, which a single `<=` -> `<` mutation breaks undetected). Rule of thumb: "no dedicated test" is not "no protection" — enumerate helper defaults, zero-value structs, and exact-count assertions in neighboring tests before signing a coverage-gap claim; and still prefer dedicated boundary cases, because incidental pins vanish silently when fixtures drift.
