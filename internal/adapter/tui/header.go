package tui

import "github.com/charmbracelet/lipgloss"

func (m *Model) renderHeader() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#121212")).
		Background(lipgloss.Color(m.theme.PrimaryHighlight)).
		Padding(0, 1).
		Render("▶ GoYT — DEC VT220 Amber Audio Terminal")
}
