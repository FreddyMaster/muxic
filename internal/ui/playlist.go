package ui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// Item represents an item in the playlist list
type Item struct {
	title, desc string
}

// Title returns the title of the item
func (i Item) Title() string { return i.title }

// Description returns the description of the item
func (i Item) Description() string { return i.desc }

// FilterValue implements the list.Item interface
func (i Item) FilterValue() string { return i.title }

// NewPlaylist creates a new playlist list model
func NewPlaylist() list.Model {
	items := []list.Item{
		Item{title: "My Favorites", desc: "Your favorite tracks"},
		Item{title: "Workout Mix", desc: "High energy tracks"},
		Item{title: "Chill Vibes", desc: "Relaxing music"},
	}

	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 0, 0, 1).
		Margin(0, 1, 0, 1)

	d.Styles.SelectedDesc = d.Styles.SelectedTitle.Copy().
		Foreground(lipgloss.Color("240"))

	l := list.New(items, d, 0, 0)
	l.Title = "Playlists"
	l.Styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.Color("62")).
		Bold(true).
		MarginLeft(2)

	return l
}
