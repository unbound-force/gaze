---
tag: input-validation
author: jay-flowers
category: gotcha
created_at: 2026-11-02T20:22:51Z
identity: input-validation-20261102T202251-jay-flowers
tier: draft
---

Negative tests for linkification invariants must use trigger values the target renderer would actually linkify. GFM §6.9 extended autolinks impose validity requirements: a www autolink needs `www.` followed by a valid domain, and an email autolink needs a valid email address whose domain carries a dot — a bare `user@host` with no dot is not a valid GFM email and will not linkify, so a negative test using it passes vacuously and pins nothing. When pinning a no-autolink escaping invariant (e.g., gaze PR #314 headerEscaper), the email-shaped fixture must be `user@host.example`-shaped, and the test should assert on the emitted bytes (no whitespace-preceded bare `www.`/scheme/email token survives escaping) since the test cannot run GitHub's renderer. This refines learning/input-validation-20261102T194011-jay-flowers, whose fixture guidance said 'user@host-branch'.
