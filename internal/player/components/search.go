package components

import (
	"github.com/charmbracelet/bubbles/table"
	"muxic/internal/util"
	"strconv"
)

type Search struct {
	Tracks      []*util.AudioFile
	IsSearching bool // Whether search is active
}

func NewSearch() *Search {
	return &Search{
		Tracks:      make([]*util.AudioFile, 0),
		IsSearching: false,
	}
}

func (s *Search) GetTracks() []*util.AudioFile {
	return s.Tracks
}

func (s *Search) ToTableRows() []table.Row {
	rows := make([]table.Row, len(s.Tracks))
	for i, t := range s.Tracks {
		rows[i] = table.Row{
			strconv.Itoa(i + 1),
			t.Title,
			t.Artist,
			t.Album,
			t.Duration,
		}
	}
	return rows
}
