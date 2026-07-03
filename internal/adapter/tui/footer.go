package tui

import (
	"fmt"
	"strings"

	"goyt/internal/domain/model"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) renderFooter(width int) string {
	var sb strings.Builder

	// Track Info Row (Line 1)
	trackTitle := "Idle"
	trackArtist := "No track loaded"
	if m.trackLoaded {
		trackTitle = m.currentTrack.Title
		trackArtist = m.currentTrack.Artist
	}

	statusIcon := "⏸"
	if m.isLoading {
		statusIcon = "⏳"
	} else if m.isPlaying {
		statusIcon = "▶"
	}

	info := fmt.Sprintf(" %s  %s - %s", statusIcon, trackArtist, trackTitle)
	if m.isLoading {
		info = fmt.Sprintf(" %s  Loading: %s - %s...", statusIcon, trackArtist, trackTitle)
	}
	sb.WriteString(info + "\n")

	// Waveform Progress Bar (Lines 2-6)
	barWidth := width - 2
	var progress string
	if m.isLoading {
		msg := " Resolving stream & buffering... "
		pad := barWidth - len(msg)
		var line string
		if pad > 0 {
			left := pad / 2
			right := pad - left
			line = lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.SecondaryHighlight)).Render(
				strings.Repeat("▰", left) + msg + strings.Repeat("▰", right),
			)
		} else {
			line = msg
		}
		progress = strings.Join([]string{
			"",
			"",
			line,
			"",
			"",
		}, "\n")
	} else if barWidth > 0 {
		// Ensure equalizerBars size matches barWidth exactly
		if len(m.equalizerBars) != barWidth {
			newBars := make([]int, barWidth)
			copy(newBars, m.equalizerBars)
			// Initialize new bars to baseline (1)
			for i := len(m.equalizerBars); i < barWidth; i++ {
				newBars[i] = 1
			}
			m.equalizerBars = newBars
		}

		pct := 0.0
		if m.duration > 0 {
			pct = m.timePos / m.duration
		}
		highlightedCount := int(pct * float64(barWidth))

		// Render 5 rows of text from top y = 4 to bottom y = 0
		var rows [5]strings.Builder
		for y := 4; y >= 0; y-- {
			for x := 0; x < barWidth; x++ {
				h := m.equalizerBars[x]

				char := ' '
				isFilled := y < h
				if isFilled {
					char = m.theme.EqualizerChar
				}

				// Styling
				var style lipgloss.Style
				if isFilled && x < highlightedCount {
					factor := float64(x)/float64(barWidth-1)*0.6 + float64(y)/4.0*0.4
					colorHex := m.interpolateColor(factor)
					style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
				} else if isFilled {
					factor := float64(x)/float64(barWidth-1)*0.6 + float64(y)/4.0*0.4
					colorHex := m.interpolateColorDim(factor)
					style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
				} else {
					style = lipgloss.NewStyle().Foreground(lipgloss.Color("#1C1C1C"))
				}
				rows[y].WriteString(style.Render(string(char)))
			}
		}

		progress = strings.Join([]string{
			rows[4].String(),
			rows[3].String(),
			rows[2].String(),
			rows[1].String(),
			rows[0].String(),
		}, "\n")
	} else {
		progress = ""
	}
	sb.WriteString(progress + "\n")

	// Time Pos & Volume Row (Line 7)
	currTimeStr := formatTime(m.timePos)
	totalTimeStr := formatTime(m.duration)
	timeInfo := fmt.Sprintf(" %s / %s", currTimeStr, totalTimeStr)
	if m.isLoading {
		timeInfo = " --:-- / --:--"
	}
	vol := fmt.Sprintf("Vol: %d%% ", m.volume)

	padSize := width - len(timeInfo) - len(vol)
	if padSize > 0 {
		sb.WriteString(timeInfo + strings.Repeat(" ", padSize) + vol)
	} else {
		sb.WriteString(timeInfo + "   " + vol)
	}

	footerStyle := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(m.theme.InactiveBorder))

	return footerStyle.Render(sb.String())
}

func (m *Model) interpolateColor(factor float64) string {
	return interpolateKeyframes(factor, m.theme.VisualizerPlayed)
}

func (m *Model) interpolateColorDim(factor float64) string {
	return interpolateKeyframes(factor, m.theme.VisualizerUnplayed)
}

func interpolateKeyframes(factor float64, keyframes []model.RGB) string {
	if len(keyframes) == 0 {
		return "#FFFFFF"
	}
	if factor <= 0 {
		return fmt.Sprintf("#%02X%02X%02X", keyframes[0].R, keyframes[0].G, keyframes[0].B)
	}
	if factor >= 1 {
		last := keyframes[len(keyframes)-1]
		return fmt.Sprintf("#%02X%02X%02X", last.R, last.G, last.B)
	}

	idx := factor * float64(len(keyframes)-1)
	low := int(idx)
	high := low + 1
	t := idx - float64(low)

	c1 := keyframes[low]
	c2 := keyframes[high]

	r := int(float64(c1.R)*(1-t) + float64(c2.R)*t)
	g := int(float64(c1.G)*(1-t) + float64(c2.G)*t)
	b := int(float64(c1.B)*(1-t) + float64(c2.B)*t)

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func formatTime(seconds float64) string {
	s := int(seconds)
	min := s / 60
	sec := s % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}
