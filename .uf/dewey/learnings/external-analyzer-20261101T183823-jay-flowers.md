---
tag: external-analyzer
author: jay-flowers
category: pattern
created_at: 2026-11-01T18:38:23Z
identity: external-analyzer-20261101T183823-jay-flowers
tier: draft
---

Gaze external-analyzer dispatch gates are duplicated per sibling command runner in cmd/gaze/main.go, so widening a gate for some commands leaves others divergent: PR #281 (design decision D7, openspec/changes/quality-classification-counts/design.md) switched runCrap and runQuality to `analyzerFlag != "" || languageFlag != ""` matching runDocscan, but runReport alone still gates module-root resolution and external dispatch on `analyzerFlag` only — `--language` alone is silently inert there (issue #282). The shared plumbing already handles language-only discovery (initExternalSession → adapter.Discover, error "no analyzer found for language %q"), so parity fixes are gate-condition widenings, not new wiring; runReport's external path also bypasses the standard CRAP/quality/classify/docscan step pipeline via the opts.AnalyzeFunc override. Triage pattern for "command X ignores flag Y" issues in gaze: diff the gate conditions across the sibling runners first, and treat cited line numbers as drift-prone (issue #282 cited ~2013; the gates sat at 2225/2294 at verification time).
