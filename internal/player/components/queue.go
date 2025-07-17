package components

import (
	"github.com/charmbracelet/bubbles/table"
	"math/rand"
	"muxic/internal/util"
	"strconv"
	"sync"
)

type Queue struct {
	Tracks       []*util.AudioFile
	CurrentIndex int
	Playing      bool
	mu           sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{}
}

func (q *Queue) Add(track *util.AudioFile) {
	q.Tracks = append(q.Tracks, track)
}

func (q *Queue) Remove(index int) {
	q.Tracks = append(q.Tracks[:index], q.Tracks[index+1:]...)
}

func (q *Queue) Next() {
	q.CurrentIndex++
	if q.CurrentIndex >= len(q.Tracks) {
		q.CurrentIndex = 0
	}
}

func (q *Queue) GetNext() *util.AudioFile {
	q.Next()
	return q.Current()
}

func (q *Queue) Previous() {
	q.CurrentIndex--
	if q.CurrentIndex < 0 {
		q.CurrentIndex = len(q.Tracks) - 1
	}
}

func (q *Queue) Shuffle() {
	rand.Shuffle(len(q.Tracks), func(i, j int) {
		q.Tracks[i], q.Tracks[j] = q.Tracks[j], q.Tracks[i]
	})
}

func (q *Queue) Current() *util.AudioFile {
	if len(q.Tracks) == 0 || q.CurrentIndex < 0 || q.CurrentIndex >= len(q.Tracks) {
		return nil
	}
	return q.Tracks[q.CurrentIndex]
}

func (q *Queue) Clear() {
	q.Tracks = nil
}

func (q *Queue) Length() int {
	return len(q.Tracks)
}

func (q *Queue) IsEmpty() bool {
	return len(q.Tracks) == 0
}

func (q *Queue) ToTableRows() []table.Row {
	rows := make([]table.Row, len(q.Tracks))
	for i, t := range q.Tracks {
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
