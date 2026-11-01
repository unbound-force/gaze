---
tag: content-pipeline
author: jay-flowers
category: pattern
created_at: 2026-10-31T23:34:56Z
identity: content-pipeline-20261031T233456-jay-flowers
tier: draft
---

Gaze curator code-mode triage specifics (init-ci-scaffold review, 2026-10-31, branch opsx/init-ci-scaffold): (1) The blog significance threshold "new CLI command or subcommand" reads to include significant new flags on existing commands — org precedent #150 (blog: gaze baseline comparison, filed for the `--baseline` flag on `gaze crap`) with tutorial companion #151, plus #51 (blog: uf init now scaffolds for Cursor). A flag that becomes the documented "preferred adoption path" for a flagship capability is blog-significant. (2) docs/architecture/ is a recurring blind spot of gaze's Documentation Validation Gate (enumerated list: README, AGENTS.md, GoDoc, spec artifacts): the PR #305 review caught docs/architecture/extending.md stale, and this review caught the Package Map row for internal/scaffold (purpose omitted CI workflow generation; "None (uses embed.FS)" dependency claim falsified by the x/mod direct-dep promotion) — grep docs/architecture/ in every content sweep, not just the files tasks 4.x enumerate. (3) When verifying the Website Documentation Gate for gaze changes, the website issues carry the "docs:" title prefix but no label (precedent #345, #352) — search by title keywords rather than the `docs` label, and confirm scope with gh issue view (body must state what changed, why it matters, pages needing updates).
