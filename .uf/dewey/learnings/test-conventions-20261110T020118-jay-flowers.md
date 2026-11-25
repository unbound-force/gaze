---
tag: test-conventions
author: jay-flowers
category: decision
created_at: 2026-11-10T02:01:18Z
identity: test-conventions-20261110T020118-jay-flowers
tier: draft
---

Gaze house pattern (confirmed assessing PR #326 feedback): tests pin the GAZE_COVERAGE_RUN env-var name literally (os.Getenv/t.Setenv guards) rather than referencing production's coverageRunEnv constant, even in package goprovider where the unexported constant is in scope. Rationale: the env name is documented user-facing contract (README.md, docs/reference/cli/{crap,report,self-check}.md, AGENTS.md idiom), so literal pinning makes an accidental constant-value change fail the guard-behavior tests instead of silently breaking documented workflows and users' pre-existing skip guards; sharing the constant would couple tests to an implementation detail (Constitution IV). DRY extraction (exporting goprovider.CoverageRunEnv or a shared skip helper) is deferred to a recorded trigger: a 6th skip-guard site. When assessing DRY findings (default.md CS-004) against one-line env-guard idioms distributed across test packages, weigh contract verification and the AGENTS.md-prescribed literal idiom before requiring extraction.
