package tui

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"goyt/internal/domain/model"
	"goyt/internal/domain/port"
	"goyt/pkg/notification"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles user inputs and mpv events.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showHelpOverlay {
			m.showHelpOverlay = false
			return m, nil
		}
		if m.confirmDeletePlaylist {
			switch msg.String() {
			case "y", "Y":
				m.confirmDeletePlaylist = false
				m.statusMessage = fmt.Sprintf("Deleting playlist %q...", m.playlistToDelete.Title)
				m.isLoadingPlaylists = true
				return m, tea.Batch(
					m.DeletePlaylistCmd(m.playlistToDelete.ID, m.playlistToDelete.Title),
					ClearStatusAfter(3*time.Second),
				)
			case "n", "N", "esc":
				m.confirmDeletePlaylist = false
				return m, nil
			default:
				return m, nil
			}
		}

		if m.isAnyInputFocused() {
			switch msg.String() {
			case "ctrl+c":
				m.player.Stop()
				return m, tea.Quit
			case "enter":
				if m.activeView == ViewSearch && m.searchInput.Focused() {
					query := m.searchInput.Value()
					if query != "" {
						m.isSearching = true
						m.searchError = nil
						m.searchResults = nil
						m.searchInput.Blur()
						return m, m.SearchCmd(query)
					}
				} else if m.activeView == ViewPlaylistSelect && m.playlistInput.Focused() {
					name := m.playlistInput.Value()
					var createCmd tea.Cmd
					if name != "" {
						m.statusMessage = fmt.Sprintf("Creating playlist %q and adding track...", name)
						createCmd = m.CreatePlaylistAndAddTrackCmd(name, "", m.trackToManage)
					}
					m.playlistInput.Reset()
					m.playlistInput.Blur()
					m.creatingPlaylist = false
					m.activeView = m.previousView
					return m, createCmd
				}
				return m, nil
			case "esc":
				if m.activeView == ViewSearch && m.searchInput.Focused() {
					m.searchInput.Blur()
				} else if m.activeView == ViewPlaylistSelect && m.playlistInput.Focused() {
					m.creatingPlaylist = false
					m.playlistInput.Reset()
					m.playlistInput.Blur()
				}
				return m, nil
			case "tab":
				if m.activeView == ViewSearch {
					m.focusSide = !m.focusSide
					if !m.focusSide {
						m.searchInput.Focus()
					} else {
						m.searchInput.Blur()
					}
				}
				return m, nil
			default:
				if m.activeView == ViewSearch && m.searchInput.Focused() {
					m.searchInput, cmd = m.searchInput.Update(msg)
					return m, cmd
				}
				if m.activeView == ViewPlaylistSelect && m.playlistInput.Focused() {
					m.playlistInput, cmd = m.playlistInput.Update(msg)
					return m, cmd
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.player.Stop()
			return m, tea.Quit

		case "?":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.showHelpOverlay = true
				return m, nil
			}

		case "c":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				card := m.generateShareCard()
				if err := clipboard.WriteAll(card); err == nil {
					m.statusMessage = "Copied 'Now Playing' card to clipboard!"
				} else {
					m.statusMessage = "Failed to copy card to clipboard."
				}
				return m, ClearStatusAfter(3*time.Second)
			}

		case "v":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.visualizerMode = (m.visualizerMode + 1) % 3
				modes := []string{"Wave", "Block Bars", "Minimal Sparkline"}
				m.statusMessage = fmt.Sprintf("Visualizer Mode: %s", modes[m.visualizerMode])
				return m, ClearStatusAfter(2*time.Second)
			}

		case "t":
			if m.activeView != ViewSearch || !m.searchInput.Focused() {
				m.themeIndex = (m.themeIndex + 1) % len(model.PresetNames)
				nextName := model.PresetNames[m.themeIndex]
				m.theme = model.PresetThemes[nextName]
				m.statusMessage = fmt.Sprintf("Theme: %s", m.theme.Name)
				return m, ClearStatusAfter(2 * time.Second)
			}

		case "r":
			if m.activeView == ViewPlaylists {
				if m.inPlaylistDetail {
					if len(m.libraryPlaylists) > 0 && m.playlistListIndex >= 0 && m.playlistListIndex < len(m.libraryPlaylists) {
						pl := m.libraryPlaylists[m.playlistListIndex]
						m.isLoadingPlaylists = true
						m.playlistsError = nil
						m.statusMessage = fmt.Sprintf("Reloading playlist %q...", pl.Title)
						return m, tea.Batch(
							m.loadPlaylistTracksCmd(pl.ID, pl.Title),
							ClearStatusAfter(3*time.Second),
						)
					}
				} else {
					m.isLoadingPlaylists = true
					m.playlistsError = nil
					m.statusMessage = "Reloading library playlists..."
					return m, tea.Batch(
						m.loadPlaylistsCmd(),
						ClearStatusAfter(3*time.Second),
					)
				}
				return m, nil
			}

			if m.trackLoaded {
				m.lyricsLoading = true
				m.lyricsError = nil
				m.plainLyrics = ""
				m.syncedLyrics = nil
				m.statusMessage = "Retrying lyrics fetch..."
				return m, tea.Batch(
					m.fetchLyricsCmd(m.currentTrack.Artist, m.currentTrack.Title, m.currentTrack.VideoID),
					ClearStatusAfter(2*time.Second),
				)
			}

		// Global playback shortcuts (when not typing in search box)
		case "space", " ":
			if m.activeView == ViewMCP && !m.focusSide {
				if m.mcpEnabled != nil {
					m.mcpEnabled.Store(!m.mcpEnabled.Load())
					stateStr := "ON"
					if !m.mcpEnabled.Load() {
						stateStr = "OFF"
					}
					m.statusMessage = fmt.Sprintf("MCP Server toggled %s", stateStr)
					return m, ClearStatusAfter(3*time.Second)
				}
				return m, nil
			}
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
								ClearStatusAfter(3*time.Second),
							)
						}
						return m, ClearStatusAfter(3*time.Second)
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
									ClearStatusAfter(3*time.Second),
								)
							}
							return m, ClearStatusAfter(3*time.Second)
						}
					} else {
						if len(m.libraryPlaylists) > 0 {
							pl := m.libraryPlaylists[m.playlistListIndex]
							m.isLoadingPlaylists = true
							m.playlistsError = nil
							m.statusMessage = fmt.Sprintf("Loading all tracks from playlist %q...", pl.Title)
							return m, tea.Batch(
								m.EnqueuePlaylistCmd(pl.ID, pl.Title),
								ClearStatusAfter(3*time.Second),
							)
						}
					}
				}
			}

		case "m", "M":
			if !m.focusSide {
				var track model.Track
				var hasTrack bool

				switch m.activeView {
				case ViewSearch:
					if !m.searchInput.Focused() && len(m.searchResults) > 0 {
						track = m.searchResults[m.searchListIndex]
						hasTrack = true
					}
				case ViewPlaylists:
					if m.inPlaylistDetail && len(m.selectedPlaylistTracks) > 0 {
						track = m.selectedPlaylistTracks[m.playlistTrackIndex]
						hasTrack = true
					}
				case ViewQueue:
					if len(m.queue.List()) > 0 {
						track = m.queue.List()[m.queueListIndex]
						hasTrack = true
					}
				}

				if hasTrack {
					m.previousView = m.activeView
					m.trackToManage = track
					m.playlistSelectIndex = 0
					m.creatingPlaylist = false
					m.activeView = ViewPlaylistSelect
					m.playlistInput.Reset()
					m.playlistInput.Blur()
					return m, nil
				}
			}

		case "d", "D":
			if !m.focusSide && m.activeView == ViewPlaylists && !m.inPlaylistDetail {
				if len(m.libraryPlaylists) > 0 {
					m.playlistToDelete = m.libraryPlaylists[m.playlistListIndex]
					m.confirmDeletePlaylist = true
					return m, nil
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
					cmds = append(cmds, m.loadPlaylistsCmd())
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
				case ViewLyrics:
					m.lyricsScrollOffset = max(0, m.lyricsScrollOffset-1)
				case ViewPlaylistSelect:
					if !m.creatingPlaylist {
						m.playlistSelectIndex = max(0, m.playlistSelectIndex-1)
					}
				}
			}
			return m, tea.Batch(cmds...)

		case "down", "j":
			if m.focusSide {
				m.sidebarIndex = min(5, m.sidebarIndex+1)
				m.activeView = ActiveView(m.sidebarIndex)
				if m.activeView == ViewPlaylists && len(m.libraryPlaylists) == 0 {
					m.isLoadingPlaylists = true
					m.playlistsError = nil
					cmds = append(cmds, m.loadPlaylistsCmd())
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
				case ViewLyrics:
					m.lyricsScrollOffset++
				case ViewPlaylistSelect:
					if !m.creatingPlaylist {
						totalOptions := len(m.libraryPlaylists) + 1
						m.playlistSelectIndex = min(totalOptions-1, m.playlistSelectIndex+1)
					}
				}
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.focusSide {
				m.focusSide = false
				if m.activeView == ViewSearch {
					if len(m.searchResults) == 0 {
						m.searchInput.Focus()
					} else {
						m.searchInput.Blur()
					}
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
							cmds = append(cmds, m.loadPlaylistTracksCmd(pl.ID, pl.Title))
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
				case ViewMCP:
					if m.mcpEnabled != nil {
						m.mcpEnabled.Store(!m.mcpEnabled.Load())
						stateStr := "ON"
						if !m.mcpEnabled.Load() {
							stateStr = "OFF"
						}
						m.statusMessage = fmt.Sprintf("MCP Server toggled %s", stateStr)
						return m, ClearStatusAfter(3*time.Second)
					}
				case ViewPlaylistSelect:
					if m.creatingPlaylist {
						name := m.playlistInput.Value()
						if name != "" {
							m.statusMessage = fmt.Sprintf("Creating playlist %q and adding track...", name)
							cmds = append(cmds, m.CreatePlaylistAndAddTrackCmd(name, "", m.trackToManage))
						}
						m.playlistInput.Reset()
						m.playlistInput.Blur()
						m.creatingPlaylist = false
						m.activeView = m.previousView
					} else if m.playlistSelectIndex == 0 {
						m.creatingPlaylist = true
						m.playlistInput.Focus()
					} else {
						pl := m.libraryPlaylists[m.playlistSelectIndex-1]
						m.statusMessage = fmt.Sprintf("Adding track to playlist %q...", pl.Title)
						cmds = append(cmds, m.AddTrackToPlaylistCmd(pl.ID, pl.Title, m.trackToManage))
						m.activeView = m.previousView
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
			if m.activeView == ViewPlaylistSelect {
				if m.creatingPlaylist {
					m.creatingPlaylist = false
					m.playlistInput.Reset()
					m.playlistInput.Blur()
				} else {
					m.activeView = m.previousView
				}
				return m, nil
			}
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

	case MCPSearchMsg:
		m.activeView = ViewSearch
		m.focusSide = false
		m.searchInput.SetValue(msg.Query)
		m.searchInput.Blur()
		m.searchResults = msg.Tracks
		m.searchContinuation = msg.Continuation
		m.searchListIndex = 0
		m.searchError = nil
		m.isSearching = false
		m.statusMessage = fmt.Sprintf("Search updated via MCP: %q", msg.Query)
		return m, ClearStatusAfter(3*time.Second)

	case MCPShowPlaylistsMsg:
		m.activeView = ViewPlaylists
		m.focusSide = false
		m.libraryPlaylists = msg.Playlists
		m.playlistListIndex = 0
		m.inPlaylistDetail = false
		m.statusMessage = "Playlists loaded via MCP"
		return m, ClearStatusAfter(3*time.Second)

	case MCPShowPlaylistDetailMsg:
		m.activeView = ViewPlaylists
		m.focusSide = false
		m.selectedPlaylistName = msg.PlaylistName
		m.selectedPlaylistTracks = msg.Tracks
		m.playlistTrackIndex = 0
		m.inPlaylistDetail = true
		m.statusMessage = fmt.Sprintf("Opened playlist details via MCP: %q", msg.PlaylistName)
		return m, ClearStatusAfter(3*time.Second)

	case MCPEnqueueTrackMsg:
		m.queue.Add(msg.Track)
		m.activeView = ViewQueue
		m.statusMessage = fmt.Sprintf("Added to queue via MCP: %s - %s", msg.Track.Artist, msg.Track.Title)
		var playCmd tea.Cmd
		if !m.trackLoaded {
			m.currentTrack = msg.Track
			m.trackLoaded = true
			m.isLoading = true
			playCmd = m.PlayTrackCmd(msg.Track)
		}
		return m, tea.Batch(playCmd, ClearStatusAfter(3*time.Second))

	case MCPEnqueuePlaylistMsg:
		if len(msg.Tracks) > 0 {
			m.queue.Add(msg.Tracks...)
			m.activeView = ViewQueue
			m.statusMessage = fmt.Sprintf("Added %d tracks from %q to queue via MCP", len(msg.Tracks), msg.PlaylistName)
			var playCmd tea.Cmd
			if !m.trackLoaded {
				firstTrack := msg.Tracks[0]
				m.queue.SetIndex(len(m.queue.List()) - len(msg.Tracks))
				m.currentTrack = firstTrack
				m.trackLoaded = true
				m.isLoading = true
				playCmd = m.PlayTrackCmd(firstTrack)
			}
			return m, tea.Batch(playCmd, ClearStatusAfter(3*time.Second))
		}
		m.statusMessage = fmt.Sprintf("Playlist %q has no tracks", msg.PlaylistName)
		return m, ClearStatusAfter(3*time.Second)

	case MCPPlayPauseMsg:
		var targetPause bool
		switch msg.Action {
		case "play":
			targetPause = false
		case "pause":
			targetPause = true
		default: // toggle
			targetPause = m.isPlaying
		}
		_ = m.player.SetPause(targetPause)
		m.isPlaying = !targetPause
		m.statusMessage = fmt.Sprintf("Playback updated via MCP: %s", msg.Action)
		return m, ClearStatusAfter(3*time.Second)

	case MCPGetPlaybackInfoMsg:
		msg.ResponseChan <- PlaybackInfo{
			CurrentTrack: m.currentTrack,
			IsPlaying:    m.isPlaying,
			Duration:     m.duration,
			TimePos:      m.timePos,
			Volume:       m.volume,
		}
		return m, nil

	case MCPConnectionsMsg:
		m.mcpConnections = msg.Count
		return m, nil

	case MCPRefreshPlaylistsMsg:
		m.isLoadingPlaylists = true
		m.playlistsError = nil
		return m, m.loadPlaylistsCmd()

	case playlistCreatedAndAddedMsg:
		m.statusMessage = fmt.Sprintf("Created playlist %q and added %q", msg.playlistName, msg.trackTitle)
		m.isLoadingPlaylists = true
		return m, tea.Batch(m.loadPlaylistsCmd(), ClearStatusAfter(4*time.Second))

	case playlistAddedMsg:
		m.statusMessage = fmt.Sprintf("Added %q to playlist %q", msg.trackTitle, msg.playlistName)
		return m, ClearStatusAfter(4*time.Second)

	case playlistAddError:
		m.statusMessage = fmt.Sprintf("Error: %v", msg.err)
		return m, ClearStatusAfter(5*time.Second)

	case playlistDeletedMsg:
		m.statusMessage = fmt.Sprintf("Deleted playlist %q", msg.playlistName)
		m.isLoadingPlaylists = true
		m.playlistListIndex = 0
		return m, tea.Batch(m.loadPlaylistsCmd(), ClearStatusAfter(4*time.Second))

	case playlistDeleteError:
		m.statusMessage = fmt.Sprintf("Error deleting playlist: %v", msg.err)
		return m, ClearStatusAfter(5*time.Second)

	case ClearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case playlistsLoaded:
		m.isLoadingPlaylists = false
		m.libraryPlaylists = msg.playlists
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
					ClearStatusAfter(3*time.Second),
				)
			}
		} else {
			m.statusMessage = fmt.Sprintf("Playlist %q is empty", msg.name)
		}
		return m, ClearStatusAfter(3*time.Second)

	case playlistTracksEnqueueError:
		m.isLoadingPlaylists = false
		m.playlistsError = msg.err
		m.statusMessage = fmt.Sprintf("Failed to enqueue playlist: %v", msg.err)
		return m, ClearStatusAfter(3*time.Second)

	case equalizerTickMsg:
		if m.isPlaying {
			for i := range m.equalizerBars {
				factor := math.Sin(float64(i) / float64(len(m.equalizerBars)-1) * math.Pi)
				noise := 0.15 + rand.Float64()*0.85
				colMaxHeight := 1.0 + m.currentIntensity*4.0*factor*noise

				target := colMaxHeight
				current := float64(m.equalizerBars[i])

				var newVal float64
				if target > current {
					newVal = math.Round(target)
				} else {
					newVal = current - 0.5
				}

				if newVal < 1 {
					newVal = 1
				} else if newVal > 5 {
					newVal = 5
				}
				m.equalizerBars[i] = int(newVal)
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
		m.recalculateSizes(msg.Width, msg.Height)
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

		if m.searchContinuation != "" && !m.isLoadingNextPage {
			m.isLoadingNextPage = true
			cmds = append(cmds, m.SearchNextPageCmd(m.searchContinuation))
		}
		return m, tea.Batch(cmds...)

	case searchNextPageMsg:
		m.isLoadingNextPage = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading more results: %v", msg.err)
			return m, ClearStatusAfter(3*time.Second)
		}
		m.searchResults = append(m.searchResults, msg.tracks...)
		m.searchContinuation = msg.continuation
		return m, nil

	case model.Track:
		m.currentTrack = msg
		m.isPlaying = true
		return m, m.triggerTrackChange(msg)

	case lyricsLoadedMsg:
		if msg.trackID == m.lyricsTrackID {
			m.lyricsLoading = false
			m.plainLyrics = msg.plainLyrics
			m.syncedLyrics = msg.syncedLyrics
			m.lyricsError = msg.err
		}
		return m, nil

	case MpvEventMsg:
		cmd := m.handleMpvEvent(port.PlayerEvent(msg))
		return m, tea.Batch(cmd, m.waitForMpvEvents())
	}

	if m.activeView == ViewSearch && m.searchInput.Focused() {
		m.searchInput, cmd = m.searchInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.activeView == ViewPlaylistSelect && m.playlistInput.Focused() {
		m.playlistInput, cmd = m.playlistInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleMpvEvent(ev port.PlayerEvent) tea.Cmd {
	var cmd tea.Cmd
	switch ev.Type {
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
		case "af-metadata/myfilter":
			if metadataMap, ok := ev.Data.(map[string]interface{}); ok {
				if rmsStr, ok := metadataMap["lavfi.astats.Overall.RMS_level"].(string); ok {
					var rms float64
					if _, err := fmt.Sscanf(rmsStr, "%f", &rms); err == nil {
						if rms < -35 {
							rms = -35
						} else if rms > -10 {
							rms = -10
						}
						m.currentIntensity = (rms + 35.0) / 25.0
					}
				}
			}
		}
	case "end-file":
		m.isLoading = false
		if ev.Reason == "eof" {
			if nextTrack, ok := m.queue.Next(); ok {
				m.currentTrack = nextTrack
				m.trackLoaded = true
				m.isLoading = true
				url := fmt.Sprintf("ytdl://%s", nextTrack.VideoID)
				_ = m.player.LoadFile(url)
				cmd = m.triggerTrackChange(nextTrack)
			} else {
				m.isPlaying = false
				m.trackLoaded = false
				m.timePos = 0
				m.duration = 0
			}
		}
	}
	return cmd
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

func (m *Model) CreatePlaylistAndAddTrackCmd(name, description string, track model.Track) tea.Cmd {
	return func() tea.Msg {
		playlistID, err := m.catalog.CreatePlaylist(name, description)
		if err != nil {
			return playlistAddError{err: err}
		}
		err = m.catalog.AddTrackToPlaylist(playlistID, track.VideoID)
		if err != nil {
			return playlistAddError{err: err}
		}
		return playlistCreatedAndAddedMsg{playlistName: name, trackTitle: track.Title}
	}
}

func (m *Model) AddTrackToPlaylistCmd(playlistID, playlistName string, track model.Track) tea.Cmd {
	return func() tea.Msg {
		err := m.catalog.AddTrackToPlaylist(playlistID, track.VideoID)
		if err != nil {
			return playlistAddError{err: err}
		}
		return playlistAddedMsg{playlistName: playlistName, trackTitle: track.Title}
	}
}

type playlistAddError struct{ err error }
type playlistCreatedAndAddedMsg struct{ playlistName, trackTitle string }
type playlistAddedMsg struct{ playlistName, trackTitle string }

func (m *Model) DeletePlaylistCmd(playlistID, playlistName string) tea.Cmd {
	return func() tea.Msg {
		err := m.catalog.DeletePlaylist(playlistID)
		if err != nil {
			return playlistDeleteError{err: err}
		}
		return playlistDeletedMsg{playlistName: playlistName}
	}
}

type playlistDeleteError struct{ err error }
type playlistDeletedMsg struct{ playlistName string }

func (m *Model) triggerTrackChange(track model.Track) tea.Cmd {
	if m.lyricsTrackID == track.VideoID {
		return nil
	}
	m.lyricsTrackID = track.VideoID
	m.plainLyrics = ""
	m.syncedLyrics = nil
	m.lyricsLoading = true
	m.lyricsError = nil
	m.lyricsScrollOffset = 0

	if m.notificationsEnabled {
		notification.Send("Now Playing", fmt.Sprintf("%s - %s", track.Artist, track.Title))
	}

	return m.fetchLyricsCmd(track.Artist, track.Title, track.VideoID)
}

func (m *Model) generateShareCard() string {
	if !m.trackLoaded {
		return "No track loaded."
	}
	var sb strings.Builder
	sb.WriteString("┌────────────────────────────────────────┐\n")
	sb.WriteString("│          🎵 NOW PLAYING ON GoYT 🎵     │\n")
	sb.WriteString("├────────────────────────────────────────┤\n")
	
	title := m.currentTrack.Title
	artist := m.currentTrack.Artist
	
	titleLine := fmt.Sprintf("│ Title:  %-30s │\n", truncateStr(title, 30))
	artistLine := fmt.Sprintf("│ Artist: %-30s │\n", truncateStr(artist, 30))
	sb.WriteString(titleLine)
	sb.WriteString(artistLine)

	pct := 0.0
	if m.duration > 0 {
		pct = m.timePos / m.duration
	}
	barWidth := 20
	filled := int(pct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}
	if filled < 0 {
		filled = 0
	}
	unfilled := barWidth - filled
	barStr := strings.Repeat("█", filled) + strings.Repeat("░", unfilled)
	timeStr := fmt.Sprintf("%s/%s", formatTime(m.timePos), formatTime(m.duration))
	progressLine := fmt.Sprintf("│ [%s] %-13s │\n", barStr, timeStr)
	sb.WriteString(progressLine)

	sb.WriteString("├────────────────────────────────────────┤\n")
	linkStr := fmt.Sprintf("https://music.youtube.com/watch?v=%s", m.currentTrack.VideoID)
	linkLine := fmt.Sprintf("│ Link: %-32s │\n", linkStr)
	sb.WriteString(linkLine)
	sb.WriteString("└────────────────────────────────────────┘\n")
	return sb.String()
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}

func (m *Model) isAnyInputFocused() bool {
	if m.focusSide {
		return false
	}
	return (m.activeView == ViewSearch && m.searchInput.Focused()) ||
		(m.activeView == ViewPlaylistSelect && m.playlistInput.Focused())
}

