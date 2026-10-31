---
tag: spec-review
author: jay-flowers
category: pattern
created_at: 2026-10-31T15:11:08Z
identity: spec-review-20261031T151108-jay-flowers
tier: draft
---

Spec-review technique (Guard, specs mode): build an explicit scenario-to-task map before verdicting — for every GIVEN/WHEN/THEN scenario in spec.md, locate the tasks.md item that VERIFIES it, not merely the one that implements it. The recurring under-delivery gap is requirements whose scenarios have implementation tasks but no assertion tasks: fallback/warning branches of newly introduced parsers, console-output requirements, and suppression conditions ("section MUST NOT print when X"). In the gaze init-ci-scaffold spec review (2026-10-31), the go.mod-directive fallback scenario and the CI next-steps output requirement both had implement tasks but zero verification tasks, while the proposal's Testability PASS claimed matrix coverage — caught only by mapping each scenario to a phase-3 test task one by one.
