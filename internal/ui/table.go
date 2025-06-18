package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func DefaultTableColumns(width int) []table.Column {
	// Fixed widths for duration column
	durationWidth := 10
	// Calculate remaining width for other columns
	remainingWidth := width - durationWidth - 2 // -2 for the separators
	// Distribute remaining width: 40% title, 40% artist, 20% album
	titleWidth := remainingWidth * 40 / 100
	artistWidth := remainingWidth * 40 / 100
	albumWidth := remainingWidth * 20 / 100

	return []table.Column{
		{Title: "Title", Width: titleWidth},
		{Title: "Artist", Width: artistWidth},
		{Title: "Album", Width: albumWidth},
		{Title: "Duration", Width: durationWidth},
	}
}

func NewTable(columns []table.Column, rows []table.Row) table.Model {
	// Create the table with initial settings.
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
	)
	t.SetStyles(DefaultTableStyles())
	return t
}

func DefaultTableStyles() table.Styles {
	// Set default styles for the table.
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	s.Cell = s.Cell.
		Padding(0, 1)
	return s
}
