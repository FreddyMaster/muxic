package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
)

func NewSearch() textinput.Model {
	// Create the table with initial settings.
	t := textinput.New()
	t.Placeholder = "Search..."
	t.Prompt = "> "
	return t
}
