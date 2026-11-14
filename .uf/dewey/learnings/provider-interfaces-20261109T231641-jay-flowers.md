---
tag: provider-interfaces
author: jay-flowers
category: context
created_at: 2026-11-09T23:16:41Z
identity: provider-interfaces-20261109T231641-jay-flowers
tier: draft
---

TestSC002_GazeCRAPloadMatchBetweenCrapAndReport (cmd/gaze/main_test.go:3295) fails on main without -short because it predates the provider-interfaces refactor: it calls crap.Analyze with crap.DefaultOptions() plus only a ContractCoverageProvider, and DefaultOptions() leaves ComplexityProvider and LineCoverageProvider nil, so Analyze returns 'ComplexityProvider is required' before any analysis or coverage generation. Its runReport leg (real pipeline, no runnerFunc, no coverProfile) is therefore never reached
