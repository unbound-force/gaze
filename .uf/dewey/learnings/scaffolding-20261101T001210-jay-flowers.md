---
tag: scaffolding
author: jay-flowers
category: gotcha
created_at: 2026-11-01T00:12:10Z
identity: scaffolding-20261101T001210-jay-flowers
tier: draft
---

File-derived placeholder grammars must enumerate the source format's legal adornments, not just hostile or malformed values. gaze init --ci's go.mod line scan (init-ci-scaffold change, PR #308) normalizes trailing whitespace and CRLF before its grammar gate but does not strip trailing // comments — legal go.mod syntax (the same syntax as // indirect require markers) — so `go 1.24 // note` fails the accepted grammar and degrades to the 1.25 fallback with a "no readable `go` directive" warning that misattributes the cause to the file. When a design chooses a minimal scanner over a real parser to avoid coupling to unrelated file validity, the spec's value-space enumeration for that placeholder must include the format's legal comment/quoting adornments alongside malformed input, and fallback warnings should say "outside the accepted grammar" rather than "unreadable" when the scan — not the file — is the narrower side.
