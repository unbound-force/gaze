---
tag: gitignore
author: jay-flowers
category: gotcha
created_at: 2026-11-24T23:27:54Z
identity: gitignore-20261124T232754-jay-flowers
tier: draft
as_of: "gaze@6675b56b89dea50273e85b513a60b545c5d2fac5"
---

A bare (unanchored) `gaze` line in .gitignore matched any file or directory named `gaze` at any depth and silently hid newly added cmd/gaze/*.go source files from `git status` — new test files could ship uncommitted. The fix removed the bare pattern while keeping the root-anchored `/gaze` (the compiled binary). Anchor build-artifact ignore patterns to the repo root so they ignore only the artifact; unanchored binary-name patterns shadow real source paths anywhere in the tree.
