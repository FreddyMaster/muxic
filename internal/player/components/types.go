package components

import (
	"errors"
	"muxic/internal/util"
	"time"
)

// PlaybackState represents the current state of audio playback
type PlaybackState int

const (
	StateStopped PlaybackState = iota
	StatePlaying
	StatePaused
)

// RepeatMode defines how playback should repeat
type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatOne
	RepeatAll
)

// ViewMode represents the different UI views
type ViewMode int

const (
	ViewLibrary ViewMode = iota
	ViewPlaylists
	ViewPlaylistTracks
	ViewQueue
	ViewSettings
)

// KeyMap defines the keyboard shortcuts for the application
type KeyMap struct {
	PlayPause     string
	Stop          string
	NextTrack     string
	PreviousTrack string
	VolumeUp      string
	VolumeDown    string
	ToggleMute    string
	ToggleRepeat  string
	ToggleShuffle string
	ToggleView    string
	Quit          string
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		PlayPause:     " ",
		Stop:          "s",
		NextTrack:     "n",
		PreviousTrack: "p",
		VolumeUp:      "+",
		VolumeDown:    "-",
		ToggleMute:    "m",
		ToggleRepeat:  "r",
		ToggleShuffle: "S",
		ToggleView:    "tab",
		Quit:          "q",
	}
}

// Config holds the application configuration
type Config struct {
	Volume         float64
	RepeatMode     RepeatMode
	Shuffle        bool
	DefaultView    ViewMode
	AutoPlay       bool
	Theme          Theme
	LibraryPath    string
	PlaylistsPath  string
	ConfigPath     string
	LastPlayedFile string
	LastPosition   time.Duration
}

// Theme defines the visual styling of the application
type Theme struct {
	PrimaryColor   string
	SecondaryColor string
	AccentColor    string
	TextColor      string
	Background     string
	BorderStyle    string
}

// PlaybackInfo contains information about the current playback
type PlaybackInfo struct {
	CurrentTrack  *util.AudioFile
	CurrentTime   time.Duration
	Duration      time.Duration
	State         PlaybackState
	Volume        float64
	IsMuted       bool
	RepeatMode    RepeatMode
	IsShuffled    bool
	QueuePosition int
	QueueLength   int
}

// Event represents a player event
type Event struct {
	Type    string
	Message string
	Time    time.Time
	Data    interface{}
}

// PlaylistInfo contains summary information about a playlist
type PlaylistInfo struct {
	ID         int
	Name       string
	TrackCount int
	Duration   time.Duration
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// LibraryStats contains statistics about the music library
type LibraryStats struct {
	TotalTracks  int
	TotalArtists int
	TotalAlbums  int
	TotalGenres  int
	TotalSize    int64
	TotalTime    time.Duration
	LastUpdated  time.Time
}

// Error types
var (
	ErrNoActivePlaylist = errors.New("no active playlist")
	ErrPlaylistEmpty    = errors.New("playlist is empty")
	ErrTrackNotFound    = errors.New("track not found")
	ErrInvalidState     = errors.New("invalid player state")
	ErrFileNotFound     = errors.New("file not found")
	ErrInvalidFormat    = errors.New("invalid audio format")
)

// Interface for player controls
type PlayerController interface {
	Play() error
	Pause() error
	Stop() error
	Next() error
	Previous() error
	Seek(pos time.Duration) error
	SetVolume(vol float64) error
	ToggleMute() error
	ToggleRepeat() error
	ToggleShuffle() error
	GetPlaybackInfo() (*PlaybackInfo, error)
}
