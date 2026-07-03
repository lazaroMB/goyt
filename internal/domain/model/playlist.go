package model

// Playlist represents a YouTube Music playlist.
type Playlist struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count string `json:"count"` // e.g. "50 songs"
}
