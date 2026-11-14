---
tag: assertion-depth
author: jay-flowers
category: pattern
created_at: 2026-11-10T00:09:27Z
identity: assertion-depth-20261110T000927-jay-flowers
tier: draft
---

Scenario-variant tests must mirror their siblings' assertion depth, not just their arrange/act structure. When a new test varies one condition over an existing code path (e.g., GoLineCoverageProvider.Coverage with a pre-generated profile while GAZE_COVERAGE_RUN=1), copying the sibling's fixture and call shape but stopping at err==nil + len(results)>0 violates go.md TC-009 [MUST] and default.md TC-003 [MUST], and lets malformed per-result data pass. In internal/provider/goprovider/goprovider_test.go the established depth for pre-generated-profile tests is the per-result loop asserting non-empty File and Percentage within [0,100] — TestCoverage_ShortWithCoverProfile carries an explicit "same assertions as the non-Short test" comment establishing the convention — while TestCoverage_NestedRunUsesPreGeneratedProfile (PR #326) initially omitted it and was flagged by review. The fix is to copy the sibling loop verbatim after the length check.
