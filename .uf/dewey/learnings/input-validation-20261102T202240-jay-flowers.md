---
tag: input-validation
author: jay-flowers
category: pattern
created_at: 2026-11-02T20:22:40Z
identity: input-validation-20261102T202240-jay-flowers
tier: draft
---

Neutralizing GFM extended autolinking in generated markdown requires breaking the trigger at the source-text level; escaping markup metacharacters cannot help because the autolink grammar fires on bare `www.`+domain, `scheme://`, and `local@domain` sequences preceded by a space, line start, or an opening delimiter — boundaries that label prefixes like "Branch: " and table-cell openers like "| " always supply. Fix-side mechanisms that preserve the visible value: wrap the value in an inline code span (code spans are never autolinked — gaze's tableName already relies on this for table cells), or, where the format forbids code spans (fixed header lines), rewrite the trigger's first character as a numeric character reference (`&#119;ww.`, `user&#64;host`) or backslash-escape the punctuation after it (`www\.`) — both render as the original text while failing the source-level grammar; validate each mechanism against the actual renderer (cmark-gfm / GitHub Step Summary) since autolink handling is implementation-dependent. Enumerate the complete trigger class (www., ://, AND bare `@` emails) once per artifact: gaze's answer.go ruleMarkup banned www. and :// but not emails, and the render_markdown.go metadata/prose channels banned none. Complements learning/input-validation-20261102T194011-jay-flowers.
