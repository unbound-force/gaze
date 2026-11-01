---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:28:33Z
identity: issue-triage-20261101T172833-jay-flowers
tier: draft
---

Automated triage-agent comments can land on the wrong issue and must be verified against the host issue's subject before being treated as triage context. On unbound-force/gaze issue #266 (a Go convention-pack t.Chdir rule proposal), a fullsend triage comment discussed entirely unrelated HTML-formatter work (#260/#8/#265) and instructed removing a `blocked` label the issue never carried. Reliable mispost signals: the comment references issue numbers and labels foreign to the host issue, and its recommended next steps do not match the issue's actual scope. When detected, disregard the comment for triage purposes — it introduces false dependencies — and flag it for maintainer cleanup rather than escalating the issue to NEEDS-CLARIFICATION, since the issue body itself remains unambiguous.
