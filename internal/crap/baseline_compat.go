package crap

import (
	"fmt"
	"path/filepath"
	"strings"
)

// maxBaselinePathExamples bounds how many offending paths a
// BaselinePathError carries. Examples are illustrative material for
// diagnostics; the full extent of the problem is conveyed by Count.
const maxBaselinePathExamples = 3

// ReconciliationStats reports the outcome of a baseline path
// reconciliation pass.
type ReconciliationStats struct {
	// Normalized is the number of baseline score entries whose absolute
	// file paths were rewritten to module-relative form. Entries that
	// were already relative (or empty) are not counted.
	Normalized int
}

// BaselinePathError reports baseline score entries whose absolute file
// paths could not be reconciled against either the current module
// directory or the baseline root. It carries the total count of
// unreconcilable entries plus a bounded sample of example paths so the
// CLI layer can render an actionable diagnostic (design D3: the CLI
// alone knows the baseline file path and composes the full message).
type BaselinePathError struct {
	// Count is the total number of baseline entries whose absolute
	// paths lie under neither the module directory nor the baseline
	// root.
	Count int

	// Examples holds up to maxBaselinePathExamples of the offending
	// absolute paths, in baseline order.
	Examples []string
}

// Error returns a self-describing diagnostic naming the unreconcilable
// entry count and, when available, the first example path. The CLI
// layer wraps this message with the baseline file path and both
// remediations (regenerate the baseline, or re-run with
// --baseline-root) rather than the error carrying them itself.
func (e *BaselinePathError) Error() string {
	if len(e.Examples) == 0 {
		return fmt.Sprintf("%d baseline entries use absolute paths outside the current module", e.Count)
	}
	return fmt.Sprintf("%d baseline entries use absolute paths outside the current module (e.g. %s)", e.Count, e.Examples[0])
}

// ReconcileBaselinePaths rewrites absolute baseline Score.File paths to
// the module-relative form used by current reports, so that
// file:function keys match across gaze versions. Baselines saved before
// path relativization was introduced carry absolute paths while current
// reports carry module-relative paths (issue #330).
//
// Each baseline score entry is processed in order:
//
//  1. Empty or non-absolute File values are untouched (already in the
//     module-relative representation).
//  2. Absolute paths are rewritten with filepath.Rel(moduleDir, ...)
//     when the result stays within moduleDir.
//  3. Otherwise, when baselineRoot is non-empty, absolute paths are
//     rewritten with filepath.Rel(baselineRoot, ...) when the result
//     stays within baselineRoot.
//  4. Entries matched by neither rule are unreconcilable.
//
// A relative path escapes its base — and the rewrite is rejected — when
// it is ".." or ".."-prefixed. The escape test is required because
// filepath.Rel succeeds with "../x.go" for sibling directories (e.g.
// filepath.Rel("/a/mod", "/a/other/x.go") yields "../other/x.go"), and
// such a rewrite would fabricate a path that does not exist in the
// module.
//
// The root order (moduleDir first, then baselineRoot) is deterministic;
// when a path lies under both roots, the moduleDir rewrite wins.
//
// The baseline Report is normalized in place and must be a freshly
// loaded value with no other holder (design R5). Only the File field is
// modified; CRAP and GazeCRAP values and all other score semantics are
// preserved (design D5).
//
// On success, stats.Normalized reports the number of rewrites and err is
// nil. When one or more entries are unreconcilable, err is a
// *BaselinePathError carrying the count and example paths; entries
// processed before the failure may already have been rewritten, so the
// caller must discard the baseline. A nil baseline is a no-op returning
// zero stats and a nil error.
//
// The function performs no I/O and reads no global state; it is
// deterministic for a given (baseline, moduleDir, baselineRoot) input.
func ReconcileBaselinePaths(baseline *Report, moduleDir, baselineRoot string) (ReconciliationStats, error) {
	var stats ReconciliationStats
	if baseline == nil {
		return stats, nil
	}

	var unreconcilable []string
	for i := range baseline.Scores {
		file := baseline.Scores[i].File
		if file == "" || !filepath.IsAbs(file) {
			// Already module-relative (or unspecified): nothing to do.
			continue
		}
		if rel, ok := relativizeWithin(moduleDir, file); ok {
			baseline.Scores[i].File = rel
			stats.Normalized++
			continue
		}
		if baselineRoot != "" {
			if rel, ok := relativizeWithin(baselineRoot, file); ok {
				baseline.Scores[i].File = rel
				stats.Normalized++
				continue
			}
		}
		unreconcilable = append(unreconcilable, file)
	}

	if len(unreconcilable) > 0 {
		return stats, &BaselinePathError{
			Count:    len(unreconcilable),
			Examples: boundedExamples(unreconcilable),
		}
	}
	return stats, nil
}

// relativizeWithin reports whether file lies under base, returning
// file's path relative to base. It reports false when file cannot be
// made relative to base or when the result would escape base (design
// D2: both conditions mean file does not lie under base).
func relativizeWithin(base, file string) (string, bool) {
	rel, err := filepath.Rel(base, file)
	if err != nil {
		return "", false
	}
	if escapesBase(rel) {
		return "", false
	}
	return rel, true
}

// escapesBase reports whether rel — a path expressed relative to some
// base directory — refers outside that base. A result of exactly ".."
// or beginning with "../" escapes; a directory merely named "..foo"
// does not, which is why the separator-suffixed prefix test is used.
func escapesBase(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// boundedExamples returns at most maxBaselinePathExamples paths from
// paths, preserving order.
func boundedExamples(paths []string) []string {
	if len(paths) <= maxBaselinePathExamples {
		return paths
	}
	return paths[:maxBaselinePathExamples]
}
