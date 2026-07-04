package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderPlaylistSelect() string {
	var sb strings.Builder
	sb.WriteString("\n  [ Add Song to Playlist ]\n\n")
	sb.WriteString(fmt.Sprintf("  Song: %s - %s\n\n", m.trackToManage.Artist, m.trackToManage.Title))

	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
	}

	if m.creatingPlaylist {
		sb.WriteString("  Create New Playlist:\n")
		sb.WriteString(fmt.Sprintf("  %s\n\n", m.playlistInput.View()))
		sb.WriteString("  [ Press Enter to Create & Add | Press Esc to Cancel ]\n")
		return sb.String()
	}

	// Playlist list
	mainHeight := m.height - 11
	overhead := 6
	maxVisible := mainHeight - 2 - overhead
	if maxVisible < 5 {
		maxVisible = 5
	}

	totalOptions := len(m.libraryPlaylists) + 1 // +1 for "Create New Playlist"

	start, end := getVisibleRange(totalOptions, maxVisible, m.playlistSelectIndex)

	scrollIndicator := ""
	if start > 0 && end < totalOptions {
		scrollIndicator = " ▲ ▼"
	} else if start > 0 {
		scrollIndicator = " ▲"
	} else if end < totalOptions {
		scrollIndicator = " ▼"
	}

	sb.WriteString(fmt.Sprintf("  Select Playlist (showing %d-%d of %d)%s:\n\n", start+1, end, totalOptions, scrollIndicator))

	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.playlistSelectIndex && !m.focusSide {
			prefix = "> "
		}

		itemStyle := lipgloss.NewStyle()
		if i == m.playlistSelectIndex && !m.focusSide {
			itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).Bold(true)
		}

		if i == 0 {
			sb.WriteString(itemStyle.Render(prefix+"[ + Create New Playlist ]") + "\n")
		} else {
			pl := m.libraryPlaylists[i-1]
			sb.WriteString(itemStyle.Render(fmt.Sprintf("%s%s (%s)", prefix, pl.Title, pl.Count)) + "\n")
		}
	}

	sb.WriteString("\n  [ Press Enter to Select | Press Esc to Go Back ]\n")
	return sb.String()
}
