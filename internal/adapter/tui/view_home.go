package tui

import "strings"

func (m *Model) renderHome() string {
	var sb strings.Builder
	sb.WriteString("\n  [ Welcome to GoYT! ]\n\n")
	sb.WriteString("  Keyboard Shortcuts:\n")
	sb.WriteString("  - Tab       : Switch focus between Sidebar and Workspace\n")
	sb.WriteString("  - Up/Down   : Navigate lists or sidebar items\n")
	sb.WriteString("  - Enter     : Select/Play items\n")
	sb.WriteString("  - /         : Focus search bar (in Search view)\n")
	sb.WriteString("  - Esc       : Clear & refocus search bar (or go back)\n")
	sb.WriteString("  - Space     : Toggle Play / Pause\n")
	sb.WriteString("  - a         : Add highlighted item to Play Queue\n")
	sb.WriteString("  - m / M     : Add highlighted track to a Playlist\n")
	sb.WriteString("  - d / D     : Delete highlighted playlist (in Playlists view)\n")
	sb.WriteString("  - n / p     : Skip to Next / Previous track\n")
	sb.WriteString("  - [ / ]     : Decrease / Increase volume\n")
	sb.WriteString("  - Left/Right: Seek 10s backward / forward\n")
	sb.WriteString("  - t         : Cycle color themes (iOS, Amber, Catppuccin, Nord, Dracula)\n")
	sb.WriteString("  - v         : Cycle visualizer modes (Wave, Block Bars, Minimal Sparkline)\n")
	sb.WriteString("  - r         : Retry fetching lyrics (in Lyrics view)\n")
	sb.WriteString("  - c         : Copy Now Playing card to clipboard\n")
	sb.WriteString("  - ?         : Toggle help screen overlay\n")
	sb.WriteString("  - q         : Exit program\n\n")
	sb.WriteString("  Note: Requires 'mpv' and 'yt-dlp' installed on your system.\n")
	return sb.String()
}
