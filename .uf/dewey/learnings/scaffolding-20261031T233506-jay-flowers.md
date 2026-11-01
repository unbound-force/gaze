---
tag: scaffolding
author: jay-flowers
category: gotcha
created_at: 2026-10-31T23:35:06Z
identity: scaffolding-20261031T233506-jay-flowers
tier: draft
---

Console guidance emitted by a scaffolding tool into a foreign repository must reference documentation by absolute URL, not repo-relative paths: the consumer's working directory does not contain the tool's docs, so "see docs/guides/x.md" is unresolvable at the moment the user reads it. In gaze's init --ci next-steps section (2026-10-31 SRE review), two pointers referenced docs/guides/ci-integration.md while the workflow file the same command wrote used the full GitHub URL for the identical document — the artifact's URL style is the correct pattern for any output the consumer reads outside the tool's own repo. The dangling-path class also propagates into docs that quote sample console output verbatim, so the fix must update the code and the documented sample together.
