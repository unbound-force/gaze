---
tag: ci-cd
author: jay-flowers
category: gotcha
created_at: 2026-11-13T17:14:59Z
identity: ci-cd-20261113T171459-jay-flowers
tier: draft
---

When documenting a pinned `uses: ...@vX.Y.Z` consumer example for a reusable workflow that also exposes a tool-version input with a default, remember the default is resolved from the workflow file at that tag, not from main. A stale example silently installs the older default: gaze's ci-integration guide pinned `@v2.3.0` while omitting `gaze-version`, and v2.3.0's baked-in default was v2.2.0 (the pre-spec-040 adapter), so copying the example recreated the very narration-passing incident the docs' new note warned against. Whenever a workflow input default is bumped, grep the docs for every `uses: ...@` example and either raise its pin or set the tool-version input explicitly (explicit matches what `gaze init --ci` emits and survives re-tagging).
