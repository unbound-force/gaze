---
tag: subprocess-testing
author: jay-flowers
category: pattern
created_at: 2026-11-09T23:16:44Z
identity: subprocess-testing-20261109T231644-jay-flowers
tier: draft
---

To assert that production code created no temp file (a guard that must short-circuit before file creation), redirect TMPDIR via t.Setenv to a dedicated t.TempDir() captured on the FIRST t.TempDir() call, then assert os.ReadDir on that directory is empty afterward. Two mechanics make this work: (1) os.CreateTemp honors the current TMPDIR at call time, so a guard placed after the temp-file step is caught by the non-empty ReadDir; (2) Go's t.TempDir() creates its numbered base directory under the TMPDIR that w
