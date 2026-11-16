---
tag: spec-accuracy
author: jay-flowers
category: gotcha
created_at: 2026-11-07T18:22:21Z
identity: spec-accuracy-20261107T182221-jay-flowers
tier: draft
---

Verification claims in spec artifacts' constitution-alignment sections must be derived from the repository's CI configuration, not from the local verification workflow. On gaze PR 317 (dogfood-ai-report-from-source), proposal.md's Testability PASS claimed "actionlint and yamllint validate the workflow syntax and expressions locally and in CI" — but actionlint runs nowhere in gaze CI: .mega-linter.yml uses an ENABLE_LINTERS whitelist (GO_GOLANGCI_LINT, MARKDOWN_MARKDOWNLINT, YAML_YAMLLINT, REPOSITORY_GITLE... [Information field was bounded at 512 chars by the summary-detail retrieval; stored verbatim.]
