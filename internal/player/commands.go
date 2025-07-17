// commands.go contains all the command handlers for the music player UI.
// In the Bubble Tea architecture, commands are functions that perform I/O
// or other side effects (like making network requests, accessing hardware,
// or reading from disk). They run in a separate goroutine so as not to
// block the main UI loop.
//
// Each command function here is a "command factory": it takes the necessary
// data and returns a `tea.Cmd`, which is a `func() tea.Msg`. The Bubble Tea
// runtime executes this function and sends the returned `tea.Msg` to the
// main `Update` function for state processing.
package player

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/player/components"
	"muxic/internal/util"
)

// --- Message Definitions ---
// Messages are the primary way different parts of the application communicate.
// Commands return messages to signal completion or report results. The `Update`
// function processes these messages to change the application state.

// searchResultMsg is a message that carries the results of a library search.
type searchResultMsg struct {
	tracks []*util.AudioFile
}

// --- Queue Messages ---

// addTrackToQueueMsg is a message that signals a request to add a specific track
// to the playback queue.
type addTrackToQueueMsg struct {
	track *util.AudioFile
}

// removeTrackFromQueueMsg is a message that signals a request to remove a track
// from the playback queue at a specific index.
type removeTrackFromQueueMsg struct {
	index int
}

// nextTrackInQueueMsg signals a request to advance to the next track in the queue.
type nextTrackInQueueMsg struct{}

// previousTrackInQueueMsg signals a request to go back to the previous track in the queue.
type previousTrackInQueueMsg struct{}

// clearQueueMsg signals a request to remove all tracks from the playback queue.
type clearQueueMsg struct{}

// viewQueueMsg signals a request to switch the UI view to the queue.
type viewQueueMsg struct{}

// --- Playlist Messages ---

// playlistCreatedMsg is sent when a new playlist has been successfully created.
// It carries the newly created playlist object.
type playlistCreatedMsg struct {
	playlist *components.Playlist
}

// playlistDeletedMsg is sent when a playlist has been successfully deleted.
// It carries the ID of the deleted playlist.
type playlistDeletedMsg struct {
	id int
}

// trackAddedToPlaylistMsg is sent when a track has been successfully added to a playlist.
type trackAddedToPlaylistMsg struct {
	playlistID int
	track      *util.AudioFile
}

// trackRemovedFromPlaylistMsg is sent when a track has been successfully removed from a playlist.
type trackRemovedFromPlaylistMsg struct {
	playlistID int
	trackIndex int
}

// playlistShuffledMsg is sent when a playlist has been successfully shuffled.
type playlistShuffledMsg struct {
	playlistID int
}

// --- Player Messages ---

// pauseMsg is sent when the audio player has been successfully paused.
type pauseMsg struct{}

// stopMsg is sent when the audio player has been successfully stopped.
type stopMsg struct{}

// playbackSeekedMsg is sent after a successful seek operation (e.g., skip forward/backward).
// It carries the new stream position and the calculated human-readable time.
type playbackSeekedMsg struct {
	newPosition   int
	newPlayedTime time.Duration
}

// volumeChangedMsg is sent after the volume has been successfully changed.
// It carries the new volume level.
type volumeChangedMsg struct {
	newVolume float64
}

// --- Command Factories ---

// AddToQueueCmd creates a command that wraps a track in a message for the Update function.
// This is a "pure" command; it has no side effects itself and simply passes data.
func AddToQueueCmd(track *util.AudioFile) tea.Cmd {
	return func() tea.Msg {
		return addTrackToQueueMsg{track: track}
	}
}

// RemoveFromQueueCmd creates a command to request removing a track at a specific index.
func RemoveFromQueueCmd(index int) tea.Cmd {
	return func() tea.Msg {
		return removeTrackFromQueueMsg{index: index}
	}
}

// PlayNextInQueueCmd creates a command to request playing the next track.
func PlayNextInQueueCmd() tea.Cmd {
	return func() tea.Msg {
		return nextTrackInQueueMsg{}
	}
}

// PlayPreviousInQueueCmd creates a command to request playing the previous track.
func PlayPreviousInQueueCmd() tea.Cmd {
	return func() tea.Msg {
		return previousTrackInQueueMsg{}
	}
}

// ClearQueueCmd creates a command to request clearing the queue.
func ClearQueueCmd() tea.Cmd {
	return func() tea.Msg {
		return clearQueueMsg{}
	}
}

// ViewQueueCmd creates a command to request switching to the queue view.
func ViewQueueCmd() tea.Cmd {
	return func() tea.Msg {
		return viewQueueMsg{}
	}
}

// CreatePlaylistCmd performs the side effect of creating a new playlist using the PlaylistManager.
// It handles potential errors and returns the newly created playlist on success.
func CreatePlaylistCmd(pm *components.PlaylistManager, name string) tea.Cmd {
	return func() tea.Msg {
		if pm == nil {
			return errors.New("cannot create playlist: playlist manager is nil")
		}

		playlist, err := pm.CreatePlaylist(name)
		if err != nil {
			return err
		}

		// Also set the new playlist as active.
		if err := pm.SetActivePlaylist(playlist.ID); err != nil {
			return err
		}

		return playlistCreatedMsg{playlist: playlist}
	}
}

// AddToPlaylistCmd performs the side effect of adding a track to a specified playlist.
func AddToPlaylistCmd(pm *components.PlaylistManager, playlistID int, track *util.AudioFile) tea.Cmd {
	return func() tea.Msg {
		if pm == nil {
			return errors.New("cannot add to playlist: playlist manager is nil")
		}
		if track == nil {
			return errors.New("cannot add to playlist: track is nil")
		}

		// Perform the core operation.
		if err := pm.AddTracks(playlistID, track); err != nil {
			log.Printf("Failed to add track to playlist: %v", err)
			return err
		}

		return trackAddedToPlaylistMsg{playlistID: playlistID, track: track}
	}
}

// RemoveFromPlaylistCmd performs the side effect of removing a track from a playlist.
func RemoveFromPlaylistCmd(pm *components.PlaylistManager, playlistID int, trackIndex int) tea.Cmd {
	return func() tea.Msg {
		if pm == nil {
			return errors.New("cannot remove from playlist: playlist manager is nil")
		}

		// Pre-condition check before performing the action.
		if trackIndex < 0 || trackIndex >= pm.ActivePlaylist.Length() {
			return errors.New("cannot remove from playlist: track index out of range")
		}

		if err := pm.RemoveTrack(playlistID, trackIndex); err != nil {
			log.Printf("Failed to remove track from playlist: %v", err)
			return err
		}

		return trackRemovedFromPlaylistMsg{playlistID: playlistID, trackIndex: trackIndex}
	}
}

func ShufflePlaylistCmd(pm *components.PlaylistManager, playlistID int) tea.Cmd {
	return func() tea.Msg {
		if pm == nil {
			return errors.New("cannot shuffle playlist: playlist manager is nil")
		}

		if err := pm.ShufflePlaylist(playlistID); err != nil {
			log.Printf("Failed to shuffle playlist: %v", err)
			return err
		}

		return playlistShuffledMsg{playlistID: playlistID}
	}
}

// PauseCmd performs the side effect of pausing playback via the audio speaker.
// It locks the speaker to ensure thread safety.
func PauseCmd(player *components.AudioPlayer) tea.Cmd {
	return func() tea.Msg {
		if player.Ctrl == nil {
			return errors.New("no active playback to pause")
		}
		speaker.Lock()
		player.Ctrl.Paused = true
		speaker.Unlock()

		return pauseMsg{}
	}
}

// StopCmd performs the side effects of clearing the speaker and closing the audio stream.
func StopCmd(player *components.AudioPlayer) tea.Cmd {
	return func() tea.Msg {
		speaker.Clear()

		if player.CurrentStreamer != nil {
			// Closing the stream is the core I/O operation that can fail.
			if err := player.CurrentStreamer.Close(); err != nil {
				log.Printf("Error closing audio streamer: %v", err)
				return err
			}
		}

		return stopMsg{}
	}
}

// SkipForwardCmd performs the side effect of seeking the audio stream forward by 10 seconds.
// It calculates the new position and returns the result in a message.
func SkipForwardCmd(player *components.AudioPlayer) tea.Cmd {
	return func() tea.Msg {
		if player.CurrentStreamer == nil {
			return errors.New("no active playback to skip forward")
		}

		// Defer unlock to ensure it's always called, even if an error occurs.
		speaker.Lock()
		defer speaker.Unlock()

		currentPos := player.CurrentStreamer.Position()
		sampleRate := int(player.SampleRate)
		newPos := currentPos + (10 * sampleRate) // 10 seconds forward
		streamerLen := player.CurrentStreamer.Len()

		if newPos > streamerLen {
			newPos = streamerLen
		}
		if err := player.CurrentStreamer.Seek(newPos); err != nil {
			return err
		}
		// Calculate the new human-readable time to send back to the model.
		newPlayedTime := time.Duration(newPos) * time.Second / time.Duration(sampleRate)

		return playbackSeekedMsg{newPosition: newPos, newPlayedTime: newPlayedTime}
	}
}

// SkipBackwardCmd performs the side effect of seeking the audio stream backward by 10 seconds.
func SkipBackwardCmd(player *components.AudioPlayer) tea.Cmd {
	return func() tea.Msg {
		if player.CurrentStreamer == nil {
			return errors.New("no active playback to skip backward")
		}

		speaker.Lock()
		defer speaker.Unlock()

		currentPos := player.CurrentStreamer.Position()
		sampleRate := int(player.SampleRate)
		newPos := currentPos - (10 * sampleRate)

		if newPos < 0 {
			newPos = 0
		}
		if err := player.CurrentStreamer.Seek(newPos); err != nil {
			return err
		}
		newPlayedTime := time.Duration(newPos) * time.Second / time.Duration(sampleRate)

		return playbackSeekedMsg{newPosition: newPos, newPlayedTime: newPlayedTime}
	}
}

// SetVolumeCmd is a reusable command that performs the side effect of setting the player volume.
// It's used by Volume Up, Volume Down, etc., which calculate the target level in the Update loop.
func SetVolumeCmd(player *components.AudioPlayer, newVolume float64) tea.Cmd {
	return func() tea.Msg {
		if player == nil || player.Volume == nil {
			return errors.New("cannot set volume: player or volume controller is nil")
		}
		player.SetVolume(newVolume)
		return volumeChangedMsg{newVolume: newVolume}
	}
}

// SearchCmd performs a synchronous search of the library. As this is a fast, in-memory
// operation, it doesn't need to be a complex command, but wrapping it maintains consistency.
func SearchCmd(query string) tea.Cmd {
	return func() tea.Msg {
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

// LoadLibraryCmd performs the initial, potentially long-running I/O operation of
// scanning the user's Music directory for audio files.
func LoadLibraryCmd() tea.Cmd {
	return func() tea.Msg {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("Failed to get home directory: %v", err)
			return err // Return the error as a message for the Update loop.
		}
		musicDir := filepath.Join(homeDir, "Music")

		tracks, err := util.GetAudioFiles(musicDir)
		if err != nil {
			log.Printf("Failed to scan audio files: %v", err)
			return err
		}

		// On success, return a message with the loaded tracks.
		return LibraryLoadedMsg{Tracks: tracks}
	}
}
