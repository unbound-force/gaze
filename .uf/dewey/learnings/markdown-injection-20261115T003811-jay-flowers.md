---
tag: markdown-injection
author: jay-flowers
category: pattern
created_at: 2026-11-15T00:38:11Z
identity: markdown-injection-20261115T003811-jay-flowers
tier: draft
---

cmark-gfm resolves HTML character references (&#119;, &#x3A;, &colon;, &commat;) into text nodes during inline parsing, BEFORE the GFM autolink extension post-processes the inline tree. Markdown injection defenses that defang only literal autolink trigger bytes (www., scheme://, dotted-email @) are therefore bypassable: a value like &#119;ww.evil.example carries no literal trigger byte for byte-level defanging to catch, yet decodes to a linkifiable www domain. The class-closing fix is unconditional backslash-escaping of & as \& at every non-code-span output sink — one rule closes every encoding family, known and future, because a backslash-escaped & cannot open a reference — while code-span content must keep & verbatim (code spans never interpret references; escaping there displays the backslash). Layering note: strings.NewReplacer performs a single non-overlapping pass in argument order, so an escapeAmp layer composed before a markup escaper never gets its inserted backslash re-escaped, and the & pair cannot disturb existing pair ordering because & is not a prefix or suffix of any other pattern.
