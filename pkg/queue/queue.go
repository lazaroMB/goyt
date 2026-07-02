package queue

import (
	"math/rand"
	"time"

	"goyt/pkg/ytmusic"
)

// Queue manages the playlist queue and playback history.
type Queue struct {
	tracks       []ytmusic.Track
	currentIndex int
	rng          *rand.Rand
}

// NewQueue creates a new empty queue.
func NewQueue() *Queue {
	return &Queue{
		tracks:       make([]ytmusic.Track, 0),
		currentIndex: -1,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Add adds tracks to the end of the queue.
func (q *Queue) Add(tracks ...ytmusic.Track) {
	q.tracks = append(q.tracks, tracks...)
	if q.currentIndex == -1 && len(q.tracks) > 0 {
		q.currentIndex = 0
	}
}

// AddNext inserts a track immediately after the current index.
func (q *Queue) AddNext(t ytmusic.Track) {
	if len(q.tracks) == 0 {
		q.tracks = append(q.tracks, t)
		q.currentIndex = 0
		return
	}

	insertIdx := q.currentIndex + 1
	q.tracks = append(q.tracks[:insertIdx], append([]ytmusic.Track{t}, q.tracks[insertIdx:]...)...)
}

// Clear clears the queue.
func (q *Queue) Clear() {
	q.tracks = make([]ytmusic.Track, 0)
	q.currentIndex = -1
}

// Current returns the current track and whether it exists.
func (q *Queue) Current() (ytmusic.Track, bool) {
	if q.currentIndex < 0 || q.currentIndex >= len(q.tracks) {
		return ytmusic.Track{}, false
	}
	return q.tracks[q.currentIndex], true
}

// Next moves to the next track and returns it.
func (q *Queue) Next() (ytmusic.Track, bool) {
	if len(q.tracks) == 0 {
		return ytmusic.Track{}, false
	}
	if q.currentIndex < len(q.tracks)-1 {
		q.currentIndex++
		return q.tracks[q.currentIndex], true
	}
	return ytmusic.Track{}, false
}

// Prev moves to the previous track and returns it.
func (q *Queue) Prev() (ytmusic.Track, bool) {
	if len(q.tracks) == 0 {
		return ytmusic.Track{}, false
	}
	if q.currentIndex > 0 {
		q.currentIndex--
		return q.tracks[q.currentIndex], true
	}
	return ytmusic.Track{}, false
}

// Shuffle shuffles the remaining items in the queue.
func (q *Queue) Shuffle() {
	if len(q.tracks) <= 1 {
		return
	}

	// Shuffle everything after the current index
	start := q.currentIndex + 1
	if start >= len(q.tracks) {
		return
	}

	subSlice := q.tracks[start:]
	q.rng.Shuffle(len(subSlice), func(i, j int) {
		subSlice[i], subSlice[j] = subSlice[j], subSlice[i]
	})
}

// List returns the current queue slice.
func (q *Queue) List() []ytmusic.Track {
	return q.tracks
}

// CurrentIndex returns the current track index.
func (q *Queue) CurrentIndex() int {
	return q.currentIndex
}

// SetIndex sets the current playback index manually.
func (q *Queue) SetIndex(idx int) bool {
	if idx >= 0 && idx < len(q.tracks) {
		q.currentIndex = idx
		return true
	}
	return false
}
