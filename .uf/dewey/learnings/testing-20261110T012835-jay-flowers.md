---
tag: testing
author: jay-flowers
category: gotcha
created_at: 2026-11-10T01:28:35Z
identity: testing-20261110T012835-jay-flowers
tier: draft
---

In Go tests, pinning TMPDIR to the first t.TempDir() result and later asserting that directory is empty is sound: testing.T.TempDir creates a per-test root under the TMPDIR in effect at the FIRST call and returns a new numbered subdirectory per invocation, so later t.TempDir() calls land as siblings of the pinned directory, never inside it. Only direct os.CreateTemp/os.MkdirTemp("", ...) calls made during the pinned window write into the pinned dir. Gaze's nested-coverage guard tests (internal/provider/goprovider/coverage_test.go, PR #326) rely on exactly this property: after t.Setenv("TMPDIR", tmp) where tmp := t.TempDir(), asserting os.ReadDir(tmp) is empty proves the guard fired before createTempProfile ran, while the unrelated t.TempDir() passed as the moduleDir argument cannot pollute the assertion. Without this TempDir semantics, such an empty-dir assertion would be a false test.
