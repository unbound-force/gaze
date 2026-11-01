---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:38:17Z
identity: issue-triage-20261101T183817-jay-flowers
tier: draft
---

Automated triage-bot comments can carry analysis for the wrong issue. On unbound-force/gaze#282 ("gaze report: honor --language alone for external analyzer dispatch"), the fullsend triage comment (id 5831893125) analyzed unrelated HTML report formatter work (#8/#260, TestQualityHTML_* test cases), declared the issue "unblocked" for that unrelated scope, and issued /fs-code next-steps guidance for it. When triaging, verify each comment's subject aligns with the issue title and body before treating it as evidence or following its next-steps guidance; anchor the assessment on the issue body plus independently corroborated repository context.
