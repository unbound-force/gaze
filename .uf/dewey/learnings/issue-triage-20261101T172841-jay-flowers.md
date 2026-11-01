---
tag: issue-triage
author: jay-flowers
category: gotcha
created_at: 2026-11-01T17:28:41Z
identity: issue-triage-20261101T172841-jay-flowers
tier: draft
---

Retro-agent issues cite repository state as it existed at retrospective time, and those citations go stale in active repositories — triage must re-verify them against the current tree rather than trusting ingest-time "verified" summaries. Triaging unbound-force/gaze#266 (propose convention-pack rule TC-013: prefer t.Chdir over os.Chdir+t.Cleanup boilerplate), both the issue body and the ingest-time review context asserted "t.Chdir already used in 2 TestLoadConfig_* instances in cmd/gaze/main_test.go"; a fresh repo-wide grep at triage time found zero t.Chdir usages and no TestLoadConfig tests in any .go file — subsequent refactoring (the implemented config-threshold-validation OpenSpec change touches those tests) had removed them. The load-bearing claims still held (go.md defines TC-001–TC-012 with no directory-change rule; internal/loader/loader_test.go:99-102 still carries the legacy os.Chdir+t.Cleanup pattern), so the issue remained VALID — but repeating the stale precedent would have misled the implementer. Rule of thumb: for any issue claiming "the codebase already does X", grep for X at triage time; drift in supporting evidence does not by itself invalidate a request whose gap and cleanup target still exist, but the drift must be called out in the triage output.
