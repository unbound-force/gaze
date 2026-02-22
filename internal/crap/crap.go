// Package crap computes CRAP (Change Risk Anti-Patterns) scores for
// Go functions by combining cyclomatic complexity with test coverage.
//
// The CRAP formula: CRAP(m) = comp^2 * (1 - cov/100)^3 + comp
// where comp = cyclomatic complexity and cov = coverage percentage.
//
// A CRAPload is the count of functions with a CRAP score at or above
// a given threshold (default 15).
package crap

import (
	"math"
)

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

// Summary holds aggregate statistics for a CRAP report.
type Summary struct {
	TotalFunctions      int              `json:"total_functions"`
	AvgComplexity       float64          `json:"avg_complexity"`
	AvgLineCoverage     float64          `json:"avg_line_coverage"`
	AvgCRAP             float64          `json:"avg_crap"`
	CRAPload            int              `json:"crapload"`
	CRAPThreshold       float64          `json:"crap_threshold"`
	GazeCRAPload        *int             `json:"gaze_crapload,omitempty"`
	GazeCRAPThreshold   *float64         `json:"gaze_crap_threshold,omitempty"`
	AvgGazeCRAP         *float64         `json:"avg_gaze_crap,omitempty"`
	AvgContractCoverage *float64         `json:"avg_contract_coverage,omitempty"`
	QuadrantCounts      map[Quadrant]int `json:"quadrant_counts,omitempty"`
	WorstCRAP           []Score          `json:"worst_crap"`
	WorstGazeCRAP       []Score          `json:"worst_gaze_crap,omitempty"`
}

// Report is the complete CRAP analysis output.
type Report struct {
	Scores  []Score `json:"scores"`
	Summary Summary `json:"summary"`
}

// Formula computes CRAP(m) = comp^2 * (1 - cov/100)^3 + comp.
// comp is cyclomatic complexity (>= 1).
// coveragePct is line coverage as a percentage (0-100).
func Formula(complexity int, coveragePct float64) float64 {
	comp := float64(complexity)
	uncov := 1.0 - coveragePct/100.0
	return comp*comp*math.Pow(uncov, 3) + comp
}

// ClassifyQuadrant determines the quadrant for a function based on
// its CRAP and GazeCRAP scores relative to independent thresholds.
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
