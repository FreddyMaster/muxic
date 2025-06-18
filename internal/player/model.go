// Package player implements the main music player interface and logic.
// It handles audio playback, playlist management, and user interactions.
package player

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"

	"muxic/internal/ui"
	"muxic/internal/util"
)

// ViewMode represents the different views in the application
type ViewMode int

// Application view modes
const (
	// ViewLibrary shows the music library
	ViewLibrary ViewMode = iota
	// ViewPlaylists shows the list of playlists
	ViewPlaylists
	// ViewPlaylistTracks shows tracks in the selected playlist
	ViewPlaylistTracks
)

// String returns a human-readable representation of the view mode
func (v ViewMode) String() string {
	switch v {
	case ViewLibrary:
		return "Library"
	case ViewPlaylists:
		return "Playlists"
	case ViewPlaylistTracks:
		return "Playlist Tracks"
	default:
		return "Unknown View"
	}
}

// IsPlaylistView returns true if the current view is related to playlists
func (v ViewMode) IsPlaylistView() bool {
	return v == ViewPlaylists || v == ViewPlaylistTracks
}

// Model represents the main application state and UI components
type Model struct {
	// UI Components
	Table        table.Model     // Main library table view
	textInput    textinput.Model // Search input field
	playlistList table.Model     // Playlist tracks table
	Progress     progress.Model  // Playback progress bar

	// Library Data
	Columns []table.Column // Table column definitions
	Rows    []table.Row    // Currently visible rows (filtered)
	AllRows []table.Row    // All available rows (unfiltered)
	Paths   []string       // File paths for audio files

	// Search State
	searchIndex         *util.SearchIndex // Search index for library
	searchResultIndices []int             // Original indices of search results
	isSearching         bool              // Whether search is active

	// Audio Playback State
	CurrentStreamer beep.StreamSeekCloser // Current audio stream
	Playing         bool                  // Whether audio is playing
	TotalSamples    int                   // Total samples in current track
	SampleRate      beep.SampleRate       // Audio sample rate
	SamplesPlayed   int                   // Samples played so far
	PlayedTime      time.Duration         // Formatted play time
	TotalTime       time.Duration         // Total track duration
	Ctrl            *beep.Ctrl            // Playback controller
	Volume          *effects.Volume       // Volume controller

	// Application State
	viewMode ViewMode // Current view

	// Layout Dimensions
	Width         int // Window width
	Height        int // Window height
	ProgressWidth int // Width of progress bar
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
	if !m.Playing || m.TotalSamples <= 0 {
		return m, tickCmd()
	}

	percent := float64(m.SamplesPlayed) / float64(m.TotalSamples)
	if percent > 1.0 {
		percent = 1.0
		m.Playing = false
	}

	cmd := m.Progress.SetPercent(percent)
	return m, tea.Batch(tickCmd(), cmd)
}

// handleWindowSize handles window resize events
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.resize(msg.Width, msg.Height)

	// Forward the window size to the active view
	var cmd tea.Cmd
	if m.viewMode.IsPlaylistView() {
		m.playlistList, cmd = m.playlistList.Update(msg)
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
	switch {

	// View switching
	case key.Matches(msg, util.DefaultKeyMap.ToggleView):
		return m.toggleView()

	// Playback controls
	case key.Matches(msg, util.DefaultKeyMap.Play):
		return m, EnterCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.Pause):
		return m, PauseCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.Stop):
		return m, StopCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.SkipBackward):
		return m, SkipBackwardCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.SkipForward):
		return m, SkipForwardCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.VolumeUp):
		return m, VolumeUpCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.VolumeDown):
		return m, VolumeDownCmd(m)
	case key.Matches(msg, util.DefaultKeyMap.VolumeMute):
		return m, VolumeMuteCmd(m)

	// Search
	case key.Matches(msg, util.DefaultKeyMap.Search):
		return m.toggleSearch()

	// Playlist management
	case key.Matches(msg, util.DefaultKeyMap.AddToPlaylist):
		return m.handleAddToPlaylist()

	case key.Matches(msg, util.DefaultKeyMap.RemoveFromPlaylist):
		return m.handleRemoveFromPlaylist()

	// Quit
	case key.Matches(msg, util.DefaultKeyMap.Quit):
		if m.viewMode.IsPlaylistView() {
			m.viewMode = ViewLibrary
			return m, nil
		}
		return m, tea.Quit

	default:
		// Forward unhandled keys to the active table
		var cmd tea.Cmd
		switch m.viewMode {
		case ViewLibrary:
			m.Table, cmd = m.Table.Update(msg)
		case ViewPlaylists, ViewPlaylistTracks:
			m.playlistList, cmd = m.playlistList.Update(msg)
		}
		return m, cmd
	}
}

// toggleSearch toggles the search input field
func (m *Model) toggleSearch() (tea.Model, tea.Cmd) {
	if m.viewMode != ViewLibrary {
		return m, nil
	}

	m.isSearching = !m.isSearching
	if m.isSearching {
		m.textInput.Focus()
	} else {
		m.textInput.Blur()
	}
	return m, nil
}

// toggleView switches between library and playlist views
func (m *Model) toggleView() (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewLibrary:
		m.viewMode = ViewPlaylists
	case ViewPlaylists, ViewPlaylistTracks:
		m.viewMode = ViewLibrary
	}
	return m, nil
}

// handleAddToPlaylist adds the selected song to the current playlist
func (m *Model) handleAddToPlaylist() (tea.Model, tea.Cmd) {
	if m.viewMode == ViewLibrary && len(m.Rows) > 0 {
		selected := m.Table.Cursor()
		if selected >= 0 && selected < len(m.Rows) {
			m.addToPlaylist(selected)
		}
	}
	return m, nil
}

// handleRemoveFromPlaylist removes the selected song from the current playlist
func (m *Model) handleRemoveFromPlaylist() (tea.Model, tea.Cmd) {
	if m.viewMode.IsPlaylistView() && len(m.playlistList.Rows()) > 0 {
		selected := m.playlistList.Cursor()
		m.removeFromPlaylist(selected)
	}
	return m, nil
}

// Update is the main update loop handling incoming messages.
// It processes messages and updates the application state accordingly.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle search input if search is active
	if m.isSearching && m.viewMode == ViewLibrary {
		return m.handleSearchInput(msg)
	}

	// Handle key presses
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKeyPress(keyMsg)
	}

	// Handle other message types
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tickMsg:
		return m.handleTick()

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
			m.isSearching = false
			m.textInput.Blur()
			return m, nil
		}

		// Update the search input
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)

		// Update search results
		query := m.textInput.Value()
		if query == "" {
			// If search is empty, show all rows with their original indices
			m.Rows = m.AllRows
			m.searchResultIndices = make([]int, len(m.AllRows))
			for i := range m.AllRows {
				m.searchResultIndices[i] = i
			}
		} else {
			// Otherwise, get the search results with their original indices
			m.Rows, m.searchResultIndices = m.searchIndex.Search(query)
		}
		m.Table.SetRows(m.Rows)
		return m, cmd

	default:
		return m, nil
	}
}

// addToPlaylist adds the currently selected song to the playlist
func (m *Model) addToPlaylist(selected int) {
	if selected < 0 || selected >= len(m.Rows) {
		return // Index out of bounds
	}

	// Get the original index in case of search filtering
	originalIdx := selected
	if m.isSearching && len(m.searchResultIndices) > selected {
		originalIdx = m.searchResultIndices[selected]
	}

	// Create a new row with an index at the beginning
	currentRows := m.playlistList.Rows()
	newRow := append([]string{fmt.Sprintf("%d", len(currentRows)+1)}, m.AllRows[originalIdx]...)

	// Add the new row and update the table
	m.playlistList.SetRows(append(currentRows, newRow))
}

// removeFromPlaylist removes the currently selected song from the playlist
func (m *Model) removeFromPlaylist(selected int) {
	// TODO: Remove the selected song from the playlist
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
	m.textInput.Width = width
}

// calculateContentHeight calculates the available height for content
func (m *Model) calculateContentHeight() int {
	// Total height minus status bar, progress bar, and padding
	height := m.Height - 4 // Adjust based on your UI elements
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
	m.Table.SetColumns(ui.DefaultTableColumns(width))
	m.Table.SetHeight(height)

	// Update playlist table
	m.playlistList.SetColumns(ui.DefaultPlaylistColumns(width))
	m.playlistList.SetHeight(height)
}

// View renders the complete UI layout as a string.
func (m *Model) View() string {
	// Compose the main UI components
	searchView := m.renderSearch()
	content := m.renderContent()
	progressBar := m.renderProgressBar()
	timeView := m.renderTimeDisplay()
	statusBar := m.renderStatusBar()

	// Combine all components
	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchView,
		content,
		progressBar,
		timeView,
		statusBar,
	)
}

// renderSearch renders the search input if active
func (m *Model) renderSearch() string {
	if m.isSearching && m.viewMode == ViewLibrary {
		return m.textInput.View()
	}
	return ""
}

// renderContent renders the main content area based on the current view mode
func (m *Model) renderContent() string {
	switch m.viewMode {
	case ViewLibrary:
		return m.Table.View()
	case ViewPlaylists, ViewPlaylistTracks:
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
		m.playlistList.View(),
	)
}

// renderProgressBar renders the playback progress bar
func (m *Model) renderProgressBar() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		Render(m.Progress.View())
}

// renderTimeDisplay renders the current and total playback time
func (m *Model) renderTimeDisplay() string {
	timeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("240")).
		Width(m.Width)

	timeText := fmt.Sprintf("%s / %s",
		formatDuration(m.PlayedTime),
		formatDuration(m.TotalTime))

	return timeStyle.Render(timeText)
}

// renderStatusBar renders the status bar with view indicator and help text
func (m *Model) renderStatusBar() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Render(fmt.Sprintf(" %s | Tab: Switch View | Q: Quit", m.viewMode))
}

func NewModel(rows []table.Row, paths []string) (*Model, error) {
	// Default width for initial table creation
	defaultWidth := 80

	// Initialize main table
	columns := ui.DefaultTableColumns(defaultWidth)
	t := ui.NewTable(columns, rows)
	p := ui.NewProgressBar()

	// Initialize search indices to match initial rows
	indices := make([]int, len(rows))
	for i := range rows {
		indices[i] = i
	}

	// Initialize the text input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "> "
	ti.CharLimit = 50
	ti.Width = 50

	// Initialize playlist table with empty rows
	playlistColumns := ui.DefaultPlaylistColumns(defaultWidth)
	playlistTable := ui.NewPlaylist(playlistColumns, []table.Row{})

	return &Model{
		Table:               t,
		textInput:           ti,
		playlistList:        playlistTable,
		Columns:             columns,
		Rows:                rows,
		AllRows:             rows,
		Paths:               paths,
		searchResultIndices: indices,
		Progress:            p,
		searchIndex:         util.NewSearchIndex(rows),
		viewMode:            ViewLibrary,
		Width:               80, // default, will be set by WindowSizeMsg
		Height:              24, // default, will be set by WindowSizeMsg
	}, nil
}

// Run starts the Bubble Tea program.
func (m *Model) Run() error {
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
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
