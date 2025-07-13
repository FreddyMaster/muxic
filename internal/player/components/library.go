package components

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"muxic/internal/util"
	"sync"
)

// Library is a singleton that holds all audio files
var (
	libraryInstance *Library
	once            sync.Once
)

type Library struct {
	Name  string
	Files []*util.AudioFile
}

// GetLibrary returns the singleton instance of the library
func GetLibrary() *Library {
	once.Do(func() {
		libraryInstance = &Library{
			Name:  "Music Library",
			Files: make([]*util.AudioFile, 0),
		}
	})
	return libraryInstance
}

// AddFile adds a file to the library if it doesn't already exist
func (l *Library) AddFile(file *util.AudioFile) bool {
	// Check if file already exists in library
	for _, f := range l.Files {
		if f.Path == file.Path {
			return false // File already exists
		}
	}
	l.Files = append(l.Files, file)
	return true
}

// GetFile returns a file by index
func (l *Library) GetFile(index int) (*util.AudioFile, error) {
	if index < 0 || index >= len(l.Files) {
		return nil, fmt.Errorf("index out of range")
	}
	return l.Files[index], nil
}

// RemoveFile removes a file from the library by index
func (l *Library) RemoveFile(index int) error {
	if index < 0 || index >= len(l.Files) {
		return fmt.Errorf("index out of range")
	}
	l.Files = append(l.Files[:index], l.Files[index+1:]...)
	return nil
}

// ToTableRows converts all files in the library to table rows
func (l *Library) ToTableRows() []table.Row {
	library := GetLibrary()
	rows := make([]table.Row, library.Length())
	for i, t := range library.Files {
		rows[i] = table.Row{
			t.Title,
			t.Artist,
			t.Album,
			t.Duration,
		}
	}
	return rows
}

// GetPaths returns all file paths in the library
func (l *Library) GetPaths() []string {
	paths := make([]string, len(l.Files))
	for i, file := range l.Files {
		paths[i] = file.Path
	}
	return paths
}

// Length returns the number of files in the library
func (l *Library) Length() int {
	return len(l.Files)
}

// Clear removes all files from the library
func (l *Library) Clear() {
	l.Files = make([]*util.AudioFile, 0)
}
