package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI layouts.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing GoYT..."
	}

	// Layout elements
	sidebarWidth := 20
	mainWidth := m.width - sidebarWidth - 4 // border padding
	mainHeight := m.height - 11            // minus header (1) and footer player (10)

	// Define styles
	mainStyle := lipgloss.NewStyle().
		Width(mainWidth).
		Height(mainHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.InactiveBorder))

	if !m.focusSide {
		mainStyle = mainStyle.BorderForeground(lipgloss.Color(m.theme.PrimaryHighlight))
	}

	// Header
	header := m.renderHeader()

	// Sidebar
	sidebarView := m.renderSidebar(sidebarWidth, mainHeight)

	// Main Panel View
	var mainView string
	switch m.activeView {
	case ViewHome:
		mainView = m.renderHome()
	case ViewSearch:
		mainView = m.renderSearch()
	case ViewPlaylists:
		mainView = m.renderPlaylists()
	case ViewQueue:
		mainView = m.renderQueue()
	case ViewMCP:
		mainView = m.renderMCP()
	case ViewPlaylistSelect:
		mainView = m.renderPlaylistSelect()
	}
	mainPanel := mainStyle.Render(mainView)

	// Horizontal layouts
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainPanel)

	// Footer (Player controls & progress bar)
	footer := m.renderFooter(m.width - 2)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
