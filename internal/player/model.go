package player

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/ui"
	"muxic/internal/util"
	"time"
)

type Model struct {
	Table           table.Model // Table UI component.
	textInput       textinput.Model
	Columns         []table.Column        // Table columns definition.
	Rows            []table.Row           // Table rows that list audio files.
	Paths           []string              // List of audio file paths.
	Width           int                   // Screen width.
	Height          int                   // Screen height.
	CurrentStreamer beep.StreamSeekCloser // Current audio streamer.
	Progress        progress.Model        // Progress bar UI component.
	ProgressWidth   int                   // Calculated width for progress bar.
	Playing         bool                  // Flag for whether audio is playing.
	TotalSamples    int                   // Total number of audio samples in current file.
	SampleRate      beep.SampleRate       // Sample rate of current audio file.
	SamplesPlayed   int                   // Number of samples that have been played.
	PlayedTime      time.Duration         // Time elapsed during playback.
	TotalTime       time.Duration         // Total playback time.
	Ctrl            *beep.Ctrl
	Volume          *effects.Volume
}

// Init initializes the application and starts the tick command.
func (m *Model) Init() tea.Cmd {
	// Start the ticker to regularly update progress.
	return tickCmd()
}

// Update is the main update loop handling incoming messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
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
			return m, tea.Quit
		case key.Matches(msg, util.DefaultKeyMap.Play):
			return m, EnterCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.Stop):
			speaker.Clear()
			m.Playing = false
		case key.Matches(msg, util.DefaultKeyMap.Pause):
			return m, PauseCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.VolumeUp):
			return m, VolumeUpCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.VolumeDown):
			return m, VolumeDownCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.VolumeMute):
			return m, VolumeMuteCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.SkipForward):
			return m, SkipForwardCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.SkipBackward):
			return m, SkipBackwardCmd(m)
		case key.Matches(msg, util.DefaultKeyMap.Search):
			return m, SearchCmd(m)
		}
	}
	m.Table, _ = m.Table.Update(msg)
	m.textInput, _ = m.textInput.Update(msg)
	return m, nil
}

func (m *Model) resize(width, height int) {
	m.Width = width
	m.Height = height
	// Calculate available width accounting for borders, padding, and column separators
	available := width - 4 // 4 for borders, 2*5=10 for padding, 4 for column separators
	if available < 40 {
		available = 40
	}
	// Fixed widths for number and year columns
	durationWidth := 10
	// Calculate remaining width for other columns
	remainingWidth := available - durationWidth - 2 // -2 for the separators
	// Distribute remaining width: 40% title, 40% artist, 20% album
	titleWidth := remainingWidth * 40 / 100
	artistWidth := remainingWidth * 40 / 100
	albumWidth := remainingWidth - titleWidth - artistWidth
	// Ensure minimum widths
	if titleWidth < 10 {
		titleWidth = 10
	}
	if artistWidth < 10 {
		artistWidth = 10
	}
	if albumWidth < 8 {
		albumWidth = 8
	}

	// Define table columns based on calculated widths
	m.Columns = []table.Column{
		{Title: "Title", Width: titleWidth},
		{Title: "Artist", Width: artistWidth},
		{Title: "Album", Width: albumWidth},
		{Title: "Duration", Width: durationWidth},
	}
	// Calculate available height for the table
	headerHeight := 4 // Title + help + empty line
	footerHeight := 4 // Progress bar + time + empty line + bottom padding
	tableHeight := height - headerHeight - footerHeight - 2
	// Ensure minimum height
	if tableHeight < 3 {
		tableHeight = 3
	}
	m.Table.SetColumns(m.Columns)
	m.Table.SetHeight(tableHeight)
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

	// Compose title, help, table view, progress bar and time display.
	progressBar := lipgloss.NewStyle().
		Width(m.Width - 4). // Match table width
		Render(m.Progress.View())

	// Build the UI
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.textInput.View(),
		"\n"+m.Table.View(),
		"\n"+progressBar,
		timesView,
	)
}

func NewModel(rows []table.Row, paths []string) (*Model, error) {
	columns := ui.DefaultTableColumns()
	t := ui.NewTable(columns, rows)
	p := ui.NewProgressBar()

	// Initialize the text input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Prompt = "🔍 "
	ti.CharLimit = 50
	ti.Width = 50

	return &Model{
		Table:     t,
		textInput: ti,
		Columns:   columns,
		Rows:      rows,
		Paths:     paths,
		Progress:  p,
		Width:     80, // default, will be set by WindowSizeMsg
		Height:    24, // default, will be set by WindowSizeMsg
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

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second/10, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
