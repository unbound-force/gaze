---
tag: review-dispatch
author: jay-flowers
category: gotcha
created_at: 2026-11-25T03:28:05Z
identity: review-dispatch-20261125T032805-jay-flowers
tier: draft
as_of: "gaze@73bb94085f82edd0182836a17ed2e0669856cc02"
evidence:
  - "session:7e4660d1-156f-482e-84ff-dfce158e8a5b"
---

The opencode glob tool returns "No files found" for patterns rooted in a dot-directory — hit during /uf.unleash Step 6 reviewer discovery on opsx/report-language-dispatch (2026-11-25), where glob '.opencode/agents/divisor-*.md' reported zero reviewers while `ls -1 .opencode/agents/` showed eleven divisor-*.md files, including divisor-gaze-testing.md whose presence is enforced by the repo's embedded-asset byte-identity test (TestEmbeddedAssetsMatchSource). Applying the fail-closed zero-reviewer rule off the glob result would have stopped a healthy pipeline. For reviewer enumeration and any dot-directory discovery in this environment, confirm with a direct bash ls before treating a zero count as real.
