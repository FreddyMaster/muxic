package player

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"time"
)

// View renders the complete UI layout as a string.
func (m *Model) View() string {
	// Main content
	content := m.renderContent()

	// Player UI
	playerUI := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderPlayerInfo(),
	)

	// Status bar
	statusBar := m.renderStatusBar()

	// Final layout
	return lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		playerUI,
		statusBar,
	)
}

// renderContent renders the main content area based on the current view mode
func (m *Model) renderContent() string {
	switch m.viewMode {
	case ViewLibrary:
		return m.renderLibraryView()
	case ViewSearch:
		return m.renderSearchView()
	case ViewPlaylistTracks:
		return m.renderPlaylistView()
	case ViewQueue:
		return m.renderQueueView()
	default:
		return ""
	}
}

// renderTitledView is a helper to render a view with a title and content.
func (m *Model) renderTitledView(title string, content ...string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		MarginBottom(1)

	// Prepend the rendered title to the content strings.
	fullContent := append([]string{titleStyle.Render(title)}, content...)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		fullContent...,
	)
}

func (m *Model) renderLibraryView() string {
	if m.isLoading {
		return m.renderTitledView("Library", "\n  Loading music library...")
	}
	return m.renderTitledView("Library", m.LibraryTable.View())
}

// renderPlaylistView renders the playlist or playlist tracks view
func (m *Model) renderPlaylistView() string {
	title := "Playlists"
	if m.viewMode == ViewPlaylistTracks {
		title = "Playlist Tracks"
	}
	return m.renderTitledView(title, m.PlaylistTable[m.ActivePlaylistIndex].View())
}

func (m *Model) renderSearchView() string {
	return m.renderTitledView("Search", m.SearchInput.View(), m.SearchTable.View())
}

func (m *Model) renderQueueView() string {
	return m.renderTitledView("Queue", m.QueueTable.View())
}

// renderProgressBar renders the playback progress bar
func (m *Model) renderProgressBar() string {
	return lipgloss.NewStyle().
		MarginTop(1).
		Render(m.Progress.View())
}

func (m *Model) renderVolumeDisplay() string {
	volumeStyle := lipgloss.NewStyle().
		Bold(true).
		MarginLeft(1).
		MarginRight(1).
		Foreground(lipgloss.Color("250"))

	if m.AudioPlayer.Volume == nil {
		return volumeStyle.Render("Volume: 50%")
	}

	if m.AudioPlayer.Volume.Silent {
		return volumeStyle.Render("Volume: Muted")
	}

	// Convert the logarithmic volume to a percentage (approximate)
	// Assuming 0.0 is 100%, and it can go higher.
	volumePercent := m.AudioPlayer.GetVolume()
	volumeText := fmt.Sprintf("Volume: %.0f%%", volumePercent)

	return volumeStyle.Render(volumeText)
}

func (m *Model) renderCurrentTrackDisplay() string {
	trackStyle := lipgloss.NewStyle().
		Bold(true).
		MarginLeft(1).
		MarginRight(1).
		Foreground(lipgloss.Color("255"))

	if m.Queue.Current() == nil {
		return ""
	}

	trackText := fmt.Sprintf("%s", m.Queue.Current().Title)

	return trackStyle.Render(trackText)
}

func (m *Model) renderArtistDisplay() string {
	artistStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		MarginLeft(1).
		MarginRight(1).
		Align(lipgloss.Center)

	if m.Queue.Current() == nil {
		return ""
	}

	artistText := fmt.Sprintf(
		"%s - %s",
		m.Queue.Current().Artist,
		m.Queue.Current().Album,
	)

	return artistStyle.Render(artistText)
}

// renderStatusBar renders the status bar with view indicator and help text
func (m *Model) renderStatusBar() string {
	return lipgloss.NewStyle().
		Width(m.Width).
		Bold(true).
		MarginTop(1).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Render(fmt.Sprintf(" %s | Tab: Switch View | Q: Quit", m.viewMode))
}

func (m *Model) renderPlayedTime() string {
	playedTimeStyle := lipgloss.NewStyle().
		MarginRight(1).
		MarginLeft(1).
		Render(formatDuration(m.AudioPlayer.PlayedTime))

	return playedTimeStyle
}

func (m *Model) renderTotalTime() string {
	totalTimeStyle := lipgloss.NewStyle().
		MarginLeft(1).
		MarginRight(1).
		Render(formatDuration(m.AudioPlayer.TotalTime))

	return totalTimeStyle
}

func (m *Model) renderPlayerInfo() string {
	trackArtistBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderCurrentTrackDisplay(),
		m.renderArtistDisplay(),
	)

	progressDisplayBlock := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.renderPlayedTime(),
		m.renderProgressBar(),
		m.renderTotalTime(),
	)

	volumeDisplayBlock := lipgloss.JoinVertical(
		lipgloss.Right,
		m.renderVolumeDisplay(),
	)

	playedTimeStr := formatDuration(m.AudioPlayer.PlayedTime)
	totalTimeStr := formatDuration(m.AudioPlayer.TotalTime)

	// Calculate widths for each block
	leftWidth := lipgloss.Width(trackArtistBlock)
	rightWidth := lipgloss.Width(volumeDisplayBlock)
	centerWidth := m.Width - leftWidth - rightWidth
	progressBarWidth := centerWidth - lipgloss.Width(playedTimeStr) - lipgloss.Width(totalTimeStr) - 12
	m.Progress.Width = progressBarWidth

	// Style and align each block
	left := lipgloss.NewStyle().Width(leftWidth).Align(lipgloss.Left).MarginTop(1).Render(trackArtistBlock)
	center := lipgloss.NewStyle().Width(centerWidth).Align(lipgloss.Center).MarginTop(1).Render(progressDisplayBlock)
	right := lipgloss.NewStyle().Width(rightWidth).Align(lipgloss.Right).MarginTop(1).Render(volumeDisplayBlock)

	return lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		left,
		center,
		right,
	)
}

func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}
