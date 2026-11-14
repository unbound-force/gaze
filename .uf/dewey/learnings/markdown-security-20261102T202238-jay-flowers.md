---
tag: markdown-security
author: jay-flowers
category: gotcha
created_at: 2026-11-02T20:22:38Z
identity: markdown-security-20261102T202238-jay-flowers
tier: draft
---

When neutralizing GFM extended autolinks in tool-rendered markdown, prefer defanging (www[.]evil.example, user[@]host) over backslash-escaping the trigger punctuation (www\., user\@host): the inserted bracket is not a domain character, so defanging breaks the autolink grammar at both the source-byte level and the rendered-text level, while backslash escapes are resolved into text before cmark-gfm's autolink extension post-processes the inline tree, so a reconstructed www. can still match and link. Validate whatever mechanism is chosen once against GitHub's actual renderer (keep the repo stdlib-only — no markdown-renderer dependency for CI), then pin it with byte-level negative tests asserting no boundary-preceded www. / user@host / :// survives in the rendered bytes; escaping invariants stated only in code comments (like gaze PR #314's headerEscaper at render_markdown.go:77) are unenforced until such a test exists.
