---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:16:37Z
identity: issue-triage-20261101T181637-jay-flowers
tier: draft
---

Cleanup issues that enumerate file lists rot between filing and implementation. unbound-force/gaze #244 (stale /review-council refs after the uf. namespace migration) listed four files to update; at triage time only the constitution (lines 10/133) still carried the stale form — .opencode/skills/review-context/SKILL.md:19, .opencode/agents/cobalt-crush-dev.md:127, and all of docs/ had already been remediated by later work. The durable contract in such issues is the grep-based acceptance criterion, not the file list: re-verify every enumerated claim against the working tree before implementing. Command-rename sweep pattern-design gotchas: require the leading slash so the still-legitimate slash-free workflow-id namespace (command: "review-council" enums in dispatch schemas, checkpoint step ids) does not false-positive; expect path-form matches (../review-council in .uf/dewey/sources.yaml, command/review-council.md in immutable spec records) to surface as a separate class needing current-vs-historical judgment rather than a command-reference fix; and governance-doc fixes must follow the target document's own amendment protocol (constitution PATCH bump for name-only refinements, since gate semantics are unchanged).
