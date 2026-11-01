---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-11-01T15:59:31Z
identity: content-pipeline-20261101T155931-jay-flowers
tier: draft
---

gaze repo documentation contract for CI/CD-only changes: changes confined to .github/** (plus spec artifacts under openspec/ or specs/) are categorically exempt from the Website Documentation Gate, blog, and tutorial checks per AGENTS.md. The complete in-repo documentation obligation for such changes is exactly two surfaces: (a) the AGENTS.md CI/CD section workflow description and (b) an AGENTS.md Recent Changes entry in the established house style. The repo has no root CHANGELOG.md (release notes are GoReleaser-generated from conventional commits), so a missing CHANGELOG.md entry is never a valid finding. docs/guides/ci-integration.md documents the consumer-facing reusable-workflow contract and only needs updating when that contract changes — not when gaze's own release pipeline gains internal gates (e.g., validate-workflow-floor, validate-module-major). Curator reviews of CI/CD-only changes should verify the two AGENTS.md surfaces and stop; flagging website/blog/tutorial gaps for them is a false positive.
