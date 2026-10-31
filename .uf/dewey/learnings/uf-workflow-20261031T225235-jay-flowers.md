---
tag: uf-workflow
author: jay-flowers
category: pattern
created_at: 2026-10-31T22:52:35Z
identity: uf-workflow-20261031T225235-jay-flowers
tier: draft
---

Gate completion by artifact adoption (opsx/init-ci-scaffold branch, 2026-10-31): when a /uf.unleash OpenSpec pipeline resumes with its review-council gate task (e.g. task 5.6 "Review Council Gate") still unchecked but a consolidated review-dispatch artifact already exists under .uf/artifacts/dispatch/ whose input_context.head_sha EXACTLY equals the current HEAD and whose verdict is approving (APPROVE or APPROVE WITH ADVISORIES), the gate is satisfied by reading the artifact and marking the checkbox — never by re-dispatching the council. Strict head-SHA equality is the trust anchor: it proves the reviewers judged the exact committed bytes, so a re-run only burns review budget, and any new commit before the PR would invalidate the match and force a full re-review. On this branch the artifact was ecba6bf6-8491-4cc4-a33b-340efc9f3ddc (APPROVE WITH ADVISORIES, 8/8 runs, 0 HIGH/CRITICAL, head 71d1065) and adopting it completed the pipeline with zero wasted review spend. General rule: before running any review gate, check .uf/artifacts/dispatch/ for an approving artifact pinned to the current HEAD; also note that uf_unleash_detect_resume's spec_review derivation can false-negative on unstaged tasks.md checkbox edits (git-exec trimming corrupts porcelain parsing — see learning uf-workflow-20261031T185049), so checkpoint state plus artifact head-SHA equality is the more reliable completion evidence.
