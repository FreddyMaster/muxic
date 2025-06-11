package util

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/dhowden/tag"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

// formatDuration formats a time.Duration as a string in the format "HH:MM:SS" or "MM:SS".
// If the duration is greater than 1 hour, the "HH" is included, otherwise it is omitted.
func formatDuration(d time.Duration) string {
	// Calculate the total number of seconds.
	totalSeconds := int(d.Seconds())

	// Calculate the hours, minutes and seconds.
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60

	// If the duration is greater than 1 hour, include the "HH".
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	// Otherwise, omit the "HH".
	return fmt.Sprintf("%02d:%02d", m, s)
}

// Add a cache for file metadata
var (
	metadataCache = make(map[string]struct {
		title, artist, album, duration string
	})
	cacheMutex sync.RWMutex
)

// readAudioMetadata extracts metadata from the audio file at the specified path.
// If metadata is missing, it provides default values.
// The function caches the results for faster access, and the cache is cleared when the program exits.
func readAudioMetadata(path, defaultName string) (string, string, string, string) {
	// Check if the file is in the cache
	cacheMutex.RLock()
	if cached, exists := metadataCache[path]; exists {
		cacheMutex.RUnlock()
		return cached.title, cached.artist, cached.album, cached.duration
	}
	cacheMutex.RUnlock()

	// Open file once for both metadata and duration
	f, err := os.Open(path)
	if err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}
	defer f.Close()

	// Get file info for modification time
	fileInfo, err := f.Stat()
	if err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}

	// Read metadata
	meta, err := tag.ReadFrom(f)
	if err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}

	// Get duration without reopening the file
	if _, err := f.Seek(0, 0); err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}

	duration := getFileDurationFromReader(f, fileInfo)

	// Get metadata with fallbacks
	title := defaultName
	if t := meta.Title(); t != "" {
		title = t
	}
	artist := "Unknown"
	if a := meta.Artist(); a != "" {
		artist = a
	}
	album := "Unknown"
	if a := meta.Album(); a != "" {
		album = a
	}

	// Cache the results
	cacheMutex.Lock()
	metadataCache[path] = struct {
		title, artist, album, duration string
	}{
		title:    title,
		artist:   artist,
		album:    album,
		duration: duration,
	}
	cacheMutex.Unlock()

	return title, artist, album, duration
}

// getFileDurationFromReader reads the duration of an audio file from the provided file.
// The file is read from the beginning, decoded, and then closed.
// It returns the duration of the audio file as a string in the format "HH:MM:SS".
func getFileDurationFromReader(f *os.File, fileInfo os.FileInfo) string {
	// Seek to the beginning of the file
	if _, err := f.Seek(0, 0); err != nil {
		return "0:00"
	}

	// Decode the file
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return "0:00"
	}
	defer func(streamer beep.StreamSeekCloser) {
		err := streamer.Close()
		if err != nil {
			fmt.Println("Error closing streamer:", err)
		}
	}(streamer)

	// Calculate the total samples
	totalSamples := streamer.Len()

	// Calculate the duration from the sample rate and the length of the streamer
	duration := format.SampleRate.D(totalSamples)

	// Return the duration as a string in the format "HH:MM:SS"
	return formatDuration(duration)
}

// GetAudioRows scans the specified directory for audio files, retrieves their metadata,
// and returns a slice of table rows and corresponding file paths.
func GetAudioRows(dir string) ([]table.Row, []string, error) {
	// Read directory entries.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}

	// Struct to hold the result of processing each file.
	type result struct {
		row   table.Row
		path  string
		index int
	}

	var wg sync.WaitGroup
	results := make(chan result, len(entries)) // Channel to collect results.
	var rows []table.Row
	var paths []string

	// Process files in parallel.
	for i, entry := range entries {
		if entry.IsDir() || !isAudioFile(entry.Name()) {
			continue
		}

		wg.Add(1)
		go func(idx int, entry os.DirEntry) {
			defer wg.Done()

			// Construct file path and read metadata.
			path := filepath.Join(dir, entry.Name())
			title, artist, album, duration := readAudioMetadata(path, entry.Name())

			// Send result to channel.
			results <- result{
				row:   table.Row{title, artist, album, duration},
				path:  path,
				index: idx,
			}
		}(i, entry)
	}

	// Close results channel when all workers are done.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results from channel.
	tempRows := make([]table.Row, len(entries))
	tempPaths := make([]string, len(entries))
	count := 0

	// Populate temporary slices with results.
	for res := range results {
		tempRows[res.index] = res.row
		tempPaths[res.index] = res.path
		count++
	}

	// Initialize slices for non-empty entries.
	rows = make([]table.Row, 0, count)
	paths = make([]string, 0, count)

	// Filter out empty entries.
	for i := 0; i < len(tempRows); i++ {
		if tempPaths[i] != "" {
			rows = append(rows, tempRows[i])
			paths = append(paths, tempPaths[i])
		}
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
