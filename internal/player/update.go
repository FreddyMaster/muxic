package player

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"muxic/internal/player/components"
	"muxic/internal/ui"
	"muxic/internal/util"
	"time"
)

// Update is the main update loop handling incoming messages.
// It processes messages and updates the application state accordingly.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Clear error on any message except tick messages
	if _, isTick := msg.(tickMsg); !isTick {
		m.Error = nil
	}

	// Check if current track finished playing
	if m.AudioPlayer != nil &&
		m.AudioPlayer.Playing &&
		m.AudioPlayer.GetProgress() >= 0.99 { // 99% played
		return m, PlayNextInQueueCmd(m)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case progress.FrameMsg:
		return m.handleProgressFrame(msg)
	case tickMsg:
		return m.handleTick()
	case LibraryLoadedMsg:
		library := components.GetLibrary()
		for _, track := range msg.Tracks {
			library.AddFile(track)
		}
		m.LibraryTable.SetRows(library.ToTableRows())
		m.isLoading = false
		return m, nil
	case performSearchMsg:
		return m, SearchCmd(m.SearchInput.Value())
	case searchResultMsg:
		m.Search.Tracks = msg.tracks
		m.UpdateSearchTable()
		return m, nil
	default:
		return m, nil
	}
}

// tickCmd returns a command that sends a tickMsg after a short duration
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// handleTick updates the progress bar based on playback progress
func (m *Model) handleTick() (tea.Model, tea.Cmd) {
	if !m.AudioPlayer.Playing || m.AudioPlayer.TotalSamples <= 0 {
		return m, tickCmd()
	}

	percent := float64(m.AudioPlayer.SamplesPlayed) / float64(m.AudioPlayer.TotalSamples)
	if percent > 1.0 {
		percent = 1.0
		m.AudioPlayer.Playing = false
	}
	// Update progress bar
	progressCmd := m.Progress.SetPercent(percent)
	return m, tea.Batch(tickCmd(), progressCmd)
}

// UpdatePlaylistTable updates the playlist table with the current playlist tracks
func (m *Model) UpdatePlaylistTable() {
	if m.PlaylistManager == nil || m.PlaylistManager.ActivePlaylist == nil {
		return
	}

	rows := m.PlaylistManager.ToTableRows(m.PlaylistManager.ActivePlaylist.ID)
	if m.ActivePlaylistIndex >= 0 && m.ActivePlaylistIndex < len(m.PlaylistTable) {
		tbl := &m.PlaylistTable[m.ActivePlaylistIndex]
		tbl.SetRows(rows)
		tbl.UpdateViewport()
	}
}

func (m *Model) UpdateSearchTable() {
	if m.Search == nil {
		return
	}
	rows := m.Search.ToTableRows()
	m.SearchTable.SetRows(rows)
	m.SearchTable.UpdateViewport()
}

func (m *Model) UpdateQueueTable() {
	if m.Queue == nil {
		return
	}
	rows := m.Queue.ToTableRows()
	m.QueueTable.SetRows(rows)
	m.QueueTable.UpdateViewport()
}

// handleWindowSize handles window resize events
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.resize(msg.Width, msg.Height)

	// Forward the window size to the active view
	var cmd tea.Cmd
	if m.viewMode.IsPlaylistView() {
		m.PlaylistTable[m.ActivePlaylistIndex], cmd = m.PlaylistTable[m.ActivePlaylistIndex].Update(msg)
	}
	return m, cmd
}

// handleProgressFrame processes progress bar frame updates
func (m *Model) handleProgressFrame(msg progress.FrameMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var progressModel tea.Model

	// Update the progress bar and type assert the result
	progressModel, cmd = m.Progress.Update(msg)
	m.Progress = progressModel.(progress.Model)

	return m, cmd
}

// updateTableLayouts updates the layout of all tables
func (m *Model) updateTableLayouts(width, height int) {
	// Update main table
	m.LibraryTable.SetColumns(ui.DefaultLibraryTableColumns(width))
	m.LibraryTable.SetHeight(height)

	// Update Search table
	m.SearchTable.SetColumns(ui.DefaultSearchTableColumns(width))
	m.SearchTable.SetHeight(height)

	// Update playlist table
	m.PlaylistTable[m.ActivePlaylistIndex].SetColumns(ui.DefaultPlaylistTableColumns(width))
	m.PlaylistTable[m.ActivePlaylistIndex].SetHeight(height)

	// Update queue table
	m.QueueTable.SetColumns(ui.DefaultQueueTableColumns(width))
	m.QueueTable.SetHeight(height)
}

// handleKeyPress processes keyboard input
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// First, try to handle the key press with the active table
	var cmd tea.Cmd
	switch m.viewMode {
	case ViewLibrary:
		m.LibraryTable, cmd = m.LibraryTable.Update(msg)
		m.ActiveFileIndex = m.LibraryTable.Cursor()
		// If the table handled the key, return early
		if cmd != nil {
			return m, cmd
		}
	case ViewSearch:
		if m.Search.IsSearching {
			// Stop any existing timer
			if m.searchTimer != nil {
				m.searchTimer.Stop()
			}

			// When searching, forward keys to the text input
			var inputCmd tea.Cmd
			m.SearchInput, inputCmd = m.SearchInput.Update(msg)

			// Start a new timer
			m.searchTimer = time.NewTimer(200 * time.Millisecond)

			// The command will wait for the timer and then send a performSearchMsg
			debounceCmd := func() tea.Msg {
				<-m.searchTimer.C
				return performSearchMsg{}
			}

			return m, tea.Batch(inputCmd, debounceCmd)
		} else {
			// When not searching, forward keys to the table
			m.SearchTable, cmd = m.SearchTable.Update(msg)
			m.ActiveFileIndex = m.SearchTable.Cursor()
			if cmd != nil {
				return m, cmd
			}
		}

	case ViewPlaylists, ViewPlaylistTracks:
		m.PlaylistTable[m.ActivePlaylistIndex], cmd = m.PlaylistTable[m.ActivePlaylistIndex].Update(msg)
		m.ActiveFileIndex = m.PlaylistTable[m.ActivePlaylistIndex].Cursor()
		// If the table handled the key, return early
		if cmd != nil {
			return m, cmd
		}
	case ViewQueue:
		m.QueueTable, cmd = m.QueueTable.Update(msg)
		m.ActiveFileIndex = m.QueueTable.Cursor()
		// If the table handled the key, return early
		if cmd != nil {
			return m, cmd
		}
	}

	// If the table didn't handle the key, try application commands
	switch {
	// View switching
	case key.Matches(msg, util.DefaultKeyMap.ToggleView):
		return m.toggleView()

	// Playback controls - only handle these if we're not in a text input
	case key.Matches(msg, util.DefaultKeyMap.Play):
		return m, PlayCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.Pause):
		return m, PauseCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.Stop):
		return m, StopCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.SkipBackward):
		return m, SkipBackwardCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.SkipForward):
		return m, SkipForwardCmd(m)

	// Volume controls
	case key.Matches(msg, util.DefaultKeyMap.VolumeUp):
		return m, VolumeUpCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.VolumeDown):
		return m, VolumeDownCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.VolumeMute):
		return m, VolumeMuteCmd(m)

	// Track navigation
	case key.Matches(msg, util.DefaultKeyMap.NextTrack):
		return m, NextTrackCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.PreviousTrack):
		return m, PreviousTrackCmd(m)

	// Search
	case key.Matches(msg, util.DefaultKeyMap.Search):
		if m.viewMode == ViewSearch {
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

	// Playlist management
	case key.Matches(msg, util.DefaultKeyMap.CreatePlaylist):
		return m, CreatePlaylistCmd(m, "New Playlist")
	case key.Matches(msg, util.DefaultKeyMap.AddToPlaylist):
		return m, AddToPlaylistCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.RemoveFromPlaylist):
		return m, RemoveFromPlaylistCmd(m)

	// Queue management
	case key.Matches(msg, util.DefaultKeyMap.AddToQueue):
		return m, AddToQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.RemoveFromQueue):
		return m, RemoveFromQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.ViewQueue):
		return m, ViewQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.PlayNext):
		return m, PlayNextInQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.PlayPrevious):
		return m, PlayPreviousInQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.ClearQueue):
		return m, ClearQueueCmd(m)

	// Quit
	case key.Matches(msg, util.DefaultKeyMap.Quit):
		return m, tea.Quit

	// If we get here, the key wasn't handled by the application
	default:
		return m, nil
	}
}

// toggleView switches between library and playlist views
func (m *Model) toggleView() (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewLibrary:
		m.viewMode = ViewSearch
		return m, nil

	case ViewSearch:
		m.viewMode = ViewPlaylists
		return m, nil

	case ViewPlaylists:
		m.viewMode = ViewQueue
		return m, nil

	case ViewQueue:
		m.viewMode = ViewLibrary
		return m, nil

	default:
		m.viewMode = ViewLibrary
		return m, nil
	}
}
