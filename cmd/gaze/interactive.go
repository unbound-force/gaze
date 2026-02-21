package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/jflowers/gaze/internal/taxonomy"
)

// keyMap defines keybindings for the interactive TUI.
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Quit     key.Binding
	Help     key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Quit, k.Help}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Quit, k.Help},
	}
}

var defaultKeyMap = keyMap{
	Up:       key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("^/k", "up")),
	Down:     key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("v/j", "down")),
	PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
	PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
	Quit:     key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}

// Styles for the TUI.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			MarginBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	tuiHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))

	tuiBorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63"))

	tierP0Style = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	tierP1Style = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	tierP2Style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
)

// analyzeModel is the Bubble Tea model for browsing analysis results.
type analyzeModel struct {
	results  []taxonomy.AnalysisResult
	viewport viewport.Model
	help     help.Model
	keys     keyMap
	ready    bool
	content  string
}

func newAnalyzeModel(results []taxonomy.AnalysisResult) analyzeModel {
	h := help.New()
	content := renderAnalyzeContent(results)
	return analyzeModel{
		results: results,
		help:    h,
		keys:    defaultKeyMap,
		content: content,
	}
}

func renderAnalyzeContent(results []taxonomy.AnalysisResult) string {
	var sb strings.Builder

	totalEffects := 0
	for _, r := range results {
		totalEffects += len(r.SideEffects)
	}

	sb.WriteString(titleStyle.Render(
		fmt.Sprintf("Gaze Analysis: %d function(s), %d side effect(s)",
			len(results), totalEffects)))
	sb.WriteString("\n\n")

	for _, result := range results {
		name := result.Target.QualifiedName()
		sb.WriteString(tuiHeaderStyle.Render(fmt.Sprintf("=== %s ===", name)))
		sb.WriteString("\n")
		sb.WriteString(statusStyle.Render(fmt.Sprintf("    %s", result.Target.Location)))
		sb.WriteString("\n")

		if len(result.SideEffects) == 0 {
			sb.WriteString(statusStyle.Render("    No side effects detected."))
			sb.WriteString("\n\n")
			continue
		}

		// Build side effects table.
		rows := make([][]string, 0, len(result.SideEffects))
		for _, e := range result.SideEffects {
			desc := e.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			rows = append(rows, []string{
				string(e.Tier),
				string(e.Type),
				desc,
			})
		}

		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(tuiBorderStyle).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return tuiHeaderStyle
				}
				if col == 0 && row >= 0 && row < len(rows) {
					switch rows[row][0] {
					case "P0":
						return tierP0Style
					case "P1":
						return tierP1Style
					case "P2":
						return tierP2Style
					}
				}
				return lipgloss.NewStyle()
			}).
			Headers("TIER", "TYPE", "DESCRIPTION").
			Rows(rows...)

		sb.WriteString(t.String())
		sb.WriteString("\n\n")
	}

	return sb.String()
}

func (m analyzeModel) Init() tea.Cmd {
	return nil
}

func (m analyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 0
		footerHeight := 2
		verticalMargin := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMargin
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m analyzeModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	footer := statusStyle.Render(
		fmt.Sprintf(" %3.f%% ", m.viewport.ScrollPercent()*100)) +
		" " + m.help.View(m.keys)

	return m.viewport.View() + "\n" + footer
}

// runInteractiveAnalyze launches the Bubble Tea TUI for browsing
// analysis results.
func runInteractiveAnalyze(results []taxonomy.AnalysisResult) error {
	model := newAnalyzeModel(results)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
