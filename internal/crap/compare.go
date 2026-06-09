package crap

import (
	"encoding/json"
	"fmt"
	"io"
)

// CompareOptions configures the baseline comparison algorithm.
type CompareOptions struct {
	// Epsilon is the minimum score delta to trigger a regression
	// or improvement classification. Deltas within [-epsilon,
	// +epsilon] are classified as unchanged.
	Epsilon float64

	// NewFunctionThreshold is the CRAP score above which a new
	// function (not in the baseline) is classified as a
	// violation. Functions at or below this threshold are
	// informational "new" entries.
	NewFunctionThreshold float64
}

// LoadBaseline deserializes a crap.Report from JSON read from r.
// It uses standard json.Unmarshal which ignores unknown fields,
// providing forward and backward compatibility with baselines
// created by older or newer gaze versions (D5, R6).
func LoadBaseline(r io.Reader) (*Report, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading baseline: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("baseline is empty")
	}

	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parsing baseline JSON: %w", err)
	}

	return &report, nil
}

// scoreKey builds the composite lookup key for matching functions
// between baseline and current results. Uses file + ":" + function
// per D2.
func scoreKey(s Score) string {
	return s.File + ":" + s.Function
}

// Compare performs a deterministic per-function comparison between
// baseline and current CRAP reports. It classifies each function as
// regression, improvement, unchanged, new, new_violation, or
// removed. The result includes a pass/fail gate: passed is true
// when there are zero regressions and zero new-function violations
// (D7).
//
// The algorithm is a pure function with no I/O or global state (D1).
func Compare(baseline *Report, current *Report, opts CompareOptions) *ComparisonResult {
	// Build lookup map from baseline scores.
	baselineMap := make(map[string]Score, len(baseline.Scores))
	for _, s := range baseline.Scores {
		baselineMap[scoreKey(s)] = s
	}

	result := &ComparisonResult{
		Report: current,
	}

	// Track which baseline entries have been matched so we can
	// identify removed functions from what remains.
	matched := make(map[string]bool, len(baseline.Scores))

	for _, cur := range current.Scores {
		key := scoreKey(cur)
		base, found := baselineMap[key]
		if !found {
			// Function not in baseline: new or new_violation
			// status determined in buildComparisonSummary (SC-004).
			result.NewFunctions = append(result.NewFunctions, cur)
			continue
		}

		matched[key] = true

		delta := FunctionDelta{
			Baseline:  base,
			Current:   cur,
			CRAPDelta: cur.CRAP - base.CRAP,
		}

		// Compute GazeCRAP delta only when baseline had coverage
		// (D5: skip when baseline GazeCRAP is nil or zero).
		var hasGazeDelta bool
		if base.GazeCRAP != nil && *base.GazeCRAP > 0 &&
			cur.GazeCRAP != nil {
			gd := *cur.GazeCRAP - *base.GazeCRAP
			delta.GazeCRAPDelta = &gd
			hasGazeDelta = true
		}

		// Classify the function's change status.
		delta.Status = classifyDelta(delta.CRAPDelta,
			delta.GazeCRAPDelta, hasGazeDelta, opts.Epsilon)

		result.Deltas = append(result.Deltas, delta)
	}

	// Remaining unmatched baseline entries are removed functions
	// (SC-005).
	for _, base := range baseline.Scores {
		key := scoreKey(base)
		if !matched[key] {
			result.RemovedFunctions = append(result.RemovedFunctions, base)
		}
	}

	// Compute summary counts.
	result.Summary = buildComparisonSummary(result, opts)

	return result
}

// classifyDelta determines the FunctionStatus for a matched function
// based on CRAP and GazeCRAP deltas relative to epsilon.
//
// When signals conflict (e.g., CRAP regresses but GazeCRAP improves),
// regression takes precedence (SC-003). This is conservative: a
// regression in any metric is a regression overall.
func classifyDelta(crapDelta float64, gazeCRAPDelta *float64, hasGazeDelta bool, epsilon float64) FunctionStatus {
	crapRegression := crapDelta > epsilon
	crapImprovement := crapDelta < -epsilon

	var gazeRegression, gazeImprovement bool
	if hasGazeDelta && gazeCRAPDelta != nil {
		gazeRegression = *gazeCRAPDelta > epsilon
		gazeImprovement = *gazeCRAPDelta < -epsilon
	}

	// Any regression signal wins (conservative approach per
	// SC-003 conflicting signals scenario).
	if crapRegression || gazeRegression {
		return StatusRegression
	}

	// Improvement only if no regression and at least one metric
	// improved. If CRAP improved but GazeCRAP regressed, the
	// regression check above already caught it.
	if crapImprovement || gazeImprovement {
		return StatusImprovement
	}

	return StatusUnchanged
}

// buildComparisonSummary computes aggregate counts from the
// comparison result and determines pass/fail.
func buildComparisonSummary(result *ComparisonResult, opts CompareOptions) ComparisonSummary {
	summary := ComparisonSummary{
		Epsilon:              opts.Epsilon,
		NewFunctionThreshold: opts.NewFunctionThreshold,
		RemovedFunctions:     len(result.RemovedFunctions),
	}

	for _, d := range result.Deltas {
		switch d.Status {
		case StatusRegression:
			summary.Regressions++
		case StatusImprovement:
			summary.Improvements++
		case StatusUnchanged:
			summary.Unchanged++
		}
	}

	for _, s := range result.NewFunctions {
		if s.CRAP > opts.NewFunctionThreshold {
			summary.NewViolations++
		} else {
			summary.NewFunctions++
		}
	}

	// Pass when zero regressions and zero new-function violations
	// (D7).
	summary.Passed = summary.Regressions == 0 && summary.NewViolations == 0

	return summary
}
