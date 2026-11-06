---
tag: spec-contracts
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:17:19Z
identity: spec-contracts-20261102T211719-jay-flowers
tier: draft
---

Spec 040 introduced contracts/ artifacts as line-by-line implementation contracts for parsing/extraction code. Clause-level half-implementation gotcha: a behavioral clause with an observability tail ("lines that fail to unmarshal are skipped AND COUNTED") is easy to half-land — the primary behavior (skip) works, every fixture passes because fixtures cover only the all-or-nothing paths (not_json.txt exercised the zero-events failure, not partial degradation), and the observability artifact (the count) silently never exists. Prevention: when implementing from a contracts/ artifact, diff each numbered contract clause against the implementation before marking the task complete, and add a fixture for every clause with a measurable artifact (counts, suffixes, orderings) — especially the partial-degradation path between total failure and clean success.
