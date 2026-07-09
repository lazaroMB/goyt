package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderQueue() string {
	var sb strings.Builder

	tracks := m.queue.List()
	if len(tracks) == 0 {
		sb.WriteString("  Play Queue:\n\n")
		sb.WriteString("  Queue is empty. Go search and add songs!\n")
		return sb.String()
	}

	maxVisible := m.mainHeight - 2 - 2
	if maxVisible < 1 {
		maxVisible = 1
	}

	currIdx := m.queue.CurrentIndex()
	start, end := getVisibleRange(len(tracks), maxVisible, m.queueListIndex)

	scrollIndicator := ""
	if start > 0 && end < len(tracks) {
		scrollIndicator = " ▲ ▼"
	} else if start > 0 {
		scrollIndicator = " ▲"
	} else if end < len(tracks) {
		scrollIndicator = " ▼"
	}

	sb.WriteString(fmt.Sprintf("  Play Queue (showing %d-%d of %d)%s:\n\n", start+1, end, len(tracks), scrollIndicator))

	for i := start; i < end; i++ {
		track := tracks[i]
		prefix := "  "
		if i == currIdx {
			prefix = "▶ "
		} else if i == m.queueListIndex && !m.focusSide {
			prefix = "> "
		}

		itemStyle := lipgloss.NewStyle()
		if i == currIdx {
			itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.SecondaryHighlight)).Bold(true)
		} else if i == m.queueListIndex && !m.focusSide {
			itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).Bold(true)
		}

		cachedIndicator := ""
		if cached, _ := m.cacheMgr.IsCached(track.VideoID); cached {
			cachedIndicator = " ↓"
		}
		line := fmt.Sprintf("%s%s - %s (%s)%s", prefix, track.Artist, track.Title, track.Duration, cachedIndicator)
		sb.WriteString(itemStyle.Render(line) + "\n")
	}

	return sb.String()
}
