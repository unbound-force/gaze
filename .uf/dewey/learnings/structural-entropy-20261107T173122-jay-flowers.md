---
tag: structural-entropy
author: jay-flowers
category: pattern
created_at: 2026-11-07T17:31:22Z
identity: structural-entropy-20261107T173122-jay-flowers
tier: draft
---

Entropy-divisor recovery when the pinned head SHA is missing from the local object store: the PR branch was absent from origin's refs/heads and refs/pull/317/head was absent too (fork topology — this checkout's origin is github.com/jflowers/gaze while the PR lives on the upstream repo), but `git fetch origin <full-40-hex-head-SHA>` retrieved the exact reviewed commit anyway: the fork's object store held the head even though no branch advertised it, and GitHub serves fetch-by-SHA for held objects. The comman
