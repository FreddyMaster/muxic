package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

func DefaultTableColumns() []table.Column {
	return []table.Column{
		{Title: "Title", Width: 20},
		{Title: "Artist", Width: 16},
		{Title: "Album", Width: 16},
		{Title: "Year", Width: 6},
		{Title: "Duration", Width: 8},
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
