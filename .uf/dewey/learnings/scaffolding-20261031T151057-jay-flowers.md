---
tag: scaffolding
author: jay-flowers
category: pattern
created_at: 2026-10-31T15:10:57Z
identity: scaffolding-20261031T151057-jay-flowers
tier: draft
---

When a scaffold generates executable artifacts (CI workflow YAML) from file-derived values, every substituted value needs a specified grammar and fallback in the spec. "Verbatim passthrough" of a value like go.mod's `go` directive into a YAML double-quoted scalar permits malformed output and quote-escape corruption: a value containing `"` terminates the scalar (silent truncation or an unparseable workflow), and a raw CR from a CRLF file triggers YAML flow folding. CLI-flag placeholders are safe only when explicitly validated first (e.g., semver.IsValid yields a YAML-safe character set) — the file-derived placeholder is the usual gap. Spec reviews of scaffolding features should enumerate each placeholder's valid/invalid value space with the same rigor applied to flags.
