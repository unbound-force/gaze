---
tag: test-quality
author: jay-flowers
category: pattern
created_at: 2026-11-25T03:27:54Z
identity: test-quality-20261125T032754-jay-flowers
tier: draft
as_of: "gaze@73bb94085f82edd0182836a17ed2e0669856cc02"
---

When a flag's correct handling routes to a code path whose failure mode is error EA and its previous (buggy) handling routes to a path failing with error EB, the regression test should assert BOTH that the error contains EA AND that it does not contain EB. The negative assertion is the discriminator proving the old path was not taken; without it, a partial revert producing any third error still passes an "error is non-nil" bar. Gaze's TestXxxWithLanguageOnly_TriggersExternalPath family (cmd/gaze/external_analyzer_test.go, crap/quality/report variants) applies this to pin gate conditions: asserting "no analyzer found for language" present plus "finding module root" absent proves the external-analyzer gate (not the Go-native module-root gate) produced the failure — and a single such test pins both the dispatch gate and the module-dir-resolution gate, because each gate's non-widening yields a distinguishable error (module-root error vs. a Go-native loader error on the empty fixture). Caveat discovered while reviewing the report variant: the family asserts the error class but not the sentinel language name, so the "error names the requested language" contract clause is unpinned everywhere — when copying the pattern, also assert the sentinel appears in the message.
