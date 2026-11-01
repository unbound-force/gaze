---
tag: ci-asset-drift
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:20:19Z
identity: ci-asset-drift-20261101T172019-jay-flowers
tier: draft
---

Verified while triaging gaze issue #263 (fast CI job for embed.FS drift): the issue's illustrative shell snippet globs `internal/scaffold/assets/**/*.md` + `**/*.jsonc`, but bash without `shopt -s globstar` treats `**` as `*` (single directory level), so root-level `internal/scaffold/assets/dcp.jsonc` — 1 of the 9-file mirror manifest enumerated in `TestAssetPaths_Returns9Files` (internal/scaffold/scaffold_test.go:381) — is silently uncovered. Additionally, the snippet's `if [ -f "$src" ] && ! diff -q ...` guard passes silently when the `.opencode/` counterpart is missing entirely, whereas `TestEmbeddedAssetsMatchSource` (scaffold_test.go:342) fails hard on a missing source. Fast CI drift checks over embedded scaffold assets should mirror the enumerated manifest instead of globbing: the manifest also encodes the deliberate `ci/` subtree exclusion (`internal/scaffold/assets/ci/gaze-ai-report.yml` has no `.opencode/` counterpart by design, enforced by `TestAssetPaths_ExcludesCISubtree`), which a naive glob sweep would false-positive on. The fast check must also add the secondary sync explicitly (`internal/aireport/assets/agents/gaze-reporter.md` ↔ `internal/scaffold/assets/agents/gaze-reporter.md`, enforced by `TestEmbeddedPromptMatchesScaffold`) — the issue's prose mentions it but its example script omits it.
