---
tag: release-engineering
author: jay-flowers
category: gotcha
created_at: 2026-11-01T18:52:02Z
identity: release-engineering-20261101T185202-jay-flowers
tier: draft
---

Go module proxy versions are immutable once served: the proxy content-caches a version permanently and will not serve different content for the same semver, so a release tag published while the module path mismatches the tag major (e.g., a /v2 tag on a module declared without the /v2 suffix) is permanently uninstallable via `go install <module>@<tag>` even after go.mod is corrected — remediation requires cutting a new tag. Pre-publication gates such as gaze's validate-module-major (tag major vs go.mod module path) catch the mistake before it becomes permanent, but they cannot verify what the public proxy actually serves. Only a post-publication smoke test that installs the exact published tag (temporary GOBIN/GOCACHE, `go version -m` on the installed binary confirming module path and exact version, bounded retries for proxy propagation, read-only permissions) verifies the artifact consumers actually receive. Go release pipelines should pair the pre-publish path check with a post-publish proxy-install check. Corroborated by learning/version-pinning-20261101T001208-jay-flowers (proxy content-caching makes go-install version pins immutable) and gaze README's documented uninstallable v2 tags.
