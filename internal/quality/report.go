package quality

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// qualityOutput is the top-level JSON structure for quality reports.
type qualityOutput struct {
	Reports []taxonomy.QualityReport `json:"quality_reports"`
	Summary *taxonomy.PackageSummary `json:"quality_summary"`
}

// WriteJSON serializes quality reports and summary as formatted JSON.
func WriteJSON(w io.Writer, reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) error {
	output := qualityOutput{
		Reports: reports,
		Summary: summary,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// WriteText writes a human-readable quality report with lipgloss styling.
func WriteText(w io.Writer, reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) error {
	// Styles.
	header := lipgloss.NewStyle().Bold(true)
	good := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))    // green
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))    // yellow
	bad := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))     // red
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // gray

	for i, r := range reports {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}

		// Header line.
		_, _ = fmt.Fprintln(w, header.Render(fmt.Sprintf(
			"=== %s -> %s ===",
			r.TestFunction,
			r.TargetFunction.QualifiedName())))

		_, _ = fmt.Fprintf(w, "    Test: %s\n", r.TestLocation)
		_, _ = fmt.Fprintf(w, "    Target: %s\n", r.TargetFunction.Location)

		// Contract Coverage.
		covPct := r.ContractCoverage.Percentage
		covStyle := good
		if covPct < 50 {
			covStyle = bad
		} else if covPct < 80 {
			covStyle = warn
		}
		_, _ = fmt.Fprintf(w, "    Contract Coverage: %s (%d/%d)\n",
			covStyle.Render(fmt.Sprintf("%.0f%%", covPct)),
			r.ContractCoverage.CoveredCount,
			r.ContractCoverage.TotalContractual)

		// Over-Specification.
		overCount := r.OverSpecification.Count
		overStyle := good
		if overCount > 0 {
			overStyle = warn
		}
		if overCount > 3 {
			overStyle = bad
		}
		_, _ = fmt.Fprintf(w, "    Over-Specified: %s\n",
			overStyle.Render(fmt.Sprintf("%d", overCount)))

		// Detection Confidence.
		detConf := r.AssertionDetectionConfidence
		detStyle := good
		if detConf < 70 {
			detStyle = warn
		}
		if detConf < 50 {
			detStyle = bad
		}
		_, _ = fmt.Fprintf(w, "    Detection Confidence: %s\n",
			detStyle.Render(fmt.Sprintf("%d%%", detConf)))

		// Gaps.
		if len(r.ContractCoverage.Gaps) > 0 {
			_, _ = fmt.Fprintln(w, muted.Render("    Gaps (untested contractual effects):"))
			for _, gap := range r.ContractCoverage.Gaps {
				_, _ = fmt.Fprintf(w, "      - %s: %s (%s)\n",
					gap.Type, gap.Description, gap.Location)
			}
		}

		// Suggestions.
		if len(r.OverSpecification.Suggestions) > 0 {
			_, _ = fmt.Fprintln(w, muted.Render("    Suggestions:"))
			for _, s := range r.OverSpecification.Suggestions {
				_, _ = fmt.Fprintf(w, "      - %s\n", s)
			}
		}

		// Ambiguous effects.
		if len(r.AmbiguousEffects) > 0 {
			_, _ = fmt.Fprintf(w, "    Ambiguous effects (excluded): %d\n",
				len(r.AmbiguousEffects))
		}

		// Unmapped assertions.
		if len(r.UnmappedAssertions) > 0 {
			_, _ = fmt.Fprintf(w, "    Unmapped assertions: %d\n",
				len(r.UnmappedAssertions))
		}
	}

	// Package summary.
	if summary != nil && summary.TotalTests > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, header.Render("=== Package Summary ==="))
		_, _ = fmt.Fprintf(w, "    Tests analyzed: %d\n", summary.TotalTests)
		_, _ = fmt.Fprintf(w, "    Average contract coverage: %.0f%%\n",
			summary.AverageContractCoverage)
		_, _ = fmt.Fprintf(w, "    Total over-specifications: %d\n",
			summary.TotalOverSpecifications)
		_, _ = fmt.Fprintf(w, "    Assertion detection confidence: %d%%\n",
			summary.AssertionDetectionConfidence)

		if len(summary.WorstCoverageTests) > 0 {
			_, _ = fmt.Fprintln(w, muted.Render("    Lowest coverage tests:"))
			for _, worst := range summary.WorstCoverageTests {
				_, _ = fmt.Fprintf(w, "      - %s: %.0f%% (%d/%d)\n",
					worst.TestFunction,
					worst.ContractCoverage.Percentage,
					worst.ContractCoverage.CoveredCount,
					worst.ContractCoverage.TotalContractual)
			}
		}
	}

	return nil
}
