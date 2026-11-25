---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T21:17:34Z
identity: test-quality-20261102T211734-jay-flowers
tier: draft
---

Contract clauses that promise an observable signal die silently unless a test binds them to an assertion in the same change. In gaze PR #314, the spec-040 opencode-adapter contract states non-JSON stream lines are 'skipped and counted', but parseEvents shipped a bare `continue` — no counter anywhere — and the fixture matrix was all-or-nothing (all-valid streams plus one all-invalid not_json.txt), so no test could fail on the missing count. Two audit rules for skip-on-error stream parsers: (1) count the skips inside the parsing function and surface the count on every outcome — failure errors AND the success path — because the headline degradation scenario (mostly junk lines, one decodable answer) yields a *successful* extraction that error-path threading alone never touches; (2) add a mixed fixture (valid events interleaved with junk lines) asserting the surfaced count as a specific value, since all-or-nothing fixture matrices structurally cannot see partial degradation.
