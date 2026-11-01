---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:28:36Z
identity: issue-triage-20261101T172836-jay-flowers
tier: draft
---

While triaging gaze issue #266 (t.Chdir convention-pack rule), a fullsend triage-agent comment (id 5714498227) analyzed the HTML-formatter topic (#260/#8/#265) — entirely unrelated to the issue's subject — indicating the automation posted its analysis to the wrong issue. When a triage/automation comment's subject matter does not match the issue body, treat the issue body as authoritative for the validity assessment, record the comment as unrelated noise in the reasoning, and do not let the mismatch push the verdict toward NEEDS-CLARIFICATION; flag the apparent misdirection for human cleanup rather than acting on the comment's content.
