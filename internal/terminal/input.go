package terminal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"
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

// ListenForESC listens for ESC key press in a goroutine and sends signal when pressed
// Returns a channel that will receive true when ESC is pressed
func ListenForESC() chan bool {
	escChan := make(chan bool, 1)

	go func() {
		// Save the current terminal state
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			// If we can't set raw mode, just close the channel and return
			close(escChan)
			return
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		// Read single bytes from stdin
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				break
			}

			// ESC key is byte 27
			if buf[0] == 27 {
				escChan <- true
				break
			}
		}

		close(escChan)
	}()

	return escChan
}

// StopESCListener restores terminal to normal mode
func StopESCListener() {
	// This will happen automatically when the goroutine exits
	// but we provide this function for consistency
}
