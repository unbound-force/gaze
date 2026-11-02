---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:46:51Z
identity: test-quality-20261102T004651-jay-flowers
tier: draft
---

Pattern in internal/scaffold/ufconfig_test.go (ufMatrixFixtureNoTriage, ufMatrixFixtureTriageNotSequence): fixtures are derived from a base fixture through one package-level strings.Replace var shared by setup and expectation. Pair it with a guard that fails loudly if the Replace needle stops matching, so the derived fixture cannot silently equal the base.
