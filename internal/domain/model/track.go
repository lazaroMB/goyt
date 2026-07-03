package model

// Track represents a YouTube Music track.
type Track struct {
	VideoID  string `json:"videoId"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration string `json:"duration"` // format "m:ss"
}
