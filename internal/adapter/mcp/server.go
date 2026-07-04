package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"goyt/internal/adapter/tui"
	"goyt/internal/domain/model"
	"goyt/internal/domain/port"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	catalog    port.MusicCatalogPort
	program    *tea.Program
	port       int
	enabled    *atomic.Bool
	activeConn int32
}

func NewServer(catalog port.MusicCatalogPort, program *tea.Program, portNum int, enabled *atomic.Bool) *Server {
	return &Server{
		catalog: catalog,
		program: program,
		port:    portNum,
		enabled: enabled,
	}
}

type trackingHandler struct {
	next            http.Handler
	enabled         *atomic.Bool
	program         *tea.Program
	activeConnCount *int32
}

func (h *trackingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/sse" {
		if r.Method == http.MethodGet {
			if !h.enabled.Load() {
				http.Error(w, "MCP server is disabled", http.StatusServiceUnavailable)
				return
			}
			count := atomic.AddInt32(h.activeConnCount, 1)
			h.program.Send(tui.MCPConnectionsMsg{Count: int(count)})
			defer func() {
				count := atomic.AddInt32(h.activeConnCount, -1)
				h.program.Send(tui.MCPConnectionsMsg{Count: int(count)})
			}()
		} else if r.Method == http.MethodPost {
			if !h.enabled.Load() {
				http.Error(w, "MCP server is disabled", http.StatusServiceUnavailable)
				return
			}
		}
	}

	h.next.ServeHTTP(w, r)
}

func (s *Server) Start() error {
	mcpServer := server.NewMCPServer("goyt-mcp", "1.0.0")

	// 1. Search Music Tool
	searchTool := mcp.NewTool("search_music",
		mcp.WithDescription("Search for tracks/songs on YouTube Music. Displays results in TUI."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query (e.g. song name, artist)"),
		),
	)
	mcpServer.AddTool(searchTool, s.handleSearch)

	// 2. List Playlists Tool
	listPlaylistsTool := mcp.NewTool("list_playlists",
		mcp.WithDescription("List liked playlists in the user's library. Displays playlists in TUI."),
	)
	mcpServer.AddTool(listPlaylistsTool, s.handleListPlaylists)

	// 3. Get Playlist Tracks Tool
	getPlaylistTracksTool := mcp.NewTool("get_playlist_tracks",
		mcp.WithDescription("Get tracks from a specific playlist. Displays details in TUI."),
		mcp.WithString("playlist_id",
			mcp.Required(),
			mcp.Description("The ID of the playlist (e.g. PL... or VL...)"),
		),
	)
	mcpServer.AddTool(getPlaylistTracksTool, s.handleGetPlaylistTracks)

	// 4. Add Track to Play Queue Tool
	addTrackToQueueTool := mcp.NewTool("add_track_to_queue",
		mcp.WithDescription("Add a specific track to the play queue. Displays queue in TUI and triggers playback if needed."),
		mcp.WithString("video_id",
			mcp.Required(),
			mcp.Description("The video ID of the track"),
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("The title of the track"),
		),
		mcp.WithString("artist",
			mcp.Required(),
			mcp.Description("The artist name"),
		),
		mcp.WithString("duration",
			mcp.Description("Duration of the track in format 'm:ss'"),
		),
	)
	mcpServer.AddTool(addTrackToQueueTool, s.handleEnqueueTrack)

	// 5. Add Playlist to Play Queue Tool
	addPlaylistToQueueTool := mcp.NewTool("add_playlist_to_queue",
		mcp.WithDescription("Fetch all tracks from a playlist and add them to the play queue. Displays queue in TUI and triggers playback if needed."),
		mcp.WithString("playlist_id",
			mcp.Required(),
			mcp.Description("The ID of the playlist"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the playlist to display in status message"),
		),
	)
	mcpServer.AddTool(addPlaylistToQueueTool, s.handleEnqueuePlaylist)

	// 6. Play Pause Playback Tool
	playPauseTool := mcp.NewTool("play_pause",
		mcp.WithDescription("Pause, resume, or toggle music playback."),
		mcp.WithString("action",
			mcp.Description("Action to take ('play', 'pause', 'toggle'). Defaults to 'toggle'."),
		),
	)
	mcpServer.AddTool(playPauseTool, s.handlePlayPause)

	// 7. Get Current Playback Info Tool
	getPlaybackInfoTool := mcp.NewTool("get_playback_info",
		mcp.WithDescription("Get information about the currently playing track, status, volume, and progress."),
	)
	mcpServer.AddTool(getPlaybackInfoTool, s.handleGetPlaybackInfo)

	// 8. Create Playlist Tool
	createPlaylistTool := mcp.NewTool("create_playlist",
		mcp.WithDescription("Create a new private playlist in the library."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the playlist to create"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description for the playlist"),
		),
	)
	mcpServer.AddTool(createPlaylistTool, s.handleCreatePlaylist)

	// 9. Add Track to Playlist Tool
	addTrackToPlaylistTool := mcp.NewTool("add_track_to_playlist",
		mcp.WithDescription("Add a specific song/track to a new or existing playlist."),
		mcp.WithString("playlist_id",
			mcp.Required(),
			mcp.Description("The ID of the playlist (e.g. PL... or VL...)"),
		),
		mcp.WithString("video_id",
			mcp.Required(),
			mcp.Description("The YouTube video ID of the track to add"),
		),
	)
	mcpServer.AddTool(addTrackToPlaylistTool, s.handleAddTrackToPlaylist)

	// Create and start the Streamable HTTP server
	httpServer := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/sse"),
	)

	wrappedHandler := &trackingHandler{
		next:            httpServer,
		enabled:         s.enabled,
		program:         s.program,
		activeConnCount: &s.activeConn,
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: wrappedHandler,
	}

	log.Printf("Starting MCP Streamable HTTP server on http://localhost:%d/sse", s.port)
	return srv.ListenAndServe()
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tracks, token, err := s.catalog.Search(query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search: %v", err)), nil
	}

	// Fetch another page if continuation token is present to return 40 tracks total, matching TUI behavior
	if token != "" {
		nextTracks, nextToken, err := s.catalog.SearchNextPage(token)
		if err == nil {
			tracks = append(tracks, nextTracks...)
			token = nextToken
		}
	}

	// Dispatch message to Bubble Tea TUI
	s.program.Send(tui.MCPSearchMsg{
		Query:        query,
		Tracks:       tracks,
		Continuation: token,
	})

	// Format response for LLM
	var resultText string
	if len(tracks) == 0 {
		resultText = fmt.Sprintf("No tracks found for search query: %q", query)
	} else {
		resultText = fmt.Sprintf("Found %d tracks for query %q. TUI search view updated.\n\nTracks list:\n", len(tracks), query)
		for _, t := range tracks {
			resultText += fmt.Sprintf("- Title: %s, Artist: %s, VideoID: %s, Duration: %s\n", t.Title, t.Artist, t.VideoID, t.Duration)
		}
	}
	return mcp.NewToolResultText(resultText), nil
}

func (s *Server) handleListPlaylists(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playlists, err := s.catalog.GetLibraryPlaylists()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch playlists: %v", err)), nil
	}

	// Dispatch message to Bubble Tea TUI
	s.program.Send(tui.MCPShowPlaylistsMsg{
		Playlists: playlists,
	})

	var resultText string
	if len(playlists) == 0 {
		resultText = "No playlists found in the user's library."
	} else {
		resultText = fmt.Sprintf("Found %d playlists. TUI playlist view updated.\n\nPlaylists list:\n", len(playlists))
		for _, p := range playlists {
			resultText += fmt.Sprintf("- Title: %s, ID: %s, Count: %s\n", p.Title, p.ID, p.Count)
		}
	}
	return mcp.NewToolResultText(resultText), nil
}

func (s *Server) handleGetPlaylistTracks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playlistID, err := request.RequireString("playlist_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tracks, err := s.catalog.GetPlaylistTracks(playlistID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch playlist tracks: %v", err)), nil
	}

	playlistName := "Playlist " + playlistID
	// Dispatch message to Bubble Tea TUI
	s.program.Send(tui.MCPShowPlaylistDetailMsg{
		PlaylistName: playlistName,
		Tracks:       tracks,
	})

	var resultText string
	if len(tracks) == 0 {
		resultText = fmt.Sprintf("Playlist %s is empty.", playlistID)
	} else {
		resultText = fmt.Sprintf("Found %d tracks in playlist %s. TUI playlist detail view updated.\n\nTracks list:\n", len(tracks), playlistID)
		for _, t := range tracks {
			resultText += fmt.Sprintf("- Title: %s, Artist: %s, VideoID: %s, Duration: %s\n", t.Title, t.Artist, t.VideoID, t.Duration)
		}
	}
	return mcp.NewToolResultText(resultText), nil
}

func (s *Server) handleEnqueueTrack(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	videoID, err := request.RequireString("video_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	title, err := request.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	artist, err := request.RequireString("artist")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	duration := "0:00"
	if argsMap, ok := request.Params.Arguments.(map[string]any); ok {
		if durVal, exists := argsMap["duration"]; exists {
			if durStr, ok := durVal.(string); ok {
				duration = durStr
			}
		}
	}

	track := model.Track{
		VideoID:  videoID,
		Title:    title,
		Artist:   artist,
		Duration: duration,
	}

	// Dispatch message to Bubble Tea TUI
	s.program.Send(tui.MCPEnqueueTrackMsg{
		Track: track,
	})

	resultText := fmt.Sprintf("Successfully added track %s - %s (ID: %s) to play queue. TUI queue view updated.", artist, title, videoID)
	return mcp.NewToolResultText(resultText), nil
}

func (s *Server) handleEnqueuePlaylist(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playlistID, err := request.RequireString("playlist_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tracks, err := s.catalog.GetPlaylistTracks(playlistID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch playlist tracks: %v", err)), nil
	}

	if len(tracks) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Playlist %q is empty, cannot add to queue.", name)), nil
	}

	// Dispatch message to Bubble Tea TUI
	s.program.Send(tui.MCPEnqueuePlaylistMsg{
		PlaylistName: name,
		Tracks:       tracks,
	})

	resultText := fmt.Sprintf("Successfully fetched %d tracks from playlist %q (ID: %s) and added them to the play queue. TUI queue view updated.", len(tracks), name, playlistID)
	return mcp.NewToolResultText(resultText), nil
}

func (s *Server) handlePlayPause(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	action := "toggle"
	if argsMap, ok := request.Params.Arguments.(map[string]any); ok {
		if actVal, exists := argsMap["action"]; exists {
			if actStr, ok := actVal.(string); ok {
				action = actStr
			}
		}
	}

	s.program.Send(tui.MCPPlayPauseMsg{
		Action: action,
	})

	return mcp.NewToolResultText(fmt.Sprintf("Sent play/pause command (%s) to TUI.", action)), nil
}

func (s *Server) handleGetPlaybackInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ch := make(chan tui.PlaybackInfo, 1)
	s.program.Send(tui.MCPGetPlaybackInfoMsg{ResponseChan: ch})

	select {
	case info := <-ch:
		var text string
		if info.CurrentTrack.VideoID == "" {
			text = "No track is currently loaded."
		} else {
			status := "paused"
			if info.IsPlaying {
				status = "playing"
			}
			text = fmt.Sprintf("Current Track: %s - %s\nStatus: %s\nPosition: %.0fs / %.0fs\nVolume: %d%%\nVideoID: %s",
				info.CurrentTrack.Artist, info.CurrentTrack.Title, status, info.TimePos, info.Duration, info.Volume, info.CurrentTrack.VideoID)
		}
		return mcp.NewToolResultText(text), nil
	case <-time.After(1 * time.Second):
		return mcp.NewToolResultError("Timed out getting playback info from TUI"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Server) handleCreatePlaylist(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	description := ""
	if argsMap, ok := request.Params.Arguments.(map[string]any); ok {
		if descVal, exists := argsMap["description"]; exists {
			if descStr, ok := descVal.(string); ok {
				description = descStr
			}
		}
	}

	playlistID, err := s.catalog.CreatePlaylist(name, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create playlist: %v", err)), nil
	}

	// Request TUI to reload playlists cache
	s.program.Send(tui.MCPRefreshPlaylistsMsg{})

	return mcp.NewToolResultText(fmt.Sprintf("Successfully created private playlist %q (ID: %s).", name, playlistID)), nil
}

func (s *Server) handleAddTrackToPlaylist(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	playlistID, err := request.RequireString("playlist_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	videoID, err := request.RequireString("video_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	err = s.catalog.AddTrackToPlaylist(playlistID, videoID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add track to playlist: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully added track (ID: %s) to playlist (ID: %s).", videoID, playlistID)), nil
}
