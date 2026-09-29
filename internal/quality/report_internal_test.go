package quality

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TestWriteContractCoverage_Thresholds verifies the style threshold boundaries
// for contract coverage: <50 → bad, >=50 && <80 → warn, >=80 → good.
func TestWriteContractCoverage_Thresholds(t *testing.T) {
	tests := []struct {
		name      string
		covPct    float64
		wantStyle func(s qualityStyles) lipgloss.Style
	}{
		{name: "49_bad", covPct: 49, wantStyle: func(s qualityStyles) lipgloss.Style { return s.bad }},
		{name: "50_warn", covPct: 50, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "79_warn", covPct: 79, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "80_good", covPct: 80, wantStyle: func(s qualityStyles) lipgloss.Style { return s.good }},
	}

	s := newQualityStyles()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cc := taxonomy.ContractCoverage{
				Percentage:       tt.covPct,
				CoveredCount:     1,
				TotalContractual: 2,
			}
			writeContractCoverage(&buf, cc, s)
			out := buf.String()
			if !strings.Contains(out, "Contract Coverage:") {
				t.Errorf("expected 'Contract Coverage:' in output, got %q", out)
			}
			// Verify the correct style was applied to the percentage value.
			wantRendered := tt.wantStyle(s).Render(fmt.Sprintf("%.0f%%", tt.covPct))
			if !strings.Contains(out, wantRendered) {
				t.Errorf("expected styled rendering %q in output, got %q", wantRendered, out)
			}
		})
	}
}

// TestWriteOverSpecification_Thresholds verifies the style threshold boundaries
// for over-specification: 0 → good, >0 → warn, >3 → bad.
func TestWriteOverSpecification_Thresholds(t *testing.T) {
	tests := []struct {
		name      string
		overCount int
		wantStyle func(s qualityStyles) lipgloss.Style
	}{
		{name: "0_good", overCount: 0, wantStyle: func(s qualityStyles) lipgloss.Style { return s.good }},
		{name: "1_warn", overCount: 1, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "3_warn_boundary", overCount: 3, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "4_bad", overCount: 4, wantStyle: func(s qualityStyles) lipgloss.Style { return s.bad }},
	}

	s := newQualityStyles()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeOverSpecification(&buf, tt.overCount, s)
			out := buf.String()
			if !strings.Contains(out, "Over-Specified:") {
				t.Errorf("expected 'Over-Specified:' in output, got %q", out)
			}
			// Verify the correct style was applied to the count value.
			wantRendered := tt.wantStyle(s).Render(fmt.Sprintf("%d", tt.overCount))
			if !strings.Contains(out, wantRendered) {
				t.Errorf("expected styled rendering %q in output, got %q", wantRendered, out)
			}
		})
	}
}

// TestWriteDetectionConfidence_Thresholds verifies the style threshold boundaries
// for detection confidence: <50 → bad, >=50 && <70 → warn, >=70 → good.
func TestWriteDetectionConfidence_Thresholds(t *testing.T) {
	tests := []struct {
		name      string
		detConf   int
		wantStyle func(s qualityStyles) lipgloss.Style
	}{
		{name: "49_bad", detConf: 49, wantStyle: func(s qualityStyles) lipgloss.Style { return s.bad }},
		{name: "50_warn", detConf: 50, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "69_warn", detConf: 69, wantStyle: func(s qualityStyles) lipgloss.Style { return s.warn }},
		{name: "70_good", detConf: 70, wantStyle: func(s qualityStyles) lipgloss.Style { return s.good }},
	}

	s := newQualityStyles()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeDetectionConfidence(&buf, tt.detConf, s)
			out := buf.String()
			if !strings.Contains(out, "Detection Confidence:") {
				t.Errorf("expected 'Detection Confidence:' in output, got %q", out)
			}
			// Verify the correct style was applied to the confidence value.
			wantRendered := tt.wantStyle(s).Render(fmt.Sprintf("%d%%", tt.detConf))
			if !strings.Contains(out, wantRendered) {
				t.Errorf("expected styled rendering %q in output, got %q", wantRendered, out)
			}
		})
	}
}

// TestWriteSSADiagnostics_Rendering verifies that SSA diagnostics render
// correctly when degraded, and produce no output otherwise.
func TestWriteSSADiagnostics_Rendering(t *testing.T) {
	s := newQualityStyles()

	t.Run("degraded_with_packages", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{
			SSADegraded:         true,
			SSADegradedPackages: []string{"example.com/pkg/a", "example.com/pkg/b"},
		}
		writeSSADiagnostics(&buf, summary, s)
		out := buf.String()
		if !strings.Contains(out, "SSA construction failed") {
			t.Errorf("expected 'SSA construction failed' in output, got %q", out)
		}
		if !strings.Contains(out, "2 package(s)") {
			t.Errorf("expected '2 package(s)' in output, got %q", out)
		}
		if !strings.Contains(out, "example.com/pkg/a") {
			t.Errorf("expected package name in output, got %q", out)
		}
		if !strings.Contains(out, "partial (AST-only)") {
			t.Errorf("expected 'partial (AST-only)' in output, got %q", out)
		}
	})

	t.Run("nil_summary", func(t *testing.T) {
		var buf bytes.Buffer
		writeSSADiagnostics(&buf, nil, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for nil summary, got %q", buf.String())
		}
	})

	t.Run("not_degraded", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{SSADegraded: false}
		writeSSADiagnostics(&buf, summary, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for non-degraded summary, got %q", buf.String())
		}
	})

	t.Run("degraded_empty_packages", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{
			SSADegraded:         true,
			SSADegradedPackages: []string{},
		}
		writeSSADiagnostics(&buf, summary, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for degraded with empty packages, got %q", buf.String())
		}
	})
}

// TestWritePackageSummary_WorstCoverage verifies that the package summary
// renders worst coverage tests correctly.
func TestWritePackageSummary_WorstCoverage(t *testing.T) {
	s := newQualityStyles()

	t.Run("with_worst_coverage", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{
			TotalTests:                   5,
			AverageContractCoverage:      65,
			TotalOverSpecifications:      2,
			AssertionDetectionConfidence: 80,
			WorstCoverageTests: []taxonomy.QualityReport{
				{
					TestFunction: "TestFoo",
					ContractCoverage: taxonomy.ContractCoverage{
						Percentage:       25,
						CoveredCount:     1,
						TotalContractual: 4,
					},
				},
			},
		}
		writePackageSummary(&buf, summary, s)
		out := buf.String()
		if !strings.Contains(out, "Package Summary") {
			t.Errorf("expected 'Package Summary' in output, got %q", out)
		}
		if !strings.Contains(out, "Tests analyzed: 5") {
			t.Errorf("expected 'Tests analyzed: 5' in output, got %q", out)
		}
		if !strings.Contains(out, "Lowest coverage tests:") {
			t.Errorf("expected 'Lowest coverage tests:' in output, got %q", out)
		}
		if !strings.Contains(out, "TestFoo") {
			t.Errorf("expected 'TestFoo' in output, got %q", out)
		}
		if !strings.Contains(out, "25%") {
			t.Errorf("expected '25%%' in output, got %q", out)
		}
	})

	t.Run("nil_summary", func(t *testing.T) {
		var buf bytes.Buffer
		writePackageSummary(&buf, nil, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for nil summary, got %q", buf.String())
		}
	})

	t.Run("zero_tests", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{TotalTests: 0}
		writePackageSummary(&buf, summary, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for zero tests, got %q", buf.String())
		}
	})
}

// TestWriteGapsSection_WithoutHint verifies that a gap without a corresponding
// hint renders correctly (no hint line emitted).
func TestWriteGapsSection_WithoutHint(t *testing.T) {
	s := newQualityStyles()

	t.Run("gap_without_hint", func(t *testing.T) {
		var buf bytes.Buffer
		cc := taxonomy.ContractCoverage{
			Gaps: []taxonomy.SideEffect{
				{Type: "ReturnValue", Description: "returns int", Location: "foo.go:10"},
			},
			GapHints: []string{""}, // empty hint
		}
		writeGapsSection(&buf, cc, s)
		out := buf.String()
		if !strings.Contains(out, "ReturnValue") {
			t.Errorf("expected 'ReturnValue' in output, got %q", out)
		}
		if strings.Contains(out, "hint:") {
			t.Errorf("expected no hint line for empty hint, got %q", out)
		}
	})

	t.Run("gap_with_short_hints_slice", func(t *testing.T) {
		var buf bytes.Buffer
		cc := taxonomy.ContractCoverage{
			Gaps: []taxonomy.SideEffect{
				{Type: "ReturnValue", Description: "returns int", Location: "foo.go:10"},
				{Type: "ErrorReturn", Description: "returns error", Location: "foo.go:11"},
			},
			GapHints: []string{"check err"}, // shorter than Gaps
		}
		writeGapsSection(&buf, cc, s)
		out := buf.String()
		if !strings.Contains(out, "check err") {
			t.Errorf("expected hint 'check err' for first gap, got %q", out)
		}
		// Only one hint line should appear (for the first gap only).
		if strings.Count(out, "hint:") != 1 {
			t.Errorf("expected exactly 1 hint line, got %d in %q", strings.Count(out, "hint:"), out)
		}
	})

	t.Run("no_gaps", func(t *testing.T) {
		var buf bytes.Buffer
		cc := taxonomy.ContractCoverage{}
		writeGapsSection(&buf, cc, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for no gaps, got %q", buf.String())
		}
	})
}

// TestWriteDiscardedReturns_WithoutHint verifies that a discarded return
// without a corresponding hint renders correctly (no hint line emitted).
func TestWriteDiscardedReturns_WithoutHint(t *testing.T) {
	s := newQualityStyles()

	t.Run("discarded_without_hint", func(t *testing.T) {
		var buf bytes.Buffer
		cc := taxonomy.ContractCoverage{
			DiscardedReturns: []taxonomy.SideEffect{
				{Type: "ReturnValue", Description: "returns int", Location: "foo.go:10"},
			},
			DiscardedReturnHints: []string{""}, // empty hint
		}
		writeDiscardedReturns(&buf, cc, s)
		out := buf.String()
		if !strings.Contains(out, "ReturnValue") {
			t.Errorf("expected 'ReturnValue' in output, got %q", out)
		}
		if strings.Contains(out, "hint:") {
			t.Errorf("expected no hint line for empty hint, got %q", out)
		}
	})

	t.Run("no_discarded", func(t *testing.T) {
		var buf bytes.Buffer
		cc := taxonomy.ContractCoverage{}
		writeDiscardedReturns(&buf, cc, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for no discarded returns, got %q", buf.String())
		}
	})
}

// TestWriteUnmappedAssertions_WithoutReason verifies that an unmapped assertion
// without a reason renders correctly (no reason suffix).
func TestWriteUnmappedAssertions_WithoutReason(t *testing.T) {
	t.Run("without_reason", func(t *testing.T) {
		var buf bytes.Buffer
		assertions := []taxonomy.AssertionMapping{
			{
				AssertionLocation: "foo_test.go:10",
				AssertionType:     "equality",
				UnmappedReason:    "",
			},
		}
		writeUnmappedAssertions(&buf, assertions)
		out := buf.String()
		if !strings.Contains(out, "foo_test.go:10") {
			t.Errorf("expected location in output, got %q", out)
		}
		if strings.Contains(out, "[") {
			t.Errorf("expected no reason bracket for empty reason, got %q", out)
		}
	})

	t.Run("with_reason", func(t *testing.T) {
		var buf bytes.Buffer
		assertions := []taxonomy.AssertionMapping{
			{
				AssertionLocation: "foo_test.go:10",
				AssertionType:     "equality",
				UnmappedReason:    "no matching effect",
			},
		}
		writeUnmappedAssertions(&buf, assertions)
		out := buf.String()
		if !strings.Contains(out, "[no matching effect]") {
			t.Errorf("expected '[no matching effect]' in output, got %q", out)
		}
	})

	t.Run("no_assertions", func(t *testing.T) {
		var buf bytes.Buffer
		writeUnmappedAssertions(&buf, nil)
		if buf.Len() != 0 {
			t.Errorf("expected no output for nil assertions, got %q", buf.String())
		}
	})
}

// TestWriteSkippedTests_Rendering verifies the skipped tests section
// in the text report.
func TestWriteSkippedTests_Rendering(t *testing.T) {
	s := newQualityStyles()

	t.Run("nil_summary", func(t *testing.T) {
		var buf bytes.Buffer
		writeSkippedTests(&buf, nil, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for nil summary, got %q", buf.String())
		}
	})

	t.Run("zero_skipped", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{SkippedTests: 0}
		writeSkippedTests(&buf, summary, s)
		if buf.Len() != 0 {
			t.Errorf("expected no output for zero skipped, got %q", buf.String())
		}
	})

	t.Run("some_skipped", func(t *testing.T) {
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{
			SkippedTests:     3,
			SkippedTestNames: []string{"TestAlpha", "TestBeta", "TestGamma"},
		}
		writeSkippedTests(&buf, summary, s)
		out := buf.String()
		if !strings.Contains(out, "3 test function(s) skipped") {
			t.Errorf("expected '3 test function(s) skipped' in output, got %q", out)
		}
		if !strings.Contains(out, "TestAlpha") {
			t.Errorf("expected 'TestAlpha' in output, got %q", out)
		}
		if !strings.Contains(out, "TestGamma") {
			t.Errorf("expected 'TestGamma' in output, got %q", out)
		}
		if !strings.Contains(out, "--target") {
			t.Errorf("expected '--target' hint in output, got %q", out)
		}
	})

	t.Run("count_exceeds_names_length", func(t *testing.T) {
		// Invariant violation: SkippedTests > len(SkippedTestNames).
		// The defensive guard (i < len(summary.SkippedTestNames)) must
		// prevent out-of-bounds access.
		var buf bytes.Buffer
		summary := &taxonomy.PackageSummary{
			SkippedTests:     5,
			SkippedTestNames: []string{"TestAlpha", "TestBeta"},
		}
		writeSkippedTests(&buf, summary, s)
		out := buf.String()
		// Header should show the count from SkippedTests (5), not len(names).
		if !strings.Contains(out, "5 test function(s) skipped") {
			t.Errorf("expected '5 test function(s) skipped' in header, got %q", out)
		}
		// Available names should appear.
		if !strings.Contains(out, "TestAlpha") {
			t.Errorf("expected 'TestAlpha' in output, got %q", out)
		}
		if !strings.Contains(out, "TestBeta") {
			t.Errorf("expected 'TestBeta' in output, got %q", out)
		}
		// No panic or out-of-bounds — the function should complete normally.
	})

	t.Run("truncation_at_20", func(t *testing.T) {
		var buf bytes.Buffer
		names := make([]string, 25)
		for i := range names {
			names[i] = fmt.Sprintf("TestFunc%02d", i)
		}
		summary := &taxonomy.PackageSummary{
			SkippedTests:     25,
			SkippedTestNames: names,
		}
		writeSkippedTests(&buf, summary, s)
		out := buf.String()
		// First 20 should appear.
		if !strings.Contains(out, "TestFunc00") {
			t.Errorf("expected 'TestFunc00' in output, got %q", out)
		}
		if !strings.Contains(out, "TestFunc19") {
			t.Errorf("expected 'TestFunc19' in output, got %q", out)
		}
		// 21st should NOT appear as a list item.
		if strings.Contains(out, "TestFunc20") {
			t.Errorf("expected 'TestFunc20' to be truncated, got %q", out)
		}
		// Header should show the full count (25), not truncated count.
		if !strings.Contains(out, "25 test function(s) skipped") {
			t.Errorf("expected '25 test function(s) skipped' in header, got %q", out)
		}
		// Truncation message.
		if !strings.Contains(out, "... and 5 more") {
			t.Errorf("expected '... and 5 more' in output, got %q", out)
		}
	})
}

// TestWriteText_MultiReportSeparator verifies that a blank line separator
// appears between multiple reports.
func TestWriteText_MultiReportSeparator(t *testing.T) {
	reports := []taxonomy.QualityReport{
		{
			TestFunction: "TestA",
			TargetFunction: taxonomy.FunctionTarget{
				Function: "A",
				Package:  "pkg",
			},
			ContractCoverage: taxonomy.ContractCoverage{Percentage: 100},
		},
		{
			TestFunction: "TestB",
			TargetFunction: taxonomy.FunctionTarget{
				Function: "B",
				Package:  "pkg",
			},
			ContractCoverage: taxonomy.ContractCoverage{Percentage: 100},
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, reports, nil); err != nil {
		t.Fatalf("WriteText() error = %v", err)
	}
	out := buf.String()

	// The separator between reports is a blank line (two consecutive newlines
	// with only whitespace between them).
	testAIdx := strings.Index(out, "TestA")
	testBIdx := strings.Index(out, "TestB")
	if testAIdx < 0 || testBIdx < 0 {
		t.Fatalf("expected both TestA and TestB in output, got %q", out)
	}
	between := out[testAIdx:testBIdx]
	if !strings.Contains(between, "\n\n") {
		t.Errorf("expected blank line separator between reports, got %q", between)
	}
}
