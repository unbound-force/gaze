---
tag: markdownlint
author: jay-flowers
category: gotcha
created_at: 2026-11-16T03:56:29Z
identity: markdownlint-20261116T035629-jay-flowers
tier: draft
as_of: "gaze@6d77d4d0f15cefa35fabf8d0de5eae2b5bffe5f1"
evidence:
  - "docs/concepts/scoring.md"
  - "docs/guides/doc-annotations.md"
  - "openspec/changes/reduce-markdownlint-noise-from-generated/tasks.md"
---

markdownlint --fix (markdownlint-cli2 v0.24.1) can semantically damage documentation while making it lint-clean. Two concrete damages from a 23-finding scoped auto-fix pass: (1) bare multiplication asterisks in prose (a*b dimensions) get escaped as literal \* or re-associated with adjacent words as emphasis markers, corrupting math notation in docs/concepts/scoring.md; (2) double-backtick code spans with inner spaces get their spacing mangled in docs/guides/doc-annotations.md, breaking intended rendering. Rule: after any bulk markdownlint auto-fix pass, hand-review the full diff for semantic damage around math operators, emphasis-adjacent words, and code spans — do not trust green lint as proof of meaning preservation. Auto-fix passes must also stay scoped to user-owned files: in repos with embedded byte-identical mirrors (e.g. internal/scaffold/assets/ mirroring .opencode/), a repo-wide --fix breaks TestEmbeddedAssetsMatchSource.
