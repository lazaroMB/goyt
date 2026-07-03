package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"goyt/pkg/player"
	"goyt/pkg/queue"
	"goyt/pkg/ytmusic"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ActiveView represents the active view tab.
type ActiveView int

const (
	ViewHome ActiveView = iota
	ViewSearch
	ViewPlaylists
	ViewQueue
)

// MpvEventMsg wraps mpv IPC events.
type MpvEventMsg player.Event

// TickMsg updates the TUI timer.
type TickMsg time.Time

// ClearStatusMsg triggers clearing the TUI toast status message.
type ClearStatusMsg struct{}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// Model is the main Bubble Tea model.
type Model struct {
	client     *ytmusic.Client
	player     *player.Player
	queue      *queue.Queue
	activeView ActiveView
	focusSide  bool // true = sidebar, false = main pane

	// UI Component State
	sidebarIndex   int
	searchListIndex int
	queueListIndex  int
	searchInput     textinput.Model
	isSearching     bool
	searchError     error

	// Search Results
	searchResults []ytmusic.Track
	searchContinuation string
	isLoadingNextPage  bool

	// Playlist View State
	libraryPlaylists       []ytmusic.Playlist
	selectedPlaylistTracks []ytmusic.Track
	selectedPlaylistName   string
	playlistListIndex      int
	playlistTrackIndex     int
	inPlaylistDetail       bool
	isLoadingPlaylists     bool
	playlistsError         error

	// Player State (synced from mpv)
	isPlaying    bool
	duration     float64
	timePos      float64
	volume       int
	currentTrack ytmusic.Track
	trackLoaded  bool
	isLoading    bool
	statusMessage string

	// Equalizer State
	equalizerBars []int

	// Window Dimensions
	width  int
	height int
}

// NewModel initializes the Bubble Tea model.
func NewModel(client *ytmusic.Client, p *player.Player, q *queue.Queue) *Model {
	ti := textinput.New()
	ti.Placeholder = "Search songs, artists..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	return &Model{
		client:        client,
		player:        p,
		queue:         q,
		activeView:    ViewHome,
		focusSide:     true,
		sidebarIndex:  0,
		searchInput:   ti,
		volume:        70,
		equalizerBars: make([]int, 8),
	}
}

// Custom command messages for playlists
type playlistsLoadError struct{ err error }
type playlistTracksLoaded struct {
	tracks []ytmusic.Track
	name   string
}
type playlistTracksLoadError struct{ err error }
type playlistTracksEnqueueLoaded struct {
	tracks []ytmusic.Track
	name   string
}
type playlistTracksEnqueueError struct{ err error }

// LoadPlaylistsCmd fetches the authenticated user's library playlists.
func (m *Model) LoadPlaylistsCmd() tea.Cmd {
	return func() tea.Msg {
		playlists, err := m.client.GetLibraryPlaylists()
		if err != nil {
			return playlistsLoadError{err}
		}
		return playlists
	}
}

// LoadPlaylistTracksCmd fetches the tracks of a playlist.
func (m *Model) LoadPlaylistTracksCmd(playlistID string, name string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.GetPlaylistTracks(playlistID)
		if err != nil {
			return playlistTracksLoadError{err}
		}
		return playlistTracksLoaded{tracks: tracks, name: name}
	}
}

// EnqueuePlaylistCmd fetches the tracks of a playlist to add them directly to queue.
func (m *Model) EnqueuePlaylistCmd(playlistID string, name string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.client.GetPlaylistTracks(playlistID)
		if err != nil {
			return playlistTracksEnqueueError{err}
		}
		return playlistTracksEnqueueLoaded{tracks: tracks, name: name}
	}
}

type equalizerTickMsg time.Time

func (m *Model) tickEqualizer() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return equalizerTickMsg(t)
	})
}

// Init sets up the initial commands.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.waitForMpvEvents(),
		m.tickEqualizer(),
	)
}

func (m *Model) waitForMpvEvents() tea.Cmd {
	return func() tea.Msg {
		ev := <-m.player.Events()
		return MpvEventMsg(ev)
	}
}

type searchResultsMsg struct {
	tracks       []ytmusic.Track
	continuation string
	query        string
	err          error
}

type searchNextPageMsg struct {
	tracks       []ytmusic.Track
	continuation string
	err          error
}

// SearchCmd performs a search asynchronously.
func (m *Model) SearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		tracks, continuation, err := m.client.Search(query)
		if err != nil {
			return searchResultsMsg{err: err}
		}
		return searchResultsMsg{tracks: tracks, continuation: continuation, query: query}
	}
}

// SearchNextPageCmd loads the next page of search results.
func (m *Model) SearchNextPageCmd(token string) tea.Cmd {
	return func() tea.Msg {
		tracks, continuation, err := m.client.SearchNextPage(token)
		if err != nil {
			return searchNextPageMsg{err: err}
		}
		return searchNextPageMsg{tracks: tracks, continuation: continuation}
	}
}

// PlayTrackCmd loads a track into the player.
func (m *Model) PlayTrackCmd(track ytmusic.Track) tea.Cmd {
	return func() tea.Msg {
		// Use ytdl:// prefix to let mpv's internal yt-dlp hook resolve the stream URL
		url := fmt.Sprintf("ytdl://%s", track.VideoID)
		err := m.player.LoadFile(url)
		if err != nil {
			return err
		}
		// Reset duration/progress indicators
		return track
	}
}

// Update handles user inputs and mpv events.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.player.Stop()
			return m, tea.Quit

		// Global playback shortcuts (when not typing in search box)
		case "space", " ":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.isPlaying = !m.isPlaying
				_ = m.player.SetPause(!m.isPlaying)
				return m, nil
			}
		case "n":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				if nextTrack, ok := m.queue.Next(); ok {
					m.currentTrack = nextTrack
					m.trackLoaded = true
					m.isLoading = true
					cmds = append(cmds, m.PlayTrackCmd(nextTrack))
				}
				return m, tea.Batch(cmds...)
			}
		case "p":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				if prevTrack, ok := m.queue.Prev(); ok {
					m.currentTrack = prevTrack
					m.trackLoaded = true
					m.isLoading = true
					cmds = append(cmds, m.PlayTrackCmd(prevTrack))
				}
				return m, tea.Batch(cmds...)
			}
		case "[":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.volume = max(0, m.volume-5)
				_ = m.player.SetVolume(m.volume)
				return m, nil
			}
		case "]":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.volume = min(100, m.volume+5)
				_ = m.player.SetVolume(m.volume)
				return m, nil
			}
		case "left":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				_ = m.player.Seek(-10)
				return m, nil
			}
		case "right":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				_ = m.player.Seek(10)
				return m, nil
			}

		case "a":
			if !m.focusSide {
				switch m.activeView {
				case ViewSearch:
					if !m.searchInput.Focused() && len(m.searchResults) > 0 {
						track := m.searchResults[m.searchListIndex]
						m.queue.Add(track)
						m.statusMessage = fmt.Sprintf("Added to queue: %s - %s", track.Artist, track.Title)
						
						if !m.trackLoaded {
							m.currentTrack = track
							m.trackLoaded = true
							m.isLoading = true
							return m, tea.Batch(
								m.PlayTrackCmd(track),
								clearStatusAfter(3*time.Second),
							)
						}
						return m, clearStatusAfter(3*time.Second)
					}
				case ViewPlaylists:
					if m.inPlaylistDetail {
						if len(m.selectedPlaylistTracks) > 0 {
							track := m.selectedPlaylistTracks[m.playlistTrackIndex]
							m.queue.Add(track)
							m.statusMessage = fmt.Sprintf("Added to queue: %s - %s", track.Artist, track.Title)
							
							if !m.trackLoaded {
								m.currentTrack = track
								m.trackLoaded = true
								m.isLoading = true
								return m, tea.Batch(
									m.PlayTrackCmd(track),
									clearStatusAfter(3*time.Second),
								)
							}
							return m, clearStatusAfter(3*time.Second)
						}
					} else {
						if len(m.libraryPlaylists) > 0 {
							pl := m.libraryPlaylists[m.playlistListIndex]
							m.isLoadingPlaylists = true
							m.playlistsError = nil
							m.statusMessage = fmt.Sprintf("Loading all tracks from playlist %q...", pl.Title)
							return m, tea.Batch(
								m.EnqueuePlaylistCmd(pl.ID, pl.Title),
								clearStatusAfter(3*time.Second),
							)
						}
					}
				}
			}

		case "tab":
			m.focusSide = !m.focusSide
			if !m.focusSide && m.activeView == ViewSearch {
				m.searchInput.Focus()
			} else {
				m.searchInput.Blur()
			}
			return m, nil

		case "up", "k":
			if m.focusSide {
				m.sidebarIndex = max(0, m.sidebarIndex-1)
				m.activeView = ActiveView(m.sidebarIndex)
				if m.activeView == ViewPlaylists && len(m.libraryPlaylists) == 0 {
					m.isLoadingPlaylists = true
					m.playlistsError = nil
					cmds = append(cmds, m.LoadPlaylistsCmd())
				}
			} else {
				switch m.activeView {
				case ViewSearch:
					if !m.searchInput.Focused() {
						m.searchListIndex = max(0, m.searchListIndex-1)
					}
				case ViewPlaylists:
					if m.inPlaylistDetail {
						m.playlistTrackIndex = max(0, m.playlistTrackIndex-1)
					} else {
						m.playlistListIndex = max(0, m.playlistListIndex-1)
					}
				case ViewQueue:
					m.queueListIndex = max(0, m.queueListIndex-1)
				}
			}
			return m, tea.Batch(cmds...)

		case "down", "j":
			if m.focusSide {
				m.sidebarIndex = min(3, m.sidebarIndex+1)
				m.activeView = ActiveView(m.sidebarIndex)
				if m.activeView == ViewPlaylists && len(m.libraryPlaylists) == 0 {
					m.isLoadingPlaylists = true
					m.playlistsError = nil
					cmds = append(cmds, m.LoadPlaylistsCmd())
				}
			} else {
				switch m.activeView {
				case ViewSearch:
					if !m.searchInput.Focused() {
						m.searchListIndex = min(len(m.searchResults)-1, m.searchListIndex+1)
						if m.searchListIndex >= len(m.searchResults)-5 && m.searchContinuation != "" && !m.isLoadingNextPage {
							m.isLoadingNextPage = true
							cmds = append(cmds, m.SearchNextPageCmd(m.searchContinuation))
						}
					}
				case ViewPlaylists:
					if m.inPlaylistDetail {
						m.playlistTrackIndex = min(len(m.selectedPlaylistTracks)-1, m.playlistTrackIndex+1)
					} else {
						m.playlistListIndex = min(len(m.libraryPlaylists)-1, m.playlistListIndex+1)
					}
				case ViewQueue:
					m.queueListIndex = min(len(m.queue.List())-1, m.queueListIndex+1)
				}
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.focusSide {
				m.focusSide = false
				if m.activeView == ViewSearch {
					m.searchInput.Focus()
				}
			} else {
				switch m.activeView {
				case ViewSearch:
					if m.searchInput.Focused() {
						// Trigger search
						query := m.searchInput.Value()
						if query != "" {
							m.isSearching = true
							m.searchError = nil
							m.searchResults = nil
							m.searchInput.Blur()
							cmds = append(cmds, m.SearchCmd(query))
						}
					} else {
						// Select track from search results
						if len(m.searchResults) > 0 {
							track := m.searchResults[m.searchListIndex]
							m.queue.Add(track)
							// Play immediately, setting current index to the newly added track
							m.queue.SetIndex(len(m.queue.List()) - 1)
							m.currentTrack = track
							m.trackLoaded = true
							m.isLoading = true
							cmds = append(cmds, m.PlayTrackCmd(track))
						}
					}
				case ViewPlaylists:
					if !m.inPlaylistDetail {
						if len(m.libraryPlaylists) > 0 {
							pl := m.libraryPlaylists[m.playlistListIndex]
							m.isLoadingPlaylists = true
							m.playlistsError = nil
							cmds = append(cmds, m.LoadPlaylistTracksCmd(pl.ID, pl.Title))
						}
					} else {
						if len(m.selectedPlaylistTracks) > 0 {
							track := m.selectedPlaylistTracks[m.playlistTrackIndex]
							m.queue.Add(track)
							m.queue.SetIndex(len(m.queue.List()) - 1)
							m.currentTrack = track
							m.trackLoaded = true
							m.isLoading = true
							cmds = append(cmds, m.PlayTrackCmd(track))
						}
					}
				case ViewQueue:
					if len(m.queue.List()) > 0 {
						if m.queue.SetIndex(m.queueListIndex) {
							if track, ok := m.queue.Current(); ok {
								m.currentTrack = track
								m.trackLoaded = true
								m.isLoading = true
								cmds = append(cmds, m.PlayTrackCmd(track))
							}
						}
					}
				}
			}
			return m, tea.Batch(cmds...)

		case "/":
			if m.activeView == ViewSearch && !m.focusSide && !m.searchInput.Focused() {
				m.searchInput.Focus()
				m.searchInput.SetValue(m.searchInput.Value())
				return m, nil
			}

		case "backspace":
			if m.activeView == ViewSearch && m.searchInput.Focused() {
				break
			}
			if m.activeView == ViewPlaylists && m.inPlaylistDetail {
				m.inPlaylistDetail = false
				return m, nil
			}
			return m, nil

		case "esc":
			if m.activeView == ViewPlaylists && m.inPlaylistDetail {
				m.inPlaylistDetail = false
				return m, nil
			}
			if m.activeView == ViewSearch {
				if m.searchInput.Focused() {
					m.searchInput.Blur()
				} else {
					m.searchInput.SetValue("")
					m.searchInput.Focus()
				}
				return m, nil
			}
			return m, nil
		}

	case ClearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case []ytmusic.Playlist:
		m.isLoadingPlaylists = false
		m.libraryPlaylists = msg
		m.playlistListIndex = 0
		return m, nil

	case playlistsLoadError:
		m.isLoadingPlaylists = false
		m.playlistsError = msg.err
		return m, nil

	case playlistTracksLoaded:
		m.isLoadingPlaylists = false
		m.selectedPlaylistTracks = msg.tracks
		m.selectedPlaylistName = msg.name
		m.playlistTrackIndex = 0
		m.inPlaylistDetail = true
		return m, nil

	case playlistTracksLoadError:
		m.isLoadingPlaylists = false
		m.playlistsError = msg.err
		return m, nil

	case playlistTracksEnqueueLoaded:
		m.isLoadingPlaylists = false
		if len(msg.tracks) > 0 {
			m.queue.Add(msg.tracks...)
			m.statusMessage = fmt.Sprintf("Added %d songs from %q to queue", len(msg.tracks), msg.name)
			
			if !m.trackLoaded {
				firstTrack := msg.tracks[0]
				m.queue.SetIndex(len(m.queue.List()) - len(msg.tracks))
				m.currentTrack = firstTrack
				m.trackLoaded = true
				m.isLoading = true
				return m, tea.Batch(
					m.PlayTrackCmd(firstTrack),
					clearStatusAfter(3*time.Second),
				)
			}
		} else {
			m.statusMessage = fmt.Sprintf("Playlist %q is empty", msg.name)
		}
		return m, clearStatusAfter(3*time.Second)
	case playlistTracksEnqueueError:
		m.isLoadingPlaylists = false
		m.playlistsError = msg.err
		m.statusMessage = fmt.Sprintf("Failed to enqueue playlist: %v", msg.err)
		return m, clearStatusAfter(3*time.Second)

	case equalizerTickMsg:
		if m.isPlaying {
			for i := range m.equalizerBars {
				// Random step of -2, -1, 0, +1, +2
				delta := rand.Intn(5) - 2
				newVal := m.equalizerBars[i] + delta
				if newVal < 1 {
					newVal = 1
				} else if newVal > 5 {
					newVal = 5
				}
				if newVal == 1 && rand.Float32() < 0.2 {
					newVal = rand.Intn(3) + 2
				}
				m.equalizerBars[i] = newVal
			}
		} else {
			for i := range m.equalizerBars {
				if m.equalizerBars[i] > 1 {
					m.equalizerBars[i]--
				}
			}
		}
		return m, m.tickEqualizer()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case searchResultsMsg:
		m.isSearching = false
		if msg.err != nil {
			m.searchError = msg.err
			return m, nil
		}
		m.searchResults = msg.tracks
		m.searchContinuation = msg.continuation
		m.searchListIndex = 0
		m.searchError = nil

		var cmds []tea.Cmd
		if m.searchContinuation != "" && !m.isLoadingNextPage {
			m.isLoadingNextPage = true
			cmds = append(cmds, m.SearchNextPageCmd(m.searchContinuation))
		}
		return m, tea.Batch(cmds...)

	case searchNextPageMsg:
		m.isLoadingNextPage = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading more results: %v", msg.err)
			return m, clearStatusAfter(3*time.Second)
		}
		m.searchResults = append(m.searchResults, msg.tracks...)
		m.searchContinuation = msg.continuation
		return m, nil

	case ytmusic.Track:
		m.currentTrack = msg
		m.isPlaying = true
		return m, nil

	case MpvEventMsg:
		m.handleMpvEvent(player.Event(msg))
		return m, m.waitForMpvEvents()
	}

	if m.activeView == ViewSearch && m.searchInput.Focused() {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleMpvEvent(ev player.Event) {
	switch ev.Event {
	case "start-file":
		m.isLoading = true
	case "file-loaded":
		m.isLoading = false
	case "property-change":
		switch ev.Name {
		case "time-pos":
			if val, ok := ev.Data.(float64); ok {
				m.timePos = val
			}
		case "duration":
			if val, ok := ev.Data.(float64); ok {
				m.duration = val
			}
		case "pause":
			if val, ok := ev.Data.(bool); ok {
				m.isPlaying = !val
			}
		case "volume":
			if val, ok := ev.Data.(float64); ok {
				m.volume = int(val)
			}
		}
	case "end-file":
		m.isLoading = false
		// Auto-play next song on EOF (only if it ended naturally via eof, not manual stops)
		if ev.Reason == "eof" {
			if nextTrack, ok := m.queue.Next(); ok {
				m.currentTrack = nextTrack
				m.trackLoaded = true
				m.isLoading = true
				// We can't return commands from this helper method directly,
				// but we can spawn the action by writing back to player.
				// To keep it simple, we load the next file
				url := fmt.Sprintf("ytdl://%s", nextTrack.VideoID)
				_ = m.player.LoadFile(url)
			} else {
				m.isPlaying = false
				m.trackLoaded = false
				m.timePos = 0
				m.duration = 0
			}
		}
	}
}

// View renders the TUI layouts.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing GoYT..."
	}

	// Layout elements
	sidebarWidth := 20
	mainWidth := m.width - sidebarWidth - 4 // border padding
	mainHeight := m.height - 10            // minus header and footer player

	// Define styles
	sidebarStyle := lipgloss.NewStyle().
		Width(sidebarWidth).
		Height(mainHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF0000"))

	mainStyle := lipgloss.NewStyle().
		Width(mainWidth).
		Height(mainHeight - 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3C3C3C"))

	if !m.focusSide {
		mainStyle = mainStyle.BorderForeground(lipgloss.Color("#FF0000"))
		sidebarStyle = sidebarStyle.BorderForeground(lipgloss.Color("#3C3C3C"))
	}

	// Header
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#FF0000")).
		Padding(0, 1).
		Render("▶ GoYT — YouTube Music CLI Player (Go)")

	// Sidebar Views
	var sbBuilder strings.Builder
	views := []string{"  Home  ", "  Search  ", "  Playlists  ", "  Queue  "}
	for i, v := range views {
		if i == m.sidebarIndex {
			selectedStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#121212")).
				Background(lipgloss.Color("#FF0000")).
				Bold(true)
			sbBuilder.WriteString(selectedStyle.Render(v) + "\n")
		} else {
			sbBuilder.WriteString(v + "\n")
		}
	}
	sidebarView := sidebarStyle.Render(sbBuilder.String())

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
	}
	mainPanel := mainStyle.Render(mainView)

	// Horizontal layouts
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainPanel)

	// Footer (Player controls & progress bar)
	footer := m.renderFooter(m.width - 2)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

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
	sb.WriteString("  - n / p     : Skip to Next / Previous track\n")
	sb.WriteString("  - [ / ]     : Decrease / Increase volume\n")
	sb.WriteString("  - Left/Right: Seek 10s backward / forward\n")
	sb.WriteString("  - q         : Exit program\n\n")
	sb.WriteString("  Note: Requires 'mpv' and 'yt-dlp' installed on your system.\n")
	return sb.String()
}

func (m *Model) renderSearch() string {
	var sb strings.Builder
	sb.WriteString("  Search Songs:\n")
	sb.WriteString(fmt.Sprintf("  %s\n\n", m.searchInput.View()))

	if m.statusMessage != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  "+m.statusMessage) + "\n\n")
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
	mainHeight := m.height - 10
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
			itemStyle = itemStyle.Foreground(lipgloss.Color("#FF0000")).Bold(true)
		}
		line := fmt.Sprintf("%s%s - %s (%s)", prefix, track.Artist, track.Title, track.Duration)
		sb.WriteString(itemStyle.Render(line) + "\n")
	}

	return sb.String()
}

func (m *Model) renderQueue() string {
	var sb strings.Builder

	tracks := m.queue.List()
	if len(tracks) == 0 {
		sb.WriteString("  Play Queue:\n\n")
		sb.WriteString("  Queue is empty. Go search and add songs!\n")
		return sb.String()
	}

	mainHeight := m.height - 10
	maxVisible := mainHeight - 2 - 3
	if maxVisible < 5 {
		maxVisible = 5
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
			itemStyle = itemStyle.Foreground(lipgloss.Color("#00FF00")).Bold(true)
		} else if i == m.queueListIndex && !m.focusSide {
			itemStyle = itemStyle.Foreground(lipgloss.Color("#FF0000")).Bold(true)
		}

		line := fmt.Sprintf("%s%s - %s (%s)", prefix, track.Artist, track.Title, track.Duration)
		sb.WriteString(itemStyle.Render(line) + "\n")
	}

	return sb.String()
}

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
		if pad > 0 {
			left := pad / 2
			right := pad - left
			progress = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")).Render(
				strings.Repeat("▰", left) + msg + strings.Repeat("▰", right),
			)
		} else {
			progress = msg
		}
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
				
				// Filled circle if y < h, space otherwise
				char := ' '
				isFilled := y < h
				if isFilled {
					char = '●'
				}

				// Styling
				var style lipgloss.Style
				if isFilled && x < highlightedCount {
					// Active played LED: full diagonal rainbow gradient color
					factor := float64(x)/float64(barWidth-1)*0.6 + float64(y)/4.0*0.4
					colorHex := interpolateColor(factor)
					style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
				} else if isFilled {
					// Active unplayed LED: dim gradient color
					factor := float64(x)/float64(barWidth-1)*0.6 + float64(y)/4.0*0.4
					colorHex := interpolateColorDim(factor)
					style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex))
				} else {
					// Inactive LED: very dim empty circle
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
		BorderForeground(lipgloss.Color("#3C3C3C"))

	return footerStyle.Render(sb.String())
}

type rgb struct {
	r, g, b int
}

func interpolateColor(factor float64) string {
	keyframes := []rgb{
		{0, 242, 254},  // Cyan
		{79, 172, 254}, // Blue
		{0, 255, 135},  // Lime Green
		{254, 200, 96}, // Gold/Orange
		{255, 8, 68},   // Red
	}
	return interpolateKeyframes(factor, keyframes)
}

func interpolateColorDim(factor float64) string {
	keyframes := []rgb{
		{0, 60, 64},   // Dark Cyan
		{20, 43, 64},  // Dark Blue
		{0, 64, 34},   // Dark Green
		{64, 50, 24},  // Dark Gold/Orange
		{64, 2, 17},   // Dark Red
	}
	return interpolateKeyframes(factor, keyframes)
}

func interpolateKeyframes(factor float64, keyframes []rgb) string {
	if factor <= 0 {
		return fmt.Sprintf("#%02X%02X%02X", keyframes[0].r, keyframes[0].g, keyframes[0].b)
	}
	if factor >= 1 {
		last := keyframes[len(keyframes)-1]
		return fmt.Sprintf("#%02X%02X%02X", last.r, last.g, last.b)
	}

	idx := factor * float64(len(keyframes)-1)
	low := int(idx)
	high := low + 1
	t := idx - float64(low)

	c1 := keyframes[low]
	c2 := keyframes[high]

	r := int(float64(c1.r)*(1-t) + float64(c2.r)*t)
	g := int(float64(c1.g)*(1-t) + float64(c2.g)*t)
	b := int(float64(c1.b)*(1-t) + float64(c2.b)*t)

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

func formatTime(seconds float64) string {
	s := int(seconds)
	min := s / 60
	sec := s % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

	mainHeight := m.height - 10

	if !m.inPlaylistDetail {
		overhead := 4
		if m.statusMessage != "" {
			overhead = 6
		}
		maxVisible := mainHeight - 2 - overhead
		if maxVisible < 5 {
			maxVisible = 5
		}

		if len(m.libraryPlaylists) == 0 {
			sb.WriteString("  Your Playlists:\n\n")
			if m.statusMessage != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  "+m.statusMessage) + "\n\n")
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
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  "+m.statusMessage) + "\n\n")
		}

		for i := start; i < end; i++ {
			pl := m.libraryPlaylists[i]
			prefix := "  "
			if i == m.playlistListIndex && !m.focusSide {
				prefix = "> "
			}
			itemStyle := lipgloss.NewStyle()
			if i == m.playlistListIndex && !m.focusSide {
				itemStyle = itemStyle.Foreground(lipgloss.Color("#FF0000")).Bold(true)
			}
			line := fmt.Sprintf("%s%s (%s)", prefix, pl.Title, pl.Count)
			sb.WriteString(itemStyle.Render(line) + "\n")
		}
		sb.WriteString("\n  [ Press Enter to open | Press 'a' to add all songs to queue ]\n")
	} else {
		overhead := 4
		if m.statusMessage != "" {
			overhead = 6
		}
		maxVisible := mainHeight - 2 - overhead
		if maxVisible < 5 {
			maxVisible = 5
		}

		if len(m.selectedPlaylistTracks) == 0 {
			sb.WriteString(fmt.Sprintf("  Playlist: %s\n\n", m.selectedPlaylistName))
			if m.statusMessage != "" {
				sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  "+m.statusMessage) + "\n\n")
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
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("  "+m.statusMessage) + "\n\n")
		}

		for i := start; i < end; i++ {
			track := m.selectedPlaylistTracks[i]
			prefix := "  "
			if i == m.playlistTrackIndex && !m.focusSide {
				prefix = "> "
			}
			itemStyle := lipgloss.NewStyle()
			if i == m.playlistTrackIndex && !m.focusSide {
				itemStyle = itemStyle.Foreground(lipgloss.Color("#FF0000")).Bold(true)
			}
			line := fmt.Sprintf("%s%s - %s (%s)", prefix, track.Artist, track.Title, track.Duration)
			sb.WriteString(itemStyle.Render(line) + "\n")
		}
		sb.WriteString("\n  [ Press Enter to play track | Press 'a' to add to queue | Press Esc to go back ]\n")
	}

	return sb.String()
}
