package tui

import (
	"fmt"
	"sync/atomic"
	"time"

	"goyt/internal/domain/model"
	"goyt/internal/domain/port"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ActiveView int

const (
	ViewHome ActiveView = iota
	ViewSearch
	ViewPlaylists
	ViewQueue
	ViewMCP
)

type ClearStatusMsg struct{}

func ClearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return ClearStatusMsg{}
	})
}

// MCPSearchMsg updates the search view with results retrieved via MCP
type MCPSearchMsg struct {
	Query        string
	Tracks       []model.Track
	Continuation string
}

// MCPShowPlaylistsMsg switches the view to list all playlists
type MCPShowPlaylistsMsg struct {
	Playlists []model.Playlist
}

// MCPShowPlaylistDetailMsg switches the view to details of a specific playlist
type MCPShowPlaylistDetailMsg struct {
	PlaylistName string
	Tracks       []model.Track
}

// MCPEnqueueTrackMsg adds a track to the queue
type MCPEnqueueTrackMsg struct {
	Track model.Track
}

// MCPEnqueuePlaylistMsg adds all tracks of a playlist to the queue
type MCPEnqueuePlaylistMsg struct {
	PlaylistName string
	Tracks       []model.Track
}

// MCPPlayPauseMsg requests to update play/pause state
type MCPPlayPauseMsg struct {
	Action string // "play", "pause", "toggle"
}

// PlaybackInfo holds current player status
type PlaybackInfo struct {
	CurrentTrack model.Track
	IsPlaying    bool
	Duration     float64
	TimePos      float64
	Volume       int
}

// MCPGetPlaybackInfoMsg requests status from the TUI
type MCPGetPlaybackInfoMsg struct {
	ResponseChan chan PlaybackInfo
}

// MCPConnectionsMsg notifies the TUI of active client connections count
type MCPConnectionsMsg struct {
	Count int
}

type Model struct {
	theme      model.Theme
	catalog    port.MusicCatalogPort
	player     port.AudioPlayerPort
	queue      *model.Queue
	activeView ActiveView
	focusSide  bool // true = sidebar, false = main pane

	// UI Component State
	sidebarIndex    int
	searchListIndex int
	queueListIndex  int
	searchInput     textinput.Model
	isSearching     bool
	searchError     error

	// Search Results
	searchResults      []model.Track
	searchContinuation string
	isLoadingNextPage  bool

	// Playlist View State
	libraryPlaylists       []model.Playlist
	selectedPlaylistTracks []model.Track
	selectedPlaylistName   string
	playlistListIndex      int
	playlistTrackIndex     int
	inPlaylistDetail       bool
	isLoadingPlaylists     bool
	playlistsError         error

	// Player State (synced from mpv)
	isPlaying     bool
	duration      float64
	timePos       float64
	volume        int
	currentTrack  model.Track
	trackLoaded   bool
	isLoading     bool
	statusMessage string

	// Equalizer State
	equalizerBars    []int
	currentIntensity float64

	// MCP State
	mcpEnabled     *atomic.Bool
	mcpConnections int

	// Window Dimensions
	width  int
	height int
}

func NewModel(catalog port.MusicCatalogPort, player port.AudioPlayerPort, q *model.Queue, theme *model.Theme, mcpEnabled *atomic.Bool) *Model {
	ti := textinput.New()
	ti.Placeholder = "Search songs, artists..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	var t model.Theme
	if theme != nil {
		t = *theme
	} else {
		// Fallback defaults
		t = model.Theme{
			PrimaryHighlight:   "#FFB000",
			SecondaryHighlight: "#FFD700",
			InactiveBorder:     "#3C3C3C",
			VisualizerPlayed: []model.RGB{
				{R: 211, G: 84, B: 0},
				{R: 254, G: 153, B: 0},
				{R: 255, G: 215, B: 0},
			},
			VisualizerUnplayed: []model.RGB{
				{R: 62, G: 35, B: 0},
				{R: 90, G: 57, B: 0},
				{R: 122, G: 82, B: 0},
			},
			EqualizerChar:      '●',
		}
	}

	return &Model{
		theme:          t,
		catalog:        catalog,
		player:         player,
		queue:          q,
		activeView:     ViewHome,
		focusSide:      true,
		sidebarIndex:   0,
		searchInput:    ti,
		volume:         70,
		equalizerBars:  make([]int, 80), // default to 80 columns like we updated earlier
		mcpEnabled:     mcpEnabled,
		mcpConnections: 0,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickEqualizer(),
		m.waitForMpvEvents(),
		m.loadPlaylistsCmd(),
	)
}

// Commands & helper definitions

type MpvEventMsg port.PlayerEvent

func (m *Model) waitForMpvEvents() tea.Cmd {
	return func() tea.Msg {
		ev := <-m.player.Events()
		return MpvEventMsg(ev)
	}
}

type equalizerTickMsg struct{}

func (m *Model) tickEqualizer() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return equalizerTickMsg{}
	})
}

// Custom command messages for playlists
type playlistsLoaded struct {
	playlists []model.Playlist
}
type playlistsLoadError struct{ err error }
type playlistTracksLoaded struct {
	tracks []model.Track
	name   string
}
type playlistTracksLoadError struct{ err error }

type playlistTracksEnqueueLoaded struct {
	tracks []model.Track
	name   string
}
type playlistTracksEnqueueError struct{ err error }

func (m *Model) EnqueuePlaylistCmd(playlistID string, name string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.catalog.GetPlaylistTracks(playlistID)
		if err != nil {
			return playlistTracksEnqueueError{err}
		}
		return playlistTracksEnqueueLoaded{tracks: tracks, name: name}
	}
}

func (m *Model) loadPlaylistsCmd() tea.Cmd {
	return func() tea.Msg {
		playlists, err := m.catalog.GetLibraryPlaylists()
		if err != nil {
			return playlistsLoadError{err: err}
		}
		return playlistsLoaded{playlists: playlists}
	}
}

func (m *Model) loadPlaylistTracksCmd(playlistID, name string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := m.catalog.GetPlaylistTracks(playlistID)
		if err != nil {
			return playlistTracksLoadError{err: err}
		}
		return playlistTracksLoaded{tracks: tracks, name: name}
	}
}

type searchResultsMsg struct {
	tracks       []model.Track
	continuation string
	err          error
}

func (m *Model) SearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		tracks, continuation, err := m.catalog.Search(query)
		return searchResultsMsg{tracks: tracks, continuation: continuation, err: err}
	}
}

type searchNextPageMsg struct {
	tracks       []model.Track
	continuation string
	err          error
}

func (m *Model) SearchNextPageCmd(token string) tea.Cmd {
	return func() tea.Msg {
		tracks, continuation, err := m.catalog.SearchNextPage(token)
		return searchNextPageMsg{tracks: tracks, continuation: continuation, err: err}
	}
}

// PlayTrackCmd loads a track into the player.
func (m *Model) PlayTrackCmd(track model.Track) tea.Cmd {
	return func() tea.Msg {
		// Use ytdl:// prefix to let mpv's internal yt-dlp hook resolve the stream URL
		url := fmt.Sprintf("ytdl://%s", track.VideoID)
		err := m.player.LoadFile(url)
		if err != nil {
			return err
		}
		return track
	}
}

func (m *Model) PlayNextCmd() tea.Cmd {
	return func() tea.Msg {
		if nextTrack, ok := m.queue.Next(); ok {
			return m.PlayTrackCmd(nextTrack)()
		}
		return nil
	}
}
