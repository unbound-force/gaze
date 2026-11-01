---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:49:20Z
identity: issue-triage-20261101T174920-jay-flowers
tier: draft
---

When triaging new metric/gate feature requests for gaze (e.g., issue #271 branch coverage), evaluate whether the existing -coverprofile data path (LineCoverageProvider / the CI pattern documented in disk-website/content/docs/getting-started/tester.md) can derive the metric before accepting custom AST instrumentation proposals. Instrumentation-based approaches (source rewriting to track execution) conflict with the constitution's Minimal Assumptions principle — no source annotation or restructuring — and force acceptance tests into the slow e2e suite (subprocess go test execution, testing.Short() guarded), while cross-referencing coverprofile statement data against AST-enumerated branch arms keeps the analysis unit-testable against fixture profiles and reuses data the tool already consumes. The same testability asymmetry applies to any future execution-derived metric proposal.
