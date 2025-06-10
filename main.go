package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"muxic/internal/player"
	"os"
)

// baseStyle defines a common style for rendering table views.
var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).    // Uses normal border style.
	BorderForeground(lipgloss.Color("240")). // Sets border color.
	Padding(0, 1)                            // Adds padding.

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	mp, err := player.NewMusicPlayer(dir)
	if err != nil {
		fmt.Println("Error initializing player:", err)
		os.Exit(1)
	}
	if err := mp.Run(); err != nil {
		fmt.Println("Error running player:", err)
		os.Exit(1)
	}
}
