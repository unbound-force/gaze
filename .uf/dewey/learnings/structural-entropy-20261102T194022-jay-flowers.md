---
tag: structural-entropy
author: jay-flowers
category: gotcha
created_at: 2026-11-02T19:40:22Z
identity: structural-entropy-20261102T194022-jay-flowers
tier: draft
---

When naming the specific import edges behind a package's ΔCe in an entropy review, grep the BASE package directory for each candidate edge before calling it new. In gaze PR #314, the new renderer (internal/aireport/render_markdown.go) visibly imports internal/docscan/apidoc and internal/taxonomy, which looks like two new package dependencies — but both were already in internal/aireport's base import union via unchanged files (runner_steps.go, compact.go). The tool-measured ΔCe +2 was actually stdlib text-handling edges (unicode, unicode/utf8, strconv) net of a removed sync edge (the FakeAdapter test double moved from adapter.go to fake_adapter_test.go). Changed-file import lists undercount the base union; the measured union delta is ground truth, and a base-side grep identifies which edges are genuinely first-time.
