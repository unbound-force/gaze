---
tag: uf-unleash
author: jay-flowers
category: gotcha
created_at: 2026-11-16T18:38:18Z
identity: uf-unleash-20261116T183818-jay-flowers
tier: draft
as_of: "gaze@745e597410ff162c7a3ae03ddce988e471b24e3f"
---

In the opsx/gaze-command-tool-owned /uf.run-pipeline-plan run (2026-11-16, gaze repo), the first /uf.unleash delegation died on its first model call with a 401 "invalid API key" from api.enmaas.devshift.net — the child's model-matrix resolution (classifier cascade → meta-prompt pack selection) picked an enmaas identity whose API key was invalid, while the host session's model worked fine. Two recoveries: (1) dispatch-based tools were unaffected — dispatch_review_runs reviewer children ran the pinned Muse-Spark model and authenticated fine; (2) uf_delegate_command calls succeed again when passed an explicit model pin (enmaas/rits/zai-org/glm-5-3, the host model identity) — the explicit model wins the precedence chain outright, skipping matrix resolution entirely. Equally important: the 401-killed child had already completed tier routing, opsx-propose artifacts, and implement phases 1-4 green before dying mid-pipeline — never infer "no work happened" from a DELEGATE_FAILED envelope; re-anchor on uf_unleash_branch_gate + get_progress + uf_unleash_detect_resume, where disk state is authoritative. The child had also executed implement phases BEFORE spec review (out of pipeline order); uf_unleash_detect_resume correctly reported resume_from=spec_review by reconciling checkpoint claims against dispatch artifacts, and the specs-mode council handled the diff carrying both artifacts and already-implemented code without issue.
