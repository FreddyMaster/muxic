package components

import (
	"errors"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/util"
	"time"
)

// AudioPlayer represents the state of the audio player
type AudioPlayer struct {
	CurrentStreamer beep.StreamSeekCloser // Current audio stream
	Playing         bool                  // Whether audio is playing
	Looping         bool                  // Whether audio is looping
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

func (a *AudioPlayer) Play(track *util.AudioFile) error {

	// If a track is already playing, stop it
	if a.IsPlaying() {

		a.Stop()
	}

	// Open and set up the new track
	streamer, format, totalSamples, err := util.OpenAudioFile(track.Path)
	if err != nil {
		return err
	}

	// Update AudioPlayer state
	a.CurrentStreamer = streamer
	a.SampleRate = format.SampleRate
	a.TotalSamples = totalSamples
	a.SamplesPlayed = 0
	a.PlayedTime = 0
	a.TotalTime = time.Duration(totalSamples) * time.Second / time.Duration(format.SampleRate)

	// Create a reference to the streamer to prevent garbage collection
	wrappedStreamer := beep.Seq(streamer, beep.Callback(func() {
		a.Playing = false
	}))

	// Wrap the streamer to track progress
	progressStreamer := beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = wrappedStreamer.Stream(samples)
		a.SamplesPlayed += n
		a.PlayedTime = time.Duration(a.SamplesPlayed) * time.Second /
			time.Duration(a.SampleRate)
		return
	})

	// Set up volume and control, reusing existing volume settings if they exist
	currentVolume := 0.0 // Default volume if not set
	if a.Volume != nil {
		currentVolume = a.Volume.Volume
	}

	a.Volume = &effects.Volume{
		Streamer: progressStreamer,
		Base:     1,             // Exponential scale base
		Volume:   currentVolume, // Use the current volume setting
		Silent:   false,
	}
	a.Ctrl = &beep.Ctrl{Streamer: a.Volume}

	// Start playback
	speaker.Play(a.Ctrl)
	a.Playing = true
	a.Ctrl.Paused = false

	return nil
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

func (a *AudioPlayer) isLooping() bool {
	return a.Looping
}
