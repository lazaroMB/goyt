package cache

import (
	"context"
	"fmt"
	"goyt/internal/domain/model"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LocalCacheManager struct {
	cacheDir       string
	maxSize        int64 // in bytes
	cachedTracks   map[string]string
	downloading    map[string]bool
	mu             sync.RWMutex
	onComplete     func(videoID string)
}

func NewLocalCacheManager(onComplete func(videoID string)) *LocalCacheManager {
	return &LocalCacheManager{
		maxSize:      500 * 1024 * 1024, // 500 MB default
		cachedTracks: make(map[string]string),
		downloading:  make(map[string]bool),
		onComplete:   onComplete,
	}
}

func (c *LocalCacheManager) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	userCache, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get user cache dir: %w", err)
	}

	c.cacheDir = filepath.Join(userCache, "goyt")
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Verify yt-dlp is in PATH
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return fmt.Errorf("yt-dlp not found in PATH, caching disabled: %w", err)
	}

	return c.scanCacheDir()
}

// scanCacheDir builds the initial file map. Must be called under lock.
func (c *LocalCacheManager) scanCacheDir() error {
	files, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	c.cachedTracks = make(map[string]string)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if strings.HasSuffix(name, ".part") || strings.HasSuffix(name, ".ytdl") || strings.HasSuffix(name, ".temp") {
			continue
		}

		ext := filepath.Ext(name)
		videoID := strings.TrimSuffix(name, ext)
		// YouTube video IDs are typically 11 characters
		if len(videoID) == 11 {
			c.cachedTracks[videoID] = filepath.Join(c.cacheDir, name)
		}
	}
	return nil
}

func (c *LocalCacheManager) IsCached(videoID string) (bool, string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path, ok := c.cachedTracks[videoID]
	return ok, path
}

func (c *LocalCacheManager) PreBuffer(track model.Track) {
	c.mu.Lock()
	if _, ok := c.cachedTracks[track.VideoID]; ok {
		c.mu.Unlock()
		return
	}
	if c.downloading[track.VideoID] {
		c.mu.Unlock()
		return
	}
	c.downloading[track.VideoID] = true
	c.mu.Unlock()

	// Start asynchronous pre-buffering
	go func() {
		defer func() {
			c.mu.Lock()
			delete(c.downloading, track.VideoID)
			c.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		outputPath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.%%(ext)s", track.VideoID))
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", track.VideoID)

		// Execute yt-dlp to download raw audio
		cmd := exec.CommandContext(ctx, "yt-dlp",
			"-f", "bestaudio",
			"--no-playlist",
			"-o", outputPath,
			videoURL,
		)

		if err := cmd.Run(); err != nil {
			// Download failed
			return
		}

		// Find the resulting file name (since it has dynamic extension)
		c.mu.Lock()
		defer c.mu.Unlock()

		files, err := os.ReadDir(c.cacheDir)
		if err != nil {
			return
		}

		var foundPath string
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if strings.HasPrefix(name, track.VideoID) &&
				!strings.HasSuffix(name, ".part") &&
				!strings.HasSuffix(name, ".ytdl") {
				foundPath = filepath.Join(c.cacheDir, name)
				break
			}
		}

		if foundPath != "" {
			c.cachedTracks[track.VideoID] = foundPath
			c.pruneCacheLRU()
			if c.onComplete != nil {
				c.onComplete(track.VideoID)
			}
		}
	}()
}

// pruneCacheLRU deletes oldest files if size exceeds limit. Must be called under lock.
func (c *LocalCacheManager) pruneCacheLRU() {
	type fileInfo struct {
		path    string
		size    int64
		modTime time.Time
		videoID string
	}

	var files []fileInfo
	var totalSize int64

	for videoID, path := range c.cachedTracks {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		totalSize += stat.Size()
		files = append(files, fileInfo{
			path:    path,
			size:    stat.Size(),
			modTime: stat.ModTime(),
			videoID: videoID,
		})
	}

	if totalSize <= c.maxSize {
		return
	}

	// Sort by modTime ascending (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	targetSize := int64(float64(c.maxSize) * 0.8) // Prune down to 80% of max size
	for _, f := range files {
		if totalSize <= targetSize {
			break
		}
		if err := os.Remove(f.path); err == nil {
			totalSize -= f.size
			delete(c.cachedTracks, f.videoID)
		}
	}
}

func (c *LocalCacheManager) ClearAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		_ = os.Remove(filepath.Join(c.cacheDir, f.Name()))
	}

	c.cachedTracks = make(map[string]string)
	c.downloading = make(map[string]bool)
	return nil
}

func (c *LocalCacheManager) SetOnComplete(onComplete func(videoID string)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onComplete = onComplete
}

