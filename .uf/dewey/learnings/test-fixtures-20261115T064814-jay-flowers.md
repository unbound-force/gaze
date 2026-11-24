---
tag: test-fixtures
author: jay-flowers
category: gotcha
created_at: 2026-11-15T06:48:14Z
identity: test-fixtures-20261115T064814-jay-flowers
tier: draft
as_of: "gaze@7965494232385e920cd63a717a79469c8b7eac5e"
---

A suppression assertion ("X must not appear in output Y") is vacuous when the test fixture cannot produce X. In gaze PR #345, TestRunCrap_ExplicitIncompatibleBaseline asserted that the normalization notice is suppressed on the baseline-incompatibility error path, but its fixture was a single fully-unreconcilable baseline entry — reconciliation failed with Normalized == 0, so the notice could not fire even if the print were wrongly moved before the error check; the assertion could never fail. Fix: make the fixture mixed (one reconcilable + one unreconcilable entry) so the suppressed output is genuinely producible and the assertion has teeth.
