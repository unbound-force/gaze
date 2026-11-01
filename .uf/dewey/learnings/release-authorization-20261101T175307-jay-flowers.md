---
tag: release-authorization
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:07Z
identity: release-authorization-20261101T175307-jay-flowers
tier: draft
---

Release-pipeline authorization gates must be ordered before every job with an irreversible side effect, not merely before the jobs that consume the protected resource. In gaze's release.yml, the reusable preflight job creates the annotated tag via the GitHub API — tag creation, not release publication, is the first irreversible action, so an authorization gate placed after preflight would fail safely yet leave an orphaned tag. When triaging or reviewing release-hardening proposals, verify the proposed gate ordering against each job's actual side-effect map (tag creation, release creation, signing, cask push) rather than against job names.
