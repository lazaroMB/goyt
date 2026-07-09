package port

import "goyt/internal/domain/model"

type CacheManager interface {
	// Start initializes the cache manager (scans directory, builds file map)
	Start() error
	// IsCached returns true and the absolute file path if the video is fully cached
	IsCached(videoID string) (bool, string)
	// PreBuffer starts background download of the next track's raw audio
	PreBuffer(track model.Track)
	// ClearAll deletes all cached files
	ClearAll() error
	// SetOnComplete registers a callback for when a track download completes
	SetOnComplete(onComplete func(videoID string))
}
