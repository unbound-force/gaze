package report

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles defines the visual theme for terminal report output.
// Lipgloss automatically degrades to no-color when output is not a TTY.
type Styles struct {
	// Header is used for section headers (e.g. "=== FuncName ===").
	Header lipgloss.Style

	// SubHeader is used for secondary information lines.
	SubHeader lipgloss.Style

	// TierP0 through TierP4 color-code side effect tiers.
	TierP0 lipgloss.Style
	TierP1 lipgloss.Style
	TierP2 lipgloss.Style
	TierP3 lipgloss.Style
	TierP4 lipgloss.Style

	// TableHeader styles the header row of tables.
	TableHeader lipgloss.Style

	// TableCell styles regular table cells.
	TableCell lipgloss.Style

	// CRAPBad styles CRAP scores at or above threshold.
	CRAPBad lipgloss.Style

	// CRAPGood styles CRAP scores below threshold.
	CRAPGood lipgloss.Style

	// SummaryLabel styles summary line labels.
	SummaryLabel lipgloss.Style

	// SummaryValue styles summary line values.
	SummaryValue lipgloss.Style

	// Pass styles PASS indicators.
	Pass lipgloss.Style

	// Fail styles FAIL indicators.
	Fail lipgloss.Style

	// Border is used for table borders.
	Border lipgloss.Style

	// Muted is used for de-emphasized text.
	Muted lipgloss.Style
}

// DefaultStyles returns the default color scheme for terminal reports.
func DefaultStyles() Styles {
	return Styles{
		Header:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		SubHeader: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),

		TierP0: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		TierP1: lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		TierP2: lipgloss.NewStyle().Foreground(lipgloss.Color("220")),
		TierP3: lipgloss.NewStyle().Foreground(lipgloss.Color("75")),
		TierP4: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),

		TableHeader: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		TableCell:   lipgloss.NewStyle().PaddingRight(1),

		CRAPBad:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		CRAPGood: lipgloss.NewStyle().Foreground(lipgloss.Color("40")),

		SummaryLabel: lipgloss.NewStyle().Bold(true).Width(20),
		SummaryValue: lipgloss.NewStyle(),

		Pass: lipgloss.NewStyle().Foreground(lipgloss.Color("40")).Bold(true),
		Fail: lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),

		Border: lipgloss.NewStyle().Foreground(lipgloss.Color("63")),

		Muted: lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
	}
}

// TierStyle returns the appropriate style for a given tier string.
func (s Styles) TierStyle(tier string) lipgloss.Style {
	switch tier {
	case "P0":
		return s.TierP0
	case "P1":
		return s.TierP1
	case "P2":
		return s.TierP2
	case "P3":
		return s.TierP3
	case "P4":
		return s.TierP4
	default:
		return s.Muted
	}
}
