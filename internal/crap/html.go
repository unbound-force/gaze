package crap

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"sort"
)

//go:embed crap.html.tmpl
var crapTemplateFS embed.FS

type crapHTMLData struct {
	Scores     []crapHTMLScore
	Summary    crapHTMLSummary
	Comparison *crapHTMLComparison
}

type crapHTMLScore struct {
	Package                   string
	Function                  string
	File                      string
	Line                      int
	Complexity                int
	LineCoverage              float64
	CRAP                      float64
	ContractCoverage          optionalFloat
	GazeCRAP                  optionalFloat
	Quadrant                  string
	FixStrategy               string
	ContractCoverageReason    string
	EffectConfidenceAvailable bool
	EffectConfidenceMin       int
	EffectConfidenceMax       int
}

type crapHTMLSummary struct {
	TotalFunctions      int
	AvgComplexity       float64
	AvgLineCoverage     float64
	AvgCRAP             float64
	CRAPload            int
	CRAPThreshold       float64
	GazeCRAPload        optionalInt
	GazeCRAPThreshold   optionalFloat
	AvgGazeCRAP         optionalFloat
	AvgContractCoverage optionalFloat
	Quadrants           []crapHTMLCount
	Remediations        []crapHTMLCount
	WorstCRAP           []crapHTMLScore
	WorstGazeCRAP       []crapHTMLScore
	RecommendedActions  []crapHTMLAction
	SSADegradedPackages []string
}

type crapHTMLCount struct {
	Name  string
	Count int
}

type crapHTMLAction struct {
	Function   string
	Package    string
	File       string
	Line       int
	Strategy   string
	CRAP       float64
	GazeCRAP   optionalFloat
	Complexity int
	Quadrant   string
}

type crapHTMLComparison struct {
	Passed                       bool
	Regressions                  int
	Improvements                 int
	NewFunctions                 int
	NewViolations                int
	RemovedFunctions             int
	Unchanged                    int
	Epsilon                      float64
	NewFunctionThreshold         float64
	NewFunctionGazeCRAPThreshold float64
	RegressionDeltas             []crapHTMLDelta
	ImprovementDeltas            []crapHTMLDelta
	New                          []crapHTMLScore
	NewViolation                 []crapHTMLScore
	Removed                      []crapHTMLScore
}

type crapHTMLDelta struct {
	Package          string
	Function         string
	File             string
	Line             int
	BaselineCRAP     float64
	CurrentCRAP      float64
	CRAPDelta        float64
	BaselineGazeCRAP optionalFloat
	CurrentGazeCRAP  optionalFloat
	GazeCRAPDelta    optionalFloat
}

type optionalFloat struct {
	Available bool
	Value     float64
}

type optionalInt struct {
	Available bool
	Value     int
}

// WriteHTML writes a CRAP report as a complete, self-contained HTML document
// to w. Source and diagnostic values remain plain strings so html/template
// applies contextual escaping. Writer and template failures are wrapped with
// operation-specific context.
func WriteHTML(w io.Writer, report *Report) error {
	return writeHTML(w, buildCRAPHTMLData(report, nil))
}

// WriteComparisonHTML writes a CRAP report and its baseline comparison as a
// complete, self-contained HTML document to w. It preserves unavailable
// metrics instead of presenting them as measured zero values. Writer and
// template failures are wrapped with operation-specific context.
func WriteComparisonHTML(w io.Writer, result *ComparisonResult) error {
	return writeHTML(w, buildCRAPHTMLData(result.Report, result))
}

func writeHTML(w io.Writer, data crapHTMLData) error {
	tmpl, err := template.ParseFS(crapTemplateFS, "crap.html.tmpl")
	if err != nil {
		return fmt.Errorf("parsing CRAP HTML template: %w", err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing CRAP HTML template: %w", err)
	}
	return nil
}

func buildCRAPHTMLData(report *Report, comparison *ComparisonResult) crapHTMLData {
	scores := append([]Score(nil), report.Scores...)
	sort.SliceStable(scores, func(i, j int) bool { return scores[i].CRAP > scores[j].CRAP })

	data := crapHTMLData{
		Scores:  buildCRAPHTMLScores(scores),
		Summary: buildCRAPHTMLSummary(report.Summary),
	}
	if comparison != nil {
		data.Comparison = buildCRAPHTMLComparison(comparison)
	}
	return data
}

func buildCRAPHTMLScores(scores []Score) []crapHTMLScore {
	views := make([]crapHTMLScore, 0, len(scores))
	for _, score := range scores {
		view := crapHTMLScore{
			Package:                score.Package,
			Function:               score.Function,
			File:                   score.File,
			Line:                   score.Line,
			Complexity:             score.Complexity,
			LineCoverage:           score.LineCoverage,
			CRAP:                   score.CRAP,
			ContractCoverage:       optionalFloatValue(score.ContractCoverage),
			GazeCRAP:               optionalFloatValue(score.GazeCRAP),
			ContractCoverageReason: optionalStringValue(score.ContractCoverageReason),
		}
		if score.Quadrant != nil {
			view.Quadrant = string(*score.Quadrant)
		}
		if score.FixStrategy != nil {
			view.FixStrategy = string(*score.FixStrategy)
		}
		if score.EffectConfidenceRange != nil {
			view.EffectConfidenceAvailable = true
			view.EffectConfidenceMin = score.EffectConfidenceRange[0]
			view.EffectConfidenceMax = score.EffectConfidenceRange[1]
		}
		views = append(views, view)
	}
	return views
}

func buildCRAPHTMLSummary(summary Summary) crapHTMLSummary {
	view := crapHTMLSummary{
		TotalFunctions:      summary.TotalFunctions,
		AvgComplexity:       summary.AvgComplexity,
		AvgLineCoverage:     summary.AvgLineCoverage,
		AvgCRAP:             summary.AvgCRAP,
		CRAPload:            summary.CRAPload,
		CRAPThreshold:       summary.CRAPThreshold,
		GazeCRAPload:        optionalIntValue(summary.GazeCRAPload),
		GazeCRAPThreshold:   optionalFloatValue(summary.GazeCRAPThreshold),
		AvgGazeCRAP:         optionalFloatValue(summary.AvgGazeCRAP),
		AvgContractCoverage: optionalFloatValue(summary.AvgContractCoverage),
		WorstCRAP:           buildCRAPHTMLScores(summary.WorstCRAP),
		WorstGazeCRAP:       buildCRAPHTMLScores(summary.WorstGazeCRAP),
		SSADegradedPackages: summary.SSADegradedPackages,
	}

	if len(summary.QuadrantCounts) > 0 {
		for _, quadrant := range []Quadrant{Q1Safe, Q2ComplexButTested, Q3SimpleButUnderspecified, Q4Dangerous} {
			view.Quadrants = append(view.Quadrants, crapHTMLCount{Name: string(quadrant), Count: summary.QuadrantCounts[quadrant]})
		}
	}
	for _, strategy := range []FixStrategy{FixDecompose, FixAddTests, FixAddAssertions, FixDecomposeAndTest} {
		if count := summary.FixStrategyCounts[strategy]; count > 0 {
			view.Remediations = append(view.Remediations, crapHTMLCount{Name: string(strategy), Count: count})
		}
	}
	for _, action := range summary.RecommendedActions {
		actionView := crapHTMLAction{
			Function:   action.Function,
			Package:    action.Package,
			File:       action.File,
			Line:       action.Line,
			Strategy:   string(action.FixStrategy),
			CRAP:       action.CRAP,
			GazeCRAP:   optionalFloatValue(action.GazeCRAP),
			Complexity: action.Complexity,
		}
		if action.Quadrant != nil {
			actionView.Quadrant = string(*action.Quadrant)
		}
		view.RecommendedActions = append(view.RecommendedActions, actionView)
	}
	return view
}

func buildCRAPHTMLComparison(result *ComparisonResult) *crapHTMLComparison {
	summary := result.Summary
	view := &crapHTMLComparison{
		Passed:                       summary.Passed,
		Regressions:                  summary.Regressions,
		Improvements:                 summary.Improvements,
		NewFunctions:                 summary.NewFunctions,
		NewViolations:                summary.NewViolations,
		RemovedFunctions:             summary.RemovedFunctions,
		Unchanged:                    summary.Unchanged,
		Epsilon:                      summary.Epsilon,
		NewFunctionThreshold:         summary.NewFunctionThreshold,
		NewFunctionGazeCRAPThreshold: summary.NewFunctionGazeCRAPThreshold,
		Removed:                      buildCRAPHTMLScores(result.RemovedFunctions),
	}
	for _, delta := range result.Deltas {
		deltaView := crapHTMLDelta{
			Package:          delta.Current.Package,
			Function:         delta.Current.Function,
			File:             delta.Current.File,
			Line:             delta.Current.Line,
			BaselineCRAP:     delta.Baseline.CRAP,
			CurrentCRAP:      delta.Current.CRAP,
			CRAPDelta:        delta.CRAPDelta,
			BaselineGazeCRAP: optionalFloatValue(delta.Baseline.GazeCRAP),
			CurrentGazeCRAP:  optionalFloatValue(delta.Current.GazeCRAP),
			GazeCRAPDelta:    optionalFloatValue(delta.GazeCRAPDelta),
		}
		switch delta.Status {
		case StatusRegression:
			view.RegressionDeltas = append(view.RegressionDeltas, deltaView)
		case StatusImprovement:
			view.ImprovementDeltas = append(view.ImprovementDeltas, deltaView)
		}
	}
	for _, score := range result.NewFunctions {
		if isNewFunctionViolation(score, summary.NewFunctionThreshold, summary.NewFunctionGazeCRAPThreshold) {
			view.NewViolation = append(view.NewViolation, buildCRAPHTMLScores([]Score{score})...)
		} else {
			view.New = append(view.New, buildCRAPHTMLScores([]Score{score})...)
		}
	}
	return view
}

func optionalFloatValue(value *float64) optionalFloat {
	if value == nil {
		return optionalFloat{}
	}
	return optionalFloat{Available: true, Value: *value}
}

func optionalIntValue(value *int) optionalInt {
	if value == nil {
		return optionalInt{}
	}
	return optionalInt{Available: true, Value: *value}
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
