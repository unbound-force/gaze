package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/jflowers/gaze/internal/taxonomy"
)

// WriteText writes analysis results as human-readable styled text
// to the writer. Output uses lipgloss for color and formatting when
// the output is a TTY; degrades gracefully for pipes and CI.
func WriteText(w io.Writer, results []taxonomy.AnalysisResult) error {
	s := DefaultStyles()

	for i, result := range results {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if err := writeOneResult(w, result, s); err != nil {
			return err
		}
	}

	// Summary line.
	total := 0
	for _, r := range results {
		total += len(r.SideEffects)
	}
	fmt.Fprintf(w, "\n%s\n",
		s.Header.Render(fmt.Sprintf(
			"%d function(s) analyzed, %d side effect(s) detected",
			len(results), total)))

	return nil
}

func writeOneResult(w io.Writer, result taxonomy.AnalysisResult, s Styles) error {
	// Header.
	name := result.Target.QualifiedName()
	fmt.Fprintln(w, s.Header.Render(fmt.Sprintf("=== %s ===", name)))
	fmt.Fprintln(w, s.SubHeader.Render(fmt.Sprintf("    %s", result.Target.Signature)))
	fmt.Fprintln(w, s.SubHeader.Render(fmt.Sprintf("    %s", result.Target.Location)))

	if len(result.SideEffects) == 0 {
		fmt.Fprintln(w, s.Muted.Render("    No side effects detected."))
		return nil
	}

	fmt.Fprintln(w)

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

	fmt.Fprintln(w, t)

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
	fmt.Fprintf(w, "    Summary: %s\n", strings.Join(parts, ", "))

	return nil
}
