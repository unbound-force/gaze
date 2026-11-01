---
tag: release-verification
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:52:10Z
identity: release-verification-20261101T185210-jay-flowers
tier: draft
---

Offline pre-publication release gates (tag-vs-go.mod module-path consistency, semver/workflow-floor checks) and static workflow-configuration tests both stop at the publication boundary: neither verifies that a published release is actually consumable through the public Go module proxy. A released tag can pass every offline gate yet fail `go install module/vN/cmd/...@vN.x.y` — gaze's README documents exactly this failure class (early v2 tags carried an incompatible module path and fail `go install github.com/unbound-force/gaze/v2/cmd/gaze@latest`). The closure is a distinct coverage class: a post-publication smoke test that installs the exact published tag from proxy.golang.org into temporary GOBIN/module caches (isolation), runs `go version -m` on the installed binary, and asserts both the module path and the exact requested version, with bounded retries for normal proxy propagation. The network dependency belongs in a post-publication or manually dispatched read-only workflow, never in the pre-merge suite. Verified in triage of unbound-force/gaze#297: zero `proxy.golang.org` references exist under .github, and the only `go install` of a released tag (reusable_gaze_ai_report.yml) is a pinned tooling install for CI, not verification of the newly published tag — incidental future consumption of a release is not regression protection for its publication. This complements the established static-workflow-configuration-test strategy for gates that cannot run end-to-end (learning/ci-testing-20261101T175311): pre-publication gates get static config tests; post-publication availability gets a real end-to-end install check.
