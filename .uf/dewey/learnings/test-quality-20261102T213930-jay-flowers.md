---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T21:39:30Z
identity: test-quality-20261102T213930-jay-flowers
tier: draft
---

Coverage-claim audits must span both recorded fixtures and synthetic in-memory cases before calling a behavior untested — "no fixture does X" is not "X is untested". On gaze PR #314 (address-feedback item 4), the reviewer's claim "no fixture exercises \r\n endings" was literally true of testdata/opencode_streams/ but behaviorally false: TestExtractFinalText_Synthetic's "crlf framing is tolerated" case builds a \r\n-terminated stream in memory and asserts the answer survives, so a CRLF regression would fail the suite. The adjacent claim (no fixture mixes valid events with junk lines) did survive the audit — the feedback was accepted on that core — but implementing the \r\n half as a gap would have duplicated existing coverage. Rule: when a reviewer scopes a coverage claim to fixtures, grep the *_test.go files for the same behavior constructed synthetically before accepting the implied gap; recorded-fixture coverage and synthetic-case coverage are two halves of one audit, and claims anchored on only one half can be right about fixtures and wrong about protection.
