package quality

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
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

// qualityStyles bundles the lipgloss styles used by the quality text report.
type qualityStyles struct {
	header lipgloss.Style
	good   lipgloss.Style
	warn   lipgloss.Style
	bad    lipgloss.Style
	muted  lipgloss.Style
}

// newQualityStyles returns the standard quality report style palette.
func newQualityStyles() qualityStyles {
	return qualityStyles{
		header: lipgloss.NewStyle().Bold(true),
		good:   lipgloss.NewStyle().Foreground(lipgloss.Color("2")),   // green
		warn:   lipgloss.NewStyle().Foreground(lipgloss.Color("3")),   // yellow
		bad:    lipgloss.NewStyle().Foreground(lipgloss.Color("1")),   // red
		muted:  lipgloss.NewStyle().Foreground(lipgloss.Color("240")), // gray
	}
}

// WriteText writes a human-readable quality report with lipgloss styling.
func WriteText(w io.Writer, reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) error {
	s := newQualityStyles()

	for i, r := range reports {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		writeReportHeader(w, r, s)
		writeContractCoverage(w, r.ContractCoverage, s)
		writeOverSpecification(w, r.OverSpecification.Count, s)
		writeDetectionConfidence(w, r.AssertionDetectionConfidence, s)
		writeGapsSection(w, r.ContractCoverage, s)
		writeDiscardedReturns(w, r.ContractCoverage, s)
		writeSuggestionsSection(w, r.OverSpecification.Suggestions, s)
		writeAmbiguousEffects(w, r.AmbiguousEffects)
		writeUnmappedAssertions(w, r.UnmappedAssertions)
	}

	writeSSADiagnostics(w, summary, s)
	writeSkippedTests(w, summary, s)
	writePackageSummary(w, summary, s)

	return nil
}

// writeReportHeader writes the test-to-target header and location lines.
func writeReportHeader(w io.Writer, r taxonomy.QualityReport, s qualityStyles) {
	_, _ = fmt.Fprintln(w, s.header.Render(fmt.Sprintf(
		"=== %s -> %s ===",
		r.TestFunction,
		r.TargetFunction.QualifiedName())))
	_, _ = fmt.Fprintf(w, "    Test: %s\n", r.TestLocation)
	_, _ = fmt.Fprintf(w, "    Target: %s\n", r.TargetFunction.Location)
}

// writeContractCoverage writes the contract coverage metric with threshold-based styling.
func writeContractCoverage(w io.Writer, cc taxonomy.ContractCoverage, s qualityStyles) {
	covPct := cc.Percentage
	covStyle := s.good
	if covPct < 50 {
		covStyle = s.bad
	} else if covPct < 80 {
		covStyle = s.warn
	}
	_, _ = fmt.Fprintf(w, "    Contract Coverage: %s (%d/%d)\n",
		covStyle.Render(fmt.Sprintf("%.0f%%", covPct)),
		cc.CoveredCount,
		cc.TotalContractual)
}

// writeOverSpecification writes the over-specification count with threshold-based styling.
func writeOverSpecification(w io.Writer, overCount int, s qualityStyles) {
	overStyle := s.good
	if overCount > 0 {
		overStyle = s.warn
	}
	if overCount > 3 {
		overStyle = s.bad
	}
	_, _ = fmt.Fprintf(w, "    Over-Specified: %s\n",
		overStyle.Render(fmt.Sprintf("%d", overCount)))
}

// writeDetectionConfidence writes the detection confidence metric with threshold-based styling.
func writeDetectionConfidence(w io.Writer, detConf int, s qualityStyles) {
	detStyle := s.good
	if detConf < 70 {
		detStyle = s.warn
	}
	if detConf < 50 {
		detStyle = s.bad
	}
	_, _ = fmt.Fprintf(w, "    Detection Confidence: %s\n",
		detStyle.Render(fmt.Sprintf("%d%%", detConf)))
}

// writeGapsSection writes the untested contractual effects gaps list with optional hints.
func writeGapsSection(w io.Writer, cc taxonomy.ContractCoverage, s qualityStyles) {
	if len(cc.Gaps) > 0 {
		_, _ = fmt.Fprintln(w, s.muted.Render("    Gaps (untested contractual effects):"))
		for i, gap := range cc.Gaps {
			_, _ = fmt.Fprintf(w, "      - %s: %s (%s)\n",
				gap.Type, gap.Description, gap.Location)
			if i < len(cc.GapHints) && cc.GapHints[i] != "" {
				_, _ = fmt.Fprintf(w, "        hint: %s\n", cc.GapHints[i])
			}
		}
	}
}

// writeDiscardedReturns writes the definitively unasserted discarded returns list with optional hints.
func writeDiscardedReturns(w io.Writer, cc taxonomy.ContractCoverage, s qualityStyles) {
	if len(cc.DiscardedReturns) > 0 {
		_, _ = fmt.Fprintln(w, s.muted.Render("    Discarded returns (definitively unasserted):"))
		for i, dr := range cc.DiscardedReturns {
			_, _ = fmt.Fprintf(w, "      - %s: %s (%s)\n",
				dr.Type, dr.Description, dr.Location)
			if i < len(cc.DiscardedReturnHints) && cc.DiscardedReturnHints[i] != "" {
				_, _ = fmt.Fprintf(w, "        hint: %s\n", cc.DiscardedReturnHints[i])
			}
		}
	}
}

// writeSuggestionsSection writes the over-specification suggestions list.
func writeSuggestionsSection(w io.Writer, suggestions []string, s qualityStyles) {
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(w, s.muted.Render("    Suggestions:"))
		for _, sg := range suggestions {
			_, _ = fmt.Fprintf(w, "      - %s\n", sg)
		}
	}
}

// writeAmbiguousEffects writes the ambiguous effects list.
func writeAmbiguousEffects(w io.Writer, effects []taxonomy.SideEffect) {
	if len(effects) > 0 {
		_, _ = fmt.Fprintf(w, "    Ambiguous effects (excluded from metrics): %d\n",
			len(effects))
		for _, ae := range effects {
			_, _ = fmt.Fprintf(w, "      - %s: %s (%s)\n",
				ae.Type, ae.Description, ae.Location)
		}
	}
}

// writeUnmappedAssertions writes the unmapped assertions list with optional reasons.
func writeUnmappedAssertions(w io.Writer, assertions []taxonomy.AssertionMapping) {
	if len(assertions) > 0 {
		_, _ = fmt.Fprintf(w, "    Unmapped assertions: %d\n",
			len(assertions))
		for _, ua := range assertions {
			if ua.UnmappedReason != "" {
				_, _ = fmt.Fprintf(w, "      - %s  %s  [%s]\n",
					ua.AssertionLocation, ua.AssertionType, ua.UnmappedReason)
			} else {
				_, _ = fmt.Fprintf(w, "      - %s  %s\n",
					ua.AssertionLocation, ua.AssertionType)
			}
		}
	}
}

// writeSSADiagnostics writes SSA construction failure warnings if applicable.
func writeSSADiagnostics(w io.Writer, summary *taxonomy.PackageSummary, s qualityStyles) {
	if summary != nil && summary.SSADegraded && len(summary.SSADegradedPackages) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "    %s SSA construction failed for %d package(s):\n",
			s.warn.Render("⚠"),
			len(summary.SSADegradedPackages))
		for _, pkg := range summary.SSADegradedPackages {
			_, _ = fmt.Fprintf(w, "      - %s\n", pkg)
		}
		_, _ = fmt.Fprintln(w, s.muted.Render("    Quality metrics for these packages are partial (AST-only)."))
	}
}

// MaxSkippedTestDisplay is the maximum number of skipped test names
// to display in text output before truncating. Shared between the
// quality text report and the CLI empty-results summary.
const MaxSkippedTestDisplay = 20

// writeSkippedTests writes a section listing test functions that were
// skipped because no target function could be resolved (e.g., BDD/Ginkgo
// suites using dynamic dispatch).
func writeSkippedTests(w io.Writer, summary *taxonomy.PackageSummary, s qualityStyles) {
	if summary == nil || summary.SkippedTests == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "    %s %d test function(s) skipped (no target resolved):\n",
		s.warn.Render("⚠"),
		summary.SkippedTests)
	limit := summary.SkippedTests
	if limit > MaxSkippedTestDisplay {
		limit = MaxSkippedTestDisplay
	}
	for i := 0; i < limit && i < len(summary.SkippedTestNames); i++ {
		_, _ = fmt.Fprintf(w, "      - %s\n", summary.SkippedTestNames[i])
	}
	if summary.SkippedTests > MaxSkippedTestDisplay {
		_, _ = fmt.Fprintf(w, "      ... and %d more\n", summary.SkippedTests-MaxSkippedTestDisplay)
	}
	_, _ = fmt.Fprintln(w, s.muted.Render("    Use --target=FuncName to specify target functions manually."))
}

// writePackageSummary writes the package-level summary footer.
func writePackageSummary(w io.Writer, summary *taxonomy.PackageSummary, s qualityStyles) {
	if summary != nil && summary.TotalTests > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, s.header.Render("=== Package Summary ==="))
		_, _ = fmt.Fprintf(w, "    Tests analyzed: %d\n", summary.TotalTests)
		_, _ = fmt.Fprintf(w, "    Average contract coverage: %.0f%%\n",
			summary.AverageContractCoverage)
		_, _ = fmt.Fprintf(w, "    Total over-specifications: %d\n",
			summary.TotalOverSpecifications)
		_, _ = fmt.Fprintf(w, "    Assertion detection confidence: %d%%\n",
			summary.AssertionDetectionConfidence)

		if len(summary.WorstCoverageTests) > 0 {
			_, _ = fmt.Fprintln(w, s.muted.Render("    Lowest coverage tests:"))
			for _, worst := range summary.WorstCoverageTests {
				_, _ = fmt.Fprintf(w, "      - %s: %.0f%% (%d/%d)\n",
					worst.TestFunction,
					worst.ContractCoverage.Percentage,
					worst.ContractCoverage.CoveredCount,
					worst.ContractCoverage.TotalContractual)
			}
		}
	}
}
