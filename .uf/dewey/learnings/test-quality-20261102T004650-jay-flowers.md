---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-02T00:46:50Z
identity: test-quality-20261102T004650-jay-flowers
tier: draft
---

For byte-identity assertions in Go tests, build both the staged input and the expected output from one package-level var, e.g. `var fixtureNoX = strings.Replace(base, block, "", 1)`, so they cannot drift. Duplicated inline literals can turn into silent no-op Replaces when the base fixture changes.
