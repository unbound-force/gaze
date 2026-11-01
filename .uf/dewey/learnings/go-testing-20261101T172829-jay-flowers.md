---
tag: go-testing
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:28:29Z
identity: go-testing-20261101T172829-jay-flowers
tier: draft
---

Manual `os.Chdir` + `t.Cleanup(func(){ _ = os.Chdir(orig) })` boilerplate in Go tests is a concurrency-correctness and error-handling defect, not just verbosity: (1) the restore error is silently discarded (a spirit violation of default-pack CS-006's no-silent-error-swallowing rule), so a failed restore leaves every subsequent test in the package running from the wrong working directory with no signal; (2) the process-wide working directory is shared OS state that the Go race detector does not instrument, so a test using manual `os.Chdir` that later gains `t.Parallel` introduces a cwd race that `-race` cannot catch. `t.Chdir` (Go 1.24+) closes both by construction — it surfaces chdir failures via t.Fatal and refuses to run in parallel tests. Review findings on this pattern (e.g., gaze issue #266's proposed TC-013 convention-pack rule) should cite these hazards rather than merely line-count savings.
