---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:30:30Z
identity: issue-triage-20261101T183030-jay-flowers
tier: draft
---

Gate-scoping triage pattern (from gaze #295): lint/CI configuration is a protected gatekeeping value, so triage must distinguish authorized gate scoping from unauthorized gate weakening. A change request is governance-positive when it is human-initiated (member-authored issue, not an agent working around a failing check), narrows coverage only to generated/tool-owned/historical paths, explicitly preserves blocking enforcement for user-facing files, requires a documented per-path exemption rationale, and baselines remaining violations instead of erasing them. Precedent in gaze: .mega-linter.yml already excludes the byte-identical embedded scaffold asset mirrors (internal/scaffold/assets/) from MARKDOWN_MARKDOWNLINT with a documented mirror-sync rationale, so extending exclusions to .uf/dewey/ learning records and OpenSpec artifacts extends an established pattern rather than inventing one.
