---
tag: input-validation
author: jay-flowers
category: gotcha
created_at: 2026-11-02T23:26:36Z
identity: input-validation-20261102T232636-jay-flowers
tier: draft
---

Defanging literal autolink triggers (www., @, ://) in tool-rendered markdown does not close the injection surface: cmark-gfm resolves HTML character references (&#119;w→www via &#119;ww., &#64;→@, &#46;→., &colon;→:, &period;→.) into text nodes during inline parsing, BEFORE the autolink extension post-processes the inline tree — the same ordering that makes backslash-escaping the trigger punctuation ineffective. An attacker-controlled value like a branch named &#119;ww.evil.example or user&#64;host.example therefore reconstructs its trigger after byte-level defanging and still linkifies in GitHub Step Summaries (verified as a residual gap on gaze PR #314 after commit f764b07 pinned literal-trigger defanging). Close the class at the choke point: escape & outside code spans (a backslash-escaped \& cannot open an entity reference; entities are never interpreted inside code spans anyway, so a naive &-escape there would display the backslash) or defang &# and the named entities that decode to trigger characters alongside the literal triggers, and pin each entity encoding of each trigger character with a byte-level negative fixture. When enumerating an autolink defense, the trigger-character alphabet must include every source-level encoding the renderer decodes before autolinking: literal bytes, backslash-escaped bytes, and character references.
