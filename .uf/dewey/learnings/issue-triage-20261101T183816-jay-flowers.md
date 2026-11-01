---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:38:16Z
identity: issue-triage-20261101T183816-jay-flowers
tier: draft
---

Automated triage-bot comments require subject-matter verification before influencing a triage verdict. On unbound-force/gaze issue #282 (gaze report --language-only external analyzer dispatch), the fullsend triage-agent comment discussed HTML report formatter work — issue #260's foundation, parent #8, and HTML formatter test cases — content belonging to a different issue. Triage assessments should rest on the issue body and caller-provided repository context; when a comment's subject diverges from the issue, note the mismatch explicitly and exclude the comment's framing (unblocked-status claims, proposed test cases, next-step guidance) from the verdict rather than importing it.
