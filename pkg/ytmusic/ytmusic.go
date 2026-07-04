package ytmusic

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	youtube "github.com/wslyyy/youtube-go"
)

// Track represents a YouTube Music track.
type Track struct {
	VideoID  string `json:"videoId"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration string `json:"duration"` // format "m:ss"
}

// Client wraps the InnerTube API for YouTube Music.
type Client struct {
	it *youtube.InnerTube
}

// NewClient initializes a client configured for YouTube Music.
func NewClient(httpClient *http.Client) (*Client, error) {
	// WEB_REMIX is the InnerTube client name for YouTube Music
	it, err := youtube.NewInnerTube(httpClient, "WEB_REMIX", "1.20241022.01.00", "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36", "https://music.youtube.com/", nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create InnerTube client: %w", err)
	}
	return &Client{it: it}, nil
}

// Search searches for tracks based on a query, returning tracks and the next continuation token if available.
func (c *Client) Search(query string) ([]Track, string, error) {
	params := "EgWKAQIIAWoMEA4QChADEAQQCRAF"
	results, err := c.it.Search(&query, &params, nil)
	if err != nil {
		return nil, "", err
	}

	return parseTracks(results), findContinuationToken(results), nil
}

// SearchNextPage fetches the next page of search results using a continuation token.
func (c *Client) SearchNextPage(token string) ([]Track, string, error) {
	results, err := c.it.Search(nil, nil, &token)
	if err != nil {
		return nil, "", err
	}

	return parseTracks(results), findContinuationToken(results), nil
}

func findContinuationToken(data map[string]interface{}) string {
	var token string
	var collect func(val interface{})
	collect = func(val interface{}) {
		if token != "" {
			return
		}
		switch v := val.(type) {
		case map[string]interface{}:
			// Check nextContinuationData
			if nextCont, exists := v["nextContinuationData"]; exists {
				if nextContMap, ok := nextCont.(map[string]interface{}); ok {
					if tok, ok := nextContMap["continuation"].(string); ok {
						token = tok
						return
					}
				}
			}
			// Check continuationItemRenderer
			if contItem, exists := v["continuationItemRenderer"]; exists {
				if contItemMap, ok := contItem.(map[string]interface{}); ok {
					if contEndpoint, exists := contItemMap["continuationEndpoint"]; exists {
						if contEndpointMap, ok := contEndpoint.(map[string]interface{}); ok {
							if command, exists := contEndpointMap["continuationCommand"]; exists {
								if commandMap, ok := command.(map[string]interface{}); ok {
									if tok, ok := commandMap["token"].(string); ok {
										token = tok
										return
									}
								}
							}
						}
					}
				}
			}
			for _, valItem := range v {
				collect(valItem)
			}
		case []interface{}:
			for _, valItem := range v {
				collect(valItem)
			}
		}
	}
	collect(data)
	return token
}

// GetSuggestions retrieves suggestions (up next / radio) for a video.
func (c *Client) GetSuggestions(videoID string) ([]Track, error) {
	// Next endpoint returns recommendations and playlist info
	results, err := c.it.Next(&videoID, nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}

	return parseTracks(results), nil
}

var durationRegexp = regexp.MustCompile(`^\d+:\d+(:\d+)?$`)

// parseTracks recursively finds musicResponsiveListItemRenderer and extracts Track structs.
func parseTracks(data map[string]interface{}) []Track {
	var tracks []Track
	var collect func(val interface{})

	collect = func(val interface{}) {
		switch v := val.(type) {
		case map[string]interface{}:
			if item, exists := v["musicResponsiveListItemRenderer"]; exists {
				if track, ok := extractTrack(item); ok {
					// Avoid duplicates
					duplicate := false
					for _, t := range tracks {
						if t.VideoID == track.VideoID {
							duplicate = true
							break
						}
					}
					if !duplicate {
						tracks = append(tracks, track)
					}
				}
			}
			for _, valItem := range v {
				collect(valItem)
			}
		case []interface{}:
			for _, valItem := range v {
				collect(valItem)
			}
		}
	}

	collect(data)
	return tracks
}

func extractTrack(v interface{}) (Track, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return Track{}, false
	}

	// 1. Extract VideoID
	videoID := ""
	if playlistItemData, ok := m["playlistItemData"].(map[string]interface{}); ok {
		if vid, ok := playlistItemData["videoId"].(string); ok {
			videoID = vid
		}
	}

	// Fallback to overlay play button endpoint
	if videoID == "" {
		if overlay, ok := m["overlay"].(map[string]interface{}); ok {
			if thumbOverlay, ok := overlay["musicItemThumbnailOverlayRenderer"].(map[string]interface{}); ok {
				if content, ok := thumbOverlay["content"].(map[string]interface{}); ok {
					if playBtn, ok := content["musicPlayButtonRenderer"].(map[string]interface{}); ok {
						if endpoint, ok := playBtn["playNavigationEndpoint"].(map[string]interface{}); ok {
							if watch, ok := endpoint["watchEndpoint"].(map[string]interface{}); ok {
								if vid, ok := watch["videoId"].(string); ok {
									videoID = vid
								}
							}
						}
					}
				}
			}
		}
	}

	if videoID == "" {
		return Track{}, false
	}

	// 2. Extract Title and metadata columns
	flexColumns, ok := m["flexColumns"].([]interface{})
	if !ok || len(flexColumns) == 0 {
		return Track{}, false
	}

	title := ""
	artist := "Unknown Artist"
	duration := "0:00"

	// Col 0: Title info
	if col0, ok := flexColumns[0].(map[string]interface{}); ok {
		if r, ok := col0["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
			if text, ok := r["text"].(map[string]interface{}); ok {
				title = getRunsText(text["runs"])
			}
		}
	}

	if title == "" {
		return Track{}, false
	}

	// Col 1: Metadata runs (Artist • Album/Views • Duration)
	if len(flexColumns) > 1 {
		if col1, ok := flexColumns[1].(map[string]interface{}); ok {
			if r, ok := col1["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
				if text, ok := r["text"].(map[string]interface{}); ok {
					runs, _ := text["runs"].([]interface{})
					
					var textParts []string
					for _, item := range runs {
						if runMap, ok := item.(map[string]interface{}); ok {
							if txt, ok := runMap["text"].(string); ok {
								trimmed := strings.TrimSpace(txt)
								if trimmed != "" && trimmed != "•" {
									textParts = append(textParts, trimmed)
								}
							}
						}
					}

					if len(textParts) > 0 {
						artist = textParts[0]
					}
					
					// Find duration by searching for time pattern at the end
					for i := len(textParts) - 1; i >= 0; i-- {
						if durationRegexp.MatchString(textParts[i]) {
							duration = textParts[i]
							break
						}
					}
				}
			}
		}
	}

	return Track{
		VideoID:  videoID,
		Title:    title,
		Artist:   artist,
		Duration: duration,
	}, true
}

func getRunsText(runsObj interface{}) string {
	runs, ok := runsObj.([]interface{})
	if !ok {
		return ""
	}
	var sb strings.Builder
	for _, run := range runs {
		if rMap, ok := run.(map[string]interface{}); ok {
			if txt, ok := rMap["text"].(string); ok {
				sb.WriteString(txt)
			}
		}
	}
	return sb.String()
}

// Playlist represents a YouTube Music playlist.
type Playlist struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count string `json:"count"` // e.g. "50 songs"
}

// GetLibraryPlaylists fetches the authenticated user's library playlists.
func (c *Client) GetLibraryPlaylists() ([]Playlist, error) {
	browseID := "FEmusic_liked_playlists"
	results, err := c.it.Browse(&browseID, nil, nil)
	if err != nil {
		return nil, err
	}
	


	if hasSignInMessage(results) {
		return nil, fmt.Errorf("session unauthenticated; please update your Cookie in config.json")
	}
	return parsePlaylists(results), nil
}

func hasSignInMessage(data map[string]interface{}) bool {
	var found bool
	var collect func(val interface{})
	collect = func(val interface{}) {
		if found {
			return
		}
		switch v := val.(type) {
		case map[string]interface{}:
			if item, exists := v["messageRenderer"]; exists {
				if itemMap, ok := item.(map[string]interface{}); ok {
					if btn, ok := itemMap["button"].(map[string]interface{}); ok {
						if btnRen, ok := btn["buttonRenderer"].(map[string]interface{}); ok {
							if nav, ok := btnRen["navigationEndpoint"].(map[string]interface{}); ok {
								if _, hasSignIn := nav["signInEndpoint"]; hasSignIn {
									found = true
									return
								}
							}
						}
					}
				}
			}
			for _, valItem := range v {
				collect(valItem)
			}
		case []interface{}:
			for _, valItem := range v {
				collect(valItem)
			}
		}
	}
	collect(data)
	return found
}

// GetPlaylistTracks fetches the tracks inside a playlist.
func (c *Client) GetPlaylistTracks(playlistID string) ([]Track, error) {
	browseID := playlistID
	if !strings.HasPrefix(browseID, "VL") {
		browseID = "VL" + browseID
	}
	results, err := c.it.Browse(&browseID, nil, nil)
	if err != nil {
		return nil, err
	}
	return parseTracks(results), nil
}

// parsePlaylists extracts gridPlaylistRenderer, musicResponsiveListItemRenderer and musicTwoRowItemRenderer items representing playlists.
func parsePlaylists(data map[string]interface{}) []Playlist {
	var playlists []Playlist
	var collect func(val interface{})

	collect = func(val interface{}) {
		switch v := val.(type) {
		case map[string]interface{}:
			if item, exists := v["musicResponsiveListItemRenderer"]; exists {
				if pl, ok := extractPlaylist(item); ok {
					playlists = append(playlists, pl)
				}
			}
			if item, exists := v["gridPlaylistRenderer"]; exists {
				if pl, ok := extractGridPlaylist(item); ok {
					playlists = append(playlists, pl)
				}
			}
			if item, exists := v["musicTwoRowItemRenderer"]; exists {
				if pl, ok := extractTwoRowPlaylist(item); ok {
					playlists = append(playlists, pl)
				}
			}
			for _, valItem := range v {
				collect(valItem)
			}
		case []interface{}:
			for _, valItem := range v {
				collect(valItem)
			}
		}
	}

	collect(data)
	return playlists
}

func extractTwoRowPlaylist(v interface{}) (Playlist, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return Playlist{}, false
	}

	playlistID := ""
	title := ""

	// Extract Title & playlistID
	if titleObj, ok := m["title"].(map[string]interface{}); ok {
		if runs, ok := titleObj["runs"].([]interface{}); ok && len(runs) > 0 {
			if runMap, ok := runs[0].(map[string]interface{}); ok {
				title, _ = runMap["text"].(string)
				if nav, ok := runMap["navigationEndpoint"].(map[string]interface{}); ok {
					if browse, ok := nav["browseEndpoint"].(map[string]interface{}); ok {
						if bid, ok := browse["browseId"].(string); ok {
							playlistID = bid
						}
					}
				}
			}
		}
	}

	// Fallback playlistID from navigationEndpoint in the root
	if playlistID == "" {
		if nav, ok := m["navigationEndpoint"].(map[string]interface{}); ok {
			if browse, ok := nav["browseEndpoint"].(map[string]interface{}); ok {
				if bid, ok := browse["browseId"].(string); ok {
					playlistID = bid
				}
			}
		}
	}

	if playlistID == "" || title == "" {
		return Playlist{}, false
	}

	// Subtitle as Count (e.g. "Playlist • 15 songs")
	count := "Playlist"
	if subtitle, ok := m["subtitle"].(map[string]interface{}); ok {
		count = getRunsText(subtitle["runs"])
	}

	return Playlist{
		ID:    playlistID,
		Title: title,
		Count: count,
	}, true
}

func extractPlaylist(v interface{}) (Playlist, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return Playlist{}, false
	}

	playlistID := ""
	flexColumns, _ := m["flexColumns"].([]interface{})
	for _, col := range flexColumns {
		if colMap, ok := col.(map[string]interface{}); ok {
			if r, ok := colMap["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
				if text, ok := r["text"].(map[string]interface{}); ok {
					if runs, ok := text["runs"].([]interface{}); ok {
						for _, run := range runs {
							if runMap, ok := run.(map[string]interface{}); ok {
								if endpoint, ok := runMap["navigationEndpoint"].(map[string]interface{}); ok {
									if browse, ok := endpoint["browseEndpoint"].(map[string]interface{}); ok {
										if bid, ok := browse["browseId"].(string); ok && (strings.HasPrefix(bid, "VL") || strings.HasPrefix(bid, "PL")) {
											playlistID = bid
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Fallback to playlistItemData or other fields
	if playlistID == "" {
		if playlistItemData, ok := m["playlistItemData"].(map[string]interface{}); ok {
			if pid, ok := playlistItemData["playlistId"].(string); ok {
				playlistID = pid
			}
		}
	}

	title := ""
	if len(flexColumns) > 0 {
		if col0, ok := flexColumns[0].(map[string]interface{}); ok {
			if r, ok := col0["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
				if text, ok := r["text"].(map[string]interface{}); ok {
					title = getRunsText(text["runs"])
				}
			}
		}
	}

	if playlistID == "" || title == "" {
		return Playlist{}, false
	}

	count := "Tracks"
	if len(flexColumns) > 1 {
		if col1, ok := flexColumns[1].(map[string]interface{}); ok {
			if r, ok := col1["musicResponsiveListItemFlexColumnRenderer"].(map[string]interface{}); ok {
				if text, ok := r["text"].(map[string]interface{}); ok {
					count = getRunsText(text["runs"])
				}
			}
		}
	}

	return Playlist{
		ID:    playlistID,
		Title: title,
		Count: count,
	}, true
}

func extractGridPlaylist(v interface{}) (Playlist, bool) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return Playlist{}, false
	}

	playlistID := ""
	if nav, ok := m["navigationEndpoint"].(map[string]interface{}); ok {
		if browse, ok := nav["browseEndpoint"].(map[string]interface{}); ok {
			if bid, ok := browse["browseId"].(string); ok {
				playlistID = bid
			}
		}
	}

	title := ""
	if titleObj, ok := m["title"].(map[string]interface{}); ok {
		title = getRunsText(titleObj["runs"])
	}

	if playlistID == "" || title == "" {
		return Playlist{}, false
	}

	count := "Playlist"
	if subtitle, ok := m["shortBylineText"].(map[string]interface{}); ok {
		count = getRunsText(subtitle["runs"])
	}

	return Playlist{
		ID:    playlistID,
		Title: title,
		Count: count,
	}, true
}

// CreatePlaylist creates a new private playlist on YouTube Music and returns its ID.
func (c *Client) CreatePlaylist(title, description string) (string, error) {
	body := map[string]interface{}{
		"title":         title,
		"description":   description,
		"privacyStatus": "PRIVATE",
	}
	resp, err := c.it.Call("PLAYLIST/CREATE", nil, body)
	if err != nil {
		return "", err
	}
	if playlistID, ok := resp["playlistId"].(string); ok {
		return playlistID, nil
	}
	return "", fmt.Errorf("playlistId not found in response")
}

// AddTrackToPlaylist adds a video/track to an existing playlist.
func (c *Client) AddTrackToPlaylist(playlistID, videoID string) error {
	body := map[string]interface{}{
		"playlistId": playlistID,
		"actions": []interface{}{
			map[string]interface{}{
				"action":       "ACTION_ADD_VIDEO",
				"addedVideoId": videoID,
			},
		},
	}
	_, err := c.it.Call("BROWSE/EDIT_PLAYLIST", nil, body)
	return err
}

// DeletePlaylist deletes a playlist by its ID on YouTube Music.
// Library browseIds use a "VL" prefix (e.g. "VLPL...") but the
// playlist/delete endpoint expects the raw ID (e.g. "PL...").
func (c *Client) DeletePlaylist(playlistID string) error {
	id := strings.TrimPrefix(playlistID, "VL")
	body := map[string]interface{}{
		"playlistId": id,
	}
	_, err := c.it.Call("PLAYLIST/DELETE", nil, body)
	return err
}
