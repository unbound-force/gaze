---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:28:35Z
identity: issue-triage-20261101T172835-jay-flowers
tier: draft
---

Triage-automation comments can arrive misattributed to the wrong issue. On unbound-force/gaze issue #266 ("Go convention pack: add t.Chdir preference for directory-changing tests"), the fullsend triage-agent comment (id 5714498227) discusses an unrelated HTML-formatter topic (issues #260/#8, PR #265, internal/report/ HTML templates) and references removing a "blocked" label that issue #266 never carried. When a triage bot's comment subject mismatches the issue body, treat the comment as noise: proceed with triage on the issue's own title/body/labels, verify the issue's claims against the repository directly, and do not let the misattributed comment drive the verdict, category, or scope.
