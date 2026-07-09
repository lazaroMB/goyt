package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderPlaylists() string {
	var sb strings.Builder
	if m.isLoadingPlaylists {
		sb.WriteString("  Loading Playlists...\n")
		if m.statusMessage != "" {
			sb.WriteString("\n  " + m.statusMessage + "\n")
		}
		return sb.String()
	}

	if m.playlistsError != nil {
		sb.WriteString(fmt.Sprintf("  Error loading playlists: %v\n\n", m.playlistsError))
		sb.WriteString("  Make sure your Cookie is correctly configured in ~/.config/goyt/config.json\n")
		return sb.String()
	}

	if m.confirmDeletePlaylist {
		sb.WriteString("  Your Playlists:\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Error)).Bold(true).Render(
			fmt.Sprintf("  ⚠️  Are you sure you want to delete playlist %q?", m.playlistToDelete.Title),
		) + "\n\n")
		sb.WriteString("  [ Press 'y' to Confirm | Press 'n' or Esc to Cancel ]\n")
		return sb.String()
	}

	if !m.inPlaylistDetail {
		overhead := 3
		if m.statusMessage != "" {
			overhead = 5
		}
		maxVisible := m.mainHeight - 2 - overhead
		if maxVisible < 1 {
			maxVisible = 1
		}

		if len(m.libraryPlaylists) == 0 {
			sb.WriteString("  Your Playlists:\n\n")
			if m.statusMessage != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
			}
			sb.WriteString("  No playlists found in your library.\n")
			return sb.String()
		}

		start, end := getVisibleRange(len(m.libraryPlaylists), maxVisible, m.playlistListIndex)

		scrollIndicator := ""
		if start > 0 && end < len(m.libraryPlaylists) {
			scrollIndicator = " ▲ ▼"
		} else if start > 0 {
			scrollIndicator = " ▲"
		} else if end < len(m.libraryPlaylists) {
			scrollIndicator = " ▼"
		}

		sb.WriteString(fmt.Sprintf("  Your Playlists (showing %d-%d of %d)%s:\n\n", start+1, end, len(m.libraryPlaylists), scrollIndicator))

		if m.statusMessage != "" {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
		}

		for i := start; i < end; i++ {
			pl := m.libraryPlaylists[i]
			prefix := "  "
			if i == m.playlistListIndex && !m.focusSide {
				prefix = "> "
			}
			itemStyle := lipgloss.NewStyle()
			if i == m.playlistListIndex && !m.focusSide {
				itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).Bold(true)
			}
			line := fmt.Sprintf("%s%s (%s)", prefix, pl.Title, pl.Count)
			sb.WriteString(itemStyle.Render(line) + "\n")
		}
		sb.WriteString("\n  [ Press Enter to open | Press 'a' to add all songs to queue | Press 'r' to reload ]\n")
	} else {
		overhead := 3
		if m.statusMessage != "" {
			overhead = 5
		}
		maxVisible := m.mainHeight - 2 - overhead
		if maxVisible < 1 {
			maxVisible = 1
		}

		if len(m.selectedPlaylistTracks) == 0 {
			sb.WriteString(fmt.Sprintf("  Playlist: %s\n\n", m.selectedPlaylistName))
			if m.statusMessage != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
			}
			sb.WriteString("  No tracks in this playlist.\n")
			sb.WriteString("  [ Press Esc to go back ]\n")
			return sb.String()
		}

		start, end := getVisibleRange(len(m.selectedPlaylistTracks), maxVisible, m.playlistTrackIndex)

		scrollIndicator := ""
		if start > 0 && end < len(m.selectedPlaylistTracks) {
			scrollIndicator = " ▲ ▼"
		} else if start > 0 {
			scrollIndicator = " ▲"
		} else if end < len(m.selectedPlaylistTracks) {
			scrollIndicator = " ▼"
		}

		sb.WriteString(fmt.Sprintf("  Playlist: %s (showing %d-%d of %d)%s\n\n", m.selectedPlaylistName, start+1, end, len(m.selectedPlaylistTracks), scrollIndicator))

		if m.statusMessage != "" {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
		}

		for i := start; i < end; i++ {
			track := m.selectedPlaylistTracks[i]
			prefix := "  "
			if i == m.playlistTrackIndex && !m.focusSide {
				prefix = "> "
			}
			itemStyle := lipgloss.NewStyle()
			if i == m.playlistTrackIndex && !m.focusSide {
				itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).Bold(true)
			}
			cachedIndicator := ""
			if cached, _ := m.cacheMgr.IsCached(track.VideoID); cached {
				cachedIndicator = " ↓"
			}
			line := fmt.Sprintf("%s%s - %s (%s)%s", prefix, track.Artist, track.Title, track.Duration, cachedIndicator)
			sb.WriteString(itemStyle.Render(line) + "\n")
		}
		sb.WriteString("\n  [ Press Enter to play track | Press 'a' to add to queue | Press 'r' to reload | Press Esc to go back ]\n")
	}

	return sb.String()
}
