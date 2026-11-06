---
tag: documentation
author: jay-flowers
category: pattern
created_at: 2026-11-02T23:26:40Z
identity: documentation-20261102T232640-jay-flowers
tier: draft
---

When a change redefines what a command IS (not just which flags it accepts), grep the docs tree for the command's characterization adjectives — "AI-formatted", "AI-powered", "AI report pipeline" — in see-also sections and landing pages, not only for contract phrases like format enumerations and invocation strings. Spec 040's implementation sweep followed the format-family/data-flow grep from the earlier documentation learning and still left four stale one-liners: docs/reference/cli/crap.md:184 and docs/index.md:44 (adjective characterizations), docs/reference/cli/docscan.md:112 (same file updated at line 3 but not in its see-also blurb), and docs/reference/cli/analyze.md:112 (format enumeration). Highest-yield targets: the See Also block at the bottom of every sibling command reference, the docs index page, and any file where the diff updated one characterization of the command but not another. Demonstrated by the spec 040 council review (PR 314): the Curator's characterization-adjective grep caught all four after the implementation's contract-phrase grep came up clean.
