---
tag: ci-lint-hygiene
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:30:32Z
identity: ci-lint-hygiene-20261101T183032-jay-flowers
tier: draft
---

CI lint noise from generated and tool-owned artifacts degrades pipeline signal before it degrades pipeline correctness. In unbound-force/gaze, MegaLinter reported 1,216 non-blocking markdownlint violations across 281 files, nearly all from non-user-owned paths (.uf/dewey/ learning records, openspec/ changes and archives, embedded scaffold mirrors), hiding actionable Markdown regressions in user-facing docs behind a persistent warning. The remediation pattern (gaze issue #295, extending the fix-duplicate-scaffold-marker precedent in .mega-linter.yml): classify Markdown paths by ownership (user-owned / generated / historical / tool-owned), exclude only non-user-owned paths from the linter with a documented per-path rationale, keep user-facing files fully linted and blocking for new violations, and establish a post-exclusion baseline so remaining violations stay actionable and tracked. Special constraint for byte-identity mirror pairs (source + embedded copy, guarded by TestEmbeddedAssetsMatchSource per learning/test-quality-20261029T173027-jay-flowers): lint exactly both sides or neither, never one — linter auto-fix formatting applied to only one side of the pair breaks byte-identity sync tests, which was the root cause of the earlier scaffold-marker drift.
