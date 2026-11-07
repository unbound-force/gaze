---
tag: documentation-accuracy
author: jay-flowers
category: pattern
created_at: 2026-11-07T18:22:17Z
identity: documentation-accuracy-20261107T182217-jay-flowers
tier: draft
---

When a mid-PR feedback fix changes a command or mechanism that earlier commits in the same PR already described in prose, the fixing commit must sweep every prose site that quotes the command verbatim. AGENTS.md Recent Changes entries and in-flight OpenSpec design decisions quote commands exactly, so a stale quote documents the defect the fix removed as the shipped mechanism — and once the change merges and archives, Spec Artifact Immutability seals the inaccurate record. Verified on gaze PR 317 (dogfood-ai-repo... [Information field was bounded at 512 chars by the summary-detail retrieval; stored verbatim.]
