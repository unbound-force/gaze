---
tag: release-gates
author: jay-flowers
category: pattern
created_at: 2026-11-13T18:21:03Z
identity: release-gates-20261113T182103-jay-flowers
tier: draft
---

When a reusable workflow's input default must equal the release tag, an exact-equality release gate already guarantees the default is canonical: the release tag is grammar-validated first, so a malformed default can never equal a valid tag and every release fails closed (exit 2 for unparsable/zero-default, exit 1 for mismatch). The paired repository-invariant test therefore only needs to prove the default is *parsable* (invoke the gate with a tag that cannot match and assert exit 1); asserting the default's canonical form would require duplicating the tag grammar on both sides. Gaze PR #335 implemented this as validate-ai-report-default.sh plus test-validate-ai-report-default.sh; the test comment's phrase "in canonical tag form" overstates what the probe proves, without weakening the gate.
