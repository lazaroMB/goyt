package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderSidebar(sidebarWidth, mainHeight int) string {
	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(mainHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.PrimaryHighlight)).
		Background(lipgloss.Color(m.theme.Background)).
		Foreground(lipgloss.Color(m.theme.TextPrimary))

	if !m.focusSide {
		sidebarStyle = sidebarStyle.BorderForeground(lipgloss.Color(m.theme.InactiveBorder))
	}

	var sbBuilder strings.Builder
	mcpText := "  MCP [ON]  "
	if m.mcpEnabled != nil && !m.mcpEnabled.Load() {
		mcpText = "  MCP [OFF]  "
	}
	views := []string{"  Home  ", "  Search  ", "  Playlists  ", "  Queue  ", mcpText}
	for i, v := range views {
		if i == m.sidebarIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.TextOnAccent)).
				Background(lipgloss.Color(m.theme.PrimaryHighlight)).
				Bold(true)
			sbBuilder.WriteString(selectedStyle.Render(v) + "\n")
		} else {
			unselectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.TextSecondary))
			sbBuilder.WriteString(unselectedStyle.Render(v) + "\n")
		}
	}
	return sidebarStyle.Render(sbBuilder.String())
}
