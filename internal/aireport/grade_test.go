package aireport

import (
	"math"
	"testing"
)

// TestGrade_RubricBoundaries checks every boundary of every dimension:
// the value on the bound earns the grade, and one tenth (one point for
// contract coverage) past the bound earns the next grade down.
func TestGrade_RubricBoundaries(t *testing.T) {
	type dim struct {
		name        string
		bounds      [len(gradeLadder)]int
		lowerBetter bool
		step        int
	}
	dims := []dim{
		{"crapload", crapLoadBounds, true, 1},
		{"gaze_crapload", gazeCRAPLoadBounds, true, 1},
		{"avg_line_coverage", lineCoverageBounds, false, 1},
		{"contract_coverage", contractCoverageBounds, false, 1},
		{"complexity", complexityBounds, true, 1},
	}
	nextDown := func(i int) Grade {
		if i+1 < len(gradeLadder) {
			return gradeLadder[i+1]
		}
		return GradeF
	}
	for _, d := range dims {
		for i, b := range d.bounds {
			var onBound, past Grade
			if d.lowerBetter {
				onBound = gradeLowerIsBetter(b, d.bounds)
				past = gradeLowerIsBetter(b+d.step, d.bounds)
			} else {
				onBound = gradeHigherIsBetter(b, d.bounds)
				past = gradeHigherIsBetter(b-d.step, d.bounds)
			}
			if onBound != gradeLadder[i] {
				t.Errorf("%s: value %d on bound graded %q, want %q", d.name, b, onBound, gradeLadder[i])
			}
			if past != nextDown(i) {
				t.Errorf("%s: value one step past %d graded %q, want %q", d.name, b, past, nextDown(i))
			}
		}
	}
}

// TestGrade_PublishedBoundaryValues pins the rubric table literally so
// an accidental edit to a bound fails loudly.
func TestGrade_PublishedBoundaryValues(t *testing.T) {
	want := map[string][len(gradeLadder)]int{
		"crapload":          {20, 40, 60, 80, 100, 125, 150, 200, 300},
		"gaze_crapload":     {20, 40, 60, 80, 100, 125, 150, 200, 300},
		"avg_line_coverage": {900, 850, 800, 750, 700, 650, 600, 550, 450},
		"contract_coverage": {80, 70, 60, 50, 40, 30, 20, 15, 10},
		"complexity":        {30, 40, 50, 60, 70, 80, 90, 100, 120},
	}
	got := map[string][len(gradeLadder)]int{
		"crapload":          crapLoadBounds,
		"gaze_crapload":     gazeCRAPLoadBounds,
		"avg_line_coverage": lineCoverageBounds,
		"contract_coverage": contractCoverageBounds,
		"complexity":        complexityBounds,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("%s bounds = %v, want %v", k, got[k], w)
		}
	}
	if rubricVersion != "1" {
		t.Errorf("rubricVersion = %q, want %q", rubricVersion, "1")
	}
}

func TestGrade_FRows(t *testing.T) {
	if g := gradeLowerIsBetter(301, crapLoadBounds); g != GradeF {
		t.Errorf("crapload 30.1%% graded %q, want F", g)
	}
	if g := gradeHigherIsBetter(449, lineCoverageBounds); g != GradeF {
		t.Errorf("line coverage 44.9%% graded %q, want F", g)
	}
	if g := gradeHigherIsBetter(9, contractCoverageBounds); g != GradeF {
		t.Errorf("contract coverage 9%% graded %q, want F", g)
	}
	if g := gradeLowerIsBetter(121, complexityBounds); g != GradeF {
		t.Errorf("complexity 12.1 graded %q, want F", g)
	}
	if g := gradeHigherIsBetter(1000, lineCoverageBounds); g != GradeA {
		t.Errorf("line coverage 100%% graded %q, want A", g)
	}
	if g := gradeLowerIsBetter(0, crapLoadBounds); g != GradeA {
		t.Errorf("crapload 0%% graded %q, want A", g)
	}
}

func TestDisplayTenths(t *testing.T) {
	cases := []struct {
		in   float64
		want int
	}{
		{79.96, 800},
		{79.94, 799},
		{80, 800},
		{0, 0},
		{0.15, 1}, // binary 0.1499..., displayed as 0.1
		{12.25, 122},
		{3.05, 30}, // binary 3.0499..., displayed as 3.0
		{100, 1000},
		{-0.04, 0},
	}
	for _, c := range cases {
		got, ok := displayTenths(c.in)
		if !ok || got != c.want {
			t.Errorf("displayTenths(%v) = %d,%v; want %d,true", c.in, got, ok, c.want)
		}
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, ok := displayTenths(bad); ok {
			t.Errorf("displayTenths(%v) ok = true, want false", bad)
		}
	}
}

// TestDisplayTenths_GradesDisplayedValue proves 79.96 grades as the
// displayed 80.0 (B+), not the raw value (B).
func TestDisplayTenths_GradesDisplayedValue(t *testing.T) {
	tenths, _ := displayTenths(79.96)
	if g := gradeHigherIsBetter(tenths, lineCoverageBounds); g != GradeBPlus {
		t.Errorf("79.96%% line coverage graded %q, want B+", g)
	}
}

func TestRatioTenths(t *testing.T) {
	cases := []struct {
		n, total, want int
	}{
		{0, 10, 0},
		{1, 10, 100},
		{7, 12, 583}, // 58.333 → 58.3
		{1, 8, 125},  // 12.5 exactly
		{1, 2000, 1}, // 0.05% is exactly 0.5 tenths → rounds half up to 0.1
		{1, 2001, 0}, // 0.04998% is 0.4998 tenths → rounds down to 0.0
		{3, 2000, 2}, // 0.15% is exactly 1.5 tenths → rounds half up to 0.2
		{1, 3, 333},  // 33.333 → 33.3
		{2, 3, 667},  // 66.666 → 66.7
		{10, 10, 1000},
		{5, 0, 0},
		{5, -1, 0},
	}
	for _, c := range cases {
		if got := ratioTenths(c.n, c.total); got != c.want {
			t.Errorf("ratioTenths(%d,%d) = %d, want %d", c.n, c.total, got, c.want)
		}
	}
}

func TestFormatTenths(t *testing.T) {
	cases := map[int]string{0: "0.0", 5: "0.5", 125: "12.5", 1000: "100.0", 583: "58.3"}
	for in, want := range cases {
		if got := formatTenths(in); got != want {
			t.Errorf("formatTenths(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestSeverityAndMarker(t *testing.T) {
	cases := []struct {
		grade  Grade
		sev    Severity
		marker string
	}{
		{GradeA, SeverityLow, "🟢"},
		{GradeAMinus, SeverityLow, "🟢"},
		{GradeBPlus, SeverityLow, "🟢"},
		{GradeB, SeverityModerate, "🟡"},
		{GradeBMinus, SeverityModerate, "🟡"},
		{GradeCPlus, SeverityModerate, "🟡"},
		{GradeC, SeverityModerate, "🟡"},
		{GradeCMinus, SeverityCritical, "🔴"},
		{GradeD, SeverityCritical, "🔴"},
		{GradeF, SeverityCritical, "🔴"},
	}
	for _, c := range cases {
		sev := severityForGrade(c.grade)
		if sev != c.sev {
			t.Errorf("severityForGrade(%q) = %q, want %q", c.grade, sev, c.sev)
		}
		if m := markerForSeverity(sev); m != c.marker {
			t.Errorf("markerForSeverity(%q) = %q, want %q", sev, m, c.marker)
		}
	}
	if m := markerForSeverity(SeverityUnavailable); m != "⚪" {
		t.Errorf("markerForSeverity(unavailable) = %q, want ⚪", m)
	}
}
