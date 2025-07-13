package player

import (
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/player/components"
	"muxic/internal/ui"
	"muxic/internal/util"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewMode represents the different views in the application
type ViewMode int

const (
	ViewLibrary ViewMode = iota
	ViewSearch
	ViewPlaylists
	ViewPlaylistTracks
	ViewQueue
)

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
	SearchTable         table.Model     // Search results table
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
	Search      *components.Search
	searchTimer *time.Timer

	// Audio Player
	AudioPlayer *components.AudioPlayer

	// Queue
	Queue *components.Queue

	// Application State
	viewMode  ViewMode // Current view
	isLoading bool     // Is the library currently being loaded?

	// Layout Dimensions
	Width         int // Window width
	Height        int // Window height
	ProgressWidth int // Width of progress bar

	// Errors
	Error error
}

// tickMsg is sent on each tick of the progress updater
type tickMsg time.Time

// performSearchMsg is sent when the search debounce timer fires
type performSearchMsg struct{}

// LibraryLoadedMsg is sent when the background library scan is complete.
// It contains the tracks that were loaded.
type LibraryLoadedMsg struct {
	Tracks []*util.AudioFile
}

// Init initializes the application and starts the tick command.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), LoadLibraryCmd())
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
	height := m.Height - 8 // Adjust based on your UI elements
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
	library := components.GetLibrary()

	// Initialize main table with library data
	libraryColumns := ui.DefaultLibraryTableColumns(defaultWidth)
	libraryRows := library.ToTableRows()
	libraryTable := ui.NewLibraryTable(libraryColumns, libraryRows)

	// Initialize progress bar
	progressBar := ui.NewProgressBar()

	// Initialize the text input
	searchInput := ui.NewSearch()

	// Initialize search table with empty rows
	searchRows := make([]table.Row, 0)
	searchColumns := ui.DefaultSearchTableColumns(defaultWidth)
	searchTable := ui.NewSearchTable(searchColumns, searchRows)

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
		SearchTable:         searchTable,
		PlaylistTable:       playlists,
		QueueTable:          queueTable,
		LibraryColumns:      libraryColumns,
		ActivePlaylistIndex: 0,  // Default to the first playlist
		ActiveFileIndex:     -1, // Initialize to -1 to indicate no selection
		PlaylistManager:     playlistManager,
		Progress:            progressBar,
		viewMode:            ViewLibrary,
		isLoading:           true, // Start in loading state
		Width:               80,   // default, will be set by WindowSizeMsg
		Height:              24,   // default, will be set by WindowSizeMsg
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
