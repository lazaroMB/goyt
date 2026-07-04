package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderSearch() string {
	var sb strings.Builder
	sb.WriteString("  Search Songs:\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", m.searchInput.View()))

	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Success)).Render("  "+m.statusMessage) + "\n\n")
	}

	if m.isSearching {
		sb.WriteString("  Searching YouTube Music...\n")
		return sb.String()
	}

	if m.searchError != nil {
		sb.WriteString(fmt.Sprintf("  Error: %v\n", m.searchError))
		return sb.String()
	}

	if len(m.searchResults) == 0 {
		sb.WriteString("  Type search query and press Enter.\n")
		return sb.String()
	}

	// Calculate maximum visible rows based on panel height and overhead
	overhead := 5
	if m.statusMessage != "" {
		overhead = 7
	}
	mainHeight := m.height - 11
	maxVisible := mainHeight - 2 - overhead
	if maxVisible < 5 {
		maxVisible = 5
	}

	start, end := getVisibleRange(len(m.searchResults), maxVisible, m.searchListIndex)

	scrollIndicator := ""
	if start > 0 && end < len(m.searchResults) {
		scrollIndicator = " ▲ ▼"
	} else if start > 0 {
		scrollIndicator = " ▲"
	} else if end < len(m.searchResults) {
		scrollIndicator = " ▼"
	}

	// Render search results header
	sb.WriteString(fmt.Sprintf("  Results (showing %d-%d of %d)%s:\n", start+1, end, len(m.searchResults), scrollIndicator))

	for i := start; i < end; i++ {
		track := m.searchResults[i]
		prefix := "  "
		if i == m.searchListIndex && !m.focusSide && !m.searchInput.Focused() {
			prefix = "> "
		}
		itemStyle := lipgloss.NewStyle()
		if i == m.searchListIndex && !m.focusSide && !m.searchInput.Focused() {
			itemStyle = itemStyle.Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).Bold(true)
		}
		line := fmt.Sprintf("%s%s - %s (%s)", prefix, track.Artist, track.Title, track.Duration)
		sb.WriteString(itemStyle.Render(line) + "\n")
	}

	return sb.String()
}

func getVisibleRange(totalItems, maxVisible, selectedIndex int) (start, end int) {
	if totalItems <= maxVisible {
		return 0, totalItems
	}

	start = selectedIndex - maxVisible/2
	if start < 0 {
		start = 0
	}
	if start+maxVisible > totalItems {
		start = totalItems - maxVisible
	}
	return start, start + maxVisible
}
