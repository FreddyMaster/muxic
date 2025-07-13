package util

import (
	"fmt"
	"github.com/dhowden/tag"
	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AudioFile represents a single audio file with its metadata
type AudioFile struct {
	Title    string
	Artist   string
	Album    string
	Duration string
	Path     string
	FileName string
}

// OpenAudioFile opens an MP3 file and decodes it to return the audio streamer, format, and total samples.
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

// ReadAudioMetadata extracts metadata from the audio file at the specified path.
func ReadAudioMetadata(path, defaultName string) (string, string, string, string) {
	// Check if the file is in the cache
	cacheMutex.RLock()
	if cached, exists := metadataCache[path]; exists {
		cacheMutex.RUnlock()
		return cached.title, cached.artist, cached.album, cached.duration
	}
	cacheMutex.RUnlock()

	// Default values
	title := defaultName
	artist := "Unknown"
	album := "Unknown"
	duration := "0:00"

	// Open file for reading
	f, err := os.Open(path)
	if err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}

	// Get file info
	fileInfo, err := f.Stat()
	if err != nil {
		return defaultName, "Unknown", "Unknown", "0:00"
	}

	// Read metadata
	meta, err := tag.ReadFrom(f)
	if err == nil {
		if t := meta.Title(); t != "" {
			title = t
		}
		if a := meta.Artist(); a != "" {
			artist = a
		}
		if a := meta.Album(); a != "" {
			album = a
		}
	}

	// Get duration
	if _, err := f.Seek(0, 0); err == nil {
		duration = getFileDurationFromReader(f, fileInfo)
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
// It returns the duration of the audio file as a string in the format "HH:MM:SS".
// Note: The file should be opened and closed by the caller.
func getFileDurationFromReader(f *os.File, fileInfo os.FileInfo) string {
	// Seek to the beginning of the file in case it was read before
	if _, err := f.Seek(0, 0); err != nil {
		return "0:00"
	}

	// Decode the file
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return "0:00"
	}
	defer func() {
		err := streamer.Close()
		if err != nil {
			log.Println(err)
		}
	}()

	// Calculate the total samples
	totalSamples := streamer.Len()

	// Calculate the duration from the sample rate and the length of the streamer
	duration := format.SampleRate.D(totalSamples)

	// Return the duration as a string in the format "HH:MM:SS"
	return formatDuration(duration)
}

// GetAudioFiles scans the specified directory for audio files and returns a slice of AudioFile
func GetAudioFiles(dir string) ([]*AudioFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	type result struct {
		file  *AudioFile
		index int
		err   error
	}

	var wg sync.WaitGroup
	results := make(chan result, len(entries))
	var audioFiles []*AudioFile

	// Process files in parallel
	for i, entry := range entries {
		if entry.IsDir() || !isAudioFile(entry.Name()) {
			continue
		}

		wg.Add(1)
		go func(idx int, entry os.DirEntry) {
			defer wg.Done()

			path := filepath.Join(dir, entry.Name())
			title, artist, album, duration := ReadAudioMetadata(path, entry.Name())

			results <- result{
				file: &AudioFile{
					Title:    title,
					Artist:   artist,
					Album:    album,
					Duration: duration,
					Path:     path,
					FileName: entry.Name(),
				},
				index: idx,
			}
		}(i, entry)
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	tempFiles := make([]*AudioFile, len(entries))
	var count int

	for res := range results {
		if res.err != nil {
			continue
		}
		tempFiles[res.index] = res.file
		count++
	}

	// Filter out nil entries
	audioFiles = make([]*AudioFile, 0, count)
	for _, file := range tempFiles {
		if file != nil {
			audioFiles = append(audioFiles, file)
		}
	}

	return audioFiles, nil
}
