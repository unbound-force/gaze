---
tag: ci-verification
author: jay-flowers
category: gotcha
created_at: 2026-11-15T01:36:56Z
identity: ci-verification-20261115T013656-jay-flowers
tier: draft
---

A CI check that claims to verify installability 'from proxy.golang.org' by running `go install module@version` does NOT actually prove proxy availability: the default GOPROXY (`https://proxy.golang.org,direct`) falls through to `direct` (VCS clone) on a proxy 404/410, so a tag pushed to GitHub but not yet propagated to the proxy still installs. To strictly verify the proxy path, pin `GOPROXY=https://proxy.golang.org` (keeping GOSUMDB=sum.golang.org); otherwise the check proves default-source installability and the docs/spec must say so.
