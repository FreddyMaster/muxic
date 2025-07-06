package components

import (
	"errors"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"time"
)

// AudioPlayer represents the state of the audio player
type AudioPlayer struct {
	CurrentStreamer beep.StreamSeekCloser // Current audio stream
	Playing         bool                  // Whether audio is playing
	SamplesPlayed   int                   // Samples played so far
	TotalSamples    int                   // Total samples in current track
	SampleRate      beep.SampleRate       // Audio sample rate
	PlayedTime      time.Duration         // Formatted play time
	TotalTime       time.Duration         // Total track duration
	Ctrl            *beep.Ctrl            // Playback controller
	Volume          *effects.Volume       // Volume controller
}

func NewAudioPlayer() *AudioPlayer {
	return &AudioPlayer{
		// Initialize with default values
		CurrentStreamer: nil,
		Playing:         false,
		SamplesPlayed:   0,
		TotalSamples:    0,
		SampleRate:      0,
		PlayedTime:      0,
		TotalTime:       0,
		Ctrl:            nil,
		Volume:          nil,
	}
}

func (a *AudioPlayer) Play() {
	if a.Ctrl != nil {
		a.Ctrl.Paused = false
		a.Playing = true
	}
}

func (a *AudioPlayer) Pause() {
	if a.Ctrl != nil {
		a.Ctrl.Paused = true
		a.Playing = false
	}
}

func (a *AudioPlayer) Stop() {
	if a.CurrentStreamer != nil {
		_ = a.CurrentStreamer.Close()
		a.CurrentStreamer = nil
	}
	a.Playing = false
	a.SamplesPlayed = 0
	a.TotalSamples = 0
	a.PlayedTime = 0
}

func (a *AudioPlayer) SetVolume(volume float64) {
	if a.Volume != nil {
		a.Volume.Volume = volume
	}
}

func (a *AudioPlayer) SeekTo(pos time.Duration) error {
	// Check if a track is playing
	if a.CurrentStreamer != nil {
		return errors.New("no track is playing")
	}

	// Convert position to sample position
	samplePos := int(pos.Seconds() * float64(a.SampleRate))
	if err := a.CurrentStreamer.Seek(samplePos); err != nil {
		return err
	}

	a.SamplesPlayed = samplePos
	a.PlayedTime = pos
	return nil
}

func (a *AudioPlayer) GetProgress() float64 {
	if a.TotalSamples <= 0 {
		return 0
	}
	return float64(a.SamplesPlayed) / float64(a.TotalSamples)
}
func (a *AudioPlayer) GetPlayedTime() time.Duration {
	return a.PlayedTime
}

func (a *AudioPlayer) GetTotalTime() time.Duration {
	return a.TotalTime
}

func (a *AudioPlayer) IsPlaying() bool {
	return a.Playing && a.Ctrl != nil && !a.Ctrl.Paused
}
