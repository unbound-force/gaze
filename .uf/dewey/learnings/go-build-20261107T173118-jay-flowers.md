---
tag: go-build
author: jay-flowers
category: reference
created_at: 2026-11-07T17:31:18Z
identity: go-build-20261107T173118-jay-flowers
tier: draft
---

When stamping build identity in CI via `go install -ldflags "-X main.version=... -X main.date=..."`, derive the date from `git show -s --format=%cI HEAD` rather than raw commit metadata or environment strings: %cI emits strict ISO 8601 (no spaces, no control characters) regardless of how the committer date was stored, so even a crafted GIT_COMMITTER_DATE normalizes to a single token that cannot inject additional linker flags when interpolated inside the quoted -ldflags string. GITHUB_SHA is hex and inherent
