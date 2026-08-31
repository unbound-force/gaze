---
tag: adapter-call-unmarshal-helper
author: yvonne-devlin
category: pattern
created_at: 2026-08-27T15:11:58Z
identity: adapter-call-unmarshal-helper-20260827T151158-yvonne-devlin
tier: draft
---

Pattern/design: a collapsed generic error-wrapping helper (e.g. callAndUnmarshal[T] that maps Call→transport-err→protocol-err→unmarshal into uniform "%s protocol call"/"%s protocol error"/"parsing %s result" templates) CANNOT preserve per-branch legacy error wording when a call site's historical strings diverge from the template. In internal/adapter/, the 3 batch sites whose method constants ARE the human prefix (complexity/coverage/analyze) migrated cleanly because the templates reproduce their legacy strings 1:1. But session.go Initialize used THREE distinct non-template strings ("initialize handshake"/"initialize error"/"parsing initialize result") that the collapsed single-error helper cannot reproduce without fragile prefix translation. Resolution sanctioned by design D2: Initialize RETAINS its original inline per-branch error handling (and its per-branch s.client.Close() cleanup) rather than delegating — so effective helper adoption was 3 sites, not the originally-planned 4. Lesson: when planning a DRY error-helper extraction, audit each call site's exact error strings first; sites with multiple distinct historical prefixes are legitimate exclusions, and preserving exact operator-facing error strings (log/alert grep patterns) outranks maximizing migration count.
