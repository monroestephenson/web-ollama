package terminal

import (
	"fmt"
	"os"
	"time"
)

// Display handles terminal output with colors and formatting
type Display struct {
	spinnerActive bool
	spinnerDone   chan bool
}

// NewDisplay creates a new display instance
func NewDisplay() *Display {
	return &Display{
		spinnerActive: false,
		spinnerDone:   make(chan bool),
	}
}

// Color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

// PrintWelcome displays the welcome message
func (d *Display) PrintWelcome(modelName string) {
	fmt.Printf("%s╔════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║   web-ollama - AI with Web Search     ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Printf("\n%sModel: %s%s\n", colorGray, modelName, colorReset)
	fmt.Printf("%sType your questions or '/exit' to quit%s\n\n", colorGray, colorReset)
}

// PrintGoodbye displays the goodbye message
func (d *Display) PrintGoodbye() {
	fmt.Printf("\n%sGoodbye! 👋%s\n", colorCyan, colorReset)
}

// PrintError displays an error message
func (d *Display) PrintError(err error) {
	fmt.Printf("%s✗ Error: %v%s\n", colorRed, err, colorReset)
}

// PrintInfo displays an info message
func (d *Display) PrintInfo(msg string) {
	fmt.Printf("%sℹ %s%s\n", colorCyan, msg, colorReset)
}

// PrintWarning displays a warning message
func (d *Display) PrintWarning(msg string) {
	fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
}

// PrintSuccess displays a success message
func (d *Display) PrintSuccess(msg string) {
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

// ShowSpinner displays a spinner with a message
func (d *Display) ShowSpinner(msg string) {
	if d.spinnerActive {
		d.StopSpinner()
	}

	d.spinnerActive = true
	d.spinnerDone = make(chan bool)

	go func() {
		spinnerChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-d.spinnerDone:
				// Clear the spinner line
				fmt.Printf("\r%s\r", clearLine())
				return
			default:
				fmt.Printf("\r%s%s %s%s", colorCyan, spinnerChars[i], msg, colorReset)
				i = (i + 1) % len(spinnerChars)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// StopSpinner stops the currently active spinner
func (d *Display) StopSpinner() {
	if d.spinnerActive {
		d.spinnerActive = false
		d.spinnerDone <- true
		time.Sleep(10 * time.Millisecond) // Give time for goroutine to clean up
	}
}

// WriteChunk writes a chunk of text without a newline (for streaming)
func (d *Display) WriteChunk(text string) {
	fmt.Print(text)
}

// WriteNewline writes a newline
func (d *Display) WriteNewline() {
	fmt.Println()
}

// PrintPrompt displays the user input prompt
func (d *Display) PrintPrompt() {
	fmt.Printf("\n%s> %s", colorGreen, colorReset)
}

// PrintAssistantPrefix prints the assistant response prefix
func (d *Display) PrintAssistantPrefix() {
	fmt.Printf("\n%sAssistant:%s ", colorBlue, colorReset)
}

// clearLine returns ANSI escape code to clear the current line
func clearLine() string {
	return "\033[2K"
}

// PrintSearchSources displays the sources that were searched
func (d *Display) PrintSearchSources(count int) {
	fmt.Printf("%s📚 Gathered information from %d sources%s\n", colorGray, count, colorReset)
}

// Cleanup ensures the display is in a good state before exit
func (d *Display) Cleanup() {
	d.StopSpinner()
}

// IsTerminal checks if stdout is a terminal
func IsTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
