package util

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	Pause        key.Binding
	Stop         key.Binding
	SkipBackward key.Binding
	SkipForward  key.Binding
	VolumeUp     key.Binding
	VolumeDown   key.Binding
	VolumeMute   key.Binding
	Play         key.Binding
	Quit         key.Binding
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
		key.WithHelp("enter", "play"),
	),
	Pause: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "pause"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
	),
	SkipBackward: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("←/h", "skip backward"),
	),
	SkipForward: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("→/l", "skip forward"),
	),
	VolumeUp: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "volume up"),
	),
	VolumeDown: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "volume down"),
	),
	VolumeMute: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "mute"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
