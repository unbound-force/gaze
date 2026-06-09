package crap

import (
	"encoding/json"
	"fmt"
	"io"
)

// comparisonScoreJSON is a local struct that merges a Score with
// comparison delta fields for JSON output. The crap.Score type is
// NOT modified — delta fields only appear in the comparison JSON
// output path (D3, SC-007).
type comparisonScoreJSON struct {
	// Inline all Score fields.
	Score

	// BaselineCRAP is the CRAP score from the baseline report.
	BaselineCRAP *float64 `json:"baseline_crap,omitempty"`

	// BaselineGazeCRAP is the GazeCRAP score from the baseline.
	BaselineGazeCRAP *float64 `json:"baseline_gaze_crap,omitempty"`

	// CRAPDelta is current CRAP minus baseline CRAP.
	CRAPDelta *float64 `json:"crap_delta,omitempty"`

	// GazeCRAPDelta is current GazeCRAP minus baseline GazeCRAP.
	GazeCRAPDelta *float64 `json:"gaze_crap_delta,omitempty"`

	// Status classifies this function's change direction.
	Status FunctionStatus `json:"status,omitempty"`
}

// newFunctionJSON is a local struct for new functions in comparison
// JSON output. Includes all Score fields plus a status field.
type newFunctionJSON struct {
	Score
	Status FunctionStatus `json:"status"`
}

// removedFunctionJSON is a local struct for removed functions in
// comparison JSON output. Includes all baseline Score fields plus
// a status field.
type removedFunctionJSON struct {
	Score
	Status FunctionStatus `json:"status"`
}

// comparisonOutputJSON is the top-level JSON envelope for comparison
// output. It assembles scores with delta annotations, new/removed
// function lists, comparison summary, and the normal CRAP summary.
type comparisonOutputJSON struct {
	Scores           []comparisonScoreJSON `json:"scores"`
	NewFunctions     []newFunctionJSON     `json:"new_functions"`
	RemovedFunctions []removedFunctionJSON `json:"removed_functions"`
	Comparison       ComparisonSummary     `json:"comparison"`
	Summary          Summary               `json:"summary"`
}

// WriteComparisonJSON writes a merged JSON output combining the
// current CRAP report scores with per-function comparison delta
// fields. This is a separate output path from WriteJSON — the CLI
// chooses which writer based on whether a baseline was loaded (D3).
//
// The JSON structure includes:
//   - scores: current scores enriched with baseline_crap, crap_delta,
//     baseline_gaze_crap, gaze_crap_delta, and status fields
//   - new_functions: functions not in baseline (status: new or new_violation)
//   - removed_functions: functions in baseline but not current (status: removed)
//   - comparison: aggregate counts and pass/fail
//   - summary: the normal CRAP summary from the current report
func WriteComparisonJSON(w io.Writer, result *ComparisonResult) error {
	// Build the delta lookup map for fast per-score enrichment.
	deltaMap := make(map[string]FunctionDelta, len(result.Deltas))
	for _, d := range result.Deltas {
		key := scoreKey(d.Current)
		deltaMap[key] = d
	}

	// Build the new-function set for status classification.
	newFuncSet := make(map[string]bool, len(result.NewFunctions))
	for _, nf := range result.NewFunctions {
		newFuncSet[scoreKey(nf)] = true
	}

	// Assemble enriched scores from the current report's scores.
	scores := make([]comparisonScoreJSON, 0, len(result.Report.Scores))
	for _, s := range result.Report.Scores {
		key := scoreKey(s)

		// Skip scores that are in the new-functions list — they
		// appear in the separate new_functions section.
		if newFuncSet[key] {
			continue
		}

		cs := comparisonScoreJSON{Score: s}

		if d, ok := deltaMap[key]; ok {
			baselineCRAP := d.Baseline.CRAP
			cs.BaselineCRAP = &baselineCRAP
			cs.CRAPDelta = &d.CRAPDelta

			if d.Baseline.GazeCRAP != nil {
				cs.BaselineGazeCRAP = d.Baseline.GazeCRAP
			}
			if d.GazeCRAPDelta != nil {
				cs.GazeCRAPDelta = d.GazeCRAPDelta
			}
			cs.Status = d.Status
		}

		scores = append(scores, cs)
	}

	// Assemble new functions with status.
	newFuncs := make([]newFunctionJSON, 0, len(result.NewFunctions))
	for _, nf := range result.NewFunctions {
		status := StatusNew
		if nf.CRAP > result.Summary.NewFunctionThreshold {
			status = StatusNewViolation
		}
		newFuncs = append(newFuncs, newFunctionJSON{
			Score:  nf,
			Status: status,
		})
	}

	// Assemble removed functions with status.
	removedFuncs := make([]removedFunctionJSON, 0, len(result.RemovedFunctions))
	for _, rf := range result.RemovedFunctions {
		removedFuncs = append(removedFuncs, removedFunctionJSON{
			Score:  rf,
			Status: StatusRemoved,
		})
	}

	output := comparisonOutputJSON{
		Scores:           scores,
		NewFunctions:     newFuncs,
		RemovedFunctions: removedFuncs,
		Comparison:       result.Summary,
		Summary:          result.Report.Summary,
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// WriteComparisonText writes the comparison report as human-readable
// text. It first writes the normal CRAP report via WriteText, then
// appends comparison-specific sections: summary line, counts,
// regressions table, improvements table, new functions list, and
// removed functions list. Empty sections are omitted (SC-008).
func WriteComparisonText(w io.Writer, result *ComparisonResult) error {
	// Write the base CRAP report first.
	if err := WriteText(w, result.Report); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w)

	// Comparison summary line: PASS or FAIL.
	passLabel := "PASS"
	if !result.Summary.Passed {
		passLabel = "FAIL"
	}
	_, _ = fmt.Fprintf(w, "--- Baseline Comparison: %s ---\n", passLabel)

	// Counts line.
	_, _ = fmt.Fprintf(w, "%d regressions, %d improvements, %d new, %d removed, %d unchanged\n",
		result.Summary.Regressions,
		result.Summary.Improvements,
		result.Summary.NewFunctions+result.Summary.NewViolations,
		result.Summary.RemovedFunctions,
		result.Summary.Unchanged,
	)

	// Regressions table.
	writeComparisonDeltaTable(w, "Regressions", result.Deltas, StatusRegression)

	// Improvements table.
	writeComparisonDeltaTable(w, "Improvements", result.Deltas, StatusImprovement)

	// New functions with violations.
	writeComparisonNewFunctions(w, result.NewFunctions, result.Summary.NewFunctionThreshold)

	// Removed functions.
	writeComparisonRemovedFunctions(w, result.RemovedFunctions)

	return nil
}

// writeComparisonDeltaTable writes a table of function deltas
// filtered by the given status. Omitted if no deltas match.
func writeComparisonDeltaTable(w io.Writer, title string, deltas []FunctionDelta, status FunctionStatus) {
	var matching []FunctionDelta
	for _, d := range deltas {
		if d.Status == status {
			matching = append(matching, d)
		}
	}
	if len(matching) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "\n%s:\n", title)
	_, _ = fmt.Fprintf(w, "  %-40s  %-10s  %-10s  %-10s\n",
		"Function", "Baseline", "Current", "Delta")
	for _, d := range matching {
		label := fmt.Sprintf("%s (%s)",
			d.Current.Function, shortenPath(d.Current.File))
		_, _ = fmt.Fprintf(w, "  %-40s  %-10.1f  %-10.1f  %+.1f\n",
			label, d.Baseline.CRAP, d.Current.CRAP, d.CRAPDelta)
	}
}

// writeComparisonNewFunctions writes the new functions section.
// Functions above the threshold are marked as violations.
// Omitted if there are no new functions.
func writeComparisonNewFunctions(w io.Writer, newFuncs []Score, threshold float64) {
	if len(newFuncs) == 0 {
		return
	}

	// Separate violations from informational new functions.
	var violations, informational []Score
	for _, s := range newFuncs {
		if s.CRAP > threshold {
			violations = append(violations, s)
		} else {
			informational = append(informational, s)
		}
	}

	if len(violations) > 0 {
		_, _ = fmt.Fprintf(w, "\nNew Functions (violations):\n")
		for _, s := range violations {
			_, _ = fmt.Fprintf(w, "  %-40s  CRAP: %.1f  [VIOLATION]\n",
				fmt.Sprintf("%s (%s)", s.Function, shortenPath(s.File)),
				s.CRAP)
		}
	}

	if len(informational) > 0 {
		_, _ = fmt.Fprintf(w, "\nNew Functions:\n")
		for _, s := range informational {
			_, _ = fmt.Fprintf(w, "  %-40s  CRAP: %.1f\n",
				fmt.Sprintf("%s (%s)", s.Function, shortenPath(s.File)),
				s.CRAP)
		}
	}
}

// writeComparisonRemovedFunctions writes the removed functions
// section. Omitted if no functions were removed.
func writeComparisonRemovedFunctions(w io.Writer, removed []Score) {
	if len(removed) == 0 {
		return
	}

	_, _ = fmt.Fprintf(w, "\nRemoved Functions:\n")
	for _, s := range removed {
		_, _ = fmt.Fprintf(w, "  %-40s  CRAP: %.1f\n",
			fmt.Sprintf("%s (%s)", s.Function, shortenPath(s.File)),
			s.CRAP)
	}
}
