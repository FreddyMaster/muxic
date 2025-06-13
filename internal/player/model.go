package player

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"muxic/internal/ui"
	"muxic/internal/util"
	"time"
)

type Model struct {
	Table               table.Model // Table UI component.
	textInput           textinput.Model
	playlistList        list.Model
	Columns             []table.Column        // Table columns definition.
	Rows                []table.Row           // Table rows that list audio files.
	AllRows             []table.Row           // All available rows
	Paths               []string              // List of audio file paths.
	searchResultIndices []int                 // Original indices of search results
	Width               int                   // Screen width.
	Height              int                   // Screen height.
	Progress            progress.Model        // Progress bar component.
	ProgressWidth       int                   // Width of the progress bar.
	CurrentStreamer     beep.StreamSeekCloser // Current audio streamer.
	Playing             bool                  // Flag for whether audio is playing.
	TotalSamples        int                   // Total number of audio samples in current file.
	SampleRate          beep.SampleRate       // Sample rate of current audio file.
	SamplesPlayed       int                   // Number of samples that have been played.
	PlayedTime          time.Duration         // Time elapsed during playback.
	TotalTime           time.Duration         // Total playback time.
	Ctrl                *beep.Ctrl
	Volume              *effects.Volume
	searchIndex         *util.SearchIndex
	isSearching         bool
	viewMode            ViewMode
}

type ViewMode int

const (
	ViewLibrary ViewMode = iota
	ViewPlaylists
	ViewPlaylistTracks
)

type tickMsg time.Time

// Init initializes the application and starts the tick command.
func (m *Model) Init() tea.Cmd {
	// Start the ticker to regularly update progress.
	return tickCmd()
}

// Update is the main update loop handling incoming messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		// Forward the window size message to the playlist list
		if m.viewMode == ViewPlaylists || m.viewMode == ViewPlaylistTracks {
			var cmd tea.Cmd
			m.playlistList, cmd = m.playlistList.Update(msg)
			return m, cmd
		}
		return m, nil

	case tickMsg:
		if m.Playing && m.TotalSamples > 0 {
			percent := float64(m.SamplesPlayed) / float64(m.TotalSamples)
			if percent > 1.0 {
				percent = 1.0
				m.Playing = false
			}
			cmd := m.Progress.SetPercent(percent)
			return m, tea.Batch(tickCmd(), cmd)
		}
		return m, tickCmd()

	case progress.FrameMsg:
		pm, cmd := m.Progress.Update(msg)
		m.Progress = pm.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, util.DefaultKeyMap.Quit):
			if m.viewMode == ViewPlaylists || m.viewMode == ViewPlaylistTracks {
				m.viewMode = ViewLibrary
				return m, nil
			}
			return m, tea.Quit

		// Forward key messages to the playlist list when in playlist view
		case m.viewMode == ViewPlaylists || m.viewMode == ViewPlaylistTracks:
			var cmd tea.Cmd
			m.playlistList, cmd = m.playlistList.Update(msg)
			return m, cmd

		// View switching
		case key.Matches(msg, util.DefaultKeyMap.ToggleView):
			if m.viewMode == ViewLibrary {
				m.viewMode = ViewPlaylists
			} else {
				m.viewMode = ViewLibrary
			}
			return m, nil

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
			if m.viewMode == ViewLibrary {
				m.isSearching = !m.isSearching
				if m.isSearching {
					m.textInput.Focus()
				} else {
					m.textInput.Blur()
				}
			}
			return m, nil
		}
	}

	// Update the appropriate component based on current view mode and focus
	if m.isSearching && m.viewMode == ViewLibrary {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyEscape:
				m.isSearching = false
				m.textInput.Blur()
				return m, nil
			}

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
		}
	}

	switch m.viewMode {
	case ViewLibrary:
		m.Table, cmd = m.Table.Update(msg)
	case ViewPlaylists, ViewPlaylistTracks:
		m.playlistList, cmd = m.playlistList.Update(msg)
	}

	return m, cmd
}

func (m *Model) resize(width, height int) {
	m.Width = width
	m.Height = height

	// Update table columns with new widths
	available := width - 4 // Account for borders
	if available < 40 {
		available = 40
	}

	// Fixed widths for duration column
	durationWidth := 10
	// Calculate remaining width for other columns
	remainingWidth := available - durationWidth - 2 // -2 for the separators
	// Distribute remaining width: 40% title, 40% artist, 20% album
	titleWidth := remainingWidth * 40 / 100
	artistWidth := remainingWidth * 40 / 100
	albumWidth := remainingWidth * 20 / 100

	m.Columns = []table.Column{
		{Title: "Title", Width: titleWidth},
		{Title: "Artist", Width: artistWidth},
		{Title: "Album", Width: albumWidth},
		{Title: "Duration", Width: durationWidth},
	}

	// Calculate available height for the content
	headerHeight := 4 // Title + help + empty line
	footerHeight := 4 // Progress bar + time + empty line + bottom padding

	contentHeight := height - headerHeight - footerHeight - 2
	if contentHeight < 3 {
		contentHeight = 3
	}

	// Update table dimensions
	m.Table.SetColumns(m.Columns)
	m.Table.SetHeight(contentHeight)

	m.playlistList.SetSize(available-4, contentHeight) // Subtract some padding

	// Update progress bar width
	m.ProgressWidth = width - 4
	if m.ProgressWidth < 10 {
		m.ProgressWidth = 10
	}
	m.Progress.Width = m.ProgressWidth
}

// View renders the complete UI layout as a string.
func (m *Model) View() string {
	// Create a styled time display.
	timeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("240"))
	timeText := fmt.Sprintf("%s / %s", formatDuration(m.PlayedTime), formatDuration(m.TotalTime))
	timesView := timeStyle.Render(timeText)

	// Show current view indicator
	var viewIndicator string
	switch m.viewMode {
	case ViewLibrary:
		viewIndicator = "Library"
	case ViewPlaylists:
		viewIndicator = "Playlists"
	case ViewPlaylistTracks:
		viewIndicator = "Tracks"
	}

	// Show search input or status
	var searchView string
	if m.isSearching && m.viewMode == ViewLibrary {
		searchView = m.textInput.View()
	} else if query := m.textInput.Value(); query != "" && m.viewMode == ViewLibrary {
		searchView = fmt.Sprintf("Search: %s (Press / to search)", query)
	} else if m.viewMode == ViewLibrary {
		searchView = "Press / to search, Tab to switch view"
	} else {
		searchView = fmt.Sprintf("View: %s (Tab to switch)", viewIndicator)
	}

	// Compose the main content based on view mode
	var content string
	switch m.viewMode {
	case ViewLibrary:
		content = m.Table.View()
	case ViewPlaylists, ViewPlaylistTracks:
		// Add some styling to the playlist list
		content = lipgloss.NewStyle().
			Margin(1, 2).
			Width(m.Width - 4).
			MaxWidth(m.Width - 4).
			Render(m.playlistList.View())
	}

	// Progress bar
	progressBar := lipgloss.NewStyle().
		Width(m.Width - 4).
		Render(m.Progress.View())

	// Build the UI
	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchView,
		"\n"+content,
		"\n"+progressBar,
		timesView,
	)
}

func NewModel(rows []table.Row, paths []string) (*Model, error) {
	columns := ui.DefaultTableColumns()
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

	playlistList := ui.NewPlaylist()

	return &Model{
		Table:               t,
		textInput:           ti,
		playlistList:        playlistList,
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

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
