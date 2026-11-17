---
tag: markdown-escaping
author: jay-flowers
category: gotcha
created_at: 2026-11-15T02:42:49Z
identity: markdown-escaping-20261115T024249-jay-flowers
tier: draft
---

Byte-level backslash escaping does not compose in markdown renderers. When an escaper inserts a backslash before a byte (gaze's entity defense: & becomes backslash-&) but passes literal backslashes through unescaped (proseEscaper, escapeCellMarkup), an attacker value with an odd run of backslashes before an entity (user-backslash-&#64;host.example) renders as user-backslash-@host.example: cmark-gfm consumes the double backslash as an escaped backslash and the character reference still decodes. Sinks that DO escape backslash (headerEscaper) are immune, so the same hostile input is safe in one sink and decodes in another. Escape-matrix fixtures that share one want string across sinks must exclude the bytes the sinks treat differently — which silently excludes exactly the divergent, least-safe interaction. Pin backslash-adjacent hostile inputs per sink, and state safety invariants only as narrowly as the parser's escape-consumption semantics allow; a byte-level 'preceded by a backslash' check does not equal 'escaped' in CommonMark.
