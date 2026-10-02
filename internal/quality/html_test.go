package quality

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

func TestWriteHTML_CompleteStructureAndAvailableFields(t *testing.T) {
	report := taxonomy.QualityReport{
		TestFunction: "TestStore_Save",
		TestLocation: "store_test.go:21:1",
		TargetFunction: taxonomy.FunctionTarget{
			Package:   "example.com/store",
			Function:  "Save",
			Receiver:  "*Store",
			Signature: "func (s *Store) Save(item Item) (int64, error)",
			Location:  "store.go:42:1",
		},
		ContractCoverage: taxonomy.ContractCoverage{
			Percentage:       50,
			CoveredCount:     1,
			TotalContractual: 2,
			Gaps: []taxonomy.SideEffect{{
				Type:        taxonomy.ReturnValue,
				Description: "returns the saved item ID",
				Location:    "store.go:49:2",
			}},
			GapHints: []string{"compare the returned ID"},
			DiscardedReturns: []taxonomy.SideEffect{{
				Type:        taxonomy.ErrorReturn,
				Description: "returns a persistence error",
				Location:    "store.go:49:9",
			}},
			DiscardedReturnHints: []string{"capture and check the error"},
		},
		OverSpecification: taxonomy.OverSpecificationScore{
			Count:       1,
			Ratio:       0.25,
			Suggestions: []string{"assert on the public result instead"},
		},
		AmbiguousEffects: []taxonomy.SideEffect{{
			Type:        taxonomy.LogWrite,
			Description: "writes an audit message",
			Location:    "store.go:45:2",
		}},
		UnmappedAssertions: []taxonomy.AssertionMapping{{
			AssertionLocation: "store_test.go:28:2",
			AssertionType:     taxonomy.AssertionEquality,
			UnmappedReason:    taxonomy.UnmappedReasonNoEffectMatch,
		}},
		AssertionCount:               4,
		AssertionDetectionConfidence: 75,
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:                   2,
		AverageContractCoverage:      60,
		TotalOverSpecifications:      1,
		AssertionDetectionConfidence: 85,
		WorstCoverageTests:           []taxonomy.QualityReport{report},
		ClassificationCounts: &taxonomy.ClassificationCounts{
			Contractual: 3,
			Incidental:  1,
			Ambiguous:   2,
		},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, []taxonomy.QualityReport{report}, summary); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	required := []string{
		"<!DOCTYPE html>", `<html lang="en">`, "<head>", "</head>", "<body>", "</body>", "</html>",
		`<meta charset="utf-8">`, `<meta name="viewport"`, "<title>Gaze Quality Report</title>", "<style>",
		"1 test-target pair(s) resolved", "Package Summary", "Tests analyzed</span><strong>2",
		"Average contract coverage</span><strong>60%", "Total over-specifications</span><strong>1",
		"Assertion detection confidence</span><strong>85%", "Classification Counts", "Contractual</span><strong>3",
		"Incidental</span><strong>1", "Ambiguous</span><strong>2", "Lowest Coverage Tests", "50% (1/2)",
		"Test Results", "TestStore_Save", "store_test.go:21:1", "(*Store).Save", "store.go:42:1",
		"func (s *Store) Save(item Item) (int64, error)", "Contract coverage</span><strong>50% (1/2)",
		"Over-specification</span><strong>1 (25%)", "Assertions detected</span><strong>4",
		"Assertion detection confidence</span><strong>75%", "Gaps", "returns the saved item ID",
		"store.go:49:2", "compare the returned ID", "Discarded Returns", "returns a persistence error",
		"store.go:49:9", "capture and check the error", "Remediation Suggestions",
		"assert on the public result instead", "Ambiguous Effects", "writes an audit message",
		"Unmapped Assertions", "store_test.go:28:2", "equality", "no_effect_match",
	}
	for _, value := range required {
		if !strings.Contains(html, value) {
			t.Errorf("expected HTML to contain %q", value)
		}
	}
}

func TestWriteHTML_AbsentOptionalSections(t *testing.T) {
	report := taxonomy.QualityReport{
		TestFunction: "TestParse",
		TestLocation: "parse_test.go:10:1",
		TargetFunction: taxonomy.FunctionTarget{
			Function:  "Parse",
			Signature: "func Parse(string) error",
			Location:  "parse.go:7:1",
		},
		ContractCoverage: taxonomy.ContractCoverage{
			Percentage:       100,
			CoveredCount:     1,
			TotalContractual: 1,
		},
		AssertionCount:               1,
		AssertionDetectionConfidence: 100,
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:                   1,
		AverageContractCoverage:      100,
		AssertionDetectionConfidence: 100,
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, []taxonomy.QualityReport{report}, summary); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	for _, heading := range []string{
		"Gaps", "Discarded Returns", "Remediation Suggestions", "Ambiguous Effects",
		"Unmapped Assertions", "Classification Counts", "Lowest Coverage Tests",
		"Unavailable analysis", "SSA degradation", "Skipped tests",
	} {
		if strings.Contains(html, heading) {
			t.Errorf("optional section %q should be absent", heading)
		}
	}
	if !strings.Contains(html, "Contract coverage</span><strong>100% (1/1)") {
		t.Error("required available contract coverage is missing")
	}
}

func TestWriteHTML_EmptyDegradedDiagnostics(t *testing.T) {
	names := make([]string, 22)
	for index := range names {
		names[index] = "TestSkipped" + string(rune('A'+index))
	}
	summary := &taxonomy.PackageSummary{
		Reason:                       "test_mapping_unavailable",
		TotalTests:                   22,
		AssertionDetectionConfidence: 40,
		SSADegraded:                  true,
		SSADegradedPackages:          []string{"example.com/store", "example.com/payments"},
		SkippedTests:                 22,
		SkippedTestNames:             names,
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, nil, summary); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	for _, value := range []string{
		"No test-target pairs were resolved.", "Quality metrics are unavailable.", "test_mapping_unavailable",
		"SSA analysis degraded.", "example.com/store", "example.com/payments",
		"22 test function(s) skipped", "TestSkippedA", "TestSkippedT", "… and 2 more", "--target=FuncName",
		"Tests analyzed</span><strong>22",
	} {
		if !strings.Contains(html, value) {
			t.Errorf("expected diagnostic HTML to contain %q", value)
		}
	}
	for _, unavailableMetric := range []string{"Average contract coverage", "Total over-specifications"} {
		if strings.Contains(html, unavailableMetric) {
			t.Errorf("degraded HTML must not fabricate %q", unavailableMetric)
		}
	}
	if strings.Contains(html, "TestSkippedU") {
		t.Error("skipped test names beyond the display limit must be omitted")
	}
}

func TestWriteHTML_MixedDegradedAndHealthyReportsPreserveAvailableMetrics(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestHealthy",
			TargetFunction: taxonomy.FunctionTarget{
				Function: "Healthy",
			},
			ContractCoverage: taxonomy.ContractCoverage{
				Percentage:       75,
				CoveredCount:     3,
				TotalContractual: 4,
			},
			OverSpecification: taxonomy.OverSpecificationScore{Count: 1, Ratio: 0.25},
		},
		{TestFunction: "TestDegraded"},
	}
	summary := &taxonomy.PackageSummary{
		SSADegraded:         true,
		SSADegradedPackages: []string{"example.com/degraded"},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, reports, summary); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if !strings.Contains(html, "Contract coverage</span><strong>75% (3/4)") {
		t.Error("healthy report metrics must remain visible when another package is degraded")
	}
	if !strings.Contains(html, "Target unavailable.") {
		t.Error("degraded report without a target must retain unavailable behavior")
	}
	if strings.Contains(html, "Contract coverage</span><strong>0% (0/0)") {
		t.Error("report without a target must not render unavailable metrics as zero")
	}
}

func TestWriteHTML_AdversarialEscapingAndSelfContainment(t *testing.T) {
	adversarial := `<script>alert("quality")</script>&<img src=x onerror=alert(1)>`
	report := taxonomy.QualityReport{
		TestFunction: adversarial,
		TestLocation: adversarial,
		TargetFunction: taxonomy.FunctionTarget{
			Function:  adversarial,
			Receiver:  adversarial,
			Signature: adversarial,
			Location:  adversarial,
		},
		ContractCoverage: taxonomy.ContractCoverage{
			Gaps: []taxonomy.SideEffect{{
				Type:        taxonomy.SideEffectType(adversarial),
				Description: adversarial,
				Location:    adversarial,
			}},
			GapHints: []string{adversarial},
		},
		OverSpecification: taxonomy.OverSpecificationScore{Suggestions: []string{adversarial}},
		AmbiguousEffects: []taxonomy.SideEffect{{
			Type:        taxonomy.SideEffectType(adversarial),
			Description: adversarial,
			Location:    adversarial,
		}},
		UnmappedAssertions: []taxonomy.AssertionMapping{{
			AssertionLocation: adversarial,
			AssertionType:     taxonomy.AssertionType(adversarial),
			UnmappedReason:    taxonomy.UnmappedReasonType(adversarial),
		}},
	}
	summary := &taxonomy.PackageSummary{
		TotalTests:          1,
		SSADegraded:         true,
		SSADegradedPackages: []string{adversarial},
		SkippedTests:        1,
		SkippedTestNames:    []string{adversarial},
		WorstCoverageTests:  []taxonomy.QualityReport{{TestFunction: adversarial}},
	}

	var output bytes.Buffer
	if err := WriteHTML(&output, []taxonomy.QualityReport{report}, summary); err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	html := output.String()
	if strings.Contains(html, adversarial) || strings.Contains(html, `<script>alert`) || strings.Contains(html, `<img src=x`) {
		t.Error("source-derived markup was rendered as active HTML")
	}
	if !strings.Contains(html, `&lt;script&gt;alert(&#34;quality&#34;)&lt;/script&gt;&amp;&lt;img`) {
		t.Error("expected adversarial content to appear as escaped text")
	}
	for _, forbidden := range []string{"<link ", "<script", "<img", "@import", "http://", "https://"} {
		if strings.Contains(html, forbidden) {
			t.Errorf("self-contained HTML must not contain %q", forbidden)
		}
	}
	if !strings.Contains(html, "<style>") {
		t.Error("self-contained HTML must include inline styling")
	}
}

func TestWriteHTML_DeterministicBytes(t *testing.T) {
	reports := []taxonomy.QualityReport{{
		TestFunction: "TestDeterministic",
		TargetFunction: taxonomy.FunctionTarget{
			Function: "Deterministic",
		},
		AssertionCount: 1,
	}}
	summary := &taxonomy.PackageSummary{TotalTests: 1}

	var first bytes.Buffer
	if err := WriteHTML(&first, reports, summary); err != nil {
		t.Fatalf("first WriteHTML call failed: %v", err)
	}
	var second bytes.Buffer
	if err := WriteHTML(&second, reports, summary); err != nil {
		t.Fatalf("second WriteHTML call failed: %v", err)
	}

	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Error("identical ordered input produced different HTML bytes")
	}
}

func TestWriteHTML_WriterFailure(t *testing.T) {
	err := WriteHTML(qualityErrorWriter{}, nil, nil)
	if err == nil {
		t.Fatal("expected WriteHTML to return the writer failure")
	}
	if !errors.Is(err, errQualityWrite) {
		t.Errorf("expected wrapped writer error, got %v", err)
	}
	if !strings.Contains(err.Error(), "executing quality HTML template") {
		t.Errorf("expected operation-specific context, got %v", err)
	}
}

var errQualityWrite = errors.New("quality writer failed")

type qualityErrorWriter struct{}

func (qualityErrorWriter) Write([]byte) (int, error) {
	return 0, errQualityWrite
}
