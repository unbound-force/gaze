package crap

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Report styles (package-level for consistent terminal output).
var (
	crapHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	crapBorderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	crapBadStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	crapGoodStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("40"))
	crapLabelStyle  = lipgloss.NewStyle().Bold(true)
	crapMutedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// WriteJSON writes the CRAP report as formatted JSON.
func WriteJSON(w io.Writer, report *Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// WriteText writes the CRAP report as human-readable styled text.
func WriteText(w io.Writer, report *Report) error {
	if len(report.Scores) == 0 {
		fmt.Fprintln(w, crapMutedStyle.Render("No functions analyzed."))
		return nil
	}

	// Sort by CRAP score descending for display.
	sorted := make([]Score, len(report.Scores))
	copy(sorted, report.Scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CRAP > sorted[j].CRAP
	})

	// Build table rows.
	rows := make([][]string, 0, len(sorted))
	for _, s := range sorted {
		marker := ""
		if s.CRAP >= report.Summary.CRAPThreshold {
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

	threshold := report.Summary.CRAPThreshold

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(crapBorderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return crapHeaderStyle
			}
			// Color CRAP score column based on threshold.
			if col == 0 && row >= 0 && row < len(sorted) {
				if sorted[row].CRAP >= threshold {
					return crapBadStyle
				}
				return crapGoodStyle
			}
			return lipgloss.NewStyle()
		}).
		Headers("CRAP", "COMPLEXITY", "COVERAGE", "FUNCTION", "FILE").
		Rows(rows...)

	fmt.Fprintln(w, t)

	// Summary.
	fmt.Fprintln(w)
	fmt.Fprintln(w, crapHeaderStyle.Render("--- Summary ---"))
	fmt.Fprintf(w, "%s  %d\n", crapLabelStyle.Render("Functions analyzed:"), report.Summary.TotalFunctions)
	fmt.Fprintf(w, "%s  %.1f\n", crapLabelStyle.Render("Avg complexity:"), report.Summary.AvgComplexity)
	fmt.Fprintf(w, "%s  %.1f%%\n", crapLabelStyle.Render("Avg line coverage:"), report.Summary.AvgLineCoverage)
	fmt.Fprintf(w, "%s  %.1f\n", crapLabelStyle.Render("Avg CRAP score:"), report.Summary.AvgCRAP)
	fmt.Fprintf(w, "%s  %.0f\n", crapLabelStyle.Render("CRAP threshold:"), report.Summary.CRAPThreshold)

	craploadStr := fmt.Sprintf("%d", report.Summary.CRAPload)
	if report.Summary.CRAPload > 0 {
		craploadStr = crapBadStyle.Render(craploadStr) + crapMutedStyle.Render(" (functions at or above threshold)")
	}
	fmt.Fprintf(w, "%s  %s\n", crapLabelStyle.Render("CRAPload:"), craploadStr)

	// GazeCRAP and quadrant stats (when available).
	if report.Summary.GazeCRAPload != nil {
		fmt.Fprintf(w, "%s  %.0f\n", crapLabelStyle.Render("GazeCRAP threshold:"), *report.Summary.GazeCRAPThreshold)
		gazeCRAPloadStr := fmt.Sprintf("%d", *report.Summary.GazeCRAPload)
		if *report.Summary.GazeCRAPload > 0 {
			gazeCRAPloadStr = crapBadStyle.Render(gazeCRAPloadStr) + crapMutedStyle.Render(" (functions at or above threshold)")
		}
		fmt.Fprintf(w, "%s  %s\n", crapLabelStyle.Render("GazeCRAPload:"), gazeCRAPloadStr)
	}

	// Quadrant breakdown.
	if len(report.Summary.QuadrantCounts) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, crapHeaderStyle.Render("--- Quadrant Breakdown ---"))
		for _, q := range []Quadrant{Q1Safe, Q2ComplexButTested, Q3SimpleButUnderspecified, Q4Dangerous} {
			count := report.Summary.QuadrantCounts[q]
			fmt.Fprintf(w, "  %-30s  %d\n", string(q), count)
		}
	}

	// Worst offenders.
	if len(report.Summary.WorstCRAP) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, crapHeaderStyle.Render(
			fmt.Sprintf("--- Worst Offenders (top %d by CRAP) ---", len(report.Summary.WorstCRAP))))
		for i, s := range report.Summary.WorstCRAP {
			score := fmt.Sprintf("%.1f", s.CRAP)
			if s.CRAP >= threshold {
				score = crapBadStyle.Render(score)
			} else {
				score = crapGoodStyle.Render(score)
			}
			fmt.Fprintf(w, "  %d. %s  %s  %s\n",
				i+1, score, s.Function,
				crapMutedStyle.Render(fmt.Sprintf("(%s:%d)", shortenPath(s.File), s.Line)))
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
