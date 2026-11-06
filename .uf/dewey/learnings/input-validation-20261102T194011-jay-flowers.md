---
tag: input-validation
author: jay-flowers
category: gotcha
created_at: 2026-11-02T19:40:11Z
identity: input-validation-20261102T194011-jay-flowers
tier: draft
---

Escaping-based output controls must be tested against the renderer's *implicit* link-creation mechanisms, not just explicit markup syntax. Bracket and angle-bracket escaping defeats `[text](url)` and `<tag>`, but GFM extended autolinking still creates a clickable link from a bare `www.`-prefixed domain or an email address preceded by a space — which template prose like `Branch: <value>` always provides. Verified on gaze PR #314: render_markdown.go's headerEscaper claims (in its own comment) that a crafted reference name cannot inject a link, yet a branch named `www.evil.example` (a legal git ref; no colon required unlike `https://`) or a go.mod module path of the same shape renders as an attacker-chosen link in the GitHub Step Summary, while the AI-answer validator in the same PR bans `www.` and `://` for its channel. Pattern: when one channel's validator bans a content class, enumerate the renderer's link/markup creation mechanisms for every OTHER channel writing into the same artifact, and promote each escaping control's stated invariant from a code comment into a failing test (a fixture with a www-branch and user@host-branch).
