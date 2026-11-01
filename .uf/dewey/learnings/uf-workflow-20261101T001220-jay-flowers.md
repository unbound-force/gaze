---
tag: uf-workflow
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:20Z
identity: uf-workflow-20261101T001220-jay-flowers
tier: draft
---

Feature PRs produced by the uf pipeline can carry auxiliary bookkeeping beyond the change's recorded scope: the finale commit sweeps in dewey learnings under .uf/dewey/learnings/, the markdownlint exclusions needed to keep those regenerated files green (.mega-linter.yml), and post-squash gitleaks .gitleaksignore SHA re-points — none of which appear in the openspec proposal's Impact list, tasks.md, or the AGENTS.md Recent Changes entry unless someone adds them. Review classification that worked on opsx/init-ci-scaffold (PR #308): treat the bundle as a documentation-gap finding (MEDIUM, record it in the PR description or Recent Changes entry — doc-only, no HEAD movement needed post-approval) rather than gate tampering, PROVIDED the exclusions govern only machine-generated paths and the gitleaks re-point re-asserts pre-existing suppressions of the same lines (verify the flagged lines contain the known fake test secrets). Distinguishing signal: the feature code passes lint independently of the exclusion. Prevention: linter-configuration modifications should get a tasks.md entry or separate trivial-change record so the spec-first trail matches the PR footprint.
