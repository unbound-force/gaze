---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-26T23:07:25Z
identity: test-quality-20261126T230725-jay-flowers
tier: draft
as_of: "gaze@6cfd16ef8c1d55fbe397eb7a1126b9261eebdd00"
---

Golden-file tests that check for unsubstituted template placeholders by enumerating the known placeholder constants miss cross-variant copy-paste residue. If a Node workflow template accidentally contains the Go template's `__GO_VERSION__` token, a renderer that substitutes per-variant placeholder lists will replace it with that variant's (empty) context value and the token vanishes silently — the enumerated check finds nothing. Gaze's ci_test.go fixed this by switching to a generic residue scan (`regexp.MustCompile(`__[A-Z][A-Z0-9_]*__`)`) that matches any dunder token regardless of variant, so a stray placeholder from any template fails the test. When a renderer substitutes a superset of placeholders across variants, assert on the token SHAPE, not on an enumerated list of known tokens.
