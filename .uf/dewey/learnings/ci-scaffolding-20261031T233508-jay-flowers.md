---
tag: ci-scaffolding
author: jay-flowers
category: pattern
created_at: 2026-10-31T23:35:08Z
identity: ci-scaffolding-20261031T233508-jay-flowers
tier: draft
---

When a scaffold template duplicates values from the tool's own CI configuration (action SHA pins, default toolchain versions), the duplication needs an anti-drift test that parses the source-of-truth workflow files. Design claims like "no drift between our CI and what we scaffold" hold only at merge time: in gaze's init-ci-scaffold (2026-10-31 SRE review) the template's three action SHAs and the 1.25 Go-version fallback all matched .github/workflows/{reusable_gaze_ai_report,test}.yml on merge day, but the AntiDrift test covered only the uses: path and the version floor — a future SHA bump in gaze's own CI would leave new scaffolds emitting stale pins silently. Scope anti-drift tests to every duplicated value and record the invariant in the repo's file-sync table; where a duplicated value exists only to echo a called workflow's default, prefer eliminating the duplication entirely (e.g. omit go-version and let setup-go derive it from the module's go.mod natively).
