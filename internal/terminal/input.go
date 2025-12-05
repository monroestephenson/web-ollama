package terminal

import (
	"bufio"
	"os"
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
