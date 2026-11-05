---
tag: markdown-escaping
author: jay-flowers
category: gotcha
created_at: 2026-11-02T20:22:48Z
identity: markdown-escaping-20261102T202248-jay-flowers
tier: draft
---

Neutralizing GFM extended autolinks in markdown emitted to GitHub-rendered surfaces (Step Summary, PR comments): code-span wrapping is the mechanism GFM's processing order guarantees — code-span content is literal and never autolinked, and gaze's aireport package already ships a codeSpan helper that handles embedded backticks via double-backtick spans. Backslash-escaping the trigger punctuation, the idiomatic move for brackets and angle brackets, must not be assumed to work for autolink triggers: GFM resolves backslash escapes into text during inline parsing and the autolink extension can then linkify the resulting www.-prefixed domain or dotted email, so www\. can still become a link. Validate any escape-based mechanism empirically against the GFM reference implementation, and pin the chosen mechanism with byte-exact negative tests that render a www.-prefixed branch and a dotted-host user@evil.example branch through the metadata header (use a dotted host — GFM's email autolink requires a valid dotted domain, so a dotless user@host may not trigger at all).
