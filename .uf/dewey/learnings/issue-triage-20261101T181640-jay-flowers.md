---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:16:40Z
identity: issue-triage-20261101T181640-jay-flowers
tier: draft
---

Cleanup issues that enumerate specific files drift stale when parallel remediation paths fix some of those files before the issue is worked. gaze issue #244 (filed 2026-09-01) listed 4 files holding stale unprefixed /review-council references; by triage on 2026-11-01, 3 of the 4 were already clean — a uf scaffold refresh had corrected .opencode/skills/review-context/SKILL.md:19, the divisor-gaze-testing reference sweep had removed all review-council mentions from docs/reference/cli/init.md, and .opencode/agents/cobalt-crush-dev.md:127 already read /uf.review-council — leaving only .specify/memory/constitution.md:10,133. The issue stayed correctly actionable because its acceptance criterion was a mechanically verifiable project-wide grep ("zero stale matches in actionable docs") rather than the file checklist; the grep, not the enumerated list, defined remaining scope. Pattern: phrase cleanup-issue acceptance criteria as grep-verifiable invariants over the artifact set, and treat enumerated file lists as hints to re-verify at implementation time, not scope contracts. Caveat: scope such greps to actionable docs — a bare project-wide run for a renamed command also hits legitimate same-name paths (e.g., .uf/dewey/sources.yaml's `path: "../review-council"` disk-source entry pointing at the sibling unbound-force/review-council repository) and completed historical change records (specs/006 tasks.md, archived-track openspec changes).
