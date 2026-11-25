---
tag: review-insight
author: jay-flowers
category: pattern
created_at: 2026-11-25T03:28:02Z
identity: review-insight-20261125T032802-jay-flowers
tier: draft
as_of: "gaze@73bb94085f82edd0182836a17ed2e0669856cc02"
evidence:
  - "cmd/gaze/external_analyzer_test.go"
  - "openspec/changes/report-language-dispatch/specs/report-language-dispatch/spec.md"
---

The code-review council verdict for opsx/report-language-dispatch (gaze report --language dispatch, APPROVE WITH ADVISORIES, 2026-11-25) carried two LOW findings with a single shared root cause: the regression test TestReportWithLanguageOnly_TriggersExternalPath mirrored its sibling tests (TestCrapWithLanguageOnly_TriggersExternalPath, TestQualityWithLanguageOnly_TriggersExternalPath) assertion-for-assertion, and the mirroring faithfully copied the siblings' coverage gaps. First, the spec scenario "language-only inside a Go module" has no test executing its WHEN clause — the test only chdirs to a bare temp dir, so a future change re-routing dispatch on go.mod presence would silently reintroduce the inert-flag bug for module users; a companion test chdir'ing into a temp dir with a minimal go.mod (the TestReportWithExternalAnalyzer_DocscanPopulated fixture pattern at lines 867-871) would pin it. Second, the assertion checks the error class ("no analyzer found for language") but never the sentinel language name, so the spec's "names the requested language" clause is unpinned across all three sibling tests. The generalizable rule: when a spec enumerates scenarios and contract clauses, diff them against the mirrored test's executed WHEN clauses and asserted strings — mirroring a sibling test inherits its gaps as reliably as its coverage, and the cheap fixes (one extra fixture, one extra strings.Contains on a sentinel already in the test) come directly out of that diff.
