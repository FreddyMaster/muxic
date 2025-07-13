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
	AddToQueue      key.Binding
	RemoveFromQueue key.Binding
	ViewQueue       key.Binding
	PlayNext        key.Binding
	PlayPrevious    key.Binding
	ClearQueue      key.Binding
}

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
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc/⌫", "back"),
	),

	// Playback controls
	Play: key.NewBinding(
		key.WithKeys("space", "enter"),
		key.WithHelp("space/enter", "play/pause"),
	),
	Pause: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "pause"),
	),
	Stop: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "stop"),
	),
	SkipBackward: key.NewBinding(
		key.WithKeys("left", "z"),
		key.WithHelp("←/z", "rewind 5s"),
	),
	SkipForward: key.NewBinding(
		key.WithKeys("right", "c"),
		key.WithHelp("→/c", "forward 5s"),
	),
	NextTrack: key.NewBinding(
		key.WithKeys("n", "shift+right"),
		key.WithHelp("n/⇨", "next track"),
	),
	PreviousTrack: key.NewBinding(
		key.WithKeys("p", "shift+left"),
		key.WithHelp("p/⇦", "previous track"),
	),

	// Volume controls
	VolumeUp: key.NewBinding(
		key.WithKeys("=", "+"),
		key.WithHelp("= or +", "volume up"),
	),
	VolumeDown: key.NewBinding(
		key.WithKeys("-", "_"),
		key.WithHelp("- or _", "volume down"),
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
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "add to playlist"),
	),
	RemoveFromPlaylist: key.NewBinding(
		key.WithKeys("del"),
		key.WithHelp("del", "remove from playlist"),
	),

	// Queue controls
	AddToQueue: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add to queue"),
	),
	RemoveFromQueue: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "remove from queue"),
	),
	ViewQueue: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view queue"),
	),
	PlayNext: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "play next in queue"),
	),
	PlayPrevious: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "play previous in queue"),
	),
	ClearQueue: key.NewBinding(
		key.WithKeys("ctrl+shift+d"),
		key.WithHelp("ctrl+shift+d", "clear queue"),
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
