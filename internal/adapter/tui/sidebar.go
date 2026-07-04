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
		BorderForeground(lipgloss.Color(m.theme.PrimaryHighlight))

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
				Foreground(lipgloss.Color("#121212")).
				Background(lipgloss.Color(m.theme.PrimaryHighlight)).
				Bold(true)
			sbBuilder.WriteString(selectedStyle.Render(v) + "\n")
		} else {
			sbBuilder.WriteString(v + "\n")
		}
	}
	return sidebarStyle.Render(sbBuilder.String())
}
