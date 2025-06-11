package player

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"muxic/internal/util"
	"time"
)

// Enter (play selected file)
func EnterCmd(m *Model) tea.Cmd {
	selected := m.Table.Cursor()
	if selected < 0 || selected >= len(m.Paths) {
		return nil
	}
	speaker.Clear()
	if m.CurrentStreamer != nil {
		_ = m.CurrentStreamer.Close()
	}
	streamer, format, totalSamples, err := util.OpenAudioFile(m.Paths[selected])
	if err != nil {
		fmt.Println("Playback error:", err)
		return nil
	}
	if m.SampleRate != format.SampleRate {
		if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err != nil {
			fmt.Println("Speaker init error:", err)
			return nil
		}
	}
	m.CurrentStreamer = streamer
	m.SampleRate = format.SampleRate
	m.TotalSamples = totalSamples
	m.SamplesPlayed = 0
	m.PlayedTime = 0
	m.TotalTime = time.Duration(totalSamples) * time.Second / time.Duration(format.SampleRate)
	m.Playing = true

	// Wrap the streamer to update SamplesPlayed and PlayedTime
	wrappedStreamer := beep.StreamerFunc(func(samples [][2]float64) (n int, ok bool) {
		n, ok = streamer.Stream(samples)
		m.SamplesPlayed += n
		m.PlayedTime += time.Duration(n) * time.Second / time.Duration(format.SampleRate)
		return
	})
	m.Volume = util.NewVolumeCtrl(wrappedStreamer)
	m.Ctrl = util.NewAudioCtrl(m.Volume)

	speaker.Play(m.Ctrl)
	return nil
}

// Pause
func PauseCmd(m *Model) tea.Cmd {
	if m.Ctrl != nil {
		m.Ctrl.Paused = !m.Ctrl.Paused
	}
	return nil
}

// Skip Forward
func SkipForwardCmd(m *Model) tea.Cmd {
	if m.CurrentStreamer != nil {
		speaker.Lock()
		newPos := m.CurrentStreamer.Position() + 10*int(m.SampleRate)
		if newPos > m.CurrentStreamer.Len() {
			newPos = m.CurrentStreamer.Len()
		}
		err := m.CurrentStreamer.Seek(newPos)
		if err != nil {
			fmt.Println("Playback error:", err)
			return nil
		}
		m.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.SampleRate)
		m.SamplesPlayed = newPos
		speaker.Unlock()
	}
	return nil
}

// Skip Backward
func SkipBackwardCmd(m *Model) tea.Cmd {
	if m.CurrentStreamer != nil {
		speaker.Lock()
		newPos := m.CurrentStreamer.Position() - 10*int(m.SampleRate)
		if newPos < 0 {
			newPos = 0
		}
		err := m.CurrentStreamer.Seek(newPos)
		if err != nil {
			fmt.Println("Playback error:", err)
			return nil
		}
		m.PlayedTime = time.Duration(newPos) * time.Second / time.Duration(m.SampleRate)
		m.SamplesPlayed = newPos
		speaker.Unlock()
	}
	return nil
}

// Volume Up
func VolumeUpCmd(m *Model) tea.Cmd {
	if m.Volume != nil {
		speaker.Lock()
		m.Volume.Volume += 0.5
		speaker.Unlock()
	}
	return nil
}

// Volume Down
func VolumeDownCmd(m *Model) tea.Cmd {
	if m.Volume != nil {
		speaker.Lock()
		m.Volume.Volume -= 0.5
		speaker.Unlock()
	}
	return nil
}

// Volume Mute
func VolumeMuteCmd(m *Model) tea.Cmd {
	if m.Volume != nil {
		speaker.Lock()
		m.Volume.Silent = !m.Volume.Silent
		speaker.Unlock()
	}
	return nil
}

func SearchCmd(m *Model) tea.Cmd {
	if m.textInput.Focused() {
		disableSearch(m)
		return nil
	}
	enableSearch(m)
	return nil
}

func disableSearch(m *Model) tea.Cmd {
	m.textInput.Blur()
	m.Table.Focus()
	util.DefaultKeyMap.Play.SetEnabled(true)
	util.DefaultKeyMap.Pause.SetEnabled(true)
	util.DefaultKeyMap.SkipForward.SetEnabled(true)
	util.DefaultKeyMap.SkipBackward.SetEnabled(true)
	return nil
}

func enableSearch(m *Model) tea.Cmd {
	m.textInput.Focus()
	m.Table.Blur()
	util.DefaultKeyMap.Play.SetEnabled(false)
	util.DefaultKeyMap.Pause.SetEnabled(false)
	util.DefaultKeyMap.SkipForward.SetEnabled(false)
	util.DefaultKeyMap.SkipBackward.SetEnabled(false)
	return nil
}
