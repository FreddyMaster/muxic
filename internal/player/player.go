package player

import (
	"fmt"
	"muxic/internal/util"
)

type MusicPlayer struct {
	model *Model
}

func NewMusicPlayer(dir string) (*MusicPlayer, error) {
	// Get audio files from the directory
	audioFiles, err := util.GetAudioFiles(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get audio files: %w", err)
	}

	// Get the library instance and add all audio files
	library := util.GetLibrary()
	for _, file := range audioFiles {
		library.AddFile(file)
	}

	// Create the model
	model, err := NewModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Refresh the library view
	model.LibraryTable.SetRows(library.ToTableRows())

	// Set the cursor to the first item if the library is not empty
	if library.Length() > 0 {
		model.ActiveFileIndex = 0
		model.LibraryTable.SetCursor(0)
	}

	return &MusicPlayer{model: model}, nil
}

func (p *MusicPlayer) Run() error {
	return p.model.Run()
}
