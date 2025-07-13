package ui

import "github.com/charmbracelet/bubbles/progress"

func NewProgressBar() progress.Model {
	return progress.New(
		progress.WithSolidFill("White"),
		progress.WithoutPercentage(),
	)
}
