package main

import (
	"fmt"
	"muxic/internal/player"
	"os"
)

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
