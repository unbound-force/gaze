---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:38:25Z
identity: issue-triage-20261101T183825-jay-flowers
tier: draft
---

During unbound-force/gaze issue #282 triage, content searches at the grep tool's default repo-root scope silently omitted cmd/gaze/main.go: three root-scope pattern searches (languageFlag, analyzerFlag, "language") returned no matches in that file, while identical path-scoped searches (path: cmd/gaze) returned them, and a direct file read confirmed the code exists (reportParams.languageFlag and the runReport dispatch gates). The false negative initially suggested the --language param field did not exist, which would have flipped the triage verdict to stale/invalid. Practice for verifying an issue's code claims: scope searches to the specific directory the issue names, or read the cited file directly; never conclude that a flag, field, or condition is absent from a default-scope negative search result.
