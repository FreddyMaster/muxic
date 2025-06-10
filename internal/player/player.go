package player

import (
	"muxic/internal/util"
)

type MusicPlayer struct {
	model *Model
}

func NewMusicPlayer(dir string) (*MusicPlayer, error) {
	rows, paths, err := util.GetAudioRows(dir)
	if err != nil {
		return nil, err
	}
	model, err := NewModel(rows, paths)
	if err != nil {
		return nil, err
	}
	return &MusicPlayer{model: model}, nil
}

func (p *MusicPlayer) Run() error {
	return p.model.Run()
}
