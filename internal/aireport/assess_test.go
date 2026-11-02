package aireport

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/crap"
)

// assessFloatPtr returns a pointer to v.
func assessFloatPtr(v float64) *float64 { return &v }

// assessCrapReport builds a CRAP report with one score per fix
// strategy plus a healthy score.
func assessCrapReport() crap.Report {
	q4 := crap.Q4Dangerous
	q3 := crap.Q3SimpleButUnderspecified
	addTests := crap.FixAddTests
	addAssertions := crap.FixAddAssertions
	decompose := crap.FixDecompose
	decomposeAndTest := crap.FixDecomposeAndTest
	scores := []crap.Score{
		{
			Package: "api", Function: "processQueue", File: "internal/api/queue.go", Line: 89,
			Complexity: 8, LineCoverage: 0, CRAP: 72, Quadrant: &q4, FixStrategy: &addTests,
		},
		{
			Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
			Complexity: 10, LineCoverage: 50, CRAP: 22.5, GazeCRAP: assessFloatPtr(110),
			Quadrant: &q3, FixStrategy: &addAssertions,
		},
		{
			Package: "api", Function: "parseFlags", File: "internal/api/flags.go", Line: 18,
			Complexity: 16, LineCoverage: 50, CRAP: 48, Quadrant: &q4, FixStrategy: &decompose,
		},
		{
			Package: "api", Function: "runBatch", File: "internal/api/batch.go", Line: 40,
			Complexity: 18, LineCoverage: 50, CRAP: 342, Quadrant: &q4, FixStrategy: &decomposeAndTest,
		},
		{
			Package: "api", Function: "healthy", File: "internal/api/healthy.go", Line: 5,
			Complexity: 2, LineCoverage: 100, CRAP: 1,
		},
	}
	actions := []crap.RecommendedAction{
		{
			Package: "api", Function: "processQueue", File: "internal/api/queue.go", Line: 89,
			FixStrategy: addTests, CRAP: 72, GazeCRAP: assessFloatPtr(72), Complexity: 8, Quadrant: &q4,
		},
		{
			Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
			FixStrategy: addAssertions, CRAP: 22.5, GazeCRAP: assessFloatPtr(110), Complexity: 10, Quadrant: &q3,
		},
		{
			Package: "api", Function: "parseFlags", File: "internal/api/flags.go", Line: 18,
			FixStrategy: decompose, CRAP: 48, GazeCRAP: assessFloatPtr(71.3), Complexity: 16, Quadrant: &q4,
		},
		{
			Package: "api", Function: "runBatch", File: "internal/api/batch.go", Line: 40,
			FixStrategy: decomposeAndTest, CRAP: 342, GazeCRAP: assessFloatPtr(342), Complexity: 18, Quadrant: &q4,
		},
	}
	return crap.Report{
		Scores: scores,
		Summary: crap.Summary{
			TotalFunctions:      20,
			AvgComplexity:       6.24,
			AvgLineCoverage:     79.04,
			AvgCRAP:             12,
			CRAPload:            4,
			CRAPThreshold:       15,
			GazeCRAPload:        intPtr(2),
			GazeCRAPThreshold:   assessFloatPtr(15),
			RecommendedActions:  actions,
			QuadrantCounts:      map[crap.Quadrant]int{crap.Q1Safe: 16, crap.Q2ComplexButTested: 1, crap.Q3SimpleButUnderspecified: 1, crap.Q4Dangerous: 2},
			FixStrategyCounts:   map[crap.FixStrategy]int{addTests: 1, addAssertions: 1, decompose: 1, decomposeAndTest: 1},
			SSADegradedPackages: nil,
		},
	}
}

// assessPayload builds a payload from a CRAP report and a quality
// total-test count.
func assessPayload(t *testing.T, report crap.Report, totalTests int, summary ReportSummary, sections ...Section) *ReportPayload {
	t.Helper()
	crapRaw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal crap report: %v", err)
	}
	qualityRaw, err := json.Marshal(map[string]any{
		"quality_summary": map[string]any{"total_tests": totalTests},
	})
	if err != nil {
		t.Fatalf("marshal quality report: %v", err)
	}
	if len(sections) == 0 {
		sections = []Section{SectionCRAP, SectionQuality, SectionClassification, SectionHealth}
	}
	summary.AvgContractCoverage = intPtr(31)
	return &ReportPayload{
		Summary:  summary,
		Metadata: ReportMetadata{Sections: sections},
		CRAP:     crapRaw,
		Quality:  qualityRaw,
	}
}

// assessRow finds a scorecard row by dimension.
func assessRow(t *testing.T, a Assessment, dimension string) ScorecardRow {
	t.Helper()
	for _, row := range a.Scorecard {
		if row.Dimension == dimension {
			return row
		}
	}
	t.Fatalf("scorecard row %q not found", dimension)
	return ScorecardRow{}
}

func TestAssess_ScorecardOrderAndLabels(t *testing.T) {
	a := Assess(assessPayload(t, assessCrapReport(), 140, ReportSummary{}))

	if a.RubricVersion != "1" {
		t.Errorf("rubric_version = %q, want %q", a.RubricVersion, "1")
	}
	want := []struct{ dimension, label string }{
		{"crapload", "CRAPload"},
		{"gaze_crapload", "GazeCRAPload"},
		{"avg_line_coverage", "Avg Line Coverage"},
		{"contract_coverage", "Contract Coverage"},
		{"complexity", "Complexity"},
	}
	if len(a.Scorecard) != len(want) {
		t.Fatalf("scorecard has %d rows, want %d", len(a.Scorecard), len(want))
	}
	for i, w := range want {
		if a.Scorecard[i].Dimension != w.dimension || a.Scorecard[i].Label != w.label {
			t.Errorf("row %d = (%q, %q), want (%q, %q)",
				i, a.Scorecard[i].Dimension, a.Scorecard[i].Label, w.dimension, w.label)
		}
	}
}

func TestAssess_DetailTemplates(t *testing.T) {
	a := Assess(assessPayload(t, assessCrapReport(), 140, ReportSummary{}))

	tests := []struct {
		dimension string
		detail    string
		value     float64
		grade     Grade
		severity  Severity
		marker    string
	}{
		{"crapload", "4/20 functions (20.0%) at or above threshold 15", 20, GradeCMinus, SeverityCritical, "🔴"},
		{"gaze_crapload", "2/20 functions (10.0%) at or above threshold 15", 10, GradeBMinus, SeverityModerate, "🟡"},
		{"avg_line_coverage", "79.0% average line coverage", 79, GradeB, SeverityModerate, "🟡"},
		{"contract_coverage", "31% average contract coverage across 140 tests", 31, GradeCPlus, SeverityModerate, "🟡"},
		{"complexity", "6.2 average cyclomatic complexity", 6.2, GradeBMinus, SeverityModerate, "🟡"},
	}
	for _, tt := range tests {
		t.Run(tt.dimension, func(t *testing.T) {
			row := assessRow(t, a, tt.dimension)
			if row.Detail != tt.detail {
				t.Errorf("detail = %q, want %q", row.Detail, tt.detail)
			}
			if row.Value == nil || *row.Value != tt.value {
				t.Errorf("value = %v, want %v", row.Value, tt.value)
			}
			if row.Grade == nil || *row.Grade != tt.grade {
				t.Errorf("grade = %v, want %v", row.Grade, tt.grade)
			}
			if row.Severity != tt.severity || row.Marker != tt.marker {
				t.Errorf("severity/marker = %s/%s, want %s/%s", row.Severity, row.Marker, tt.severity, tt.marker)
			}
		})
	}
}

func TestAssess_UnavailableMatrix(t *testing.T) {
	tests := []struct {
		name       string
		payload    func(t *testing.T) *ReportPayload
		dimension  string
		wantDetail string
	}{
		{
			name: "crap not selected",
			payload: func(t *testing.T) *ReportPayload {
				return assessPayload(t, assessCrapReport(), 140, ReportSummary{}, SectionQuality)
			},
			dimension:  "crapload",
			wantDetail: reasonNotSelected,
		},
		{
			name: "crap analysis failed",
			payload: func(t *testing.T) *ReportPayload {
				p := assessPayload(t, assessCrapReport(), 140, ReportSummary{})
				p.Errors.CRAP = strPtr("boom")
				return p
			},
			dimension:  "avg_line_coverage",
			wantDetail: reasonAnalysisFailed,
		},
		{
			name: "no functions analyzed",
			payload: func(t *testing.T) *ReportPayload {
				report := assessCrapReport()
				report.Summary.TotalFunctions = 0
				return assessPayload(t, report, 140, ReportSummary{})
			},
			dimension:  "complexity",
			wantDetail: reasonNoFunctions,
		},
		{
			name: "gazecrap unavailable",
			payload: func(t *testing.T) *ReportPayload {
				report := assessCrapReport()
				report.Summary.GazeCRAPload = nil
				return assessPayload(t, report, 140, ReportSummary{})
			},
			dimension:  "gaze_crapload",
			wantDetail: reasonGazeCRAPUnavailable,
		},
		{
			name: "quality not selected",
			payload: func(t *testing.T) *ReportPayload {
				return assessPayload(t, assessCrapReport(), 140, ReportSummary{}, SectionCRAP)
			},
			dimension:  "contract_coverage",
			wantDetail: reasonNotSelected,
		},
		{
			name: "quality analysis failed",
			payload: func(t *testing.T) *ReportPayload {
				p := assessPayload(t, assessCrapReport(), 140, ReportSummary{})
				p.Errors.Quality = strPtr("boom")
				return p
			},
			dimension:  "contract_coverage",
			wantDetail: reasonAnalysisFailed,
		},
		{
			name: "contract coverage absent",
			payload: func(t *testing.T) *ReportPayload {
				p := assessPayload(t, assessCrapReport(), 140, ReportSummary{})
				p.Summary.AvgContractCoverage = nil
				return p
			},
			dimension:  "contract_coverage",
			wantDetail: reasonNoTests,
		},
		{
			name: "no tests analyzed",
			payload: func(t *testing.T) *ReportPayload {
				return assessPayload(t, assessCrapReport(), 0, ReportSummary{})
			},
			dimension:  "contract_coverage",
			wantDetail: reasonNoTests,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := assessRow(t, Assess(tt.payload(t)), tt.dimension)
			if row.Detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", row.Detail, tt.wantDetail)
			}
			if row.Value != nil || row.Grade != nil {
				t.Errorf("value/grade = %v/%v, want both nil", row.Value, row.Grade)
			}
			if row.Severity != SeverityUnavailable || row.Marker != "⚪" {
				t.Errorf("severity/marker = %s/%s, want unavailable/⚪", row.Severity, row.Marker)
			}
		})
	}
}

// assessSingleActionPayload builds a payload with one recommended
// action and one matching score.
func assessSingleActionPayload(t *testing.T, score crap.Score, action crap.RecommendedAction) *ReportPayload {
	t.Helper()
	report := crap.Report{
		Scores: []crap.Score{score},
		Summary: crap.Summary{
			TotalFunctions:     1,
			AvgComplexity:      1,
			AvgLineCoverage:    100,
			CRAPload:           1,
			CRAPThreshold:      15,
			RecommendedActions: []crap.RecommendedAction{action},
		},
	}
	return assessPayload(t, report, 0, ReportSummary{})
}

func TestAssess_FixStrategyTemplates(t *testing.T) {
	tests := []struct {
		name     string
		score    crap.Score
		action   crap.RecommendedAction
		wantText string
		wantSev  Severity
	}{
		{
			name: "add_tests zero coverage is critical",
			score: crap.Score{
				Package: "api", Function: "processQueue", File: "internal/api/queue.go", Line: 89,
				Complexity: 8, LineCoverage: 0, CRAP: 72,
			},
			action: crap.RecommendedAction{
				Package: "api", Function: "processQueue", File: "internal/api/queue.go", Line: 89,
				FixStrategy: crap.FixAddTests, Complexity: 8, CRAP: 72,
			},
			wantText: "Add tests for `processQueue` — 0.0% line coverage, complexity 8, CRAP 72.0 (internal/api/queue.go:89).",
			wantSev:  SeverityCritical,
		},
		{
			name: "add_assertions with gazecrap",
			score: crap.Score{
				Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
				Complexity: 10, LineCoverage: 50, CRAP: 22.5,
			},
			action: crap.RecommendedAction{
				Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
				FixStrategy: crap.FixAddAssertions, Complexity: 10, CRAP: 22.5, GazeCRAP: assessFloatPtr(110),
			},
			wantText: "Strengthen assertions for `validateInput` — tests execute it without verifying its contracts; GazeCRAP 110.0 (internal/api/validate.go:21).",
			wantSev:  SeverityModerate,
		},
		{
			name: "add_assertions without gazecrap",
			score: crap.Score{
				Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
				Complexity: 10, LineCoverage: 50, CRAP: 22.5,
			},
			action: crap.RecommendedAction{
				Package: "api", Function: "validateInput", File: "internal/api/validate.go", Line: 21,
				FixStrategy: crap.FixAddAssertions, Complexity: 10, CRAP: 22.5,
			},
			wantText: "Strengthen assertions for `validateInput` — tests execute it without verifying its contracts (internal/api/validate.go:21).",
			wantSev:  SeverityModerate,
		},
		{
			name: "decompose Q4 is critical",
			score: crap.Score{
				Package: "api", Function: "parseFlags", File: "internal/api/flags.go", Line: 18,
				Complexity: 16, LineCoverage: 50, CRAP: 48, Quadrant: quadrantPtr(crap.Q4Dangerous),
			},
			action: crap.RecommendedAction{
				Package: "api", Function: "parseFlags", File: "internal/api/flags.go", Line: 18,
				FixStrategy: crap.FixDecompose, Complexity: 16, CRAP: 48,
			},
			wantText: "Decompose `parseFlags` — complexity 16 keeps CRAP at 48.0 even at full coverage (internal/api/flags.go:18).",
			wantSev:  SeverityCritical,
		},
		{
			name: "decompose_and_test",
			score: crap.Score{
				Package: "api", Function: "runBatch", File: "internal/api/batch.go", Line: 40,
				Complexity: 18, LineCoverage: 50, CRAP: 342,
			},
			action: crap.RecommendedAction{
				Package: "api", Function: "runBatch", File: "internal/api/batch.go", Line: 40,
				FixStrategy: crap.FixDecomposeAndTest, Complexity: 18, CRAP: 342,
			},
			wantText: "Decompose `runBatch`, then add tests — complexity 18, 50.0% line coverage, CRAP 342.0 (internal/api/batch.go:40).",
			wantSev:  SeverityModerate,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Assess(assessSingleActionPayload(t, tt.score, tt.action))
			if len(a.Recommendations) != 1 {
				t.Fatalf("recommendations = %d, want 1", len(a.Recommendations))
			}
			rec := a.Recommendations[0]
			if rec.Text != tt.wantText {
				t.Errorf("text = %q, want %q", rec.Text, tt.wantText)
			}
			if rec.Severity != tt.wantSev {
				t.Errorf("severity = %s, want %s", rec.Severity, tt.wantSev)
			}
			if rec.Source != "fix_strategy" {
				t.Errorf("source = %q, want fix_strategy", rec.Source)
			}
			if rec.Function != tt.action.Function || rec.File != tt.action.File || rec.Line != tt.action.Line {
				t.Errorf("location = %s (%s:%d), want %s (%s:%d)",
					rec.Function, rec.File, rec.Line, tt.action.Function, tt.action.File, tt.action.Line)
			}
			if rec.FixStrategy != string(tt.action.FixStrategy) {
				t.Errorf("fix_strategy = %q, want %q", rec.FixStrategy, tt.action.FixStrategy)
			}
		})
	}
}

func TestAssess_RecommendationCapAndOrder(t *testing.T) {
	report := assessCrapReport()
	for i := 0; i < 3; i++ {
		report.Summary.RecommendedActions = append(report.Summary.RecommendedActions, crap.RecommendedAction{
			Package: "api", Function: "healthy", File: "internal/api/healthy.go", Line: 5,
			FixStrategy: crap.FixAddTests, Complexity: 2, CRAP: 2,
		})
	}
	p := assessPayload(t, report, 140, ReportSummary{
		SSADegraded:         true,
		SSADegradedPackages: []string{"a", "b"},
		SkippedTests:        3,
		Contractual:         4,
		Ambiguous:           9,
		Incidental:          2,
	})

	a := Assess(p)
	if len(a.Recommendations) != maxRecommendations {
		t.Fatalf("recommendations = %d, want %d", len(a.Recommendations), maxRecommendations)
	}
	wantFunctions := []string{"processQueue", "validateInput", "parseFlags", "runBatch", "healthy"}
	for i, fn := range wantFunctions {
		if a.Recommendations[i].Function != fn {
			t.Errorf("recommendation %d function = %q, want %q", i, a.Recommendations[i].Function, fn)
		}
	}
	wantSources := []string{"ssa_degraded", "skipped_tests", "ambiguous_classification"}
	for i, source := range wantSources {
		if a.Recommendations[5+i].Source != source {
			t.Errorf("recommendation %d source = %q, want %q", 5+i, a.Recommendations[5+i].Source, source)
		}
	}
}

func TestAssess_ConditionRecommendationTexts(t *testing.T) {
	p := assessPayload(t, assessCrapReport(), 140, ReportSummary{
		SSADegraded:         true,
		SSADegradedPackages: []string{"a", "b"},
		SkippedTests:        3,
		Contractual:         4,
		Ambiguous:           9,
		Incidental:          2,
	})
	a := Assess(p)

	conditionRecs := a.Recommendations[4:]
	if len(conditionRecs) != 3 {
		t.Fatalf("condition recommendations = %d, want 3", len(conditionRecs))
	}
	want := []struct {
		text     string
		severity Severity
		marker   string
	}{
		{"Resolve SSA degradation in 2 package(s) — quality and contract data are partial there.", SeverityModerate, "🟡"},
		{"Map 3 skipped test(s) to targets with `gaze quality --target` — no target function could be inferred.", SeverityLow, "🟢"},
		{"Document contracts for ambiguous side effects — 9 of 15 effects (60.0%) are ambiguous.", SeverityLow, "🟢"},
	}
	for i, w := range want {
		if conditionRecs[i].Text != w.text {
			t.Errorf("recommendation %d text = %q, want %q", i, conditionRecs[i].Text, w.text)
		}
		if conditionRecs[i].Severity != w.severity || conditionRecs[i].Marker != w.marker {
			t.Errorf("recommendation %d severity/marker = %s/%s, want %s/%s",
				i, conditionRecs[i].Severity, conditionRecs[i].Marker, w.severity, w.marker)
		}
	}
}

func TestAssess_ConditionRecommendationsOmitted(t *testing.T) {
	p := assessPayload(t, assessCrapReport(), 140, ReportSummary{
		Contractual: 9,
		Ambiguous:   3,
		Incidental:  3,
	}, SectionCRAP)
	a := Assess(p)
	for _, rec := range a.Recommendations {
		if rec.Source != "fix_strategy" {
			t.Errorf("unexpected condition recommendation %q", rec.Source)
		}
	}
}

func TestAssess_JoinMissIsSkipped(t *testing.T) {
	report := assessCrapReport()
	report.Summary.RecommendedActions = append(report.Summary.RecommendedActions, crap.RecommendedAction{
		Package: "api", Function: "ghost", File: "internal/api/ghost.go", Line: 1,
		FixStrategy: crap.FixAddTests, Complexity: 4, CRAP: 20,
	})
	a := Assess(assessPayload(t, report, 140, ReportSummary{}))

	for _, rec := range a.Recommendations {
		if rec.Function == "ghost" {
			t.Fatalf("join miss rendered a recommendation: %q", rec.Text)
		}
	}
	for _, rec := range a.Recommendations {
		if strings.Contains(rec.Text, "ghost") {
			t.Fatalf("join miss leaked fabricated data: %q", rec.Text)
		}
	}
}

func TestAssess_NilPayload(t *testing.T) {
	a := Assess(nil)
	if len(a.Scorecard) != 5 {
		t.Fatalf("scorecard rows = %d, want 5", len(a.Scorecard))
	}
	for _, row := range a.Scorecard {
		if row.Grade != nil || row.Value != nil || row.Severity != SeverityUnavailable {
			t.Errorf("row %s = %+v, want unavailable", row.Dimension, row)
		}
	}
	if len(a.Recommendations) != 0 {
		t.Errorf("recommendations = %d, want 0", len(a.Recommendations))
	}
}

// quadrantPtr returns a pointer to q.
func quadrantPtr(q crap.Quadrant) *crap.Quadrant { return &q }

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// TestCodeSpan_ExactLiterals pins the CommonMark code-span algorithm for
// every backtick-position class, including the space-padding branch for
// names that begin or end with a backtick (CommonMark forbids a backtick
// adjacent to the delimiters, so the content is wrapped in spaces).
func TestCodeSpan_ExactLiterals(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", "`plain`"},
		{"a`b", "``a`b``"},
		{"a``b`c", "```a``b`c```"},
		{"`x", "`` `x ``"},
		{"x`", "`` x` ``"},
		{"``", "``` `` ```"},
	}

	for _, tc := range cases {
		if got := codeSpan(tc.in); got != tc.want {
			t.Errorf("codeSpan(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestLongestBacktickRun pins the delimiter-width computation, including
// the zero case for plain names and multi-backtick runs.
func TestLongestBacktickRun(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"plain", 0},
		{"a`b", 1},
		{"a``b`c", 2},
		{"```", 3},
		{"`a`", 1},
	}

	for _, tc := range cases {
		if got := longestBacktickRun(tc.in); got != tc.want {
			t.Errorf("longestBacktickRun(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// TestAssess_AmbiguousBoundaryNoRecommendation pins the boundary arms of
// ambiguousClassificationRecommendation: a classification run with zero
// effects, a nonzero ambiguous minority, and an exact tie must all emit no
// ambiguous_classification recommendation. The tie is the fragile arm — a
// `<=`→`<` regression would emit a spurious "50.0%" recommendation for it.
func TestAssess_AmbiguousBoundaryNoRecommendation(t *testing.T) {
	cases := []struct {
		name    string
		summary ReportSummary
	}{
		{"zero tallies with classification present", ReportSummary{}},
		{"nonzero ambiguous minority", ReportSummary{Contractual: 9, Ambiguous: 3, Incidental: 3}},
		{"nonzero exact tie", ReportSummary{Contractual: 2, Ambiguous: 3, Incidental: 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := Assess(assessPayload(t, assessCrapReport(), 140, tc.summary))
			for _, rec := range assessment.Recommendations {
				if rec.Source == "ambiguous_classification" {
					t.Fatalf("%s: spurious ambiguous_classification recommendation: %q", tc.name, rec.Text)
				}
			}
		})
	}
}
