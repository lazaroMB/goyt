package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderMCP() string {
	var sb strings.Builder
	sb.WriteString("\n  [ Model Context Protocol (MCP) Server ]\n\n")

	// Status line
	status := "ON"
	statusColor := "#00FF00" // Green
	if m.mcpEnabled != nil && !m.mcpEnabled.Load() {
		status = "OFF"
		statusColor = "#FF0000" // Red
	}

	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Bold(true)
	sb.WriteString(fmt.Sprintf("  Server Status: %s\n\n", statusStyle.Render(status)))

	// Client connection status
	clientConnected := "No"
	clientColor := "#555555" // Gray
	if m.mcpConnections > 0 {
		clientConnected = fmt.Sprintf("Yes (%d active connection(s))", m.mcpConnections)
		clientColor = "#00FFFF" // Cyan
	}
	clientStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(clientColor)).Bold(true)
	sb.WriteString(fmt.Sprintf("  Client Connected: %s\n\n", clientStyle.Render(clientConnected)))

	// Address line
	sb.WriteString("  Local Endpoint: http://localhost:8080/sse\n\n")

	// Help / control instruction
	sb.WriteString("  Controls:\n")
	sb.WriteString("  - Tab       : Focus main panel to toggle setting\n")
	sb.WriteString("  - Space     : Toggle MCP Server ON/OFF (when panel is focused)\n")
	sb.WriteString("  - Enter     : Toggle MCP Server ON/OFF (when panel is focused)\n\n")

	// Let the user know if they need to focus
	if m.focusSide {
		sb.WriteString("  (Press [Tab] to focus this pane and enable controls)\n")
	} else {
		selectedToggleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#121212")).
			Background(lipgloss.Color(m.theme.PrimaryHighlight)).
			Padding(0, 1).
			Bold(true)
		sb.WriteString("  " + selectedToggleStyle.Render("[ TOGGLE ON/OFF ]") + "\n")
	}

	return sb.String()
}
