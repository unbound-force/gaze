---
tag: test-conventions
author: jay-flowers
category: decision
created_at: 2026-11-10T02:01:16Z
identity: test-conventions-20261110T020116-jay-flowers
tier: draft
---

Gaze guard-test idiom (PR #326 review): tests that gate on documented environment contracts (GAZE_COVERAGE_RUN; analogously testing.Short()) deliberately repeat the literal env name instead of importing the production constant (goprovider's unexported coverageRunEnv). The literal independently pins the documented contract — AGENTS.md's Test Suites section prescribes the exact idiom verbatim — so a production constant-value change fails the tests rather than silently propagating (Constitution IV: test contracts verify observable behavior, not implementation details). default.md CS-004 (DRY, >2 locations) does not override this for two-line idiomatic guard snippets, matching the un-extracted testing.Short() precedent. Recorded extraction threshold: at a sixth skip-guard site, export goprovider.CoverageRunEnv or extract a shared skip helper and update the AGENTS.md + docs/architecture/contributing.md idiom text in the same change (divisor-architect LOW finding, declined by author, PR #326).
