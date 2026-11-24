---
tag: test-architecture
author: jay-flowers
category: decision
created_at: 2026-11-10T02:01:09Z
identity: test-architecture-20261110T020109-jay-flowers
tier: draft
---

Gaze PR #326 feedback resolution (divisor-architect finding, author-acknowledged): the GAZE_COVERAGE_RUN guard idiom deliberately uses the literal env-var name at test sites (os.Getenv("GAZE_COVERAGE_RUN") != "" + t.Skip — 5 sites: cmd/gaze/main_test.go:2483/2505/3430, internal/aireport/runner_steps_test.go:30/110) while production centralizes the name in the unexported goprovider.coverageRunEnv constant (coverage.go:64). Two reasons the duplication is intentional and safe: (1) the literal pins the documented external contract — AGENTS.md, README.md, and docs/reference/cli/{crap,report,self-check}.md all name GAZE_COVERAGE_RUN, and user scripts may set it — so a constant rename or value typo cannot silently pass the guards, whereas guards referencing an exported constant would track it tautologically; (2) the correctness invariant is enforced once at the production spawn seam (generateCoverProfile returns ErrNestedCoverageRun before any temp profile or go test spawn), so a misspelled or missing test guard fails loudly inside the nested E2E run instead of diverging silently — default.md CS-004's silent-drift harm model does not apply to defense-in-depth skip guards. Recorded escalation threshold: at a 6th skip-guard site, export goprovider.CoverageRunEnv or extract a shared skip helper.
