package player

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"

	"muxic/internal/player/components"
	"muxic/internal/ui"
	"muxic/internal/util"
)

// ViewMode is a custom type (an enum) to represent which UI screen is currently visible.
// Using a custom type instead of raw integers makes the code more readable and type-safe.
type ViewMode int

// Enum definition for the different views in the application.
const (
	ViewLibrary        ViewMode = iota // The main music library view. Iota gives this a value of 0.
	ViewSearch                         // The search view with input and results.
	ViewPlaylists                      // The view listing all available playlists.
	ViewPlaylistTracks                 // The view showing tracks inside a specific playlist.
	ViewQueue                          // The playback queue view.
)

// String provides a human-readable name for each ViewMode, useful for debugging or UI labels.
func (v ViewMode) String() string {
	switch v {
	case ViewLibrary:
		return "Library"
	case ViewSearch:
		return "Search"
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

// IsPlaylistView is a helper method to check if the current view is related to playlists.
func (v ViewMode) IsPlaylistView() bool {
	return v == ViewPlaylistTracks || v == ViewPlaylists
}

// Model represents the entire state of the application at any given moment.
// It's the "single source of truth." The `Update` function receives messages
// and modifies this struct to produce a new state. The `View` function reads
// from this struct to draw the UI.
type Model struct {
	// --- UI Components ---
	// These are "sub-models" from the Bubble Tea ecosystem. Each manages its own state.
	LibraryTable  table.Model     // The component for displaying the main music library.
	SearchInput   textinput.Model // The component for the text search bar.
	SearchTable   table.Model     // The component for displaying search results.
	PlaylistTable []table.Model   // A slice of tables, one for each playlist.
	QueueTable    table.Model     // The component for displaying the playback queue.
	Progress      progress.Model  // The component for the playback progress bar.

	// --- UI State ---
	// State related to the UI's current status and layout.
	viewMode            ViewMode // The currently active view (e.g., ViewLibrary, ViewQueue).
	ActivePlaylistIndex int      // Which playlist table in the slice is currently active.
	isLoading           bool     // True if the initial library scan is in progress.
	Width               int      // Current terminal width.
	Height              int      // Current terminal height.
	ProgressWidth       int      // Calculated width for the progress bar.
	Error               error    // Stores the last error received, for display in the UI.

	// --- Data & Business Logic Components ---
	// These manage the application's core data.
	LibraryColumns  []table.Column              // Column definitions for the library table.
	PlaylistManager *components.PlaylistManager // Manages all playlist data and operations.
	Search          *components.Search          // Holds search state and results.
	Queue           *components.Queue           // Manages the playback queue.
	AudioPlayer     *components.AudioPlayer     // Manages all audio playback via beep.

	// --- Playback State ---
	// Data related to the currently playing track.
	NowPlaying    *util.AudioFile // The track currently playing or paused.
	CurrentVolume float64         // The current volume level, to be reflected in the UI.

	// Internal state for debouncing search input.
	searchTimer *time.Timer

	// Track to be added after a new playlist is created
	pendingTrackToAdd *util.AudioFile
}

// --- Custom Message Definitions ---
// These messages are defined here because they are closely tied to the Model's state updates.

// UpdateNowPlayingMsg is sent to update the 'Now Playing' information in the UI.
type UpdateNowPlayingMsg struct {
	Track *util.AudioFile
}

// PlaybackFinishedMsg is sent when a track has finished playing,
// triggering the handler to play the next track in the queue.
type PlaybackFinishedMsg struct{}

// tickMsg is sent on each "tick" of our update timer to refresh the progress bar.
type tickMsg time.Time

// performSearchMsg is sent when the search debounce timer fires, triggering a search command.
type performSearchMsg struct{}

// LibraryLoadedMsg is sent by the LoadLibraryCmd when the background scan is complete.
// It contains the tracks that were found.
type LibraryLoadedMsg struct {
	Tracks []*util.AudioFile
}

// Init is the first function called when the program starts. It's responsible for
// setting the initial state and returning the first command(s) to be executed.
func (m *Model) Init() tea.Cmd {
	// We use tea.Batch to run multiple commands concurrently at startup:
	// 1. tickCmd(): Starts the timer for progress bar updates.
	// 2. LoadLibraryCmd(): Starts scanning the music library in the background.
	return tea.Batch(tickCmd(), LoadLibraryCmd())
}

// resize is a helper method called when the window size changes. It updates the
// model's dimensions and recalculates the layout for all components.
func (m *Model) resize(width, height int) {
	m.Width = width
	m.Height = height

	contentHeight := m.calculateContentHeight()
	contentWidth := m.calculateContentWidth()

	m.updateTableLayouts(contentWidth, contentHeight)

	m.ProgressWidth = width
	m.Progress.Width = m.ProgressWidth
	m.SearchInput.Width = width
}

// HandlePlaybackFinished is the logic for what to do when a track finishes playing.
// It gets the next track from the queue and creates commands to play it and update the UI.
func (m *Model) HandlePlaybackFinished() tea.Cmd {
	if m.Queue == nil || m.Queue.IsEmpty() {
		return nil // Nothing to play.
	}

	nextTrack := m.Queue.GetNext()
	if nextTrack == nil {
		return nil // Reached the end of the queue.
	}

	// This command plays the audio. It's defined inline here as it's a core part
	// of the playback flow. It returns a message on completion or error.
	playCmd := func() tea.Msg {
		if err := m.AudioPlayer.Play(nextTrack); err != nil {
			return err
		}
		return PlaybackFinishedMsg{}
	}

	// We batch the play command with a message to update the "Now Playing" UI.
	return tea.Batch(
		playCmd,
		func() tea.Msg {
			return UpdateNowPlayingMsg{Track: nextTrack}
		},
	)
}

// calculateContentHeight calculates the available height for table content.
func (m *Model) calculateContentHeight() int {
	// Total height minus space for status bar, progress bar, etc.
	height := m.Height - 8 // This magic number should be based on your View's layout.
	if height < 3 {
		return 3 // Ensure a minimum height.
	}
	return height
}

// calculateContentWidth calculates the available width for table content.
func (m *Model) calculateContentWidth() int {
	// Total width minus borders/padding.
	width := m.Width - 4
	if width < 40 {
		return 40 // Ensure a minimum width.
	}
	return width
}

// NewModel is the constructor for our application's model. It initializes all
// components and sets up the default state of the application.
func NewModel() (*Model, error) {
	defaultWidth := 80

	// Initialize the audio speaker hardware. This must be done once.
	sr := beep.SampleRate(44100)
	if err := speaker.Init(sr, sr.N(time.Second/10)); err != nil {
		return nil, err
	}

	// Initialize all data managers and UI components with default values.
	playlistManager := components.NewPlaylistManager()
	library := components.GetLibrary()

	libraryColumns := ui.DefaultLibraryTableColumns(defaultWidth)
	libraryRows := library.ToTableRows()
	libraryTable := ui.NewLibraryTable(libraryColumns, libraryRows)

	progressBar := ui.NewProgressBar()
	searchInput := ui.NewSearch()

	searchRows := make([]table.Row, 0)
	searchColumns := ui.DefaultSearchTableColumns(defaultWidth)
	searchTable := ui.NewSearchTable(searchColumns, searchRows)

	playlistRows := make([]table.Row, 0)
	playlistColumns := ui.DefaultPlaylistTableColumns(defaultWidth)
	playlistTable := ui.NewPlaylistTable(playlistColumns, playlistRows)
	playlists := []table.Model{playlistTable}

	queueRows := make([]table.Row, 0)
	queueColumns := ui.DefaultQueueTableColumns(defaultWidth)
	queueTable := ui.NewQueueTable(queueColumns, queueRows)

	// Construct the final Model struct with all initialized components.
	return &Model{
		LibraryTable:        libraryTable,
		SearchInput:         searchInput,
		SearchTable:         searchTable,
		PlaylistTable:       playlists,
		QueueTable:          queueTable,
		LibraryColumns:      libraryColumns,
		ActivePlaylistIndex: 0,
		PlaylistManager:     playlistManager,
		Progress:            progressBar,
		viewMode:            ViewLibrary,
		isLoading:           true, // Start in a loading state until the library is scanned.
		Width:               80,
		Height:              24,
		AudioPlayer:         components.NewAudioPlayer(),
		Queue:               components.NewQueue(),
		Search:              components.NewSearch(),
	}, nil
}

// Run starts the Bubble Tea program, which takes control of the terminal.
func (m *Model) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
