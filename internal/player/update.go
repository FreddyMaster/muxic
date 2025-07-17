package player

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"muxic/internal/player/components"
	"muxic/internal/ui"
	"muxic/internal/util"
	"time"
)

// Update is the central message processing function of the application. It follows
// the Elm Architecture, where the function receives the current model and a message,
// and returns the new model state and a command to be executed.
// This function is the ONLY place where the application's state (the model `m`)
// should be mutated.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// A good practice: handle any error messages first. This allows us to
	// display errors from any command in a consistent way.
	if msg, ok := msg.(error); ok {
		m.Error = msg
		return m, nil
	}

	// The main switch statement routes incoming messages to the appropriate logic.
	// Each `case` handles a specific type of event or result from a command.
	switch msg := msg.(type) {

	// --- Core Bubble Tea Messages ---

	// tea.KeyMsg is sent when the user presses a key.
	// We delegate the complex logic to a dedicated handler function.
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	// tea.WindowSizeMsg is sent on startup and when the terminal window is resized.
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	// --- Component/Animation Messages ---

	// progress.FrameMsg is sent by the progress bar component to render the next
	// frame of its animation. We must pass this message to the progress bar's own Update method.
	case progress.FrameMsg:
		return m.handleProgressFrame(msg)

	// tickMsg is our custom message for updating the playback progress periodically.
	case tickMsg:
		return m.handleTick()

	// --- Playlist Management Messages ---

	// These messages are received after their corresponding commands have completed successfully.
	// The state mutation happens here, not in the command.
	case playlistCreatedMsg:
		m.UpdatePlaylistTable()
		return m, nil
	case trackAddedToPlaylistMsg:
		m.UpdatePlaylistTable()
		return m, nil
	case trackRemovedFromPlaylistMsg:
		m.UpdatePlaylistTable()
		return m, nil
	case playlistShuffledMsg:
		m.UpdatePlaylistTable()
		return m, nil

	// --- Queue Management Messages ---

	case addTrackToQueueMsg:
		m.Queue.Add(msg.track)
		m.UpdateQueueTable()

		// If this is the first track added, we start playback automatically.
		if m.Queue.Length() == 1 {
			return m, m.HandlePlaybackFinished()
		}
		return m, nil

	case removeTrackFromQueueMsg:
		m.Queue.Remove(msg.index)
		m.UpdateQueueTable()

		return m, nil

	case nextTrackInQueueMsg:
		m.Queue.Next()
		m.UpdateQueueTable()
		return m, nil

	case previousTrackInQueueMsg:
		m.Queue.Previous()
		m.UpdateQueueTable()
		return m, nil

	case clearQueueMsg:
		m.Queue.Clear()
		m.UpdateQueueTable()
		return m, nil

	case viewQueueMsg:
		m.viewMode = ViewQueue
		return m, nil

	// --- Audio Player State Messages ---

	case pauseMsg:
		// The command performed the side effect; now we update our model's state.
		if m.AudioPlayer != nil {
			m.AudioPlayer.Playing = false
		}
		return m, nil

	case stopMsg:
		// The command stopped the hardware; now we reset our model's state.
		if m.AudioPlayer != nil {
			m.AudioPlayer.Playing = false
			m.AudioPlayer.CurrentStreamer = nil
			m.AudioPlayer.SamplesPlayed = 0
			m.AudioPlayer.PlayedTime = 0
		}
		m.NowPlaying = nil
		// We also need to tell the progress bar component to update its view.
		progressCmd := m.Progress.SetPercent(0)
		return m, progressCmd

	case playbackSeekedMsg:
		// The command performed the seek; we receive the new position and apply it.
		if m.AudioPlayer != nil {
			m.AudioPlayer.SamplesPlayed = msg.newPosition
			m.AudioPlayer.PlayedTime = msg.newPlayedTime
		}
		return m, nil

	case volumeChangedMsg:
		// The command set the volume; now we update our model's state to reflect it.
		m.CurrentVolume = msg.newVolume
		return m, nil

	// --- Data Loading and Search Messages ---

	case LibraryLoadedMsg:
		library := components.GetLibrary()
		for _, track := range msg.Tracks {
			library.AddFile(track)
		}
		m.LibraryTable.SetRows(library.ToTableRows())
		m.isLoading = false
		return m, nil

	case performSearchMsg:
		// This message triggers the search command.
		return m, SearchCmd(m.SearchInput.Value())

	case searchResultMsg:
		// We receive the results from the search command and update the search table.
		m.Search.Tracks = msg.tracks
		m.UpdateSearchTable()
		return m, nil

	// --- Playback Flow Messages ---

	case UpdateNowPlayingMsg:
		m.NowPlaying = msg.Track
		return m, nil

	case PlaybackFinishedMsg:
		// When one track finishes, this handler decides what to play next.
		return m, m.HandlePlaybackFinished()

	// If no other case matches, we do nothing.
	default:
		return m, nil
	}
}

// tickCmd returns a command that sends a tickMsg every 100ms.
// This drives the regular updates for the playback progress bar.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// handleTick is called for every tickMsg. It calculates the current playback
// percentage and sends a command to the progress bar to update its view.
func (m *Model) handleTick() (tea.Model, tea.Cmd) {
	if !m.AudioPlayer.Playing || m.AudioPlayer.TotalSamples <= 0 {
		return m, tickCmd() // If not playing, just schedule the next tick.
	}

	percent := float64(m.AudioPlayer.SamplesPlayed) / float64(m.AudioPlayer.TotalSamples)
	if percent > 1.0 {
		percent = 1.0
	}

	// We create a command to update the progress bar component.
	// We also batch it with the next tick command to keep the loop going.
	progressCmd := m.Progress.SetPercent(percent)
	return m, tea.Batch(tickCmd(), progressCmd)
}

// --- View Update Helpers ---
// These functions centralize the logic for updating the data rows in our tables.

// UpdatePlaylistTable refreshes the rows in the active playlist table.
func (m *Model) UpdatePlaylistTable() {
	if m.PlaylistManager == nil || m.PlaylistManager.ActivePlaylist == nil {
		return
	}

	rows := m.PlaylistManager.ToTableRows(m.PlaylistManager.ActivePlaylist.ID)
	if m.ActivePlaylistIndex >= 0 && m.ActivePlaylistIndex < len(m.PlaylistTable) {
		tbl := &m.PlaylistTable[m.ActivePlaylistIndex]
		tbl.SetRows(rows)
		// Update the cursor position in the active table.
		m.UpdateCursorPosition(tbl)
	}
}

// UpdateSearchTable refreshes the rows in the search results table.
func (m *Model) UpdateSearchTable() {
	if m.Search == nil {
		return
	}
	rows := m.Search.ToTableRows()
	m.SearchTable.SetRows(rows)

}

// UpdateQueueTable refreshes the rows in the queue table.
func (m *Model) UpdateQueueTable() {
	if m.Queue == nil {
		return
	}
	rows := m.Queue.ToTableRows()
	m.QueueTable.SetRows(rows)
	m.UpdateCursorPosition(&m.QueueTable)
}

// UpdateCursorPosition updates the cursor position in the active table.
func (m *Model) UpdateCursorPosition(table *table.Model) {
	listLength := len(table.Rows())
	// If the table is empty, we don't need to update the cursor position.
	if listLength == 0 {
		return
	}

	// Update the cursor position in the active table.
	if table.Cursor() >= listLength {
		table.GotoBottom()
	}
	if table.Cursor() < 0 {
		table.GotoTop()
	}
}

// --- Handler Functions for Core Messages ---

// handleWindowSize resizes the application layout and forwards the message to sub-components.
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.resize(msg.Width, msg.Height)

	// Some components (like tables) need to know the window size themselves.
	var cmd tea.Cmd
	// This logic might need to be expanded to handle the active table in any viewMode.
	if m.viewMode.IsPlaylistView() && len(m.PlaylistTable) > 0 {
		m.PlaylistTable[m.ActivePlaylistIndex], cmd = m.PlaylistTable[m.ActivePlaylistIndex].Update(msg)
	}
	return m, cmd
}

// handleProgressFrame passes animation frame messages directly to the progress bar component.
func (m *Model) handleProgressFrame(msg progress.FrameMsg) (tea.Model, tea.Cmd) {
	// The progress bar's Update method returns a new progress bar model and potentially a command.
	newProgressModel, cmd := m.Progress.Update(msg)
	m.Progress = newProgressModel.(progress.Model) // We must update our model with the new component state.
	return m, cmd
}

// updateTableLayouts is a helper to resize all tables when the window size changes.
func (m *Model) updateTableLayouts(width, height int) {
	m.LibraryTable.SetColumns(ui.DefaultLibraryTableColumns(width))
	m.LibraryTable.SetHeight(height)
	m.SearchTable.SetColumns(ui.DefaultSearchTableColumns(width))
	m.SearchTable.SetHeight(height)
	// This assumes at least one playlist table exists.
	if len(m.PlaylistTable) > 0 {
		m.PlaylistTable[m.ActivePlaylistIndex].SetColumns(ui.DefaultPlaylistTableColumns(width))
		m.PlaylistTable[m.ActivePlaylistIndex].SetHeight(height)
	}
	m.QueueTable.SetColumns(ui.DefaultQueueTableColumns(width))
	m.QueueTable.SetHeight(height)
}

// handleKeyPress is the logical hub for all user keyboard input.
// It first delegates input to the active component (like a table or text input).
// If the component doesn't handle the key, it checks for global application-level keybindings.
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// --- Component-level Input Handling ---
	// First, give the active view's main component a chance to handle the key.
	// This is for things like scrolling up/down in a table.
	switch m.viewMode {
	case ViewLibrary:
		m.LibraryTable, cmd = m.LibraryTable.Update(msg)
	case ViewSearch:
		if m.Search.IsSearching {
			// If we're in search input mode, all keys go to the text input.
			m.SearchInput, cmd = m.SearchInput.Update(msg)
			// This is a great example of debouncing input. We wait for the user
			// to stop typing before dispatching the search command.
			debounceCmd := func() tea.Msg {
				time.Sleep(200 * time.Millisecond)
				return performSearchMsg{}
			}
			return m, tea.Batch(cmd, debounceCmd)
		} else {
			// If not actively typing, keys go to the search results table.
			m.SearchTable, cmd = m.SearchTable.Update(msg)
		}
	case ViewPlaylists, ViewPlaylistTracks:
		if len(m.PlaylistTable) > 0 {
			m.PlaylistTable[m.ActivePlaylistIndex], cmd = m.PlaylistTable[m.ActivePlaylistIndex].Update(msg)
		}
	case ViewQueue:
		m.QueueTable, cmd = m.QueueTable.Update(msg)
	}

	// If the component handled the key (e.g., table scrolling), it might return a command.
	// If so, we return early and don't check for global keybindings.
	if cmd != nil {
		return m, cmd
	}

	// --- Global Application Keybindings ---
	// If the active component did not handle the key, we check our own key map.
	switch {
	case key.Matches(msg, util.DefaultKeyMap.ToggleView):
		return m.toggleView()

	// --- Playback Controls ---
	// For each action, we first validate the state (e.g., is a track playing?).
	// If the state is valid, we dispatch the appropriate focused command.
	case key.Matches(msg, util.DefaultKeyMap.Pause):
		if m.AudioPlayer == nil || !m.AudioPlayer.Playing {
			return m, nil
		}
		return m, PauseCmd(m.AudioPlayer)

	case key.Matches(msg, util.DefaultKeyMap.Stop):
		if m.AudioPlayer == nil || !m.AudioPlayer.Playing {
			return m, nil
		}
		return m, StopCmd(m.AudioPlayer)

	case key.Matches(msg, util.DefaultKeyMap.SkipBackward):
		if m.AudioPlayer == nil || !m.AudioPlayer.Playing {
			return m, nil
		}
		return m, SkipBackwardCmd(m.AudioPlayer)

	case key.Matches(msg, util.DefaultKeyMap.SkipForward):
		if m.AudioPlayer == nil || !m.AudioPlayer.Playing {
			return m, nil
		}
		return m, SkipForwardCmd(m.AudioPlayer)

	// --- Volume Controls ---
	// Here, all the logic for calculating the new volume level lives right
	// where the event is handled. The command is only told what the target volume is.
	case key.Matches(msg, util.DefaultKeyMap.VolumeUp):
		if m.AudioPlayer == nil || m.AudioPlayer.Volume == nil {
			return m, nil
		}
		currentVol := m.AudioPlayer.GetVolume()
		newVol := currentVol + 5
		if newVol > 100 {
			newVol = 100
		}
		return m, SetVolumeCmd(m.AudioPlayer, newVol)

	case key.Matches(msg, util.DefaultKeyMap.VolumeDown):
		if m.AudioPlayer == nil || m.AudioPlayer.Volume == nil {
			return m, nil
		}
		currentVol := m.AudioPlayer.GetVolume()
		newVol := currentVol - 5
		if newVol < 0 {
			newVol = 0
		}
		return m, SetVolumeCmd(m.AudioPlayer, newVol)

	// Mute/Unmute could be improved by storing the pre-mute volume.
	case key.Matches(msg, util.DefaultKeyMap.VolumeMute):
		if m.AudioPlayer == nil || m.AudioPlayer.Volume == nil {
			return m, nil
		}
		// This is a simple toggle to 0. A more advanced implementation
		// would store the current volume and restore it on unmute.
		if m.CurrentVolume > 0 {
			m.CurrentVolume = m.CurrentVolume // Assuming m.lastVolume is a new field on Model
			return m, SetVolumeCmd(m.AudioPlayer, 0)
		} else {
			return m, SetVolumeCmd(m.AudioPlayer, m.CurrentVolume)
		}

	// --- Search ---
	case key.Matches(msg, util.DefaultKeyMap.Search):
		if m.viewMode == ViewSearch {
			// Toggle between typing-mode and selection-mode.
			m.Search.IsSearching = !m.Search.IsSearching
			if m.Search.IsSearching {
				m.SearchTable.Blur()
				m.SearchInput.Focus()
			} else {
				m.SearchInput.Blur()
				m.SearchTable.Focus()
			}
		}
		return m, nil

	// --- Playlist Management ---
	case key.Matches(msg, util.DefaultKeyMap.CreatePlaylist):
		if m.PlaylistManager == nil {
			m.PlaylistManager = components.NewPlaylistManager()
		}
		return m, CreatePlaylistCmd(m.PlaylistManager, "New Playlist")

	case key.Matches(msg, util.DefaultKeyMap.AddToPlaylist):
		// This block handles all the state validation and data gathering
		// before dispatching the clean AddToPlaylistCmd.
		if m.PlaylistManager == nil {
			m.PlaylistManager = components.NewPlaylistManager()
		}

		trackToAdd := components.GetLibrary().Files[m.LibraryTable.Cursor()]
		if m.PlaylistManager.ActivePlaylist == nil {
			// Handle case where no playlist is active by creating a default one.
			playlist, err := m.PlaylistManager.CreatePlaylist("My Playlist")
			if err != nil {
				m.Error = err
				return m, nil
			}
			m.PlaylistManager.ActivePlaylist = playlist
		}
		return m, AddToPlaylistCmd(m.PlaylistManager, m.PlaylistManager.ActivePlaylist.ID, trackToAdd)

	case key.Matches(msg, util.DefaultKeyMap.RemoveFromPlaylist):
		if m.viewMode != ViewPlaylistTracks || m.PlaylistManager.ActivePlaylist == nil {
			return m, nil
		}
		indexToRemove := m.PlaylistTable[m.ActivePlaylistIndex].Cursor()
		return m, RemoveFromPlaylistCmd(m.PlaylistManager, m.PlaylistManager.ActivePlaylist.ID, indexToRemove)

	case key.Matches(msg, util.DefaultKeyMap.ShufflePlaylist):
		if m.PlaylistManager == nil || m.PlaylistManager.ActivePlaylist == nil {
			return m, nil
		}
		return m, ShufflePlaylistCmd(m.PlaylistManager, m.PlaylistManager.ActivePlaylist.ID)

	// --- Queue Management ---
	case key.Matches(msg, util.DefaultKeyMap.AddToQueue):
		track := components.GetLibrary().Files[m.LibraryTable.Cursor()]
		return m, AddToQueueCmd(track)

	case key.Matches(msg, util.DefaultKeyMap.RemoveFromQueue):
		if m.viewMode != ViewQueue {
			return m, nil
		}
		indexToRemove := m.QueueTable.Cursor()
		return m, RemoveFromQueueCmd(indexToRemove)

	case key.Matches(msg, util.DefaultKeyMap.ViewQueue):
		return m, ViewQueueCmd()

	case key.Matches(msg, util.DefaultKeyMap.PlayNext):
		return m, PlayNextInQueueCmd()

	case key.Matches(msg, util.DefaultKeyMap.PlayPrevious):
		return m, PlayPreviousInQueueCmd()

	case key.Matches(msg, util.DefaultKeyMap.ClearQueue):
		return m, ClearQueueCmd()

	// --- Quit ---
	case key.Matches(msg, util.DefaultKeyMap.Quit):
		return m, tea.Quit

	// If we get here, the key wasn't handled by any of our bindings.
	default:
		return m, nil
	}
}

// toggleView cycles through the main views of the application.
func (m *Model) toggleView() (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewLibrary:
		m.viewMode = ViewSearch
	case ViewSearch:
		m.viewMode = ViewPlaylistTracks
	case ViewPlaylistTracks:
		m.viewMode = ViewQueue
	case ViewQueue:
		m.viewMode = ViewLibrary
	default:
		m.viewMode = ViewLibrary // Fallback to a default view.
	}
	return m, nil
}
