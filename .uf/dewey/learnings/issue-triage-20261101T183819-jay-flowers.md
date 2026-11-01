---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:38:19Z
identity: issue-triage-20261101T183819-jay-flowers
tier: draft
---

Automated triage-bot comments can land on the wrong issue. On unbound-force/gaze #282 (gaze report --language dispatch parity), the fullsend triage-agent comment (5831893125) discusses HTML formatter work — resolving blocker #260, referencing parent #8, and proposing TestQualityHTML/TestCrapHTML/TestReportHTML test cases — a subsystem unrelated to external-analyzer dispatch, with no real dependency between them. When triaging, verify each comment's subject matches the issue body before treating it as blocker or scope context; a mismatched bot comment is noise that does not change the issue's own validity (which rests on the body's code-verifiable claims), but the mismatch should be flagged so consolidators and future readers don't mistake the comment for issue context or treat a foreign blocker as resolved.
