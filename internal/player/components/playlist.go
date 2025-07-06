package components

import (
	"errors"
	"fmt"
	"math/rand"
	"muxic/internal/util"
	"sort"
)

// Playlist represents a collection of audio tracks
type Playlist struct {
	ID     int               `json:"id"`
	Name   string            `json:"name"`
	Tracks []*util.AudioFile `json:"tracks"`
}

// PlaylistManager handles multiple playlists and their state
type PlaylistManager struct {
	Playlists      []*Playlist `json:"playlists"`
	ActivePlaylist *Playlist   `json:"-"`
	ActiveTrackIdx int         `json:"active_track_idx"`
	lastID         int         `json:"-"`
}

// NewPlaylistManager creates a new playlist manager
func NewPlaylistManager() *PlaylistManager {
	return &PlaylistManager{
		Playlists: make([]*Playlist, 0),
		lastID:    0,
	}
}

// CreatePlaylist creates a new playlist with the given name
func (pm *PlaylistManager) CreatePlaylist(name string) (*Playlist, error) {
	if name == "" {
		return nil, errors.New("playlist name cannot be empty")
	}

	pm.lastID++
	playlist := &Playlist{
		ID:     pm.lastID,
		Name:   name,
		Tracks: make([]*util.AudioFile, 0),
	}
	pm.Playlists = append(pm.Playlists, playlist)
	return playlist, nil
}

// DeletePlaylist removes a playlist by ID
func (pm *PlaylistManager) DeletePlaylist(id int) error {
	for i, p := range pm.Playlists {
		if p.ID == id {
			// If we're deleting the active playlist, clear the reference
			if pm.ActivePlaylist != nil && pm.ActivePlaylist.ID == id {
				pm.ActivePlaylist = nil
				pm.ActiveTrackIdx = 0
			}
			pm.Playlists = append(pm.Playlists[:i], pm.Playlists[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("playlist with ID %d not found", id)
}

// GetPlaylist returns a playlist by ID
func (pm *PlaylistManager) GetPlaylist(id int) (*Playlist, error) {
	for _, p := range pm.Playlists {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("playlist with ID %d not found", id)
}

// SetActivePlaylist sets the currently active playlist
func (pm *PlaylistManager) SetActivePlaylist(id int) error {
	playlist, err := pm.GetPlaylist(id)
	if err != nil {
		return err
	}
	pm.ActivePlaylist = playlist
	pm.ActiveTrackIdx = 0
	return nil
}

// AddTracks adds one or more tracks to a playlist
func (pm *PlaylistManager) AddTracks(playlistID int, tracks ...*util.AudioFile) error {
	playlist, err := pm.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	playlist.Tracks = append(playlist.Tracks, tracks...)
	return nil
}

// RemoveTrack removes a track from a playlist by index
func (pm *PlaylistManager) RemoveTrack(playlistID int, trackIndex int) error {
	playlist, err := pm.GetPlaylist(playlistID)
	if err != nil {
		return err
	}
	if trackIndex < 0 || trackIndex >= len(playlist.Tracks) {
		return errors.New("track index out of range")
	}
	playlist.Tracks = append(playlist.Tracks[:trackIndex], playlist.Tracks[trackIndex+1:]...)
	return nil
}

// NextTrack moves to the next track in the active playlist
func (pm *PlaylistManager) NextTrack() (*util.AudioFile, error) {
	if pm.ActivePlaylist == nil {
		return nil, errors.New("no active playlist")
	}
	if len(pm.ActivePlaylist.Tracks) == 0 {
		return nil, errors.New("playlist is empty")
	}
	pm.ActiveTrackIdx = (pm.ActiveTrackIdx + 1) % len(pm.ActivePlaylist.Tracks)
	return pm.ActivePlaylist.Tracks[pm.ActiveTrackIdx], nil
}

// PreviousTrack moves to the previous track in the active playlist
func (pm *PlaylistManager) PreviousTrack() (*util.AudioFile, error) {
	if pm.ActivePlaylist == nil {
		return nil, errors.New("no active playlist")
	}
	if len(pm.ActivePlaylist.Tracks) == 0 {
		return nil, errors.New("playlist is empty")
	}
	pm.ActiveTrackIdx--
	if pm.ActiveTrackIdx < 0 {
		pm.ActiveTrackIdx = len(pm.ActivePlaylist.Tracks) - 1
	}
	return pm.ActivePlaylist.Tracks[pm.ActiveTrackIdx], nil
}

// GetCurrentTrack returns the currently selected track in the active playlist
func (pm *PlaylistManager) GetCurrentTrack() (*util.AudioFile, error) {
	if pm.ActivePlaylist == nil {
		return nil, errors.New("no active playlist")
	}
	if len(pm.ActivePlaylist.Tracks) == 0 {
		return nil, errors.New("playlist is empty")
	}
	return pm.ActivePlaylist.Tracks[pm.ActiveTrackIdx], nil
}

// ShufflePlaylist randomizes the order of tracks in a playlist
func (pm *PlaylistManager) ShufflePlaylist(playlistID int) error {
	playlist, err := pm.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	// Store the current track to maintain its position
	var currentTrack *util.AudioFile
	if pm.ActivePlaylist != nil && pm.ActivePlaylist.ID == playlistID && len(playlist.Tracks) > 0 {
		currentTrack = playlist.Tracks[pm.ActiveTrackIdx]
	}

	// Shuffle all tracks
	shuffled := make([]*util.AudioFile, len(playlist.Tracks))
	perm := rand.Perm(len(playlist.Tracks))
	for i, v := range perm {
		shuffled[v] = playlist.Tracks[i]
	}
	playlist.Tracks = shuffled

	// Restore current track position if possible
	if currentTrack != nil {
		for i, track := range playlist.Tracks {
			if track == currentTrack {
				pm.ActiveTrackIdx = i
				break
			}
		}
	}

	return nil
}

// SortPlaylist sorts the tracks in a playlist by a given field
func (pm *PlaylistManager) SortPlaylist(playlistID int, by string, ascending bool) error {
	playlist, err := pm.GetPlaylist(playlistID)
	if err != nil {
		return err
	}

	sort.Slice(playlist.Tracks, func(i, j int) bool {
		switch by {
		case "title":
			if ascending {
				return playlist.Tracks[i].Title < playlist.Tracks[j].Title
			}
			return playlist.Tracks[i].Title > playlist.Tracks[j].Title
		case "artist":
			if ascending {
				return playlist.Tracks[i].Artist < playlist.Tracks[j].Artist
			}
			return playlist.Tracks[i].Artist > playlist.Tracks[j].Artist
		case "album":
			if ascending {
				return playlist.Tracks[i].Album < playlist.Tracks[j].Album
			}
			return playlist.Tracks[i].Album > playlist.Tracks[j].Album
		default:
			if ascending {
				return playlist.Tracks[i].Title < playlist.Tracks[j].Title
			}
			return playlist.Tracks[i].Title > playlist.Tracks[j].Title
		}
	})

	return nil
}

func (pm *PlaylistManager) Count() []*Playlist {
	return pm.Playlists
}
