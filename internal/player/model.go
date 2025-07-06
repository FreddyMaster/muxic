package player

import (
	"fmt"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/player/components"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"muxic/internal/ui"
	"muxic/internal/util"
)

// ViewMode represents the different views in the application
type ViewMode int

const (
	ViewLibrary ViewMode = iota
	ViewPlaylists
	ViewPlaylistTracks
	ViewQueue
)

func (v ViewMode) String() string {
	switch v {
	case ViewLibrary:
		return "Library"
	case ViewPlaylists:
		return "Playlists"
	case ViewPlaylistTracks:
		return "Playlist"
	case ViewQueue:
		return "Queue"
	default:
		return "Unknown"
	}
}

func (v ViewMode) Next() ViewMode {
	return ViewMode((int(v) + 1) % 4)
}

func (v ViewMode) Prev() ViewMode {
	return ViewMode((int(v) + 3) % 4)
}

func (v ViewMode) IsPlaylistView() bool {
	return v == ViewPlaylistTracks || v == ViewPlaylists
}

type Model struct {
	// UI Components
	LibraryTable        table.Model     // Main library table view
	SearchInput         textinput.Model // Search input field
	PlaylistTable       []table.Model   // Playlist tracks table
	QueueTable          table.Model     // Queue table
	ActivePlaylistIndex int             // Index of the active playlist
	Progress            progress.Model  // Playback progress bar

	// Dump

	// Library Data
	LibraryColumns []table.Column // Table column definitions

	// Playlist Data
	PlaylistManager *components.PlaylistManager

	// Playback State
	ActiveFileIndex int

	// Search
	Search *components.Search

	// Audio Player
	AudioPlayer *components.AudioPlayer

	// Queue
	Queue *components.Queue

	// Application State
	viewMode ViewMode // Current view

	// Layout Dimensions
	Width         int // Window width
	Height        int // Window height
	ProgressWidth int // Width of progress bar

	// Erorr
	Error error
}

// tickMsg is sent on each tick of the progress updater
type tickMsg time.Time

// tickCmd returns a command that sends a tickMsg after a short duration
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Init initializes the application and starts the tick command.
func (m *Model) Init() tea.Cmd {
	return tickCmd()
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
		return m.toggleSearch()

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
	case key.Matches(msg, util.DefaultKeyMap.ViewQueue):
		return m, ViewQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.PlayNext):
		return m, PlayNextInQueueCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.ClearQueue):
		m.Queue.Clear()
		return m, tea.Printf("Queue cleared")

	// Quit
	case key.Matches(msg, util.DefaultKeyMap.Quit):
		if m.viewMode.IsPlaylistView() {
			m.viewMode = ViewLibrary
			return m, nil
		}
		return m, tea.Quit

	// If we get here, the key wasn't handled by the application
	default:
		return m, nil
	}
}

// toggleSearch toggles the search input field
func (m *Model) toggleSearch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewLibrary {
		return m, nil
	}

	m.Search.IsSearching = !m.Search.IsSearching
	if m.Search.IsSearching {
		m.SearchInput.Focus()
	} else {
		m.SearchInput.Blur()
	}
	return m, nil
}

// toggleView switches between library and playlist views
func (m *Model) toggleView() (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewLibrary:
		// When switching to playlists view, ensure the playlist table is up to date
		if m.PlaylistManager != nil && len(m.PlaylistManager.Playlists) > 0 {
			// Update the active playlist index if needed
			if m.ActivePlaylistIndex >= len(m.PlaylistManager.Playlists) {
				m.ActivePlaylistIndex = 0
			}

			// Get the current active playlist
			playlist := m.PlaylistManager.Playlists[m.ActivePlaylistIndex]

			// Convert tracks to table rows
			var rows []table.Row
			for i, t := range playlist.Tracks {
				rows = append(rows, table.Row{
					strconv.Itoa(i + 1),
					t.Title,
					t.Artist,
					t.Album,
					t.Duration,
				})
			}

			// Update the playlist table
			if m.ActivePlaylistIndex < len(m.PlaylistTable) {
				m.PlaylistTable[m.ActivePlaylistIndex].SetRows(rows)
			}
		}

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

// Update is the main update loop handling incoming messages.
// It processes messages and updates the application state accordingly.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Clear error on any message except tick messages
	if _, isTick := msg.(tickMsg); !isTick {
		m.Error = nil
	}

	// Handle search input if search is active
	if m.Search.IsSearching && m.viewMode == ViewLibrary {
		return m.handleSearchInput(msg)
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

	default:
		return m, nil
	}
}

// handleSearchInput processes input when in search mode
func (m *Model) handleSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEscape:
			m.Search.IsSearching = false
			m.SearchInput.Blur()
			return m, nil
		}

		// Update the search input
		var cmd tea.Cmd
		m.SearchInput, cmd = m.SearchInput.Update(msg)

		// Update search results
		query := m.SearchInput.Value()
		library := util.GetLibrary()

		if query == "" {
			// If search is empty, show all rows
			m.LibraryTable.SetRows(library.ToTableRows())
		} else {
			// Otherwise, filter the library based on the query
			var filteredRows []table.Row
			for _, file := range library.Files {
				// Simple case-insensitive search in title and artist
				if strings.Contains(strings.ToLower(file.Title), strings.ToLower(query)) ||
					strings.Contains(strings.ToLower(file.Artist), strings.ToLower(query)) {
					filteredRows = append(filteredRows, file.ToLibraryRow())
				}
			}
			m.LibraryTable.SetRows(filteredRows)
		}
		return m, cmd

	default:
		return m, nil
	}
}

// resize handles window resize events and updates the UI components accordingly
func (m *Model) resize(width, height int) {
	// Update dimensions
	m.Width = width
	m.Height = height

	// Calculate content dimensions
	contentHeight := m.calculateContentHeight()
	contentWidth := m.calculateContentWidth()

	// Update table layouts
	m.updateTableLayouts(contentWidth, contentHeight)

	// Update progress bar and input
	m.ProgressWidth = width
	m.Progress.Width = m.ProgressWidth
	m.SearchInput.Width = width
}

// calculateContentHeight calculates the available height for content
func (m *Model) calculateContentHeight() int {
	// Total height minus status bar, progress bar, and padding
	height := m.Height - 6 // Adjust based on your UI elements
	if height < 3 {
		return 3
	}
	return height
}

// calculateContentWidth calculates the available width for content
func (m *Model) calculateContentWidth() int {
	// Total width minus borders/padding
	width := m.Width - 4
	if width < 40 {
		return 40
	}
	return width
}

// updateTableLayouts updates the layout of all tables
func (m *Model) updateTableLayouts(width, height int) {
	// Update main table
	m.LibraryTable.SetColumns(ui.DefaultLibraryTableColumns(width))
	m.LibraryTable.SetHeight(height)

	// Update playlist table
	m.PlaylistTable[m.ActivePlaylistIndex].SetColumns(ui.DefaultPlaylistTableColumns(width))
	m.PlaylistTable[m.ActivePlaylistIndex].SetHeight(height)

	// Update queue table
	m.QueueTable.SetColumns(ui.DefaultQueueTableColumns(width))
	m.QueueTable.SetHeight(height)
}

// clearErrorMsg is a message to clear the current error
type clearErrorMsg struct{}

// ErrorView renders the error message if there is one
func (m *Model) ErrorView() string {
	if m.Error == nil {
		return ""
	}

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000")).
		Bold(true).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF0000")).
		MarginBottom(1)

	return errorStyle.Render("Error: " + m.Error.Error())
}

// View renders the complete UI layout as a string.
func (m *Model) View() string {
	var views []string

	// Add error view if there's an error
	if errorView := m.ErrorView(); errorView != "" {
		views = append(views, errorView)
	}

	// Add search view
	views = append(views, m.renderSearch())

	// Add main content
	content := m.renderContent()
	if content != "" {
		views = append(views, content)
	}

	// Add progress and status bars
	progressBar := m.renderProgressBar()
	if progressBar != "" {
		views = append(views, progressBar)
	}

	timeView := m.renderTimeDisplay()
	if timeView != "" {
		views = append(views, timeView)
	}

	statusBar := m.renderStatusBar()
	if statusBar != "" {
		views = append(views, statusBar)
	}

	// Combine all non-empty components
	return lipgloss.JoinVertical(
		lipgloss.Left,
		views...,
	)
}

// renderSearch renders the search input if active
func (m *Model) renderSearch() string {
	if m.Search != nil && m.viewMode == ViewLibrary {
		return m.SearchInput.View()
	}
	return ""
}

// renderContent renders the main content area based on the current view mode
func (m *Model) renderContent() string {
	switch m.viewMode {
	case ViewLibrary:
		return m.LibraryTable.View()
	case ViewPlaylists:
		return m.renderPlaylistView()
	default:
		return ""
	}
}

// renderPlaylistView renders the playlist or playlist tracks view
func (m *Model) renderPlaylistView() string {
	title := "Playlists"
	if m.viewMode == ViewPlaylistTracks {
		title = "Playlist Tracks"
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		m.PlaylistTable[m.ActivePlaylistIndex].View(),
	)
}

// renderProgressBar renders the playback progress bar
func (m *Model) renderProgressBar() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		MarginTop(1).
		Render(m.Progress.View())
}

// renderTimeDisplay renders the current and total playback time
func (m *Model) renderTimeDisplay() string {
	timeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("240")).
		Width(m.Width)

	timeText := fmt.Sprintf("%s / %s",
		formatDuration(m.AudioPlayer.PlayedTime),
		formatDuration(m.AudioPlayer.TotalTime))

	return timeStyle.Render(timeText)
}

// GetCurrentFilePath returns the path of the currently selected file
func (m *Model) GetCurrentFilePath() string {
	switch m.viewMode {
	case ViewLibrary:
		library := util.GetLibrary()
		if m.ActiveFileIndex < 0 || m.ActiveFileIndex >= library.Count() {
			return ""
		}
		file, err := library.GetFile(m.ActiveFileIndex)
		if err != nil {
			return ""
		}
		return file.Path

	case ViewPlaylistTracks, ViewPlaylists:
		if m.PlaylistManager == nil || m.ActiveFileIndex < 0 || m.ActiveFileIndex >= len(m.PlaylistManager.ActivePlaylist.Tracks) {
			return ""
		}
		return m.PlaylistManager.ActivePlaylist.Tracks[m.ActiveFileIndex].Path

	default:
		return ""
	}
}

// renderStatusBar renders the status bar with view indicator and help text
func (m *Model) renderStatusBar() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		Bold(true).
		MarginTop(1).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Render(fmt.Sprintf(" %s | Tab: Switch View | Q: Quit", m.viewMode))
}

func NewModel() (*Model, error) {
	// Default width for initial table creation
	defaultWidth := 80

	// Initialize the audio player
	sr := beep.SampleRate(44100)
	if err := speaker.Init(sr, sr.N(time.Second/10)); err != nil {
		return nil, err
	}

	// Initialize the Playlist
	playlistManager := components.NewPlaylistManager()

	// Get the library instance
	library := util.GetLibrary()

	// Initialize main table with library data
	libraryColumns := ui.DefaultLibraryTableColumns(defaultWidth)
	libraryTable := ui.NewLibraryTable(libraryColumns, library.ToTableRows())

	// Initialize progress bar
	progressBar := ui.NewProgressBar()

	// Initialize the text input
	searchInput := ui.NewSearch()

	// Initialize playlist table with empty rows
	playlistRows := make([]table.Row, 0)
	playlistColumns := ui.DefaultPlaylistTableColumns(defaultWidth)
	playlistTable := ui.NewPlaylistTable(playlistColumns, playlistRows)
	playlists := []table.Model{playlistTable}

	// Initialize the queue table
	queueRows := make([]table.Row, 0)
	queueColumns := ui.DefaultQueueTableColumns(defaultWidth)
	queueTable := ui.NewQueueTable(queueColumns, queueRows)

	return &Model{
		LibraryTable:        libraryTable,
		SearchInput:         searchInput,
		PlaylistTable:       playlists,
		QueueTable:          queueTable,
		LibraryColumns:      libraryColumns,
		ActivePlaylistIndex: 0,  // Default to the first playlist
		ActiveFileIndex:     -1, // Initialize to -1 to indicate no selection
		PlaylistManager:     playlistManager,
		Progress:            progressBar,
		viewMode:            ViewLibrary,
		Width:               80, // default, will be set by WindowSizeMsg
		Height:              24, // default, will be set by WindowSizeMsg
		AudioPlayer:         components.NewAudioPlayer(),
		Queue:               components.NewQueue(),
		Search:              components.NewSearch(),
	}, nil
}

// Run starts the Bubble Tea program.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
