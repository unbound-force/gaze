---
tag: invariant-testing
author: jay-flowers
category: gotcha
created_at: 2026-11-02T20:22:54Z
identity: invariant-testing-20261102T202254-jay-flowers
tier: draft
---

Escaping tests that assert their own escaping mechanism ("every [ ] < is backslash-escaped") verify the mechanism, not the invariant their comments claim ("must not produce a link"). Derive negative assertions from the stated invariant evaluated against the target renderer's FULL feature set, including implicit link-creation mechanisms: GFM extended autolinking makes bare "www." domains and "user@host" emails clickable with no bracket or angle-bracket, and label prefixes like "Branch: " always supply the trailing-space boundary that triggers it. For every escaping control, enumerate the renderer's link/markup creation mechanisms (explicit link syntax, HTML tags/comments, autolinking) and add one failing-test fixture per mechanism — an invariant stated only in a code comment is unenforced. Verified on gaze PR #314: render_markdown_test.go covered brackets/pipes/newlines while a branch named "www.evil.example" (a legal git ref, unlike colon-bearing URLs) still autolinked in the GitHub Step Summary.
