// Package report provides output formatters for Gaze analysis results.
package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TextOptions configures text report rendering.
type TextOptions struct {
	// Classify causes a classification column to be included in
	// the side effects table (requires Classification to be set
	// on side effects).
	Classify bool

	// Verbose causes the full signal breakdown to be printed
	// beneath each function's table (implies Classify).
	Verbose bool
}

// WriteText writes analysis results as human-readable styled text
// to the writer. Output uses lipgloss for color and formatting when
// the output is a TTY; degrades gracefully for pipes and CI.
func WriteText(w io.Writer, results []taxonomy.AnalysisResult) error {
	return WriteTextOptions(w, results, TextOptions{})
}

// WriteTextOptions writes analysis results with configurable options.
func WriteTextOptions(w io.Writer, results []taxonomy.AnalysisResult, opts TextOptions) error {
	s := DefaultStyles()

	for i, result := range results {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		if err := writeOneResultOpts(w, result, s, opts); err != nil {
			return err
		}
	}

	// Summary line.
	total := 0
	for _, r := range results {
		total += len(r.SideEffects)
	}
	_, _ = fmt.Fprintf(w, "\n%s\n",
		s.Header.Render(fmt.Sprintf(
			"%d function(s) analyzed, %d side effect(s) detected",
			len(results), total)))

	return nil
}

func writeOneResultOpts(w io.Writer, result taxonomy.AnalysisResult, s Styles, opts TextOptions) error {
	return writeOneResult(w, result, s, opts.Classify || opts.Verbose, opts.Verbose)
}

// writeEffectRows builds table row data from side effects. When
// showClassify is true, each row has 4 columns (Tier, Type, Description,
// Classification); otherwise 3 columns (Tier, Type, Description).
// Descriptions longer than maxDesc are truncated with "...".
// Classification cells longer than maxClassify are truncated similarly.
func writeEffectRows(effects []taxonomy.SideEffect, maxDesc int, showClassify bool) [][]string {
	const maxClassify = 16
	rows := make([][]string, 0, len(effects))
	for _, e := range effects {
		desc := e.Description
		if len(desc) > maxDesc {
			desc = desc[:maxDesc-3] + "..."
		}
		row := []string{
			string(e.Tier),
			string(e.Type),
			desc,
		}
		if showClassify {
			classCell := "—"
			if e.Classification != nil {
				label := string(e.Classification.Label)
				conf := e.Classification.Confidence
				classCell = fmt.Sprintf("%s/%d%%", label, conf)
				if len(classCell) > maxClassify {
					classCell = classCell[:maxClassify-3] + "..."
				}
			}
			row = append(row, classCell)
		}
		rows = append(rows, row)
	}
	return rows
}

// writeVerboseSignals prints the signal breakdown for each side effect
// that has a non-nil Classification with at least one Signal. Each
// effect's signals are printed with source, weight, optional reasoning,
// optional source file, and optional excerpt.
func writeVerboseSignals(w io.Writer, effects []taxonomy.SideEffect) {
	for _, e := range effects {
		if e.Classification == nil || len(e.Classification.Signals) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(w, "\n  Signals for %s (%s):\n",
			string(e.Type), e.Location)
		for _, sig := range e.Classification.Signals {
			line := fmt.Sprintf("    %s: %+d", sig.Source, sig.Weight)
			if sig.Reasoning != "" {
				line += " — " + sig.Reasoning
			}
			_, _ = fmt.Fprintln(w, line)
			if sig.SourceFile != "" {
				_, _ = fmt.Fprintf(w, "      source: %s\n", sig.SourceFile)
			}
			if sig.Excerpt != "" {
				_, _ = fmt.Fprintf(w, "      excerpt: %q\n", sig.Excerpt)
			}
		}
	}
}

// buildEffectsTable creates a lipgloss table for side effects. When
// showClassify is true, the table includes a CLASSIFICATION column
// with per-cell styling based on the classification label.
func buildEffectsTable(effects []taxonomy.SideEffect, rows [][]string, s Styles, showClassify bool) *table.Table {
	t := table.New().
		Width(76).
		Border(lipgloss.NormalBorder()).
		BorderStyle(s.Border)

	if showClassify {
		t = t.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return s.TableHeader
			}
			if col == 0 && row >= 0 && row < len(rows) {
				return s.TierStyle(rows[row][0])
			}
			if col == 3 && row >= 0 && row < len(rows) {
				label := ""
				if effects[row].Classification != nil {
					label = string(effects[row].Classification.Label)
				}
				return s.ClassificationStyle(label)
			}
			return s.TableCell
		}).
			Headers("TIER", "TYPE", "DESCRIPTION", "CLASSIFICATION")
	} else {
		t = t.StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return s.TableHeader
			}
			if col == 0 && row >= 0 && row < len(rows) {
				return s.TierStyle(rows[row][0])
			}
			return s.TableCell
		}).
			Headers("TIER", "TYPE", "DESCRIPTION")
	}

	return t.Rows(rows...)
}

// writeTierSummary writes the per-tier count summary line.
func writeTierSummary(w io.Writer, effects []taxonomy.SideEffect, s Styles) {
	tierCounts := make(map[taxonomy.Tier]int)
	for _, e := range effects {
		tierCounts[e.Tier]++
	}

	var parts []string
	for _, tier := range []taxonomy.Tier{
		taxonomy.TierP0, taxonomy.TierP1,
		taxonomy.TierP2, taxonomy.TierP3, taxonomy.TierP4,
	} {
		if c, ok := tierCounts[tier]; ok {
			styled := s.TierStyle(string(tier)).Render(
				fmt.Sprintf("%s: %d", tier, c))
			parts = append(parts, styled)
		}
	}
	_, _ = fmt.Fprintf(w, "    Summary: %s\n", strings.Join(parts, ", "))
}

func writeOneResult(w io.Writer, result taxonomy.AnalysisResult, s Styles, showClassify, verbose bool) error {
	// Header.
	name := result.Target.QualifiedName()
	_, _ = fmt.Fprintln(w, s.Header.Render(fmt.Sprintf("=== %s ===", name)))
	_, _ = fmt.Fprintln(w, s.SubHeader.Render(fmt.Sprintf("    %s", result.Target.Signature)))
	_, _ = fmt.Fprintln(w, s.SubHeader.Render(fmt.Sprintf("    %s", result.Target.Location)))

	if len(result.SideEffects) == 0 {
		_, _ = fmt.Fprintln(w, s.Muted.Render("    No side effects detected."))
		return nil
	}

	_, _ = fmt.Fprintln(w)

	// Build rows and table.
	maxDesc := 42 // Non-classify default.
	if showClassify {
		maxDesc = 26 // Budget for 4-column layout.
	}
	rows := writeEffectRows(result.SideEffects, maxDesc, showClassify)
	t := buildEffectsTable(result.SideEffects, rows, s, showClassify)
	_, _ = fmt.Fprintln(w, t)

	if verbose {
		writeVerboseSignals(w, result.SideEffects)
	}

	writeTierSummary(w, result.SideEffects, s)

	return nil
}
