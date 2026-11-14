---
tag: aireport-escapers
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:02:10Z
identity: aireport-escapers-20261102T210210-jay-flowers
tier: draft
---

In gaze's aireport renderer (spec 040), the three markdown escapers have asymmetric pair-level test coverage that misleads "untested flattening" feedback: escapeCell is fully pinned (TestEscapeCell_Markup, including "a\r\nb\nc\rd" → "a b c d"), proseEscaper is fully pinned (TestSanitizeProse_Markup "line breaks" case: "one\ntwo\r\nthree\rfour" → "one two three four"), but headerEscaper has only [, ], <, > covered via TestRenderMarkdown_MetadataEscaped — its backslash, backtick, asterisk, underscore, pipe, and all three newline pairs are unpinned. Gotcha when assessing coverage of the flattening path: writeCallout applies boundedFirstLine BEFORE sanitizeProse, so multi-line callout details never reach proseEscaper's flattening; the live newline path for proseEscaper is the AI summary/recommendations slots (writeAISummarySlot / writeAIRecommendationsSlot, no boundedFirstLine).
