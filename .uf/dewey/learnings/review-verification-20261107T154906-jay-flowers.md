---
tag: review-verification
author: jay-flowers
category: gotcha
created_at: 2026-11-07T15:49:06Z
identity: review-verification-20261107T154906-jay-flowers
tier: draft
---

Before flagging a learning's or reviewer's citation as stale or nonexistent, verify the cited anchor with a direct read of the cited path — a repo-wide exact-match grep returning nothing is not proof of absence. On gaze PR #318, three separate citations each initially appeared missing and each verified on a narrower or direct search: the spec-040 opencode-adapter contract clause 'skipped and counted' was split across two wrapped lines (spec prose wraps under 72 chars, so the exact phrase matched no single l
