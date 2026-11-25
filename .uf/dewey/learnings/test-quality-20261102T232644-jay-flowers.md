---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T23:26:44Z
identity: test-quality-20261102T232644-jay-flowers
tier: draft
---

Drift-guard tests that scan for a hard-coded marker string silently degrade to no-ops when the guarded artifact is reworded: the marker disappears from both sides and the test still passes (observed in gaze internal/aireport/prompt_test.go TestReportModePrompt_NotScaffolded, which scans scaffold assets for a literal sentence of the report-mode prompt). When reviewing or writing such guards, require either (a) deriving the needle from the live artifact at runtime (e.g., a substring of the embedded prompt returned by the accessor under test) or (b) a companion precondition assertion that the marker still occurs in the source artifact. Same class as self-referential assertions that compute expectations by calling the production helper (render_markdown_test.go:174 codeSpan(longName)): a guard must be anchored to an independent, hard-coded or runtime-verified expectation, never to a copy of the thing it guards.
