package port

import "goyt/internal/domain/model"

type MusicCatalogPort interface {
	Search(query string) ([]model.Track, string, error)
	SearchNextPage(token string) ([]model.Track, string, error)
	GetLibraryPlaylists() ([]model.Playlist, error)
	GetPlaylistTracks(playlistID string) ([]model.Track, error)
}
