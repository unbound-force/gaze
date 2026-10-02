package crap

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestWriteHTML_ScoresAndSummary(t *testing.T) {
	contractCoverage := 62.5
	gazeCRAP := 19.75
	gazeCRAPload := 1
	gazeCRAPThreshold := 12.5
	avgGazeCRAP := 10.25
	avgContractCoverage := 71.5
	quadrant := Q4Dangerous
	fixStrategy := FixAddAssertions
	reason := "all_effects_ambiguous"
	confidenceRange := [2]int{78, 79}
	report := &Report{
		Scores: []Score{{
			Package:                "example.com/store",
			Function:               "(*Store).Save",
			File:                   "store.go",
			Line:                   42,
			Complexity:             9,
			LineCoverage:           80.5,
			CRAP:                   12.75,
			ContractCoverage:       &contractCoverage,
			GazeCRAP:               &gazeCRAP,
			Quadrant:               &quadrant,
			FixStrategy:            &fixStrategy,
			ContractCoverageReason: &reason,
			EffectConfidenceRange:  &confidenceRange,
		}},
		Summary: Summary{
			TotalFunctions:      4,
			AvgComplexity:       5.5,
			AvgLineCoverage:     82.25,
			AvgCRAP:             7.75,
			CRAPload:            2,
			CRAPThreshold:       15,
			GazeCRAPload:        &gazeCRAPload,
			GazeCRAPThreshold:   &gazeCRAPThreshold,
			AvgGazeCRAP:         &avgGazeCRAP,
			AvgContractCoverage: &avgContractCoverage,
			QuadrantCounts: map[Quadrant]int{
				Q1Safe:                    1,
				Q2ComplexButTested:        2,
				Q3SimpleButUnderspecified: 3,
				Q4Dangerous:               4,
			},
			FixStrategyCounts: map[FixStrategy]int{
				FixDecompose:        1,
				FixAddTests:         2,
				FixAddAssertions:    3,
				FixDecomposeAndTest: 4,
			},
			WorstCRAP:     []Score{{Function: "WorstClassic", File: "classic.go", Line: 7, CRAP: 31.5}},
			WorstGazeCRAP: []Score{{Function: "WorstContract", File: "contract.go", Line: 8, GazeCRAP: &gazeCRAP}},
			RecommendedActions: []RecommendedAction{{
				Function:    "(*Store).Save",
				Package:     "example.com/store",
				File:        "store.go",
				Line:        42,
				FixStrategy: FixAddAssertions,
				CRAP:        12.75,
				GazeCRAP:    &gazeCRAP,
				Complexity:  9,
				Quadrant:    &quadrant,
			}},
			SSADegradedPackages: []string{"example.com/degraded"},
		},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, report); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	required := []string{
		"<!DOCTYPE html>", `<html lang="en">`, `<meta charset="utf-8">`, "<title>Gaze CRAP Report</title>",
		"<body>", "</body>", "Function Scores", "(*Store).Save", "example.com/store", "store.go:42",
		">9</td><td>80.5%</td><td>12.8</td><td>62.5%</td><td>19.8</td>",
		"Q4_Dangerous", "add_assertions", "all_effects_ambiguous (confidence 78–79)",
		"Functions analyzed</span><strong>4", "Average complexity</span><strong>5.5",
		"Average line coverage</span><strong>82.2%", "Average CRAP</span><strong>7.8",
		"CRAPload</span><strong>2", "threshold 15.0", "GazeCRAPload</span><strong>1",
		"Average GazeCRAP</span><strong>10.2", "Average contract coverage</span><strong>71.5%",
		"Quadrant Breakdown", "Q1_Safe", "Q2_ComplexButTested", "Q3_SimpleButUnderspecified",
		"Remediation Breakdown", "decompose", "add_tests", "decompose_and_test",
		"Recommended Actions", "Worst CRAP Offenders", "WorstClassic", "31.5",
		"Worst GazeCRAP Offenders", "WorstContract", "SSA analysis degraded.", "example.com/degraded",
	}
	for _, value := range required {
		if !strings.Contains(html, value) {
			t.Errorf("expected HTML to contain %q", value)
		}
	}
}

func TestWriteHTML_UnavailableGazeCRAP(t *testing.T) {
	report := &Report{
		Scores: []Score{{
			Function:     "Parse",
			File:         "parse.go",
			Line:         7,
			Complexity:   2,
			LineCoverage: 100,
			CRAP:         2,
		}},
		Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, report); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if !strings.Contains(html, `<td><span class="unavailable">Unavailable</span></td><td><span class="unavailable">Unavailable</span></td>`) {
		t.Error("unavailable contract coverage and GazeCRAP were not represented explicitly")
	}
	for _, fabricated := range []string{"GazeCRAPload</span>", "Average GazeCRAP</span>", "Worst GazeCRAP Offenders"} {
		if strings.Contains(html, fabricated) {
			t.Errorf("unavailable summary metric %q should be absent", fabricated)
		}
	}
}

func TestWriteHTML_EmptyResults(t *testing.T) {
	report := &Report{Summary: Summary{CRAPThreshold: 15}}

	var output bytes.Buffer
	if err := WriteHTML(&output, report); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	for _, value := range []string{"<!DOCTYPE html>", "Gaze CRAP Report", "No functions were analyzed.", "Functions analyzed</span><strong>0", "</html>"} {
		if !strings.Contains(html, value) {
			t.Errorf("expected empty report HTML to contain %q", value)
		}
	}
	if strings.Contains(html, "Function Scores") {
		t.Error("empty report should not render a function scores table")
	}
}

func TestWriteComparisonHTML_FailedComparisonAndCategorizedDeltas(t *testing.T) {
	baselineRegressionGaze := 10.0
	currentRegressionGaze := 14.5
	regressionGazeDelta := 4.5
	baselineImprovementGaze := 20.0
	currentImprovementGaze := 12.0
	improvementGazeDelta := -8.0
	result := &ComparisonResult{
		Report: &Report{Summary: Summary{TotalFunctions: 4, CRAPThreshold: 15}},
		Deltas: []FunctionDelta{
			{
				Baseline:      Score{CRAP: 11, GazeCRAP: &baselineRegressionGaze},
				Current:       Score{Package: "example.com/regressed", Function: "Regressed", File: "regressed.go", Line: 10, CRAP: 14, GazeCRAP: &currentRegressionGaze},
				CRAPDelta:     3,
				GazeCRAPDelta: &regressionGazeDelta,
				Status:        StatusRegression,
			},
			{
				Baseline:      Score{CRAP: 22, GazeCRAP: &baselineImprovementGaze},
				Current:       Score{Package: "example.com/improved", Function: "Improved", File: "improved.go", Line: 20, CRAP: 13, GazeCRAP: &currentImprovementGaze},
				CRAPDelta:     -9,
				GazeCRAPDelta: &improvementGazeDelta,
				Status:        StatusImprovement,
			},
		},
		NewFunctions: []Score{
			{Function: "HealthyNew", File: "healthy.go", Line: 30, CRAP: 5},
			{Function: "ViolatingNew", File: "violation.go", Line: 40, CRAP: 31},
		},
		RemovedFunctions: []Score{{Function: "Removed", File: "removed.go", Line: 50, CRAP: 8}},
		Summary: ComparisonSummary{
			Regressions:                  1,
			Improvements:                 1,
			NewFunctions:                 1,
			NewViolations:                1,
			RemovedFunctions:             1,
			Unchanged:                    2,
			Epsilon:                      0.5,
			NewFunctionThreshold:         30,
			NewFunctionGazeCRAPThreshold: 20,
			Passed:                       false,
		},
	}

	var output bytes.Buffer
	if err := WriteComparisonHTML(&output, result); err != nil {
		t.Fatalf("WriteComparisonHTML failed: %v", err)
	}

	html := output.String()
	required := []string{
		"Baseline Comparison", `<div class="status fail"><strong>FAIL</strong>`,
		"Regressions</span><strong>1", "Improvements</span><strong>1", "New functions</span><strong>1",
		"New violations</span><strong>1", "Removed functions</span><strong>1", "Unchanged</span><strong>2",
		"Delta epsilon</span><strong>0.5", "CRAP 30.0", "GazeCRAP 20.0",
		"<h3>Regressions</h3>", "Regressed", "regressed.go:10", "11.0", "14.0", "&#43;3.0", "10.0", "14.5", "&#43;4.5",
		"<h3>Improvements</h3>", "Improved", "improved.go:20", "22.0", "13.0", "-9.0", "20.0", "12.0", "-8.0",
		"New Functions (violations)", "ViolatingNew", "violation.go:40", "31.0",
		"<h3>New Functions</h3>", "HealthyNew", "healthy.go:30", "5.0",
		"Removed Functions", "Removed", "removed.go:50", "8.0",
	}
	for _, value := range required {
		if !strings.Contains(html, value) {
			t.Errorf("expected comparison HTML to contain %q", value)
		}
	}
}

func TestWriteComparisonHTML_PassedComparison(t *testing.T) {
	result := &ComparisonResult{
		Report:  &Report{Summary: Summary{CRAPThreshold: 15}},
		Summary: ComparisonSummary{Passed: true},
	}

	var output bytes.Buffer
	if err := WriteComparisonHTML(&output, result); err != nil {
		t.Fatalf("WriteComparisonHTML failed: %v", err)
	}
	if !strings.Contains(output.String(), `<div class="status pass"><strong>PASS</strong>`) {
		t.Error("passing baseline comparison status is missing")
	}
}

func TestWriteHTML_AbsentBaselineSection(t *testing.T) {
	report := &Report{
		Scores:  []Score{{Function: "Stable", File: "stable.go", Line: 3, CRAP: 2}},
		Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, report); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if !strings.Contains(html, "Function Scores") || !strings.Contains(html, "Stable") {
		t.Error("normal CRAP results are missing")
	}
	for _, baselineMarkup := range []string{"Baseline Comparison", `<div class="status pass">`, `<div class="status fail">`} {
		if strings.Contains(html, baselineMarkup) {
			t.Errorf("normal report should not contain baseline markup %q", baselineMarkup)
		}
	}
}

func TestWriteHTML_AdversarialEscapingAndSelfContainment(t *testing.T) {
	adversarial := `<script>alert("crap")</script>&<img src=x onerror=alert(1)>"'><style>bad</style>`
	reason := adversarial
	report := &Report{
		Scores: []Score{{
			Package:                adversarial,
			Function:               adversarial,
			File:                   adversarial,
			CRAP:                   1,
			ContractCoverageReason: &reason,
		}},
		Summary: Summary{
			TotalFunctions:      1,
			CRAPThreshold:       15,
			SSADegradedPackages: []string{adversarial},
			WorstCRAP:           []Score{{Function: adversarial, File: adversarial, CRAP: 1}},
			RecommendedActions: []RecommendedAction{{
				Function: adversarial,
				Package:  adversarial,
				File:     adversarial,
			}},
		},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, report); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if strings.Contains(html, adversarial) || strings.Contains(html, `<script>alert`) || strings.Contains(html, `<img src=x`) || strings.Contains(html, `<style>bad`) {
		t.Error("source-derived markup was rendered as active HTML")
	}
	if !strings.Contains(html, `&lt;script&gt;alert(&#34;crap&#34;)&lt;/script&gt;&amp;&lt;img`) {
		t.Error("expected adversarial content to appear as escaped text")
	}
	for _, forbidden := range []string{"<link ", "<script", "<img", "@import", "http://", "https://", "url("} {
		if strings.Contains(html, forbidden) {
			t.Errorf("self-contained HTML must not contain %q", forbidden)
		}
	}
	if strings.Count(html, "<style>") != 1 {
		t.Errorf("self-contained HTML should contain one inline style block, got %d", strings.Count(html, "<style>"))
	}
}

func TestWriteHTML_DeterministicBytes(t *testing.T) {
	report := &Report{
		Scores: []Score{
			{Function: "Lower", File: "lower.go", CRAP: 2},
			{Function: "Higher", File: "higher.go", CRAP: 8},
		},
		Summary: Summary{
			TotalFunctions: 2,
			CRAPThreshold:  15,
			QuadrantCounts: map[Quadrant]int{
				Q4Dangerous: 1,
				Q1Safe:      1,
			},
			FixStrategyCounts: map[FixStrategy]int{
				FixAddTests:  1,
				FixDecompose: 1,
			},
		},
	}

	var first bytes.Buffer
	if err := WriteHTML(&first, report); err != nil {
		t.Fatalf("first WriteHTML call failed: %v", err)
	}
	var second bytes.Buffer
	if err := WriteHTML(&second, report); err != nil {
		t.Fatalf("second WriteHTML call failed: %v", err)
	}

	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("identical ordered input produced different HTML bytes")
	}
}

func TestWriteHTML_WriterFailure(t *testing.T) {
	err := WriteHTML(crapErrorWriter{}, &Report{})
	if err == nil {
		t.Fatal("expected WriteHTML to return the writer failure")
	}
	if !errors.Is(err, errCRAPWrite) {
		t.Errorf("expected wrapped writer error, got %v", err)
	}
	if !strings.Contains(err.Error(), "executing CRAP HTML template") {
		t.Errorf("expected operation-specific context, got %v", err)
	}
}

var errCRAPWrite = errors.New("CRAP writer failed")

type crapErrorWriter struct{}

func (crapErrorWriter) Write([]byte) (int, error) {
	return 0, errCRAPWrite
}
