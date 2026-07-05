package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI layouts.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing GoYT..."
	}

	if m.showHelpOverlay {
		return m.renderHelpOverlay(m.width, m.height)
	}

	// Layout elements
	sidebarWidth := 20
	if m.width < 50 {
		sidebarWidth = 0
	} else if m.width < 80 {
		sidebarWidth = 14
	}

	// Header
	header := m.renderHeader()

	// Footer (Player controls & progress bar)
	footer := m.renderFooter(m.width - 2)

	var body string
	if sidebarWidth == 0 && m.focusSide {
		// Drawer/menu mode: render sidebar full screen
		body = m.renderSidebar(m.width-2, m.mainHeight)
	} else {
		// Define styles
		mainStyle := lipgloss.NewStyle().
			Width(m.mainWidth).
			Height(m.mainHeight - 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(m.theme.InactiveBorder)).
			Background(lipgloss.Color(m.theme.Surface)).
			Foreground(lipgloss.Color(m.theme.TextPrimary))

		if !m.focusSide {
			mainStyle = mainStyle.BorderForeground(lipgloss.Color(m.theme.PrimaryHighlight))
		}

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
		case ViewLyrics:
			mainView = m.renderLyrics()
		case ViewMCP:
			mainView = m.renderMCP()
		case ViewPlaylistSelect:
			mainView = m.renderPlaylistSelect()
		}
		mainPanel := mainStyle.Render(mainView)

		if sidebarWidth > 0 {
			sidebarView := m.renderSidebar(sidebarWidth, m.mainHeight)
			body = lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainPanel)
		} else {
			body = mainPanel
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderHelpOverlay(viewWidth, viewHeight int) string {
	helpText := `      GoYT Keyboard Controls
  ─────────────────────────────────────
  Tab           Toggle Sidebar / Pane
  Up/Down (j/k) Navigate lists/tabs
  Enter         Select / Play track
  /             Focus search bar
  a             Add track to queue
  m             Add track to playlist
  d             Delete playlist (confirm)
  Space         Play / Pause
  n             Next track
  p             Previous track
  [ / ]         Volume down / up
  Left/Right    Seek backward/forward
  t             Cycle color themes
  v             Cycle visualizers
  r             Retry fetching lyrics
  c             Copy Now Playing card
  ?             Toggle help screen
  q / Ctrl+C    Quit player
  ─────────────────────────────────────
  Press any key to close this menu.`

	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color(m.theme.PrimaryHighlight)).
		Background(lipgloss.Color(m.theme.Surface)).
		Foreground(lipgloss.Color(m.theme.TextPrimary)).
		Padding(1, 3).
		Width(40)

	renderedHelp := helpStyle.Render(helpText)
	return lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, renderedHelp)
}
