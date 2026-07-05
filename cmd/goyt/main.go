package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"flag"

	"goyt/internal/adapter/catalog/ytmusic"
	configJson "goyt/internal/adapter/config/json"
	"goyt/internal/adapter/mcp"
	"goyt/internal/adapter/player/mpv"
	"goyt/internal/adapter/tui"
	"goyt/internal/domain/model"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)


type authenticatedRoundTripper struct {
	cookie string
	next   http.RoundTripper
}

func (art *authenticatedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if art.cookie != "" {
		req.Header.Set("Cookie", art.cookie)

		// Enforce Same-Origin for InnerTube requests to send cookies
		if req.URL.Host == "youtubei.googleapis.com" {
			req.URL.Host = "music.youtube.com"
		}
		req.Host = "music.youtube.com"
		req.Header.Set("Host", "music.youtube.com")

		// Ensure API key is present in the query string
		q := req.URL.Query()
		if q.Get("key") == "" {
			q.Set("key", "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30")
			req.URL.RawQuery = q.Encode()
		}

		// Extract SAPISID/__Secure-3PAPISID to compute SAPISIDHASH
		sapisid := extractCookieValue(art.cookie, "__Secure-3PAPISID")
		if sapisid == "" {
			sapisid = extractCookieValue(art.cookie, "SAPISID")
		}
		if sapisid != "" {
			req.Header.Set("Authorization", getSAPISIDHash(sapisid, "https://music.youtube.com"))
		}

		// Set Google account index header and YouTube client ID
		req.Header.Set("X-Goog-AuthUser", "0")
		req.Header.Set("X-Youtube-Client-Name", "67")
		req.Header.Set("Origin", "https://music.youtube.com")
	}

	return art.next.RoundTrip(req)
}

func extractCookieValue(cookieStr, name string) string {
	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if strings.HasPrefix(trimmed, name+"=") {
			return strings.TrimPrefix(trimmed, name+"=")
		}
	}
	return ""
}

func getSAPISIDHash(sapisid string, origin string) string {
	now := time.Now().Unix()
	input := fmt.Sprintf("%d %s %s", now, sapisid, origin)
	h := sha1.New()
	h.Write([]byte(input))
	sha1Sum := fmt.Sprintf("%x", h.Sum(nil))
	return fmt.Sprintf("SAPISIDHASH %d_%s", now, sha1Sum)
}

func main() {
	// Define version flags
	versionFlag := flag.Bool("v", false, "print version information")
	flag.BoolVar(versionFlag, "version", false, "print version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("GoYT Version: %s\n", version)
		fmt.Printf("Git Commit:   %s\n", commit)
		fmt.Printf("Build Date:   %s\n", date)
		os.Exit(0)
	}

	if len(os.Args) > 1 && os.Args[1] == "now" {
		client := &http.Client{Timeout: time.Second}
		resp, err := client.Get("http://localhost:8080/now-playing")
		if err != nil {
			fmt.Println("GoYT player is not currently running.")
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Println("GoYT player returned error status.")
			os.Exit(1)
		}

		type PlaybackInfo struct {
			CurrentTrack struct {
				Title   string `json:"title"`
				Artist  string `json:"artist"`
				VideoID string `json:"video_id"`
			} `json:"CurrentTrack"`
			IsPlaying bool    `json:"IsPlaying"`
			Duration  float64 `json:"Duration"`
			TimePos   float64 `json:"TimePos"`
			Volume    int     `json:"Volume"`
		}

		var info PlaybackInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			fmt.Printf("Error reading player response: %v\n", err)
			os.Exit(1)
		}

		if info.CurrentTrack.VideoID == "" {
			fmt.Println("No track loaded.")
			os.Exit(0)
		}

		pct := 0.0
		if info.Duration > 0 {
			pct = info.TimePos / info.Duration
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

		formatTime := func(seconds float64) string {
			s := int(seconds)
			min := s / 60
			sec := s % 60
			return fmt.Sprintf("%d:%02d", min, sec)
		}

		truncateStr := func(s string, maxLen int) string {
			runes := []rune(s)
			if len(runes) > maxLen {
				return string(runes[:maxLen-3]) + "..."
			}
			return s
		}

		fmt.Println("┌────────────────────────────────────────┐")
		fmt.Println("│          🎵 NOW PLAYING ON GoYT 🎵     │")
		fmt.Println("├────────────────────────────────────────┤")
		fmt.Printf("│ Title:  %-30s │\n", truncateStr(info.CurrentTrack.Title, 30))
		fmt.Printf("│ Artist: %-30s │\n", truncateStr(info.CurrentTrack.Artist, 30))
		fmt.Printf("│ [%s] %-13s │\n", barStr, fmt.Sprintf("%s/%s", formatTime(info.TimePos), formatTime(info.Duration)))
		fmt.Println("├────────────────────────────────────────┤")
		fmt.Printf("│ Link: %-32s │\n", fmt.Sprintf("https://music.youtube.com/watch?v=%s", info.CurrentTrack.VideoID))
		fmt.Println("└────────────────────────────────────────┘")
		os.Exit(0)
	}

	// Configure logging to a file to prevent stdout/stderr corruption of the TUI
	logFile, err := os.OpenFile("goyt.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// 0. Load Configuration
	configAdapter, err := configJson.NewJsonConfigAdapter()
	if err != nil {
		fmt.Printf("Error initializing config adapter: %v\n", err)
		os.Exit(1)
	}

	cookie, err := configAdapter.LoadCookie()
	if err != nil {
		fmt.Printf("Warning: failed to load cookie: %v\n", err)
	}

	theme, err := configAdapter.LoadTheme()
	if err != nil {
		fmt.Printf("Warning: failed to load theme: %v\n", err)
	}

	// 1. Initialize Player
	p := mpv.NewMpvPlayerAdapter()
	if err := p.Start(); err != nil {
		fmt.Printf("Error starting mpv: %v\n", err)
		fmt.Println("Please make sure 'mpv' and 'yt-dlp' are installed on your system.")
		os.Exit(1)
	}
	defer p.Stop()

	// 2. Setup Authenticated HTTP Client
	var httpClient *http.Client
	if cookie != "" {
		httpClient = &http.Client{
			Transport: &authenticatedRoundTripper{
				cookie: cookie,
				next:   http.DefaultTransport,
			},
		}
	}

	// 3. Initialize YouTube Music Client
	client, err := ytmusic.NewYtMusicCatalogAdapter(httpClient)
	if err != nil {
		fmt.Printf("Error starting YouTube Music catalog client: %v\n", err)
		os.Exit(1)
	}

	// 4. Initialize Queue
	q := model.NewQueue()

	// 5. Initialize Bubble Tea UI
	mcpEnabled := &atomic.Bool{}
	mcpEnabled.Store(true) // default value is on

	notificationsEnabled, err := configAdapter.LoadNotificationsEnabled()
	if err != nil {
		notificationsEnabled = true
	}

	modelTui := tui.NewModel(client, p, q, theme, mcpEnabled, notificationsEnabled)
	program := tea.NewProgram(modelTui, tea.WithAltScreen())

	// Start MCP SSE Server in background (port 8080)
	mcpServer := mcp.NewServer(client, program, 8080, mcpEnabled, version)
	go func() {
		if err := mcpServer.Start(); err != nil {
			log.Printf("MCP Server error: %v", err)
		}
	}()

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
