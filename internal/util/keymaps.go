package util

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Play         key.Binding
	Pause        key.Binding
	Stop         key.Binding
	SkipBackward key.Binding
	SkipForward  key.Binding
	VolumeUp     key.Binding
	VolumeDown   key.Binding
	VolumeMute   key.Binding

	Quit   key.Binding
	Search key.Binding
}

// DefaultKeyMap provides the default key bindings.
var DefaultKeyMap = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("↓/j", "move down"),
	),
	Play: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "Play/Pause"),
	),
	Pause: key.NewBinding(
		key.WithKeys(" ", "p"),
		key.WithHelp("space/p", "Play/Pause"),
	),
	Stop: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+S", "stop"),
	),
	SkipBackward: key.NewBinding(
		key.WithKeys(""+
			"left", "h"),
		key.WithHelp("←/h", "Skip Backward"),
	),
	SkipForward: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "Skip Forward"),
	),
	VolumeUp: key.NewBinding(
		key.WithKeys("ctrl+up"),
		key.WithHelp("Ctrl+↑", "Volume Up"),
	),
	VolumeDown: key.NewBinding(
		key.WithKeys("ctrl+down"),
		key.WithHelp("Ctrl+↓", "Volume Down"),
	),
	VolumeMute: key.NewBinding(
		key.WithKeys("ctrl+m"),
		key.WithHelp("Ctrl+M", "Toggle Mute"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("Ctrl+C", "Quit Application"),
	),
	Search: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("Ctrl+F", "Search Tracks"),
	),
}
