package components

import (
	"github.com/charmbracelet/bubbles/table"
	"strings"
)

type Search struct {
	IsSearching bool // Whether search is active
}

type SearchIndex struct {
	rows  []table.Row
	index map[string][]int // maps search terms to row indices
}

// SearchResult contains both the row and its original index
type SearchResult struct {
	Row   table.Row
	Index int
}

func NewSearch() *Search {
	return &Search{
		IsSearching: false,
	}
}

func (si *SearchIndex) Search(query string) ([]table.Row, []int) {
	if query == "" {
		// Return all rows with their indices when query is empty
		indices := make([]int, len(si.rows))
		for i := range si.rows {
			indices[i] = i
		}
		return si.rows, indices
	}

	lowerQuery := strings.ToLower(query)
	queryWords := strings.Fields(lowerQuery)

	if len(queryWords) == 0 {
		indices := make([]int, len(si.rows))
		for i := range si.rows {
			indices[i] = i
		}
		return si.rows, indices
	}

	// Count occurrences of each row index
	rowCounts := make(map[int]int)
	for _, word := range queryWords {
		if indices, exists := si.index[word]; exists {
			for _, idx := range indices {
				rowCounts[idx]++
			}
		}
	}

	// Collect rows and their original indices that match all query words
	var resultRows []table.Row
	var resultIndices []int
	for idx, count := range rowCounts {
		if count == len(queryWords) {
			resultRows = append(resultRows, si.rows[idx])
			resultIndices = append(resultIndices, idx)
		}
	}

	return resultRows, resultIndices
}
