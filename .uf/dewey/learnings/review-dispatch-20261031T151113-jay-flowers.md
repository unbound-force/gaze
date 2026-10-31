---
tag: review-dispatch
author: jay-flowers
category: gotcha
created_at: 2026-10-31T15:11:13Z
identity: review-dispatch-20261031T151113-jay-flowers
tier: draft
---

review-dispatch plugin bug (reproduced 2026-10-31 on dispatch d276f946, previously seen on fb721404): when a plan entry is a host fallback (source "host", tier null, model null — e.g. divisor-entropy via host-fallback), executeDispatchRun defaults the null tier to "standard", resolves the review-matrix standard profile, and persists the run as a direct-model run (requested_model/provider/model_id set, resolved_parent_* null). consolidate_dispatch then fails closed with four provenance errors ("host run must retain null requested model provenance", "model provenance does not match its plan entry", "model_mismatch does not match reported provenance", "successful host run requires resolved parent model and variant"). The run itself executes correctly with the host model. Recovery that unblocked finalization: back up the run file, then repair only the six provenance fields in $TMPDIR/opencode/dispatch-<correlation_id>/run-<agent>.json to the host-run contract (requested_model/provider/model_id/variant → null; resolved_parent_model → the host model slug e.g. "enmaas/rits/zai-org/glm-5-3"; resolved_parent_variant → "default"), leaving findings/proposals/text/usage/status untouched, then re-run consolidate_dispatch with provenance. A successfully finalized precedent artifact confirmed the accepted shape (requested null + resolved_parent populated + reported_model matching). Root fix belongs in the plugin: executeDispatchRun should route host-source null-tier entries through resolveHostModel instead of the standard-tier default.
