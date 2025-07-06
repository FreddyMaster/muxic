package components

import (
	"math/rand/v2"
	"muxic/internal/util"
)

type Queue struct {
	Tracks       []util.AudioFile
	CurrentIndex int
}

func NewQueue() *Queue {
	return &Queue{}
}

func (q *Queue) Add(track util.AudioFile) {
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

func (q *Queue) Previous() {
	q.CurrentIndex--
	if q.CurrentIndex < 0 {
		q.CurrentIndex = len(q.Tracks) - 1
	}
}

func (q *Queue) Shuffle() {
	rand.Shuffle(q.Length(), func(i, j int) {
		q.Tracks[i], q.Tracks[j] = q.Tracks[j], q.Tracks[i]
	})
}

func (q *Queue) Current() util.AudioFile {
	return q.Tracks[q.CurrentIndex]
}

func (q *Queue) Clear() {
	q.Tracks = nil
}

func (q *Queue) Length() int {
	return len(q.Tracks)
}
