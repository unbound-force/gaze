---
tag: test-quality
author: jay-flowers
category: gotcha
created_at: 2026-11-02T22:00:36Z
identity: test-quality-20261102T220036-jay-flowers
tier: draft
---

Indirect coverage through golden-file or template assertions only locks the sub-cases the fixtures actually contain. When a pure formatting helper implements character-collision rules (CommonMark code spans: the delimiter run must exceed the longest internal backtick run, and content starting or ending with a backtick requires space padding), grep the fixtures for the boundary character before trusting indirect coverage — in gaze PR #314 the fixture name "parse`raw`Literal" locked single-internal-backtick widening while the space-padding branch stayed suite-unreachable (no fixture begins or ends with a backtick), so deleting the branch passed CI. Guards: add direct table-driven unit tests for every collision rule in the same change as the helper, and never use the helper itself as the expected value in a strings.Contains assertion — that oracle is circular and cannot detect a helper regression.
