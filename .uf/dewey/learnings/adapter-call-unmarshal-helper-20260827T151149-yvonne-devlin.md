---
tag: adapter-call-unmarshal-helper
author: yvonne-devlin
category: gotcha
created_at: 2026-08-27T15:11:49Z
identity: adapter-call-unmarshal-helper-20260827T151149-yvonne-devlin
tier: draft
---

Go test-binary constraint: only ONE TestMain is allowed per compiled test binary. In internal/adapter/ the existing TestMain lives in `package adapter_test` (external test package) and builds+caches the fake analyzer binary. A NEW internal-package test file (`package adapter`, required to access unexported symbols like callAndUnmarshal) compiles into the SAME test binary and therefore CANNOT declare its own TestMain, nor can it reference the external package's unexported fakeBinaryPath var. Resolution: the internal test file builds its OWN fake-analyzer copy lazily via sync.Once (package-level vars: binaryPath string, buildOnce sync.Once, buildErr error). GOTCHA: this sync.Once-based lazy build canNOT use t.Cleanup for temp-dir teardown — Once.Do runs during the FIRST SUBTEST that calls it and captures that subtest's *testing.T, so the cleanup fires when that first subtest ends, deleting the shared binary before later subtests run (observed failure: "analyzer binary ... not found" in a later subtest). Accept the small (~4-5MB) OS-temp leak instead; there is no package-level teardown hook available without a TestMain.
