---
tag: path-handling
author: jay-flowers
category: gotcha
created_at: 2026-11-15T06:02:21Z
identity: path-handling-20261115T060221-jay-flowers
tier: draft
as_of: "gaze@bff29c8a934970cf9f74962fe0a8b3f3a9b326a6"
evidence:
  - "internal/crap/baseline_compat.go#escapesBase"
  - "internal/crap/baseline_compat_test.go#dot-dot-prefixed-directory-name-is-not-an-escape"
---

When relativizing paths against a base directory, filepath.Rel succeeds for sibling directories (filepath.Rel("/a/mod", "/a/other/x.go") yields "../other/x.go"), so escape rejection must test for exactly ".." or a separator-suffixed "../" prefix — never a bare ".." prefix, which wrongly rejects in-base directories merely named "..foo". Pin the predicate with a counter-case test using a directory literally named "..foo": the simplification to strings.HasPrefix(rel, "..") is the likely future refactor that breaks it.
