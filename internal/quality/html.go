package quality

import (
	"embed"
	"fmt"
	"html/template"
	"io"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

//go:embed quality.html.tmpl
var qualityTemplateFS embed.FS

type qualityHTMLData struct {
	Reports       []qualityHTMLReport
	Summary       *qualityHTMLSummary
	ResolvedPairs int
}

type qualityHTMLReport struct {
	TestFunction                 string
	TestLocation                 string
	TargetName                   string
	TargetLocation               string
	TargetSignature              string
	TargetAvailable              bool
	MetricsAvailable             bool
	ContractCoverage             float64
	CoveredCount                 int
	TotalContractual             int
	OverSpecificationCount       int
	OverSpecificationPercentage  float64
	AssertionCount               int
	AssertionDetectionConfidence int
	Gaps                         []qualityHTMLEffect
	DiscardedReturns             []qualityHTMLEffect
	Suggestions                  []string
	AmbiguousEffects             []qualityHTMLEffect
	UnmappedAssertions           []qualityHTMLAssertion
	HasUnmappedReasons           bool
}

type qualityHTMLEffect struct {
	Type        string
	Description string
	Location    string
	Hint        string
}

type qualityHTMLAssertion struct {
	Location string
	Type     string
	Reason   string
}

type qualityHTMLSummary struct {
	Reason                       string
	MetricsAvailable             bool
	DetectionAvailable           bool
	TotalTests                   int
	AverageContractCoverage      float64
	TotalOverSpecifications      int
	AssertionDetectionConfidence int
	WorstCoverageTests           []qualityHTMLWorstTest
	SSADegraded                  bool
	SSADegradedPackages          []string
	SkippedTests                 int
	SkippedTestNames             []string
	SkippedTestsRemaining        int
	ClassificationCounts         *qualityHTMLClassificationCounts
}

type qualityHTMLWorstTest struct {
	TestFunction     string
	Coverage         float64
	CoveredCount     int
	TotalContractual int
}

type qualityHTMLClassificationCounts struct {
	Contractual int
	Incidental  int
	Ambiguous   int
}

// WriteHTML writes quality reports and their package summary as a complete,
// self-contained HTML document to w. Source and analyzer values remain plain
// strings so html/template applies contextual escaping. Writer and template
// failures are returned with operation-specific context.
func WriteHTML(w io.Writer, reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) error {
	data := buildQualityHTMLData(reports, summary)

	tmpl, err := template.ParseFS(qualityTemplateFS, "quality.html.tmpl")
	if err != nil {
		return fmt.Errorf("parsing quality HTML template: %w", err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing quality HTML template: %w", err)
	}

	return nil
}

func buildQualityHTMLData(reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) qualityHTMLData {
	data := qualityHTMLData{Summary: buildQualityHTMLSummary(summary, len(reports))}
	metricsAvailable := summary == nil || (!summary.SSADegraded && summary.Reason == "")

	for _, report := range reports {
		targetAvailable := report.TargetFunction.Function != ""
		view := qualityHTMLReport{
			TestFunction:                 report.TestFunction,
			TestLocation:                 report.TestLocation,
			TargetName:                   report.TargetFunction.QualifiedName(),
			TargetLocation:               report.TargetFunction.Location,
			TargetSignature:              report.TargetFunction.Signature,
			TargetAvailable:              targetAvailable,
			MetricsAvailable:             metricsAvailable && targetAvailable,
			ContractCoverage:             report.ContractCoverage.Percentage,
			CoveredCount:                 report.ContractCoverage.CoveredCount,
			TotalContractual:             report.ContractCoverage.TotalContractual,
			OverSpecificationCount:       report.OverSpecification.Count,
			OverSpecificationPercentage:  report.OverSpecification.Ratio * 100,
			AssertionCount:               report.AssertionCount,
			AssertionDetectionConfidence: report.AssertionDetectionConfidence,
			Gaps:                         buildQualityHTMLEffects(report.ContractCoverage.Gaps, report.ContractCoverage.GapHints),
			DiscardedReturns:             buildQualityHTMLEffects(report.ContractCoverage.DiscardedReturns, report.ContractCoverage.DiscardedReturnHints),
			Suggestions:                  report.OverSpecification.Suggestions,
			AmbiguousEffects:             buildQualityHTMLEffects(report.AmbiguousEffects, nil),
			UnmappedAssertions:           buildQualityHTMLAssertions(report.UnmappedAssertions),
		}
		for _, assertion := range view.UnmappedAssertions {
			if assertion.Reason != "" {
				view.HasUnmappedReasons = true
				break
			}
		}
		if targetAvailable {
			data.ResolvedPairs++
		}
		data.Reports = append(data.Reports, view)
	}

	return data
}

func buildQualityHTMLEffects(effects []taxonomy.SideEffect, hints []string) []qualityHTMLEffect {
	views := make([]qualityHTMLEffect, 0, len(effects))
	for index, effect := range effects {
		view := qualityHTMLEffect{
			Type:        string(effect.Type),
			Description: effect.Description,
			Location:    effect.Location,
		}
		if index < len(hints) {
			view.Hint = hints[index]
		}
		views = append(views, view)
	}
	return views
}

func buildQualityHTMLAssertions(assertions []taxonomy.AssertionMapping) []qualityHTMLAssertion {
	views := make([]qualityHTMLAssertion, 0, len(assertions))
	for _, assertion := range assertions {
		views = append(views, qualityHTMLAssertion{
			Location: assertion.AssertionLocation,
			Type:     string(assertion.AssertionType),
			Reason:   string(assertion.UnmappedReason),
		})
	}
	return views
}

func buildQualityHTMLSummary(summary *taxonomy.PackageSummary, reportCount int) *qualityHTMLSummary {
	if summary == nil {
		return nil
	}

	view := &qualityHTMLSummary{
		Reason:                       summary.Reason,
		MetricsAvailable:             summary.Reason == "" && !summary.SSADegraded && reportCount > 0,
		DetectionAvailable:           reportCount > 0,
		TotalTests:                   summary.TotalTests,
		AverageContractCoverage:      summary.AverageContractCoverage,
		TotalOverSpecifications:      summary.TotalOverSpecifications,
		AssertionDetectionConfidence: summary.AssertionDetectionConfidence,
		SSADegraded:                  summary.SSADegraded,
		SSADegradedPackages:          summary.SSADegradedPackages,
		SkippedTests:                 summary.SkippedTests,
	}

	nameLimit := len(summary.SkippedTestNames)
	if nameLimit > summary.SkippedTests {
		nameLimit = summary.SkippedTests
	}
	if nameLimit > MaxSkippedTestDisplay {
		nameLimit = MaxSkippedTestDisplay
	}
	view.SkippedTestNames = summary.SkippedTestNames[:nameLimit]
	if summary.SkippedTests > MaxSkippedTestDisplay {
		view.SkippedTestsRemaining = summary.SkippedTests - MaxSkippedTestDisplay
	}

	for _, report := range summary.WorstCoverageTests {
		view.WorstCoverageTests = append(view.WorstCoverageTests, qualityHTMLWorstTest{
			TestFunction:     report.TestFunction,
			Coverage:         report.ContractCoverage.Percentage,
			CoveredCount:     report.ContractCoverage.CoveredCount,
			TotalContractual: report.ContractCoverage.TotalContractual,
		})
	}

	if summary.ClassificationCounts != nil {
		view.ClassificationCounts = &qualityHTMLClassificationCounts{
			Contractual: summary.ClassificationCounts.Contractual,
			Incidental:  summary.ClassificationCounts.Incidental,
			Ambiguous:   summary.ClassificationCounts.Ambiguous,
		}
	}

	return view
}
