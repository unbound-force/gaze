// Package report provides output formatters for Gaze analysis results.
package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/unbound-force/gaze/internal/taxonomy"
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

	if showClassify {
		// With classification column: budget 80 cols.
		// Borders ~5, padding ~8. TIER=4, TYPE=20, DESC=26, CLASS=16.
		const maxDesc = 26
		const maxClass = 16
		rows := make([][]string, 0, len(result.SideEffects))
		for _, e := range result.SideEffects {
			desc := e.Description
			if len(desc) > maxDesc {
				desc = desc[:maxDesc-3] + "..."
			}
			classCell := "—"
			if e.Classification != nil {
				label := string(e.Classification.Label)
				conf := e.Classification.Confidence
				classCell = fmt.Sprintf("%s/%d%%", label, conf)
				if len(classCell) > maxClass {
					classCell = classCell[:maxClass-3] + "..."
				}
			}
			rows = append(rows, []string{
				string(e.Tier),
				string(e.Type),
				desc,
				classCell,
			})
		}

		t := table.New().
			Width(76).
			Border(lipgloss.NormalBorder()).
			BorderStyle(s.Border).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return s.TableHeader
				}
				if col == 0 && row >= 0 && row < len(rows) {
					return s.TierStyle(rows[row][0])
				}
				if col == 3 && row >= 0 && row < len(rows) {
					label := ""
					if result.SideEffects[row].Classification != nil {
						label = string(result.SideEffects[row].Classification.Label)
					}
					return s.ClassificationStyle(label)
				}
				return s.TableCell
			}).
			Headers("TIER", "TYPE", "DESCRIPTION", "CLASSIFICATION").
			Rows(rows...)

		_, _ = fmt.Fprintln(w, t)

		// Verbose: print signal breakdown for each side effect.
		if verbose {
			for _, e := range result.SideEffects {
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
	} else {
		// Side effects table using lipgloss/table.
		// Budget: 80 cols total. Borders take ~4 (│ on each side + 2 inner │).
		// Padding: 1 space each side per column = 6 cols for 3 columns.
		// Available: 80 - 4 - 6 = 70. TIER=4, TYPE=24, DESC=42.
		const maxDesc = 42
		rows := make([][]string, 0, len(result.SideEffects))
		for _, e := range result.SideEffects {
			desc := e.Description
			if len(desc) > maxDesc {
				desc = desc[:maxDesc-3] + "..."
			}
			rows = append(rows, []string{
				string(e.Tier),
				string(e.Type),
				desc,
			})
		}

		t := table.New().
			Width(76). // Leave 4 chars for left indent.
			Border(lipgloss.NormalBorder()).
			BorderStyle(s.Border).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return s.TableHeader
				}
				// Color the tier column based on tier value.
				if col == 0 && row >= 0 && row < len(rows) {
					return s.TierStyle(rows[row][0])
				}
				return s.TableCell
			}).
			Headers("TIER", "TYPE", "DESCRIPTION").
			Rows(rows...)

		_, _ = fmt.Fprintln(w, t)
	}

	// Tier summary.
	tierCounts := make(map[taxonomy.Tier]int)
	for _, e := range result.SideEffects {
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

	return nil
}
