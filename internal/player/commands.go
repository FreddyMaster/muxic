package player

import (
	"errors"
	"github.com/charmbracelet/bubbles/table"
	"log"
	"muxic/internal/player/components"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/util"
	"path/filepath"
	"time"
)

// AddToQueueCmd adds the currently selected track to the queue
func AddToQueueCmd(m *Model) tea.Cmd {
	filePath := m.GetCurrentFilePath()
	if filePath == "" {
		return func() tea.Msg { return errors.New("no track selected") }
	}

	// Extract metadata from the audio file
	title, artist, album, duration := util.ReadAudioMetadata(filePath, filepath.Base(filePath))

	// Create a Track with the current file path and add it to the queue
	track := util.AudioFile{
		Path:     filePath,
		Title:    title,
		Artist:   artist,
		Album:    album,
		Duration: duration,
		FileName: filepath.Base(filePath),
	}

	m.Queue.Add(track)
	return nil
}

// PlayNextInQueueCmd plays the next track in the queue
func PlayNextInQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.Queue.Next()
		return nil
	}
}

// PlayPreviousInQueueCmd plays the previous track in the queue
func PlayPreviousInQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.Queue.Previous()
		return nil
	}
}

func ViewQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.viewMode = ViewQueue
		return nil
	}
}

// CreatePlaylistCmd creates a new playlist
func CreatePlaylistCmd(m *Model, name string) tea.Cmd {
	return func() tea.Msg {
		if m.PlaylistManager == nil {
			m.PlaylistManager = components.NewPlaylistManager()
		}

		playlist, err := m.PlaylistManager.CreatePlaylist(name)
		err = m.PlaylistManager.SetActivePlaylist(playlist.ID)
		if err != nil {
			log.Printf("Error setting active playlist: %v", err)
			return err
		}
		// Return nil to indicate success
		return nil
	}
}

// AddToPlaylistCmd adds the currently selected track to the active playlist
func AddToPlaylistCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		// return error if no track is selected
		if m.ActiveFileIndex < 0 {
			err := errors.New("no track selected")
			log.Println(err)
			return err
		}

		// return error if playlist manager is nil
		if m.PlaylistManager == nil {
			m.PlaylistManager = components.NewPlaylistManager()
		}

		// Get the library instance
		library := util.GetLibrary()
		if library == nil {
			err := errors.New("library is nil")
			log.Println(err)
			return err
		}

		// Ensure there's an active playlist
		if m.PlaylistManager.ActivePlaylist == nil {
			// Create a default playlist if none exists
			playlist, err := m.PlaylistManager.CreatePlaylist("My Playlist")
			if err != nil {
				log.Printf("Failed to create default playlist: %v", err)
				return err
			}
			m.PlaylistManager.ActivePlaylist = playlist
		}

		// Get the selected track
		track := library.Files[m.ActiveFileIndex]

		// Add the track to the active playlist
		err := m.PlaylistManager.AddTracks(m.PlaylistManager.ActivePlaylist.ID, track)
		if err != nil {
			log.Printf("Failed to add track to playlist: %v", err)
			return err
		}

		// Refresh the playlist view if we're in playlist view
		if m.viewMode == ViewPlaylistTracks || m.viewMode == ViewPlaylists {
			// Convert tracks to table rows
			var rows []table.Row
			for i, t := range m.PlaylistManager.ActivePlaylist.Tracks {
				rows = append(rows, table.Row{
					strconv.Itoa(i + 1),
					t.Title,
					t.Artist,
					t.Album,
					t.Duration,
				})
			}

			// Update the playlist table
			m.PlaylistTable[m.ActivePlaylistIndex].SetRows(rows)
		}

		// Return success
		return nil
	}
}

// RemoveFromPlaylistCmd removes the selected track from the active playlist
func RemoveFromPlaylistCmd(m *Model) tea.Cmd {
	// return error if playlist is nil
	if m.PlaylistManager == nil {
		return func() tea.Msg {
			return errors.New("playlist manager is nil")
		}
	}

	// return error if not in playlist view
	if m.viewMode != ViewPlaylists {
		return func() tea.Msg {
			return errors.New("not in playlist view")
		}
	}

	// return error if active playlist index is out of range
	if m.ActiveFileIndex < 0 {
		return func() tea.Msg {
			return errors.New("invalid selection")
		}
	}

	return nil
}

// PlayCmd toggles between play and pause for the current track
func PlayCmd(m *Model) tea.Cmd {
	filePath := m.GetCurrentFilePath()
	if filePath == "" {
		return func() tea.Msg {
			return errors.New("no file selected")
		}
	}

	return playTrack(m, filePath)
}

// PauseCmd pauses the current playback
func PauseCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.Ctrl != nil {
		m.AudioPlayer.Ctrl.Paused = true
		m.AudioPlayer.Playing = false
	} else {
		return func() tea.Msg {
			return errors.New("no active playback to pause")
		}
	}
	return nil
}

// StopCmd stops the current playback
func StopCmd(m *Model) tea.Cmd {
	speaker.Clear()
	m.AudioPlayer.Playing = false
	if m.AudioPlayer.CurrentStreamer != nil {
		err := m.AudioPlayer.CurrentStreamer.Close()
		if err != nil {
			return func() tea.Msg {
				return err
			}
		}
		m.AudioPlayer.CurrentStreamer = nil
	}
	m.AudioPlayer.PlayedTime = 0
	m.AudioPlayer.SamplesPlayed = 0
	m.Progress.SetPercent(0)
	return nil
}

// SkipForwardCmd Skips 10 seconds forward
func SkipForwardCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.CurrentStreamer != nil {
		speaker.Lock()
		newPos := m.AudioPlayer.CurrentStreamer.Position() + 10*int(m.AudioPlayer.SampleRate)
		if newPos > m.AudioPlayer.CurrentStreamer.Len() {
			// If the track is longer than 10 seconds, seek to the end
			newPos = m.AudioPlayer.CurrentStreamer.Len()
		}
		if err := m.AudioPlayer.CurrentStreamer.Seek(newPos); err != nil {
			return func() tea.Msg {
				return err
			}
		}
		m.AudioPlayer.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.AudioPlayer.SampleRate)
		m.AudioPlayer.SamplesPlayed = newPos
		speaker.Unlock()
	}
	return nil
}

// SkipBackwardCmd Skip 10 seconds backward
func SkipBackwardCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.CurrentStreamer != nil {
		speaker.Lock()
		newPos := m.AudioPlayer.CurrentStreamer.Position() - 10*int(m.AudioPlayer.SampleRate)
		if newPos < 0 {
			// If the track is shorter than 10 seconds, seek to the beginning
			newPos = 0
		}
		if err := m.AudioPlayer.CurrentStreamer.Seek(newPos); err != nil {
			return func() tea.Msg {
				return err
			}
		}
		m.AudioPlayer.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.AudioPlayer.SampleRate)
		m.AudioPlayer.SamplesPlayed = newPos
		speaker.Unlock()
	}
	return nil
}

func NextTrackCmd(m *Model) tea.Cmd {
	return nil
}

func PreviousTrackCmd(m *Model) tea.Cmd {
	return nil
}

// VolumeUpCmd Increases the volume
func VolumeUpCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.Volume != nil {
		m.AudioPlayer.SetVolume(m.AudioPlayer.Volume.Volume + 0.1)
	} else {
		return func() tea.Msg {
			return errors.New("volume is nil")
		}
	}
	return nil
}

// VolumeDownCmd Decreases the volume
func VolumeDownCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.Volume != nil {
		m.AudioPlayer.SetVolume(m.AudioPlayer.Volume.Volume - 0.1)
	} else {
		return func() tea.Msg {
			return errors.New("volume is nil")
		}
	}
	return nil
}

// VolumeMuteCmd Mutes the volume
func VolumeMuteCmd(m *Model) tea.Cmd {
	if m.AudioPlayer.Volume != nil {
		m.AudioPlayer.Volume.Silent = !m.AudioPlayer.Volume.Silent
	}
	return nil
}

func playTrack(m *Model, filePath string) tea.Cmd {
	// Stop current playback
	m.AudioPlayer.Stop()

	// Open and set up the new track
	streamer, format, totalSamples, err := util.OpenAudioFile(filePath)
	if err != nil {
		return func() tea.Msg { return err }
	}

	// Update AudioPlayer state
	m.AudioPlayer.CurrentStreamer = streamer
	m.AudioPlayer.SampleRate = format.SampleRate
	m.AudioPlayer.TotalSamples = totalSamples
	m.AudioPlayer.SamplesPlayed = 0
	m.AudioPlayer.PlayedTime = 0
	m.AudioPlayer.TotalTime = time.Duration(totalSamples) * time.Second / time.Duration(format.SampleRate)
	m.AudioPlayer.PlayedTime = 0
	m.AudioPlayer.TotalTime = time.Duration(totalSamples) * time.Second / time.Duration(format.SampleRate)

	// Create a reference to the streamer to prevent garbage collection
	wrappedStreamer := beep.Seq(streamer, beep.Callback(func() {
		// This callback will be called when the streamer finishes
		m.AudioPlayer.Playing = false
	}))

	// Wrap the streamer to track progress
	progressStreamer := beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = wrappedStreamer.Stream(samples)
		m.AudioPlayer.SamplesPlayed += n
		m.AudioPlayer.PlayedTime = time.Duration(m.AudioPlayer.SamplesPlayed) * time.Second /
			time.Duration(m.AudioPlayer.SampleRate)
		return
	})

	// Set up volume and control, reusing existing volume settings if they exist
	currentVolume := 0.0 // Default volume if not set
	if m.AudioPlayer.Volume != nil {
		currentVolume = m.AudioPlayer.Volume.Volume
	}

	m.AudioPlayer.Volume = &effects.Volume{
		Streamer: progressStreamer,
		Base:     2,             // Exponential scale base
		Volume:   currentVolume, // Use the current volume setting
		Silent:   false,
	}
	m.AudioPlayer.Ctrl = &beep.Ctrl{Streamer: m.AudioPlayer.Volume}

	// Start playback
	speaker.Play(m.AudioPlayer.Ctrl)
	m.AudioPlayer.Playing = true

	return nil
}

// UpdateCursorPosition updates the cursor position in the playlist view, ensuring it stays within valid bounds.
// Returns an error if the model is nil, playlist is nil, or if the active playlist index is out of range.
func UpdateCursorPosition(m *Model) error {
	// Check if model is nil
	if m == nil {
		return errors.New("model is nil")
	}
	// Check if playlist exists
	if m.PlaylistManager == nil {
		return errors.New("playlist is nil")
	}

	// Validate active playlist index
	if m.ActivePlaylistIndex < 0 || m.ActivePlaylistIndex >= len(m.PlaylistTable) {
		return errors.New("invalid playlist index")
	}

	// Calculate new cursor position
	newCursor := m.ActiveFileIndex

	if newCursor < 0 {
		newCursor = 0
	}

	// Update the cursor position
	m.ActiveFileIndex = newCursor
	m.PlaylistTable[m.ActivePlaylistIndex].SetCursor(newCursor)

	return nil
}
