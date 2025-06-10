package ui

import "github.com/charmbracelet/bubbles/progress"

func NewProgressBar() progress.Model {
	return progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)
}
