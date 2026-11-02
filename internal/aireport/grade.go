package aireport

import (
	"strconv"
	"strings"
)

// rubricVersion identifies the grading rubric encoded in this file.
// It is published as Assessment.RubricVersion and documented in the
// "Grading rubric" section of docs/reference/cli/report.md. Changing
// any boundary below requires bumping this value.
const rubricVersion = "1"

// gradeLadder lists every non-failing grade from best to worst. The
// boundary tables below are indexed in the same order; a value that
// satisfies no boundary grades F.
var gradeLadder = [...]Grade{
	GradeA, GradeAMinus, GradeBPlus, GradeB, GradeBMinus,
	GradeCPlus, GradeC, GradeCMinus, GradeD,
}

// Boundary tables. Ratio and float dimensions are expressed in tenths
// (20 means 2.0); contract coverage is expressed in integer percentage
// points. Lower-is-better tables hold inclusive upper bounds;
// higher-is-better tables hold inclusive lower bounds.
var (
	crapLoadBounds         = [len(gradeLadder)]int{20, 40, 60, 80, 100, 125, 150, 200, 300}
	gazeCRAPLoadBounds     = [len(gradeLadder)]int{20, 40, 60, 80, 100, 125, 150, 200, 300}
	lineCoverageBounds     = [len(gradeLadder)]int{900, 850, 800, 750, 700, 650, 600, 550, 450}
	contractCoverageBounds = [len(gradeLadder)]int{80, 70, 60, 50, 40, 30, 20, 15, 10}
	complexityBounds       = [len(gradeLadder)]int{30, 40, 50, 60, 70, 80, 90, 100, 120}
)

// displayTenths returns the value v as it is displayed in the report
// (strconv.FormatFloat with one decimal), expressed as integer tenths.
// Grading the displayed value guarantees the grade never contradicts
// the number printed beside it (79.96 displays as 80.0 and grades as
// 80.0). The boolean is false for NaN and infinities.
func displayTenths(v float64) (int, bool) {
	s := strconv.FormatFloat(v, 'f', 1, 64)
	n, err := strconv.Atoi(strings.Replace(s, ".", "", 1))
	if err != nil {
		return 0, false
	}
	return n, true
}

// ratioTenths returns n/total as a percentage in integer tenths,
// rounded half up: (2*n*1000 + total) / (2*total). A non-positive
// total yields 0; callers treat total == 0 as not applicable before
// calling.
func ratioTenths(n, total int) int {
	if total <= 0 {
		return 0
	}
	return (2*n*1000 + total) / (2 * total)
}

// formatTenths renders integer tenths with exactly one decimal place.
func formatTenths(tenths int) string {
	return strconv.FormatFloat(float64(tenths)/10, 'f', 1, 64)
}

// gradeLowerIsBetter grades value against inclusive upper bounds,
// scanning from A downward. The first satisfied bound wins; a value
// above every bound grades F.
func gradeLowerIsBetter(value int, bounds [len(gradeLadder)]int) Grade {
	for i, b := range bounds {
		if value <= b {
			return gradeLadder[i]
		}
	}
	return GradeF
}

// gradeHigherIsBetter grades value against inclusive lower bounds,
// scanning from A downward. The first satisfied bound wins; a value
// below every bound grades F.
func gradeHigherIsBetter(value int, bounds [len(gradeLadder)]int) Grade {
	for i, b := range bounds {
		if value >= b {
			return gradeLadder[i]
		}
	}
	return GradeF
}

// severityForGrade maps a grade to its severity band: A, A- and B+ are
// low; B, B-, C+ and C are moderate; everything else is critical.
func severityForGrade(g Grade) Severity {
	switch g {
	case GradeA, GradeAMinus, GradeBPlus:
		return SeverityLow
	case GradeB, GradeBMinus, GradeCPlus, GradeC:
		return SeverityModerate
	default:
		return SeverityCritical
	}
}

// markerForSeverity returns the closed-vocabulary emoji marker for a
// severity: 🟢 low, 🟡 moderate, 🔴 critical, ⚪ unavailable.
func markerForSeverity(s Severity) string {
	switch s {
	case SeverityLow:
		return "🟢"
	case SeverityModerate:
		return "🟡"
	case SeverityCritical:
		return "🔴"
	default:
		return "⚪"
	}
}
