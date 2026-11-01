---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:49:19Z
identity: issue-triage-20261101T174919-jay-flowers
tier: draft
---

Qualitative performance acceptance criteria ("performance impact benchmarked and acceptable", "must not significantly slow test execution") are unverifiable as written — they defer an operational decision to merge-time judgment. When triaging such issues, the issue can still be VALID because the requested behavior is clear, but the unquantified budget should be flagged for the spec clarify phase, where it must become a concrete ceiling (e.g., "<10% test-runtime overhead") or a benchmark-based regression guard in CI. Without a number, "acceptable" cannot be enforced and degrades to reviewer discretion. Discovered while triaging gaze #271 (branch coverage CI gate), whose only non-verifiable acceptance criterion is its performance budget.
