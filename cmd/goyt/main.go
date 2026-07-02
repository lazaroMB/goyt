package main

import (
	"crypto/sha1"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"goyt/pkg/config"
	"goyt/pkg/player"
	"goyt/pkg/queue"
	"goyt/pkg/tui"
	"goyt/pkg/ytmusic"

	tea "github.com/charmbracelet/bubbletea"
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
	// 0. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: failed to load config: %v\n", err)
		cfg = &config.Config{}
	}

	// 1. Initialize Player
	p := player.NewPlayer()
	if err := p.Start(); err != nil {
		fmt.Printf("Error starting mpv: %v\n", err)
		fmt.Println("Please make sure 'mpv' and 'yt-dlp' are installed on your system.")
		os.Exit(1)
	}
	defer p.Stop()

	// 2. Setup Authenticated HTTP Client
	var httpClient *http.Client
	if cfg.Cookie != "" {
		httpClient = &http.Client{
			Transport: &authenticatedRoundTripper{
				cookie: cfg.Cookie,
				next:   http.DefaultTransport,
			},
		}
	}

	// 3. Initialize YouTube Music Client
	client, err := ytmusic.NewClient(httpClient)
	if err != nil {
		fmt.Printf("Error starting YouTube Music client: %v\n", err)
		os.Exit(1)
	}

	// 4. Initialize Queue
	q := queue.NewQueue()

	// 5. Initialize Bubble Tea UI
	model := tui.NewModel(client, p, q)
	program := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := program.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
