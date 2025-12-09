package terminal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadUserInput reads a line of input from the user
func ReadUserInput() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Trim whitespace and newline
	return strings.TrimSpace(input), nil
}

// FindMatchingFiles searches for files matching the partial path after @
func FindMatchingFiles(workingDir string, partial string) []string {
	matches := []string{}

	// Determine search directory and pattern
	searchDir := workingDir
	pattern := strings.ToLower(partial)

	if strings.Contains(partial, "/") {
		// If partial contains /, split into dir and pattern
		dir, file := filepath.Split(partial)
		searchDir = filepath.Join(workingDir, dir)
		pattern = strings.ToLower(file)
	}

	// Walk the search directory (limit depth to avoid slow searches)
	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path from working directory
		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return nil
		}

		// Skip the working directory itself
		if relPath == "." {
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// For files, check if they match
		if !info.IsDir() {
			relPathLower := strings.ToLower(relPath)

			// Match if: no pattern (show all), starts with pattern, or contains pattern
			isMatch := partial == "" ||
				strings.HasPrefix(relPathLower, pattern) ||
				strings.Contains(relPathLower, pattern) ||
				strings.Contains(strings.ToLower(info.Name()), pattern)

			if isMatch && len(matches) < 100 {
				matches = append(matches, relPath)
			}
		}

		// Limit depth to avoid scanning too deep
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > 4 {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return matches
	}

	return matches
}

// ShowFileSuggestions displays file suggestions when @ is detected
func ShowFileSuggestions(workingDir string, query string) {
	// Find all @ mentions in the query
	words := strings.Fields(query)
	for _, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			partial := strings.TrimPrefix(word, "@")
			partial = strings.Trim(partial, "\"'")

			matches := FindMatchingFiles(workingDir, partial)
			if len(matches) > 0 {
				fmt.Printf("\n💡 File suggestions for '@%s':\n", partial)
				for i, match := range matches {
					if i < 10 { // Show max 10 suggestions
						fmt.Printf("   @%s\n", match)
					}
				}
				fmt.Println()
			}
		}
	}
}
