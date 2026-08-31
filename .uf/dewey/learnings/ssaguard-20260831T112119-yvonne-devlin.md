---
tag: ssaguard
author: yvonne-devlin
category: pattern
created_at: 2026-08-31T11:21:19Z
identity: ssaguard-20260831T112119-yvonne-devlin
tier: draft
---

Consolidating a duplicated recover()-guard helper (safeSSABuild) into a shared package (internal/ssaguard) is safe and coverage-neutral when the guard has ZERO external dependencies — it takes a func() and returns any. This reverses spec-021 R3's 'keep packages dependency-light' rationale, which only applied when a helper might drag in dependencies. Key design decisions that made review pass 5/5: (1) Name the package `ssaguard` NOT `ssautil` to avoid shadowing golang.org/x/tools/go/ssa/ssautil already imported at both call sites. (2) Export the function (SafeSSABuild) so both callers' export_test.go shims are eliminated — under internal/ this adds no API surface concern. (3) Keep log.Warn/log.Debug at the CALLER recovery site, do NOT move logging into the guard — the guard has no visibility into the package being built (loses pkg.PkgPath diagnostic context) and importing a logger would violate the stdlib-only constraint. (4) Document the ssa.BuildSerially caller precondition in GoDoc as documentation-only (NOT runtime validation) — mode flags are set by callers before ssautil.AllPackages, outside the guard's scope; recover() is goroutine-scoped so omitting BuildSerially causes silent panic-escape/process crash (spec 033).
