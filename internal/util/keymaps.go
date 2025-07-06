package util

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	// Navigation
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Back  key.Binding

	// Playback controls
	Play          key.Binding
	Pause         key.Binding
	Stop          key.Binding
	SkipBackward  key.Binding
	SkipForward   key.Binding
	NextTrack     key.Binding
	PreviousTrack key.Binding

	// Volume
	VolumeUp   key.Binding
	VolumeDown key.Binding
	VolumeMute key.Binding

	// Application
	Quit key.Binding

	// Search and navigation
	Search     key.Binding
	ToggleView key.Binding

	// Playlist controls
	CreatePlaylist     key.Binding
	AddToPlaylist      key.Binding
	RemoveFromPlaylist key.Binding

	// Queue controls
	AddToQueue key.Binding
	ViewQueue  key.Binding
	PlayNext   key.Binding
	ClearQueue key.Binding
}

// DefaultKeyMap provides the default key bindings.
var DefaultKeyMap = KeyMap{
	// Navigation
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "back"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),

	// Playback controls
	Play: key.NewBinding(
		key.WithKeys(" ", "p", "enter"),
		key.WithHelp("space/p/enter", "play/pause/select"),
	),
	Pause: key.NewBinding(
		key.WithKeys(" ", "p"),
		key.WithHelp("space/p", "play/pause"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
	),
	SkipBackward: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "skip backward"),
	),
	SkipForward: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "skip forward"),
	),
	NextTrack: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next track"),
	),
	PreviousTrack: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "previous track"),
	),

	// Volume controls
	VolumeUp: key.NewBinding(
		key.WithKeys("+"),
		key.WithHelp("+", "volume up"),
	),
	VolumeDown: key.NewBinding(
		key.WithKeys("-"),
		key.WithHelp("-", "volume down"),
	),
	VolumeMute: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "toggle mute"),
	),

	// Application
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q/ctrl+c", "quit"),
	),

	// Search and navigation
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	ToggleView: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "toggle view"),
	),

	// Playlist controls
	CreatePlaylist: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new playlist"),
	),
	AddToPlaylist: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add to playlist"),
	),
	RemoveFromPlaylist: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "remove from playlist"),
	),

	// Queue controls
	AddToQueue: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "add to queue"),
	),
	ViewQueue: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view queue"),
	),
	PlayNext: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "play next in queue"),
	),
	ClearQueue: key.NewBinding(
		key.WithKeys("shift+d"),
		key.WithHelp("shift+d", "clear queue"),
	),
}

// FullHelp returns a slice of key bindings for the help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},            // Navigation
		{k.Play, k.Pause, k.Stop},                  // Playback
		{k.PreviousTrack, k.NextTrack, k.PlayNext}, // Track navigation
		{k.VolumeDown, k.VolumeUp, k.VolumeMute},   // Volume
		{k.Search, k.ToggleView, k.ViewQueue},      // UI
		{k.AddToQueue, k.ClearQueue},               // Queue controls
		{k.Quit},                                   // Application
	}
}
