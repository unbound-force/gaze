---
tag: agent-delegation
author: jay-flowers
category: pattern
created_at: 2026-11-24T21:38:52Z
identity: agent-delegation-20261124T213852-jay-flowers
tier: draft
as_of: "gaze@1212e88ae47744ece19a5f000002fa702c7e919c"
---

Bundling many review fixes into one large uf_delegate_command prompt reliably fails with child_empty_response (finish reason: length) — observed three times in one uf.unleash run (an implementation task's first attempt, a testability-fix delegation that timed out at 600s after completing its edits, and a combined A-F fix delegation). The reliable pattern: one focused delegation per file or per small fix cluster, a compact prompt with the exact spec inline (existing helpers, scenario table, verification commands), an explicit short-reply instruction (under ~15 lines, no file dumps — long final replies are themselves a finish-reason-length failure mode), and a wall timeout of 900s or more. Simple edits (YAML, markdown, .gitignore) are faster and safer applied directly by the orchestrator, and every delegation's work is verified independently: read the diff, re-run the tests, and scratch-mutate the new pins to confirm they bite.
