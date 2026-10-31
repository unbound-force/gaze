---
tag: review-dispatch
author: jay-flowers
category: pattern
created_at: 2026-10-31T22:52:38Z
identity: review-dispatch-20261031T225238-jay-flowers
tier: draft
---

Advisory deferral under APPROVE WITH ADVISORIES (opsx/init-ci-scaffold, 2026-10-31): council advisories attached to an approving verdict are non-blocking by council semantics, and fixing them immediately is actively harmful when the code-review artifact pins the head SHA — the fix commit moves HEAD, breaks the artifact's head match, and invalidates the terminal review state right before the PR. The correct completion sequence was: record the 4 MEDIUM + 12 LOW advisories in the pipeline summary for follow-up work, leave the approving artifact (ecba6bf6) as the terminal review state, and let /uf.finale submit the PR with minimal-to-no changes between APPROVE and submission, exactly as the gaze repo's AGENTS.md requires. The sanctioned exception is doc-only bookkeeping that resolves advisory findings without a commit: recording the website-issue number next to its tasks.md checkbox and adding the AGENTS.md Recent Changes entry (both were themselves LOW advisories) can stay uncommitted in the working tree for the finale to sweep into its commit, so HEAD never moves before the PR.
