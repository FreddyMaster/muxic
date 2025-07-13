package player

import (
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/speaker"
	"log"
	"muxic/internal/player/components"
	"muxic/internal/util"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type queueUpdatedMsg struct{}

type playlistUpdatedMsg struct{}

type searchResultMsg struct {
	tracks []*util.AudioFile
}

// AddToQueueCmd adds the currently selected track to the queue
func AddToQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		// Get the library instance
		library := components.GetLibrary()
		if library == nil {
			err := errors.New("library is nil")
			log.Println(err)
			return err
		}

		track := library.Files[m.ActiveFileIndex]

		m.Queue.Add(track)
		m.UpdateQueueTable()

		// Play the track if it's the first in the queue
		if m.Queue.Length() == 1 {
			if err := m.AudioPlayer.Play(track); err != nil {
				log.Printf("Error playing track: %v", err)
			}
		}
		return queueUpdatedMsg{}
	}
}

// RemoveFromQueueCmd removes the currently selected track from the queue
func RemoveFromQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.viewMode != ViewQueue {
			log.Println("not in queue mode")
			return nil
		}
		index := m.ActiveFileIndex

		m.Queue.Remove(index)

		// Update cursor position
		if index >= m.Queue.Length() {
			m.ActiveFileIndex = m.Queue.Length()
		}

		// Use the existing UpdateCursorPosition function
		if err := UpdateCursorPosition(m); err != nil {
			log.Printf("Error updating cursor position: %v", err)
			return err
		}

		m.UpdateQueueTable()

		return queueUpdatedMsg{}
	}
}

// PlayNextInQueueCmd plays the next track in the queue
func PlayNextInQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.Queue.Next()
		m.UpdateQueueTable()
		return queueUpdatedMsg{}
	}
}

// PlayPreviousInQueueCmd plays the previous track in the queue
func PlayPreviousInQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.Queue.Previous()
		m.UpdateQueueTable()
		return queueUpdatedMsg{}
	}
}

func ClearQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.Queue.Clear()
		m.UpdateQueueTable()
		return queueUpdatedMsg{}
	}
}

func ViewQueueCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		m.viewMode = ViewQueue
		return queueUpdatedMsg{}
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
		// Return playlistUpdatedMsg to indicate success and update the view
		return playlistUpdatedMsg{}
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
		library := components.GetLibrary()
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
		m.UpdatePlaylistTable()

		// Return playlistUpdatedMsg to indicate success and update the view
		return playlistUpdatedMsg{}
	}
}

// RemoveFromPlaylistCmd removes the selected track from the active playlist
func RemoveFromPlaylistCmd(m *Model) tea.Cmd {
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

		// Store the current cursor position
		oldCursor := m.ActiveFileIndex

		// Add the track to the active playlist
		err := m.PlaylistManager.RemoveTrack(m.PlaylistManager.ActivePlaylist.ID, m.ActiveFileIndex)

		if err != nil {
			log.Printf("Failed to removetrack to playlist: %v", err)
			return err
		}

		// Refresh the playlist view if we're in playlist view
		m.UpdatePlaylistTable()

		// Update cursor position
		if oldCursor >= m.PlaylistManager.ActivePlaylist.Length() {
			m.ActiveFileIndex = m.PlaylistManager.ActivePlaylist.Length() - 1
		}

		// Use the existing UpdateCursorPosition function
		if err := UpdateCursorPosition(m); err != nil {
			log.Printf("Error updating cursor position: %v", err)
			return err
		}

		// Return playlistUpdatedMsg to indicate success and update the view
		return playlistUpdatedMsg{}
	}
}

// PlayCmd toggles between play and pause for the current track
func PlayCmd(m *Model) tea.Cmd {
	filePath := m.Queue.Current()
	if filePath == nil {
		return func() tea.Msg {
			return errors.New("no track selected")
		}
	}

	return nil
}

// PauseCmd pauses the current playback
func PauseCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.AudioPlayer.Ctrl != nil {
			m.AudioPlayer.Ctrl.Paused = true
			m.AudioPlayer.Playing = false
		} else {
			return errors.New("no active playback to pause")

		}
		return nil
	}
}

// StopCmd stops the current playback
func StopCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		speaker.Clear()
		m.AudioPlayer.Playing = false
		if m.AudioPlayer.CurrentStreamer != nil {
			err := m.AudioPlayer.CurrentStreamer.Close()
			if err != nil {
				return err

			}
			m.AudioPlayer.CurrentStreamer = nil
		}
		m.AudioPlayer.PlayedTime = 0
		m.AudioPlayer.SamplesPlayed = 0
		m.Progress.SetPercent(0)
		return nil
	}
}

// SkipForwardCmd Skips 10 seconds forward
func SkipForwardCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.AudioPlayer.CurrentStreamer != nil {
			speaker.Lock()
			newPos := m.AudioPlayer.CurrentStreamer.Position() + 10*int(m.AudioPlayer.SampleRate)
			if newPos > m.AudioPlayer.CurrentStreamer.Len() {
				// If the track is longer than 10 seconds, seek to the end
				newPos = m.AudioPlayer.CurrentStreamer.Len()
			}
			if err := m.AudioPlayer.CurrentStreamer.Seek(newPos); err != nil {
				return err
			}
			m.AudioPlayer.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.AudioPlayer.SampleRate)
			m.AudioPlayer.SamplesPlayed = newPos
			speaker.Unlock()
		}
		return nil
	}
}

// SkipBackwardCmd Skip 10 seconds backward
func SkipBackwardCmd(m *Model) tea.Cmd {
	return func() tea.Msg {

		if m.AudioPlayer.CurrentStreamer != nil {
			speaker.Lock()
			newPos := m.AudioPlayer.CurrentStreamer.Position() - 10*int(m.AudioPlayer.SampleRate)
			if newPos < 0 {
				// If the track is shorter than 10 seconds, seek to the beginning
				newPos = 0
			}
			if err := m.AudioPlayer.CurrentStreamer.Seek(newPos); err != nil {
				return err

			}
			m.AudioPlayer.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.AudioPlayer.SampleRate)
			m.AudioPlayer.SamplesPlayed = newPos
			speaker.Unlock()
		}
		return nil
	}
}

func NextTrackCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

func PreviousTrackCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

// VolumeUpCmd Increases the volume
func VolumeUpCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.AudioPlayer.Volume != nil {
			m.AudioPlayer.SetVolume(m.AudioPlayer.Volume.Volume + 0.1)
		} else {
			return errors.New("volume is nil")

		}
		return nil
	}
}

// VolumeDownCmd Decreases the volume
func VolumeDownCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.AudioPlayer.Volume != nil {
			m.AudioPlayer.SetVolume(m.AudioPlayer.Volume.Volume - 0.1)
		} else {
			return errors.New("volume is nil")
		}
		return nil
	}
}

// VolumeMuteCmd Mutes the volume
func VolumeMuteCmd(m *Model) tea.Cmd {
	return func() tea.Msg {
		if m.AudioPlayer.Volume != nil {
			m.AudioPlayer.Volume.Silent = !m.AudioPlayer.Volume.Silent
		}
		return nil
	}
}

// SearchCmd performs a search based on the provided query.
func SearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
		// If the query is empty, return no results.
		if query == "" {
			return searchResultMsg{tracks: []*util.AudioFile{}}
		}

		library := components.GetLibrary()
		var filteredTracks []*util.AudioFile

		for _, track := range library.Files {
			// Simple case-insensitive search in title, artist, and album
			if strings.Contains(strings.ToLower(track.Title), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(track.Artist), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(track.Album), strings.ToLower(query)) {
				filteredTracks = append(filteredTracks, track)
			}
		}

		return searchResultMsg{tracks: filteredTracks}
	}
}

// LoadLibraryCmd scans the music library in the background and returns a message
// when it's complete.
func LoadLibraryCmd() tea.Cmd {
	return func() tea.Msg {
		// Get the user's music directory.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Failed to get home directory: %v", err)
			// Return an error message or an empty list
			return LibraryLoadedMsg{Tracks: []*util.AudioFile{}}
		}
		musicDir := filepath.Join(homeDir, "Music")

		// Scan for audio files.
		tracks, err := util.GetAudioFiles(musicDir)
		if err != nil {
			log.Printf("Failed to scan audio files: %v", err)
			// Return an error message or an empty list
			return LibraryLoadedMsg{Tracks: []*util.AudioFile{}}
		}

		return LibraryLoadedMsg{Tracks: tracks}
	}
}

// UpdateCursorPosition updates the cursor position for the current view
func UpdateCursorPosition(m *Model) error {
	if m == nil {
		return errors.New("model is nil")
	}

	var (
		currentTable *table.Model
		listLength   int
	)

	// Determine which list we're working with based on the current view
	switch m.viewMode {
	case ViewLibrary:
		// For library view, use the library table
		currentTable = &m.LibraryTable
		listLength = len(components.GetLibrary().Files)

	case ViewPlaylistTracks, ViewPlaylists:
		// For playlists, use the appropriate playlist table
		if m.ActivePlaylistIndex < 0 || m.ActivePlaylistIndex >= len(m.PlaylistTable) {
			return errors.New("invalid playlist index")
		}
		currentTable = &m.PlaylistTable[m.ActivePlaylistIndex]
		if m.viewMode == ViewPlaylistTracks && m.PlaylistManager.ActivePlaylist != nil {
			listLength = len(m.PlaylistManager.ActivePlaylist.Tracks)
		} else {
			listLength = len(m.PlaylistManager.Playlists)
		}

	case ViewQueue:
		// For queue view, use the queue table
		currentTable = &m.QueueTable
		listLength = m.Queue.Length()

	default:
		return fmt.Errorf("unsupported view mode: %v", m.viewMode)
	}

	// Ensure we have a valid table
	if currentTable == nil {
		return errors.New("current table is nil")
	}

	// Get current cursor position
	currentCursor := currentTable.Cursor()

	// Adjust cursor if it's out of bounds
	if listLength == 0 {
		currentCursor = 0
	} else if currentCursor >= listLength {
		currentCursor = listLength - 1
	} else if currentCursor < 0 {
		currentCursor = 0
	}

	// Update the cursor position
	currentTable.SetCursor(currentCursor)
	m.ActiveFileIndex = currentCursor

	return nil
}
