package ytmusic

import (
	"net/http"

	"goyt/internal/domain/model"
	"goyt/pkg/ytmusic"
)

type YtMusicCatalogAdapter struct {
	client *ytmusic.Client
}

func NewYtMusicCatalogAdapter(httpClient *http.Client) (*YtMusicCatalogAdapter, error) {
	c, err := ytmusic.NewClient(httpClient)
	if err != nil {
		return nil, err
	}
	return &YtMusicCatalogAdapter{client: c}, nil
}

func toDomainTrack(t ytmusic.Track) model.Track {
	return model.Track{
		VideoID:  t.VideoID,
		Title:    t.Title,
		Artist:   t.Artist,
		Duration: t.Duration,
	}
}

func toDomainTracks(ts []ytmusic.Track) []model.Track {
	res := make([]model.Track, len(ts))
	for i, t := range ts {
		res[i] = toDomainTrack(t)
	}
	return res
}

func toDomainPlaylist(p ytmusic.Playlist) model.Playlist {
	return model.Playlist{
		ID:    p.ID,
		Title: p.Title,
		Count: p.Count,
	}
}

func toDomainPlaylists(ps []ytmusic.Playlist) []model.Playlist {
	res := make([]model.Playlist, len(ps))
	for i, p := range ps {
		res[i] = toDomainPlaylist(p)
	}
	return res
}

func (a *YtMusicCatalogAdapter) Search(query string) ([]model.Track, string, error) {
	tracks, token, err := a.client.Search(query)
	if err != nil {
		return nil, "", err
	}
	return toDomainTracks(tracks), token, nil
}

func (a *YtMusicCatalogAdapter) SearchNextPage(token string) ([]model.Track, string, error) {
	tracks, nextToken, err := a.client.SearchNextPage(token)
	if err != nil {
		return nil, "", err
	}
	return toDomainTracks(tracks), nextToken, nil
}

func (a *YtMusicCatalogAdapter) GetLibraryPlaylists() ([]model.Playlist, error) {
	playlists, err := a.client.GetLibraryPlaylists()
	if err != nil {
		return nil, err
	}
	return toDomainPlaylists(playlists), nil
}

func (a *YtMusicCatalogAdapter) GetPlaylistTracks(playlistID string) ([]model.Track, error) {
	tracks, err := a.client.GetPlaylistTracks(playlistID)
	if err != nil {
		return nil, err
	}
	return toDomainTracks(tracks), nil
}

func (a *YtMusicCatalogAdapter) CreatePlaylist(name string, description string) (string, error) {
	return a.client.CreatePlaylist(name, description)
}

func (a *YtMusicCatalogAdapter) AddTrackToPlaylist(playlistID string, videoID string) error {
	return a.client.AddTrackToPlaylist(playlistID, videoID)
}

func (a *YtMusicCatalogAdapter) DeletePlaylist(playlistID string) error {
	return a.client.DeletePlaylist(playlistID)
}
