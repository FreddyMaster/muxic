package util

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/dhowden/tag"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// openAudioFile opens an MP3 file and decodes it to return the audio streamer, format, and total samples.
func OpenAudioFile(path string) (beep.StreamSeekCloser, beep.Format, int, error) {
	// Open file.
	f, err := os.Open(path)
	if err != nil {
		return nil, beep.Format{}, 0, err
	}
	// Decode MP3 file.
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		err := f.Close()
		if err != nil {
			return nil, beep.Format{}, 0, err
		}
		return nil, beep.Format{}, 0, err
	}
	totalSamples := streamer.Len()
	return streamer, format, totalSamples, nil
}

// isAudioFile checks if a file has an .mp3 extension (case-insensitive).
func isAudioFile(name string) bool {
	const ext = ".mp3"
	return strings.HasSuffix(strings.ToLower(name), ext)
}

// readAudioMetadata extracts metadata from the audio file at the specified path.
// If metadata is missing, it provides default values.
func readAudioMetadata(path string, defaultName string) (title, artist, album, year string) {
	title = defaultName
	artist = "Unknown"
	album = "Unknown"
	year = ""
	// Open the file to read metadata.
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Println("Error closing file:", err)
		}
	}(f)
	// Read the metadata using the tag library.
	meta, err := tag.ReadFrom(f)
	if err != nil {
		return
	}
	if meta.Title() != "" {
		title = meta.Title()
	}
	if meta.Artist() != "" {
		artist = meta.Artist()
	}
	if meta.Album() != "" {
		album = meta.Album()
	}
	if meta.Year() > 0 {
		year = fmt.Sprintf("%d", meta.Year())
	}
	return
}

// getAudioRows scans the given directory for MP3 files and prepares rows and paths for the UI table.
func GetAudioRows(dir string) ([]table.Row, []string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var rows []table.Row

	var paths []string
	counter := 1

	// Iterate over directory entries.
	for _, entry := range entries {
		// Skip directories or non-audio files.
		if entry.IsDir() || !isAudioFile(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		// Read metadata for the audio file.
		title, artist, album, year := readAudioMetadata(path, entry.Name())
		// Append a new row for this audio file.
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", counter),
			title,
			artist,
			album,
			year,
		})
		// Save the full file path.
		paths = append(paths, path)
		counter++
	}
	return rows, paths, nil
}

// NewAudioCtrl creates a new beep.Ctrl for the given streamer.
func NewAudioCtrl(streamer beep.Streamer) *beep.Ctrl {
	return &beep.Ctrl{
		Streamer: streamer,
		Paused:   false,
	}
}

func NewVolumeCtrl(streamer beep.Streamer) *effects.Volume {
	return &effects.Volume{
		Streamer: streamer,
		Base:     2, // Exponential scale base
		Volume:   0, // Default volume (2^0 = 1x)
		Silent:   false,
	}
}
