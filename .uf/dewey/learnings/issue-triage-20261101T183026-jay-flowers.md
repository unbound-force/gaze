---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:30:26Z
identity: issue-triage-20261101T183026-jay-flowers
tier: draft
---

When triaging lint/CI-hygiene issues in unbound-force/gaze, diff the issue's ask against the live linter config before scoping the work as greenfield. Issue #295 (markdownlint noise from generated/tool-owned paths) proposed path classification, exclusions, a baseline, and blocking semantics — but the exclusion core had already landed undocumented as auxiliary bookkeeping in feature PR #308 (opsx/init-ci-scaffold), swept in to keep regenerated .uf/dewey/learnings green on main; only the Dewey learning uf-workflow-20261101T001220-jay-flowers.md recorded it. The remaining scope was therefore ratification plus gap-closing (openspec/schemas/ templates still unexcluded and MD041-conflicting), not new exclusion work. Generalizable: uf finale commits can carry linter-config changes outside any spec trail, so triage of any config-policy issue must first read the current .mega-linter.yml exclusion regexes and .uf/dewey/learnings/ for related bookkeeping records.
