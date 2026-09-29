// Package crap computes CRAP scores for Go functions.
package crap

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/unbound-force/gaze/v2/internal/report"
)

// WriteJSON writes the CRAP report as formatted JSON to w.
// Returns nil on success, or an error if JSON encoding fails.
func WriteJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// writeScoreTable builds and writes the CRAP score table with
// threshold markers and color styling.
func writeScoreTable(w io.Writer, sorted []Score, threshold float64, styles report.Styles) {
	rows := make([][]string, 0, len(sorted))
	for _, s := range sorted {
		marker := ""
		if s.CRAP >= threshold {
			marker = " *"
		}
		file := shortenPath(s.File)
		rows = append(rows, []string{
			fmt.Sprintf("%.1f%s", s.CRAP, marker),
			fmt.Sprintf("%d", s.Complexity),
			fmt.Sprintf("%.1f%%", s.LineCoverage),
			s.Function,
			fmt.Sprintf("%s:%d", file, s.Line),
		})
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styles.Border).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styles.Header
			}
			if col == 0 && row >= 0 && row < len(sorted) {
				if sorted[row].CRAP >= threshold {
					return styles.CRAPBad
				}
				return styles.CRAPGood
			}
			return lipgloss.NewStyle()
		}).
		Headers("CRAP", "COMPLEXITY", "COVERAGE", "FUNCTION", "FILE").
		Rows(rows...)

	_, _ = fmt.Fprintln(w, t)
}

// writeSummarySection writes the summary statistics section including
// CRAPload, GazeCRAP threshold, and GazeCRAPload (when available).
func writeSummarySection(w io.Writer, summary Summary, styles report.Styles) {
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, styles.Header.Render("--- Summary ---"))
	_, _ = fmt.Fprintf(w, "%s  %d\n", styles.SummaryLabel.Render("Functions analyzed:"), summary.TotalFunctions)
	_, _ = fmt.Fprintf(w, "%s  %.1f\n", styles.SummaryLabel.Render("Avg complexity:"), summary.AvgComplexity)
	_, _ = fmt.Fprintf(w, "%s  %.1f%%\n", styles.SummaryLabel.Render("Avg line coverage:"), summary.AvgLineCoverage)
	_, _ = fmt.Fprintf(w, "%s  %.1f\n", styles.SummaryLabel.Render("Avg CRAP score:"), summary.AvgCRAP)
	_, _ = fmt.Fprintf(w, "%s  %.0f\n", styles.SummaryLabel.Render("CRAP threshold:"), summary.CRAPThreshold)

	craploadStr := fmt.Sprintf("%d", summary.CRAPload)
	if summary.CRAPload > 0 {
		craploadStr = styles.CRAPBad.Render(craploadStr) + styles.Muted.Render(" (functions at or above threshold)")
	}
	_, _ = fmt.Fprintf(w, "%s  %s\n", styles.SummaryLabel.Render("CRAPload:"), craploadStr)

	if summary.GazeCRAPload != nil && summary.GazeCRAPThreshold != nil {
		_, _ = fmt.Fprintf(w, "%s  %.0f\n", styles.SummaryLabel.Render("GazeCRAP threshold:"), *summary.GazeCRAPThreshold)
		gazeCRAPloadStr := fmt.Sprintf("%d", *summary.GazeCRAPload)
		if *summary.GazeCRAPload > 0 {
			gazeCRAPloadStr = styles.CRAPBad.Render(gazeCRAPloadStr) + styles.Muted.Render(" (functions at or above threshold)")
		}
		_, _ = fmt.Fprintf(w, "%s  %s\n", styles.SummaryLabel.Render("GazeCRAPload:"), gazeCRAPloadStr)
	}
}

// writeQuadrantSection writes the quadrant breakdown section.
func writeQuadrantSection(w io.Writer, counts map[Quadrant]int, styles report.Styles) {
	if len(counts) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, styles.Header.Render("--- Quadrant Breakdown ---"))
	for _, q := range []Quadrant{Q1Safe, Q2ComplexButTested, Q3SimpleButUnderspecified, Q4Dangerous} {
		count := counts[q]
		_, _ = fmt.Fprintf(w, "  %-30s  %d\n", string(q), count)
	}
}

// writeSSADiagnostics writes an SSA degradation warning when
// packages had SSA construction failures.
func writeSSADiagnostics(w io.Writer, degradedPkgs []string, styles report.Styles) {
	if len(degradedPkgs) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "  ⚠ SSA construction failed for %d package(s):\n", len(degradedPkgs))
	for _, pkg := range degradedPkgs {
		_, _ = fmt.Fprintf(w, "    - %s\n", pkg)
	}
	_, _ = fmt.Fprintln(w, styles.Muted.Render("  Contract coverage and GazeCRAP metrics are partial."))
}

// writeRemediationSection writes the remediation breakdown section
// showing how many functions need each type of fix.
func writeRemediationSection(w io.Writer, counts map[FixStrategy]int, styles report.Styles) {
	if len(counts) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, styles.Header.Render("--- Remediation Breakdown ---"))
	for _, fs := range []FixStrategy{FixDecompose, FixAddTests, FixAddAssertions, FixDecomposeAndTest} {
		count := counts[fs]
		if count > 0 {
			_, _ = fmt.Fprintf(w, "  %-20s  %d\n", string(fs), count)
		}
	}
}

// writeWorstSection writes the worst offenders section with
// threshold-based coloring and fix strategy labels.
func writeWorstSection(w io.Writer, worst []Score, threshold float64, styles report.Styles) {
	if len(worst) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, styles.Header.Render(
		fmt.Sprintf("--- Worst Offenders (top %d by CRAP) ---", len(worst))))
	for i, s := range worst {
		score := fmt.Sprintf("%.1f", s.CRAP)
		if s.CRAP >= threshold {
			score = styles.CRAPBad.Render(score)
		} else {
			score = styles.CRAPGood.Render(score)
		}
		strategyLabel := ""
		if s.FixStrategy != nil {
			strategyLabel = fmt.Sprintf(" [%s]", *s.FixStrategy)
		}
		reasonLabel := ""
		if s.ContractCoverageReason != nil {
			if *s.ContractCoverageReason == "all_effects_ambiguous" && s.EffectConfidenceRange != nil {
				reasonLabel = fmt.Sprintf(" (all effects ambiguous, confidence %d-%d)",
					s.EffectConfidenceRange[0], s.EffectConfidenceRange[1])
			} else {
				reasonLabel = fmt.Sprintf(" (%s)", *s.ContractCoverageReason)
			}
		}
		_, _ = fmt.Fprintf(w, "  %d. %s  %s%s%s  %s\n",
			i+1, score, s.Function, strategyLabel, reasonLabel,
			styles.Muted.Render(fmt.Sprintf("(%s:%d)", shortenPath(s.File), s.Line)))
	}
}

// WriteText writes the CRAP report as human-readable styled text to w.
// Returns nil on success, or an error if writing to w fails.
func WriteText(w io.Writer, rpt *Report) error {
	styles := report.DefaultStyles()

	if len(rpt.Scores) == 0 {
		_, _ = fmt.Fprintln(w, styles.Muted.Render("No functions analyzed."))
		return nil
	}

	sorted := make([]Score, len(rpt.Scores))
	copy(sorted, rpt.Scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CRAP > sorted[j].CRAP
	})

	threshold := rpt.Summary.CRAPThreshold
	writeScoreTable(w, sorted, threshold, styles)
	writeSummarySection(w, rpt.Summary, styles)
	writeSSADiagnostics(w, rpt.Summary.SSADegradedPackages, styles)
	writeQuadrantSection(w, rpt.Summary.QuadrantCounts, styles)
	writeRemediationSection(w, rpt.Summary.FixStrategyCounts, styles)
	writeWorstSection(w, rpt.Summary.WorstCRAP, threshold, styles)

	return nil
}

// shortenPath removes common Go module path prefixes and returns
// a shorter relative-looking path.
func shortenPath(path string) string {
	// Find the last occurrence of a known directory marker.
	markers := []string{"/internal/", "/cmd/", "/pkg/"}
	for _, m := range markers {
		if idx := strings.LastIndex(path, m); idx >= 0 {
			return path[idx+1:]
		}
	}

	// Fall back to just the filename.
	parts := strings.Split(path, "/")
	if len(parts) > 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return path
}
