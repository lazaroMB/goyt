package tui

import "github.com/charmbracelet/lipgloss"

func (m *Model) renderHeader() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(m.theme.TextOnAccent)).
		Background(lipgloss.Color(m.theme.PrimaryHighlight)).
		Padding(0, 1).
		Render(m.theme.HeaderTitle)
}
