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

	"github.com/unbound-force/gaze/internal/report"
)

// WriteJSON writes the CRAP report as formatted JSON.
func WriteJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// WriteText writes the CRAP report as human-readable styled text.
func WriteText(w io.Writer, rpt *Report) error {
	styles := report.DefaultStyles()

	if len(rpt.Scores) == 0 {
		_, _ = fmt.Fprintln(w, styles.Muted.Render("No functions analyzed."))
		return nil
	}

	// Sort by CRAP score descending for display.
	sorted := make([]Score, len(rpt.Scores))
	copy(sorted, rpt.Scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CRAP > sorted[j].CRAP
	})

	// Build table rows.
	rows := make([][]string, 0, len(sorted))
	for _, s := range sorted {
		marker := ""
		if s.CRAP >= rpt.Summary.CRAPThreshold {
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

	threshold := rpt.Summary.CRAPThreshold

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(styles.Border).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styles.Header
			}
			// Color CRAP score column based on threshold.
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

	// Summary.
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, styles.Header.Render("--- Summary ---"))
	_, _ = fmt.Fprintf(w, "%s  %d\n", styles.SummaryLabel.Render("Functions analyzed:"), rpt.Summary.TotalFunctions)
	_, _ = fmt.Fprintf(w, "%s  %.1f\n", styles.SummaryLabel.Render("Avg complexity:"), rpt.Summary.AvgComplexity)
	_, _ = fmt.Fprintf(w, "%s  %.1f%%\n", styles.SummaryLabel.Render("Avg line coverage:"), rpt.Summary.AvgLineCoverage)
	_, _ = fmt.Fprintf(w, "%s  %.1f\n", styles.SummaryLabel.Render("Avg CRAP score:"), rpt.Summary.AvgCRAP)
	_, _ = fmt.Fprintf(w, "%s  %.0f\n", styles.SummaryLabel.Render("CRAP threshold:"), rpt.Summary.CRAPThreshold)

	craploadStr := fmt.Sprintf("%d", rpt.Summary.CRAPload)
	if rpt.Summary.CRAPload > 0 {
		craploadStr = styles.CRAPBad.Render(craploadStr) + styles.Muted.Render(" (functions at or above threshold)")
	}
	_, _ = fmt.Fprintf(w, "%s  %s\n", styles.SummaryLabel.Render("CRAPload:"), craploadStr)

	// GazeCRAP and quadrant stats (when available).
	if rpt.Summary.GazeCRAPload != nil && rpt.Summary.GazeCRAPThreshold != nil {
		_, _ = fmt.Fprintf(w, "%s  %.0f\n", styles.SummaryLabel.Render("GazeCRAP threshold:"), *rpt.Summary.GazeCRAPThreshold)
		gazeCRAPloadStr := fmt.Sprintf("%d", *rpt.Summary.GazeCRAPload)
		if *rpt.Summary.GazeCRAPload > 0 {
			gazeCRAPloadStr = styles.CRAPBad.Render(gazeCRAPloadStr) + styles.Muted.Render(" (functions at or above threshold)")
		}
		_, _ = fmt.Fprintf(w, "%s  %s\n", styles.SummaryLabel.Render("GazeCRAPload:"), gazeCRAPloadStr)
	}

	// Quadrant breakdown.
	if len(rpt.Summary.QuadrantCounts) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, styles.Header.Render("--- Quadrant Breakdown ---"))
		for _, q := range []Quadrant{Q1Safe, Q2ComplexButTested, Q3SimpleButUnderspecified, Q4Dangerous} {
			count := rpt.Summary.QuadrantCounts[q]
			_, _ = fmt.Fprintf(w, "  %-30s  %d\n", string(q), count)
		}
	}

	// Worst offenders.
	if len(rpt.Summary.WorstCRAP) > 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, styles.Header.Render(
			fmt.Sprintf("--- Worst Offenders (top %d by CRAP) ---", len(rpt.Summary.WorstCRAP))))
		for i, s := range rpt.Summary.WorstCRAP {
			score := fmt.Sprintf("%.1f", s.CRAP)
			if s.CRAP >= threshold {
				score = styles.CRAPBad.Render(score)
			} else {
				score = styles.CRAPGood.Render(score)
			}
			_, _ = fmt.Fprintf(w, "  %d. %s  %s  %s\n",
				i+1, score, s.Function,
				styles.Muted.Render(fmt.Sprintf("(%s:%d)", shortenPath(s.File), s.Line)))
		}
	}

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
