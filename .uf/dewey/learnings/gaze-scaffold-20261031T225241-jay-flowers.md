---
tag: gaze-scaffold
author: jay-flowers
category: gotcha
created_at: 2026-10-31T22:52:41Z
identity: gaze-scaffold-20261031T225241-jay-flowers
tier: draft
---

Three gaze-scaffold gotchas from the init-ci-scaffold change (issue #306, `gaze init --ci`). (1) Any new embedded asset subtree that intentionally has NO .opencode/ mirror — like internal/scaffold/assets/ci/ for the CI workflow template — must be explicitly excluded from scaffold.Run's asset walk and from assetPaths(); TestEmbeddedAssetsMatchSource fails closed on embedded assets without a .opencode/ counterpart, and the exclusion keeps the 9-file mirror manifest unchanged (this is recorded in AGENTS.md File Sync Invariants). (2) Extending scaffold.Run with the --ci write path pushed its GazeCRAPload to the dogfood gate ceiling on gaze's own CI (max-gaze-crapload 6, zero headroom); the gate-preserving fix was decomposition — extracting the CI write path into writeCIWorkflow/appendCIWorkflowResult plus focused helpers — never touching the gate itself (AGENTS.md gatekeeping-integrity rule). One delegated fix-loop iteration closed it. (3) A locally `go build`-produced gaze binary reports version "dev", and dev builds fail closed on --ci version validation by design — smoke-testing the scaffold in a temp dir requires `--gaze-version <valid tag >= v2.3.0>` or an ldflags-injected version; the fail-closed error itself is the designed behavior to demonstrate.
