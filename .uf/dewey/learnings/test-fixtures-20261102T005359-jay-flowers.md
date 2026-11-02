---
tag: test-fixtures
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:53:59Z
identity: test-fixtures-20261102T005359-jay-flowers
tier: draft
---

Table-test fixture surgery via strings.Replace: when a table case needs a mutated copy of a shared fixture (e.g., a .uf review-matrix YAML with one advisor list removed or replaced by a non-sequence mapping), derive it once into a package-level var so the staged input and the byte-identity expectation are the same object. Gotcha: strings.Replace silently no-ops when the search literal drifts from the base fixture, so a stale literal yields a derived 'fixture' identical to the unmodified base — the mistake surfaces only as indirect downstream assertion failures (unexpected rewrite, missing warning), never at the derivation site. When multiple derived fixtures share the same search literal, extract that literal into one named constant so a base-fixture edit touches a single place. Applied in internal/scaffold/ufconfig_test.go (ufMatrixFixtureNoTriage / ufMatrixFixtureTriageNotSequence).
