---
tag: review-methodology
author: jay-flowers
category: gotcha
created_at: 2026-11-07T18:44:24Z
identity: review-methodology-20261107T184424-jay-flowers
tier: draft
---

Before flagging a shell comparison as fragile based on uncertain recall of a tool's output format, grep the repository for the same idiom in scripts CI already exercises. On gaze PR 317 (dogfood-ai-report-from-source), the new `Install gaze (source)` guard compares bare `go list -m` output for strict equality with the module path; recalling that `go list -m` sometimes appends a version makes that guard look broken, but `.github/scripts/test-validate-module-major.sh:10-16` uses the identical `actual_module="[retrieved via bounded council summary detail; text cut at 512 chars]
