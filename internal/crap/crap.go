// Package crap computes CRAP (Change Risk Anti-Patterns) scores for
// Go functions by combining cyclomatic complexity with test coverage.
//
// The CRAP formula: CRAP(m) = comp^2 * (1 - cov/100)^3 + comp
// where comp = cyclomatic complexity and cov = coverage percentage.
//
// A CRAPload is the count of functions with a CRAP score at or above
// a given threshold (default 15).
package crap

// Score holds the CRAP score for a single function.
type Score struct {
	// Package is the Go package name.
	Package string `json:"package"`

	// Function is the function or method name (e.g., "Save" or
	// "(*Store).Save").
	Function string `json:"function"`

	// File is the source file path.
	File string `json:"file"`

	// Line is the line number of the function declaration.
	Line int `json:"line"`

	// Complexity is the cyclomatic complexity.
	Complexity int `json:"complexity"`

	// LineCoverage is the line coverage percentage (0-100).
	LineCoverage float64 `json:"line_coverage"`

	// CRAP is the classic CRAP score.
	CRAP float64 `json:"crap"`

	// ContractCoverage is Gaze's contract coverage (0-100).
	// Nil when unavailable (Specs 002-003 not implemented).
	ContractCoverage *float64 `json:"contract_coverage,omitempty"`

	// GazeCRAP is the CRAP formula using contract coverage.
	// Nil when contract coverage is unavailable.
	GazeCRAP *float64 `json:"gaze_crap,omitempty"`

	// Quadrant classification (nil if GazeCRAP unavailable).
	Quadrant *Quadrant `json:"quadrant,omitempty"`

	// FixStrategy is the recommended remediation action for this
	// function. Only populated when CRAP >= CRAPThreshold (i.e.,
	// the function is in the CRAPload). Nil for healthy functions.
	FixStrategy *FixStrategy `json:"fix_strategy,omitempty"`

	// ContractCoverageReason explains why contract coverage is
	// what it is. Nil for normal coverage. Populated when the
	// reason is diagnostic (e.g., all effects are ambiguous).
	ContractCoverageReason *string `json:"contract_coverage_reason,omitempty"`

	// EffectConfidenceRange is [min, max] classification confidence
	// across all side effects. Only populated when
	// ContractCoverageReason is "all_effects_ambiguous".
	EffectConfidenceRange *[2]int `json:"effect_confidence_range,omitempty"`
}

// Quadrant classifies a function based on CRAP and GazeCRAP scores
// relative to their respective thresholds.
type Quadrant string

// Quadrant classification constants.
const (
	Q1Safe                    Quadrant = "Q1_Safe"
	Q2ComplexButTested        Quadrant = "Q2_ComplexButTested"
	Q3SimpleButUnderspecified Quadrant = "Q3_SimpleButUnderspecified"
	Q4Dangerous               Quadrant = "Q4_Dangerous"
)

// FixStrategy describes the recommended remediation action for a
// function that exceeds the CRAP threshold.
type FixStrategy string

// Fix strategy constants.
const (
	// FixDecompose indicates the function's complexity is so high
	// that even 100% coverage cannot bring CRAP below threshold.
	// The function must be split into smaller units.
	FixDecompose FixStrategy = "decompose"

	// FixAddTests indicates the function has zero line coverage
	// and complexity below the threshold. Adding tests will bring
	// CRAP below threshold.
	FixAddTests FixStrategy = "add_tests"

	// FixAddAssertions indicates the function has adequate line
	// coverage but inadequate contract coverage (Q3 quadrant).
	// Existing tests execute the code but don't verify observable
	// behavior. Add assertions to existing tests.
	FixAddAssertions FixStrategy = "add_assertions"

	// FixDecomposeAndTest indicates the function has both high
	// complexity (>= threshold) and zero coverage. It needs
	// decomposition AND tests.
	FixDecomposeAndTest FixStrategy = "decompose_and_test"
)

// RecommendedAction is a prioritized remediation entry for agents
// consuming gaze JSON output. Sorted by fix strategy priority
// (add_tests first, decompose last), then by CRAP score descending.
type RecommendedAction struct {
	Function    string      `json:"function"`
	Package     string      `json:"package"`
	File        string      `json:"file"`
	Line        int         `json:"line"`
	FixStrategy FixStrategy `json:"fix_strategy"`
	CRAP        float64     `json:"crap"`
	GazeCRAP    *float64    `json:"gaze_crap,omitempty"`
	Complexity  int         `json:"complexity"`
	Quadrant    *Quadrant   `json:"quadrant,omitempty"`
}

// Summary holds aggregate statistics for a CRAP report.
type Summary struct {
	TotalFunctions      int                 `json:"total_functions"`
	AvgComplexity       float64             `json:"avg_complexity"`
	AvgLineCoverage     float64             `json:"avg_line_coverage"`
	AvgCRAP             float64             `json:"avg_crap"`
	CRAPload            int                 `json:"crapload"`
	CRAPThreshold       float64             `json:"crap_threshold"`
	GazeCRAPload        *int                `json:"gaze_crapload,omitempty"`
	GazeCRAPThreshold   *float64            `json:"gaze_crap_threshold,omitempty"`
	AvgGazeCRAP         *float64            `json:"avg_gaze_crap,omitempty"`
	AvgContractCoverage *float64            `json:"avg_contract_coverage,omitempty"`
	QuadrantCounts      map[Quadrant]int    `json:"quadrant_counts,omitempty"`
	FixStrategyCounts   map[FixStrategy]int `json:"fix_strategy_counts,omitempty"`
	WorstCRAP           []Score             `json:"worst_crap"`
	WorstGazeCRAP       []Score             `json:"worst_gaze_crap,omitempty"`
	RecommendedActions  []RecommendedAction `json:"recommended_actions,omitempty"`

	// SSADegradedPackages lists package paths where SSA construction
	// failed. When non-empty, contract coverage and GazeCRAP metrics
	// are based on a subset of analyzed functions. Consumers should
	// caveat the metrics accordingly.
	SSADegradedPackages []string `json:"ssa_degraded_packages,omitempty"`
}

// Report is the complete CRAP analysis output.
type Report struct {
	Scores  []Score `json:"scores"`
	Summary Summary `json:"summary"`
}

// FunctionStatus classifies a function's change status when
// comparing baseline and current CRAP results.
type FunctionStatus string

const (
	// StatusRegression indicates CRAP or GazeCRAP increased
	// beyond the epsilon tolerance.
	StatusRegression FunctionStatus = "regression"

	// StatusImprovement indicates CRAP or GazeCRAP decreased
	// beyond the epsilon tolerance.
	StatusImprovement FunctionStatus = "improvement"

	// StatusUnchanged indicates the score delta is within the
	// epsilon tolerance.
	StatusUnchanged FunctionStatus = "unchanged"

	// StatusNew indicates the function was not present in the
	// baseline and its CRAP score is below the new-function
	// threshold.
	StatusNew FunctionStatus = "new"

	// StatusNewViolation indicates the function was not present
	// in the baseline and its CRAP score exceeds the new-function
	// threshold.
	StatusNewViolation FunctionStatus = "new_violation"

	// StatusRemoved indicates the function was present in the
	// baseline but absent from the current results.
	StatusRemoved FunctionStatus = "removed"
)

// FunctionDelta holds the comparison between a baseline and current
// score for a single function. The Status field classifies the
// change direction.
type FunctionDelta struct {
	// Baseline is the function's score from the baseline report.
	Baseline Score `json:"baseline"`

	// Current is the function's score from the current report.
	Current Score `json:"current"`

	// CRAPDelta is current.CRAP - baseline.CRAP.
	CRAPDelta float64 `json:"crap_delta"`

	// GazeCRAPDelta is current.GazeCRAP - baseline.GazeCRAP.
	// Nil when the baseline had no GazeCRAP data (D5: skip when
	// baseline GazeCRAP is nil or zero).
	GazeCRAPDelta *float64 `json:"gaze_crap_delta,omitempty"`

	// Status classifies this function's change direction.
	Status FunctionStatus `json:"status"`
}

// ComparisonSummary holds aggregate counts for a baseline comparison.
type ComparisonSummary struct {
	Regressions          int     `json:"regressions"`
	Improvements         int     `json:"improvements"`
	NewFunctions         int     `json:"new_functions"`
	NewViolations        int     `json:"new_violations"`
	RemovedFunctions     int     `json:"removed_functions"`
	Unchanged            int     `json:"unchanged"`
	Epsilon              float64 `json:"epsilon"`
	NewFunctionThreshold float64 `json:"new_function_threshold"`
	Passed               bool    `json:"passed"`
}

// ComparisonResult is the envelope for a baseline-vs-current
// comparison. It wraps the current Report (for the normal output
// path) and adds comparison-specific data.
type ComparisonResult struct {
	// Report is the current analysis report. Excluded from JSON
	// because the comparison JSON format inlines scores with
	// delta annotations.
	Report *Report `json:"-"`

	// Deltas holds per-function comparisons for functions found
	// in both baseline and current results.
	Deltas []FunctionDelta `json:"deltas"`

	// NewFunctions holds scores for functions present in the
	// current results but absent from the baseline.
	NewFunctions []Score `json:"new_functions"`

	// RemovedFunctions holds scores for functions present in the
	// baseline but absent from the current results.
	RemovedFunctions []Score `json:"removed_functions"`

	// Summary holds aggregate comparison counts and the pass/fail
	// determination.
	Summary ComparisonSummary `json:"comparison"`
}

// Formula computes CRAP(m) = comp^2 * (1 - cov/100)^3 + comp.
// comp is cyclomatic complexity (>= 1).
// coveragePct is line coverage as a percentage (0-100).
// Returns the CRAP score as a float64; higher scores indicate higher risk.
func Formula(complexity int, coveragePct float64) float64 {
	comp := float64(complexity)
	uncov := 1.0 - coveragePct/100.0
	return comp*comp*uncov*uncov*uncov + comp
}

// ClassifyQuadrant determines the quadrant for a function based on
// its CRAP and GazeCRAP scores relative to independent thresholds.
// Returns the Quadrant constant (Q1Safe, Q2ComplexButTested,
// Q3SimpleButUnderspecified, or Q4Dangerous).
func ClassifyQuadrant(crap, gazeCRAP, crapThreshold, gazeCRAPThreshold float64) Quadrant {
	highCRAP := crap >= crapThreshold
	highGazeCRAP := gazeCRAP >= gazeCRAPThreshold

	switch {
	case !highCRAP && !highGazeCRAP:
		return Q1Safe
	case highCRAP && !highGazeCRAP:
		return Q2ComplexButTested
	case !highCRAP && highGazeCRAP:
		return Q3SimpleButUnderspecified
	default:
		return Q4Dangerous
	}
}
