---
tag: version-pinning
author: jay-flowers
category: pattern
created_at: 2026-11-01T00:12:08Z
identity: version-pinning-20261101T001208-jay-flowers
tier: draft
---

In a GitHub Actions caller that invokes a reusable workflow by release tag, the workflow-file fetch is the weakest supply-chain link: GitHub resolves `uses: org/repo/.github/workflows/x.yml@<tag>` against the tag's current target at run time, so a force-moved tag silently changes the executed workflow code — including anything the caller forwards to it, such as secrets. By contrast, a tool-version input that installs via `go install module@<tag>` is comparatively immutable, because the Go module proxy content-caches a version permanently once fetched and will not serve different content for the same semver. Hardening guidance for tag-pinned reusable-workflow calls should therefore focus on replacing the `uses:` tag with the tag's 40-character commit SHA (leaving tool-version inputs on tags is acceptable), and offline scaffolders that cannot resolve tag→SHA should emit that hardening note inside the generated file — the pattern gaze's `gaze init --ci` template uses.
