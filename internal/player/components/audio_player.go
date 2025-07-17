package components

import (
	"errors"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/speaker"
	"math"
	"muxic/internal/util"
	"sync"
	"time"
)

const (
	// Minimum volume percentage
	minVolume = 0.0
	// Max volume percentage
	maxVolume = 100.0
	// Base for exponential volume scaling
	volumeBase = 2.0
	// Max gain in decibels
	maxGainDB = 12.0
)

// AudioPlayer represents the state of the audio player
type AudioPlayer struct {
	CurrentStreamer      beep.StreamSeekCloser // Current audio stream
	Playing              bool                  // Whether audio is playing
	Looping              bool                  // Whether audio is looping
	SamplesPlayed        int                   // Samples played so far
	TotalSamples         int                   // Total samples in current track
	SampleRate           beep.SampleRate       // Audio sample rate
	PlayedTime           time.Duration         // Formatted play time
	TotalTime            time.Duration         // Total track duration
	Ctrl                 *beep.Ctrl            // Playback controller
	Volume               *effects.Volume       // Volume controller
	CurrentVolumePercent float64               // 0-100

	// doneChan signals that playback has finished.
	doneChan chan struct{}
	// closeOnce ensures the doneChan is closed only once.
	closeOnce sync.Once
}

func NewAudioPlayer() *AudioPlayer {
	return &AudioPlayer{
		// Initialize with default values
		CurrentStreamer:      nil,
		Playing:              false,
		SamplesPlayed:        0,
		TotalSamples:         0,
		SampleRate:           0,
		PlayedTime:           0,
		TotalTime:            0,
		Ctrl:                 nil,
		Volume:               nil,
		CurrentVolumePercent: 50.0,
	}
}

func (a *AudioPlayer) Play(track *util.AudioFile) error {
	if a.IsPlaying() {
		a.Stop()
	}

	streamer, format, totalSamples, err := util.OpenAudioFile(track.Path)
	if err != nil {
		return err
	}

	a.CurrentStreamer = streamer
	a.SampleRate = format.SampleRate
	a.TotalSamples = totalSamples
	a.SamplesPlayed = 0
	a.PlayedTime = 0
	a.TotalTime = time.Duration(totalSamples) * time.Second / time.Duration(format.SampleRate)

	a.doneChan = make(chan struct{})
	a.closeOnce = sync.Once{}

	callbackStreamer := beep.Callback(func() {
		// LOGGING: This is the natural end of the song.
		a.Playing = false
		a.closeOnce.Do(func() { close(a.doneChan) })
	})

	progressStreamer := beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = streamer.Stream(samples)
		a.SamplesPlayed += n
		a.PlayedTime = time.Duration(a.SamplesPlayed) * time.Second /
			time.Duration(a.SampleRate)
		return n, ok
	})

	currentVolume := 0.0
	if a.Volume != nil {
		currentVolume = a.Volume.Volume
	}

	a.Volume = &effects.Volume{
		Streamer: progressStreamer,
		Base:     2,
		Volume:   currentVolume,
		Silent:   false,
	}
	a.Ctrl = &beep.Ctrl{Streamer: a.Volume}

	speaker.Play(beep.Seq(a.Ctrl, callbackStreamer))
	a.Playing = true
	a.Ctrl.Paused = false

	<-a.doneChan // Block here

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
		speaker.Clear()
		_ = a.CurrentStreamer.Close()
		a.CurrentStreamer = nil
	}
	a.Playing = false
	a.SamplesPlayed = 0
	a.TotalSamples = 0
	a.PlayedTime = 0

	// If a track was playing, signal it to unblock the waiting Play command.
	if a.doneChan != nil {
		a.closeOnce.Do(func() { close(a.doneChan) })
	}
}

// SetVolume sets the volume as a percentage (0-100)
func (a *AudioPlayer) SetVolume(percent float64) {
	// Clamp the percentage between 0 and 100
	if percent < minVolume {
		percent = minVolume
	} else if percent > maxVolume {
		percent = maxVolume
	}

	a.CurrentVolumePercent = percent

	if a.Volume == nil {
		return
	}

	// Lock the speaker to prevent race conditions
	speaker.Lock()
	defer speaker.Unlock()

	if percent <= 0 {
		// Mute if volume is 0 or less
		a.Volume.Silent = true
	} else {
		// Convert percentage to exponential gain
		a.Volume.Silent = false
		if percent == 100 {
			// At 100%, use max gain
			a.Volume.Volume = maxGainDB / 10 // Convert dB to beep's scale
		} else {
			// Convert percentage to gain in decibels
			scaledPercent := percent / 100
			db := 10 * math.Log10(scaledPercent)
			a.Volume.Volume = db / 2 // Convert to beep's scale
		}
	}

	// Update the streamer to apply changes
	if a.Ctrl != nil {
		a.Ctrl.Streamer = a.Volume
	}
}

// GetVolume returns the current volume percentage (0-100)
func (a *AudioPlayer) GetVolume() float64 {
	return a.CurrentVolumePercent
}

func (a *AudioPlayer) SeekTo(pos time.Duration) error {
	// Check if a track is playing
	if a.CurrentStreamer == nil {
		return errors.New("no track is playing")
	}
	
	speaker.Lock()
	defer speaker.Unlock()

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
