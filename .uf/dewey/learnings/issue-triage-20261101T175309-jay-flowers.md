---
tag: issue-triage
author: jay-flowers
category: pattern
created_at: 2026-11-01T17:53:09Z
identity: issue-triage-20261101T175309-jay-flowers
tier: draft
---

Hardening-issue triage pattern (unbound-force/gaze, issue #267): an issue that reverses an intentional prior design decision is VALID when verifiable against the documented behavior it changes — the unsigned-cask fork fallback was deliberate graceful degradation (spec 015-native-macos-signing; learning/release-pipeline-20260611T201838), and #267's scope includes updating the README/AGENTS.md/macOS-signing guidance that documented the old behavior. Record overlapping issues as strict subsets, not duplicates: #99 (hardcoded signing identity) is subsumed by #267's six-secret requirement, so "resolves #99" is the correct relationship. CI/CD pipeline changes are exempt from the Website Documentation Gate and sit below the blog/tutorial significance thresholds (no new agent, CLI command, or user-facing workflow), so release-workflow hardening issues need no unbound-force/website, blog, or tutorial tracking issue — the issue's internal doc scope is the complete documentation surface.
