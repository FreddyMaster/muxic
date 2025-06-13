package main

import (
	"muxic/internal/player"
	"os"

	"github.com/charmbracelet/log"
)

func main() {
	// Set up logging
	log.SetLevel(log.DebugLevel)
	log.Info("Starting muxic player")

	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	// Ensure the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", dir)
	}

	// Initialize and run the player
	mp, err := player.NewMusicPlayer(dir)
	if err != nil {
		log.Fatal("Error initializing player:", "error", err)
	}

	// Handle cleanup on exit
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic recovered:", "error", r)
		}
	}()

	// Run the player
	if err := mp.Run(); err != nil {
		log.Fatal("Error running player:", "error", err)
	}
}
