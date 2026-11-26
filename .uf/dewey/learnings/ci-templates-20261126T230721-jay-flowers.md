---
tag: ci-templates
author: jay-flowers
category: gotcha
created_at: 2026-11-26T23:07:21Z
identity: ci-templates-20261126T230721-jay-flowers
tier: draft
as_of: "gaze@6cfd16ef8c1d55fbe397eb7a1126b9261eebdd00"
---

Scaffolded CI templates that auto-detect a language from a single marker file (e.g. package.json -> Node) can produce workflows that fail out of the box: `npm ci` requires a package-lock.json, so pnpm/yarn/bun projects (package.json but no npm lockfile) break on the first install step. Align detection with the install command (choose the package manager by lockfile presence) or emit an inline comment stating the assumption and how to switch. Likewise, keep runner-side test tooling reproducible across variants: Go pins via go.mod, Node via `npm ci`, but a bare `pip install pytest pytest-cov` floats to latest and makes the scaffolded workflow non-reproducible.
