---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:38:26Z
identity: issue-triage-20261101T183826-jay-flowers
tier: draft
---

Automated triage comments can be misrouted to the wrong issue. On unbound-force/gaze #282 (--language-only external analyzer dispatch for `gaze report`), the fullsend triage comment (id 5831893125) discussed the unrelated HTML report formatter workstream (#8 parent, #260 blocker) and proposed HTML-formatter test cases (TestQualityHTML_*, TestCrapHTML_*, TestReportHTML_*). When triaging, cross-check each comment's subject against the issue body before treating its content — especially "Proposed Test Case" blocks — as requirements or acceptance criteria; a subject-mismatched comment is noise to flag in the assessment, not scope to adopt.
