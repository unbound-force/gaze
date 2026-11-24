package crap

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"
)

// mkScore builds a minimal Score carrying only a file path, for
// synthetic baseline reports. Reconciliation reads File only.
func mkScore(file string) Score {
	return Score{Package: "pkg", Function: "Func", File: file}
}

// mkReport builds a baseline Report whose scores carry the given file
// paths, in order.
func mkReport(files ...string) *Report {
	scores := make([]Score, len(files))
	for i, file := range files {
		scores[i] = mkScore(file)
	}
	return &Report{Scores: scores}
}

// TestReconcileBaselinePaths exercises the design D7 reconciliation
// matrix with synthetic baseline reports and t.TempDir roots, asserting
// exact rewritten strings rather than just error presence (task 1.2).
func TestReconcileBaselinePaths(t *testing.T) {
	// Roots are created up front so table cases can assert against the
	// exact absolute paths they construct. Each t.TempDir call returns
	// a distinct sibling directory, so no root lies under another
	// except where a case explicitly builds one (nested-roots case).
	moduleDir := t.TempDir()
	parent := t.TempDir()
	baselineRoot := t.TempDir()
	elsewhere := t.TempDir()
	nestedRoot := t.TempDir()

	siblingModuleDir := filepath.Join(parent, "mod")
	nestedModuleDir := filepath.Join(nestedRoot, "mod")

	tests := []struct {
		name           string
		baseline       *Report
		moduleDir      string
		baselineRoot   string
		wantNormalized int
		wantFiles      []string // expected Score.File values, in order (success cases)
		wantErrCount   int      // 0 => success; > 0 => *BaselinePathError.Count
		wantExamples   []string // expected BaselinePathError.Examples (error cases)
		wantUntouched  []string // paths that must remain verbatim (error cases)
	}{
		{
			name:           "relative path untouched",
			baseline:       mkReport("internal/crap/analyze.go"),
			moduleDir:      moduleDir,
			wantNormalized: 0,
			wantFiles:      []string{"internal/crap/analyze.go"},
		},
		{
			name:           "absolute under moduleDir rewritten",
			baseline:       mkReport(filepath.Join(moduleDir, "cmd/gaze/main.go")),
			moduleDir:      moduleDir,
			wantNormalized: 1,
			wantFiles:      []string{"cmd/gaze/main.go"},
		},
		{
			// The sibling file relativizes to ../other/x.go and the
			// parent directory to exactly ".." — both escape forms.
			name:          "absolute sibling of moduleDir rejected via escape test",
			baseline:      mkReport(filepath.Join(parent, "other", "x.go"), parent),
			moduleDir:     siblingModuleDir,
			wantErrCount:  2,
			wantExamples:  []string{filepath.Join(parent, "other", "x.go"), parent},
			wantUntouched: []string{filepath.Join(parent, "other", "x.go"), parent},
		},
		{
			// The non-escaping counter-case the escapesBase doc
			// comment documents: a path component merely named with
			// a ".." prefix (a "..foo" directory) is inside the base
			// and must reconcile. Pins the separator-suffixed
			// prefix ("../") against a future simplification to a
			// bare ".." prefix that would wrongly reject it.
			name:           "dot-dot-prefixed directory name is not an escape",
			baseline:       mkReport(filepath.Join(moduleDir, "..foo", "x.go")),
			moduleDir:      moduleDir,
			wantNormalized: 1,
			wantFiles:      []string{filepath.Join("..foo", "x.go")},
		},
		{
			name:           "absolute under baselineRoot rewritten",
			baseline:       mkReport(filepath.Join(baselineRoot, "pkg/a/util.go")),
			moduleDir:      moduleDir,
			baselineRoot:   baselineRoot,
			wantNormalized: 1,
			wantFiles:      []string{"pkg/a/util.go"},
		},
		{
			name:          "absolute under neither returns BaselinePathError",
			baseline:      mkReport(filepath.Join(elsewhere, "a.go"), filepath.Join(elsewhere, "b.go")),
			moduleDir:     moduleDir,
			wantErrCount:  2,
			wantExamples:  []string{filepath.Join(elsewhere, "a.go"), filepath.Join(elsewhere, "b.go")},
			wantUntouched: []string{filepath.Join(elsewhere, "a.go"), filepath.Join(elsewhere, "b.go")},
		},
		{
			name:           "empty file untouched",
			baseline:       mkReport("", "cmd/gaze/main.go"),
			moduleDir:      moduleDir,
			wantNormalized: 0,
			wantFiles:      []string{"", "cmd/gaze/main.go"},
		},
		{
			name:           "empty baseline is a no-op",
			baseline:       &Report{},
			moduleDir:      moduleDir,
			wantNormalized: 0,
		},
		{
			name:           "nil baseline is a no-op",
			baseline:       nil,
			moduleDir:      moduleDir,
			wantNormalized: 0,
		},
		{
			name:           "mixed relative and absolute entries",
			baseline:       mkReport("internal/crap/analyze.go", filepath.Join(moduleDir, "cmd/gaze/main.go"), "cmd/gaze/x.go"),
			moduleDir:      moduleDir,
			wantNormalized: 1,
			wantFiles:      []string{"internal/crap/analyze.go", "cmd/gaze/main.go", "cmd/gaze/x.go"},
		},
		{
			name:           "normalized counts every rewrite",
			baseline:       mkReport(filepath.Join(moduleDir, "a.go"), filepath.Join(moduleDir, "b.go"), filepath.Join(moduleDir, "c.go")),
			moduleDir:      moduleDir,
			wantNormalized: 3,
			wantFiles:      []string{"a.go", "b.go", "c.go"},
		},
		{
			// moduleDir is tried before baselineRoot even when the
			// path lies under both: the file rewrites to "x.go", not
			// "mod/x.go".
			name:           "nested roots prefer moduleDir",
			baseline:       mkReport(filepath.Join(nestedRoot, "mod", "x.go")),
			moduleDir:      nestedModuleDir,
			baselineRoot:   nestedRoot,
			wantNormalized: 1,
			wantFiles:      []string{"x.go"},
		},
		{
			// Only the first maxBaselinePathExamples offending paths
			// are carried as examples; Count still reports all four.
			name:          "unreconcilable examples are bounded",
			baseline:      mkReport(filepath.Join(elsewhere, "p.go"), filepath.Join(elsewhere, "q.go"), filepath.Join(elsewhere, "r.go"), filepath.Join(elsewhere, "s.go")),
			moduleDir:     moduleDir,
			wantErrCount:  4,
			wantExamples:  []string{filepath.Join(elsewhere, "p.go"), filepath.Join(elsewhere, "q.go"), filepath.Join(elsewhere, "r.go")},
			wantUntouched: []string{filepath.Join(elsewhere, "p.go"), filepath.Join(elsewhere, "q.go"), filepath.Join(elsewhere, "r.go"), filepath.Join(elsewhere, "s.go")},
		},
		{
			// A relative moduleDir cannot relativize an absolute path
			// (filepath.Rel fails); the entry is unreconcilable rather
			// than mis-normalized.
			name:          "relative moduleDir is not a usable base",
			baseline:      mkReport(filepath.Join(moduleDir, "x.go")),
			moduleDir:     "mod",
			wantErrCount:  1,
			wantExamples:  []string{filepath.Join(moduleDir, "x.go")},
			wantUntouched: []string{filepath.Join(moduleDir, "x.go")},
		},
		{
			name:          "absolute under neither with baselineRoot set",
			baseline:      mkReport(filepath.Join(elsewhere, "z.go")),
			moduleDir:     moduleDir,
			baselineRoot:  baselineRoot,
			wantErrCount:  1,
			wantExamples:  []string{filepath.Join(elsewhere, "z.go")},
			wantUntouched: []string{filepath.Join(elsewhere, "z.go")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := ReconcileBaselinePaths(tt.baseline, tt.moduleDir, tt.baselineRoot)

			if tt.wantErrCount == 0 {
				if err != nil {
					t.Fatalf("ReconcileBaselinePaths() error = %v, want nil", err)
				}
				if stats.Normalized != tt.wantNormalized {
					t.Errorf("Normalized = %d, want %d", stats.Normalized, tt.wantNormalized)
				}
				if tt.baseline == nil {
					return
				}
				if len(tt.wantFiles) != len(tt.baseline.Scores) {
					t.Fatalf("len(baseline.Scores) = %d, want %d",
						len(tt.baseline.Scores), len(tt.wantFiles))
				}
				for i, want := range tt.wantFiles {
					if got := tt.baseline.Scores[i].File; got != want {
						t.Errorf("Scores[%d].File = %q, want %q", i, got, want)
					}
				}
				return
			}

			if err == nil {
				t.Fatal("ReconcileBaselinePaths() error = nil, want *BaselinePathError")
			}
			var pathErr *BaselinePathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("ReconcileBaselinePaths() error = %T (%v), want *BaselinePathError", err, err)
			}
			if pathErr.Count != tt.wantErrCount {
				t.Errorf("Count = %d, want %d", pathErr.Count, tt.wantErrCount)
			}
			if !slices.Equal(pathErr.Examples, tt.wantExamples) {
				t.Errorf("Examples = %v, want %v", pathErr.Examples, tt.wantExamples)
			}
			for _, want := range tt.wantUntouched {
				kept := false
				for _, s := range tt.baseline.Scores {
					if s.File == want {
						kept = true
						break
					}
				}
				if !kept {
					t.Errorf("unreconcilable path %q was rewritten; want it left verbatim", want)
				}
			}
		})
	}
}

// TestReconcileBaselinePaths_MixedPartialRewrite verifies the documented
// error-path contract: with a baseline mixing one reconcilable and one
// unreconcilable entry, reconciliation fails with *BaselinePathError, the
// reconcilable entry has already been rewritten in place, the
// unreconcilable entry is left verbatim, and the returned stats carry the
// partial rewrite count alongside the error. The caller must discard the
// baseline on error (the CLI suppresses the normalization notice for
// exactly this reason).
func TestReconcileBaselinePaths_MixedPartialRewrite(t *testing.T) {
	moduleDir := t.TempDir()
	elsewhere := t.TempDir()
	baseline := mkReport(
		filepath.Join(moduleDir, "cmd/gaze/main.go"),
		filepath.Join(elsewhere, "a.go"),
	)

	stats, err := ReconcileBaselinePaths(baseline, moduleDir, "")

	if err == nil {
		t.Fatal("ReconcileBaselinePaths() error = nil, want *BaselinePathError")
	}
	var pathErr *BaselinePathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("ReconcileBaselinePaths() error = %T (%v), want *BaselinePathError", err, err)
	}
	if pathErr.Count != 1 {
		t.Errorf("Count = %d, want 1", pathErr.Count)
	}
	if stats.Normalized != 1 {
		t.Errorf("stats.Normalized = %d, want 1 (partial rewrite visible on the error path)", stats.Normalized)
	}
	if got, want := baseline.Scores[0].File, "cmd/gaze/main.go"; got != want {
		t.Errorf("reconcilable entry File = %q, want rewritten %q (caller discards the baseline on error)", got, want)
	}
	if got, want := baseline.Scores[1].File, filepath.Join(elsewhere, "a.go"); got != want {
		t.Errorf("unreconcilable entry File = %q, want verbatim %q", got, want)
	}
}

// TestReconcileCompare_* (tasks 2.1-2.5) are comparison-level regression
// tests that feed synthetic baseline reports with absolute paths through
// ReconcileBaselinePaths and then Compare, asserting the normalized shape
// rather than just error presence. Task 2.1 covers the reconciled
// absolute→relative match; 2.2 the unchanged high-CRAP case above the new
// threshold; 2.3 same-basename disambiguation; 2.4 relocated checkouts
// via --baseline-root; and 2.5 genuinely new/removed detection after
// reconciliation.

func TestReconcileCompare_AbsoluteMatchesRelative(t *testing.T) {
	moduleDir := t.TempDir()
	absPath := filepath.Join(moduleDir, "pkg", "foo.go")
	baseline := &Report{Scores: []Score{makeScore(absPath, "Func", 10, nil)}}
	current := &Report{Scores: []Score{makeScore("pkg/foo.go", "Func", 10, nil)}}

	stats, err := ReconcileBaselinePaths(baseline, moduleDir, "")
	if err != nil {
		t.Fatalf("ReconcileBaselinePaths() error = %v, want nil", err)
	}
	if stats.Normalized != 1 {
		t.Fatalf("Normalized = %d, want 1", stats.Normalized)
	}

	result := Compare(baseline, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(result.Deltas))
	}
	if result.Deltas[0].Status != StatusUnchanged {
		t.Errorf("Status = %v, want %v", result.Deltas[0].Status, StatusUnchanged)
	}
	if len(result.NewFunctions) != 0 {
		t.Errorf("NewFunctions = %d, want 0", len(result.NewFunctions))
	}
	if len(result.RemovedFunctions) != 0 {
		t.Errorf("RemovedFunctions = %d, want 0", len(result.RemovedFunctions))
	}
	if !result.Summary.Passed {
		t.Errorf("Passed = false, want true")
	}
	if result.Summary.Unchanged != 1 {
		t.Errorf("Summary.Unchanged = %d, want 1", result.Summary.Unchanged)
	}
}

func TestReconcileCompare_UnchangedScoreAboveThreshold(t *testing.T) {
	moduleDir := t.TempDir()
	absPath := filepath.Join(moduleDir, "pkg", "foo.go")
	baseline := &Report{Scores: []Score{makeScore(absPath, "Func", 72, nil)}}
	current := &Report{Scores: []Score{makeScore("pkg/foo.go", "Func", 72, nil)}}

	stats, err := ReconcileBaselinePaths(baseline, moduleDir, "")
	if err != nil {
		t.Fatalf("ReconcileBaselinePaths() error = %v, want nil", err)
	}
	if stats.Normalized != 1 {
		t.Fatalf("Normalized = %d, want 1", stats.Normalized)
	}

	result := Compare(baseline, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1", len(result.Deltas))
	}
	if result.Deltas[0].Status != StatusUnchanged {
		t.Errorf("Status = %v, want %v", result.Deltas[0].Status, StatusUnchanged)
	}
	if result.Summary.NewViolations != 0 {
		t.Errorf("NewViolations = %d, want 0 (unchanged high CRAP must not become a violation)", result.Summary.NewViolations)
	}
	if result.Summary.Regressions != 0 {
		t.Errorf("Regressions = %d, want 0", result.Summary.Regressions)
	}
	if !result.Summary.Passed {
		t.Errorf("Passed = false, want true (unchanged function above threshold is not a gate failure)")
	}
}

func TestReconcileCompare_SameBasenameDistinctFiles(t *testing.T) {
	moduleDir := t.TempDir()
	baseline := &Report{Scores: []Score{
		makeScore(filepath.Join(moduleDir, "pkg", "a", "util.go"), "Helper", 5, nil),
		makeScore(filepath.Join(moduleDir, "pkg", "b", "util.go"), "Helper", 5, nil),
	}}
	current := &Report{Scores: []Score{
		makeScore(filepath.Join("pkg", "a", "util.go"), "Helper", 5, nil),
		makeScore(filepath.Join("pkg", "b", "util.go"), "Helper", 5, nil),
	}}

	stats, err := ReconcileBaselinePaths(baseline, moduleDir, "")
	if err != nil {
		t.Fatalf("ReconcileBaselinePaths() error = %v, want nil", err)
	}
	if stats.Normalized != 2 {
		t.Fatalf("Normalized = %d, want 2", stats.Normalized)
	}

	result := Compare(baseline, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if len(result.Deltas) != 2 {
		t.Fatalf("len(Deltas) = %d, want 2", len(result.Deltas))
	}
	if len(result.NewFunctions) != 0 {
		t.Errorf("NewFunctions = %d, want 0 (no cross-directory conflation)", len(result.NewFunctions))
	}
	if len(result.RemovedFunctions) != 0 {
		t.Errorf("RemovedFunctions = %d, want 0", len(result.RemovedFunctions))
	}
	if result.Summary.Unchanged != 2 {
		t.Errorf("Summary.Unchanged = %d, want 2", result.Summary.Unchanged)
	}
	if !result.Summary.Passed {
		t.Errorf("Passed = false, want true")
	}
}

func TestReconcileCompare_RelocatedCheckout(t *testing.T) {
	moduleDir := t.TempDir()
	oldRoot := t.TempDir()
	absPath := filepath.Join(oldRoot, "pkg", "foo.go")

	// With baselineRoot set to the old checkout root, reconciliation
	// succeeds and the comparison passes.
	baselineOK := &Report{Scores: []Score{makeScore(absPath, "Func", 10, nil)}}
	current := &Report{Scores: []Score{makeScore("pkg/foo.go", "Func", 10, nil)}}
	stats, err := ReconcileBaselinePaths(baselineOK, moduleDir, oldRoot)
	if err != nil {
		t.Fatalf("ReconcileBaselinePaths(baselineRoot) error = %v, want nil", err)
	}
	if stats.Normalized != 1 {
		t.Fatalf("Normalized = %d, want 1", stats.Normalized)
	}
	result := Compare(baselineOK, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if len(result.Deltas) != 1 || result.Deltas[0].Status != StatusUnchanged {
		t.Fatalf("with baselineRoot: Deltas = %v, want 1 unchanged", result.Deltas)
	}
	if !result.Summary.Passed {
		t.Errorf("with baselineRoot: Passed = false, want true")
	}

	// With an empty baselineRoot the same absolute path is
	// unreconcilable and must surface as a BaselinePathError.
	baselineErr := &Report{Scores: []Score{makeScore(absPath, "Func", 10, nil)}}
	_, err = ReconcileBaselinePaths(baselineErr, moduleDir, "")
	if err == nil {
		t.Fatal("ReconcileBaselinePaths(empty baselineRoot) error = nil, want *BaselinePathError")
	}
	var pathErr *BaselinePathError
	if !errors.As(err, &pathErr) {
		t.Fatalf("error = %T (%v), want *BaselinePathError", err, err)
	}
	if pathErr.Count != 1 {
		t.Errorf("Count = %d, want 1", pathErr.Count)
	}
	if !slices.Equal(pathErr.Examples, []string{absPath}) {
		t.Errorf("Examples = %v, want [%q]", pathErr.Examples, absPath)
	}
}

func TestReconcileCompare_RealNewAndRemoved(t *testing.T) {
	moduleDir := t.TempDir()
	baseline := &Report{Scores: []Score{
		makeScore(filepath.Join(moduleDir, "pkg", "common.go"), "CommonFunc", 10, nil),
		makeScore(filepath.Join(moduleDir, "pkg", "old.go"), "OldFunc", 5, nil),
	}}
	current := &Report{Scores: []Score{
		makeScore(filepath.Join("pkg", "common.go"), "CommonFunc", 10, nil),
		makeScore(filepath.Join("pkg", "new.go"), "NewFunc", 42, nil),
	}}

	stats, err := ReconcileBaselinePaths(baseline, moduleDir, "")
	if err != nil {
		t.Fatalf("ReconcileBaselinePaths() error = %v, want nil", err)
	}
	if stats.Normalized != 2 {
		t.Fatalf("Normalized = %d, want 2 (common + old)", stats.Normalized)
	}

	result := Compare(baseline, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if len(result.Deltas) != 1 {
		t.Fatalf("len(Deltas) = %d, want 1 (CommonFunc)", len(result.Deltas))
	}
	if result.Deltas[0].Status != StatusUnchanged {
		t.Errorf("delta Status = %v, want %v", result.Deltas[0].Status, StatusUnchanged)
	}
	if len(result.NewFunctions) != 1 {
		t.Fatalf("len(NewFunctions) = %d, want 1 (NewFunc)", len(result.NewFunctions))
	}
	if result.Summary.NewViolations != 1 {
		t.Errorf("NewViolations = %d, want 1 (CRAP 42 > 30)", result.Summary.NewViolations)
	}
	if len(result.RemovedFunctions) != 1 {
		t.Fatalf("len(RemovedFunctions) = %d, want 1 (OldFunc)", len(result.RemovedFunctions))
	}
	if result.Summary.RemovedFunctions != 1 {
		t.Errorf("Summary.RemovedFunctions = %d, want 1", result.Summary.RemovedFunctions)
	}
	if result.Summary.Passed {
		t.Errorf("Passed = true, want false (new violation must fail the gate exactly as with a module-relative baseline)")
	}

	// Module-relative baseline must produce the same gate outcome.
	relativeBaseline := &Report{Scores: []Score{
		makeScore(filepath.Join("pkg", "common.go"), "CommonFunc", 10, nil),
		makeScore(filepath.Join("pkg", "old.go"), "OldFunc", 5, nil),
	}}
	relativeResult := Compare(relativeBaseline, current, CompareOptions{Epsilon: 0.5, NewFunctionThreshold: 30})
	if relativeResult.Summary.Passed != result.Summary.Passed || relativeResult.Summary.NewViolations != result.Summary.NewViolations {
		t.Errorf("reconciled vs relative baseline diverge: reconciled Passed=%v NewViolations=%d, relative Passed=%v NewViolations=%d",
			result.Summary.Passed, result.Summary.NewViolations, relativeResult.Summary.Passed, relativeResult.Summary.NewViolations)
	}
}

// TestBaselinePathError_Error pins the self-describing diagnostic
// format of the typed error for both example-carrying and count-only
// construction.
func TestBaselinePathError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *BaselinePathError
		want string
	}{
		{
			name: "count and first example path",
			err: &BaselinePathError{
				Count:    12,
				Examples: []string{"/home/alice/gaze/internal/crap/analyze.go", "/home/alice/gaze/cmd/gaze/main.go"},
			},
			want: "12 baseline entries use absolute paths outside the current module (e.g. /home/alice/gaze/internal/crap/analyze.go)",
		},
		{
			name: "count only when no examples are carried",
			err:  &BaselinePathError{Count: 3},
			want: "3 baseline entries use absolute paths outside the current module",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
