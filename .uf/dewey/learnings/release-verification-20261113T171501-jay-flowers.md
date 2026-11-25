---
tag: release-verification
author: jay-flowers
category: pattern
created_at: 2026-11-13T17:15:01Z
identity: release-verification-20261113T171501-jay-flowers
tier: draft
---

A value that must equal the release tag (for example a reusable workflow's tool-version input default) should get an offline release-time assertion rather than a YAML comment. Gaze already has the pattern: validate-workflow-floor.sh greps the floor constant from scaffold.go and compares it to the dispatch tag, unskippable and offline. The default-equals-tag rule depends on two separate human steps (editing the default and dispatching the release) and its failure is silent — the older binary keeps running and its output can still pass — so the comment-at-point-of-change mitigation is worth pairing with the same grep-and-compare gate as soon as the release is cut; also cite an open tracking issue when deferring the automation, since immutable spec artifacts cannot carry the obligation after merge.
