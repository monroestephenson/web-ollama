package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"web-ollama/internal/history"
)

// EnhancedDisplay provides a rich terminal UI with history panel
type EnhancedDisplay struct {
	width          int
	height         int
	historyWidth   int
	showThinking   bool
	thinkingBuffer strings.Builder
	responseBuffer strings.Builder
	startTime      time.Time
	tokenCount     int
	renderer       *glamour.TermRenderer
}

// NewEnhancedDisplay creates a new enhanced display
func NewEnhancedDisplay(showThinking bool) *EnhancedDisplay {
	width, height := getTerminalSize()

	// Create markdown renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-10),
	)

	return &EnhancedDisplay{
		width:        width,
		height:       height,
		historyWidth: width / 3, // Left 1/3 for history
		showThinking: showThinking,
		renderer:     renderer,
	}
}

// Color codes
const (
	colorReset      = "\033[0m"
	colorBold       = "\033[1m"
	colorDim        = "\033[2m"
	colorRed        = "\033[31m"
	colorGreen      = "\033[32m"
	colorYellow     = "\033[33m"
	colorBlue       = "\033[34m"
	colorMagenta    = "\033[35m"
	colorCyan       = "\033[36m"
	colorGray       = "\033[90m"
	colorBrightBlue = "\033[94m"
)

// ClearScreen clears the terminal
func (d *EnhancedDisplay) ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

// PrintWelcome displays enhanced welcome message
func (d *EnhancedDisplay) PrintWelcome(modelName string) {
	d.ClearScreen()
	fmt.Printf("%s%s╔══════════════════════════════════════════════════════════╗%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║                                                          ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║           web-ollama - AI with Web Search               ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║                                                          ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════════════════════╝%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("\n%s%sModel:%s %s\n", colorBold, colorGray, colorReset, modelName)
	fmt.Printf("%sCommands:%s /exit | /clear | /history | /files (list files for @reference)\n", colorGray, colorReset)
	fmt.Printf("%sFile Reference:%s Use @filename in queries (e.g., \"What does @main.go do?\")\n", colorGray, colorReset)
	fmt.Println()
}

// DrawHistoryPanel renders conversation history in left panel
func (d *EnhancedDisplay) DrawHistoryPanel(messages []history.Message) {
	// Disabled - user doesn't want history panel
	return
}

// PrintSeparator prints a visual separator
func (d *EnhancedDisplay) PrintSeparator() {
	line := strings.Repeat("─", min(d.width, 80))
	fmt.Printf("%s%s%s\n", colorDim, line, colorReset)
}

// PrintPrompt displays user input prompt
func (d *EnhancedDisplay) PrintPrompt() {
	fmt.Printf("\n%s%s❯%s ", colorBold, colorGreen, colorReset)
}

// PrintUserMessage displays a user message with timestamp
func (d *EnhancedDisplay) PrintUserMessage(content string, timestamp time.Time) {
	fmt.Printf("\n%s┌─ You · %s%s\n", colorGray, timestamp.Format("15:04:05"), colorReset)
	fmt.Printf("%s│%s %s\n", colorGray, colorReset, content)
	fmt.Printf("%s└%s\n", colorGray, colorReset)
}

// StartAssistantResponse initializes response tracking
func (d *EnhancedDisplay) StartAssistantResponse() {
	d.startTime = time.Now()
	d.tokenCount = 0
	d.thinkingBuffer.Reset()
	d.responseBuffer.Reset()

	fmt.Printf("\n%s┌─ Assistant · %s%s\n", colorGray, time.Now().Format("15:04:05"), colorReset)
}

// WriteThinking writes thinking tokens (dimmed)
func (d *EnhancedDisplay) WriteThinking(text string) {
	if d.showThinking {
		d.thinkingBuffer.WriteString(text)
		fmt.Printf("%s%s%s", colorDim, text, colorReset)
	}
}

// StartAnswer prints thinking section separator
func (d *EnhancedDisplay) StartAnswer() {
	if d.showThinking && d.thinkingBuffer.Len() > 0 {
		fmt.Printf("\n%s│%s\n%s│ ─── Answer ───%s\n%s│%s\n", colorGray, colorReset, colorGray, colorReset, colorGray, colorReset)
	}
	fmt.Printf("%s│%s ", colorGray, colorReset)
}

// WriteAnswer writes answer tokens (streams live, renders markdown at end)
func (d *EnhancedDisplay) WriteAnswer(text string) {
	d.responseBuffer.WriteString(text)
	d.tokenCount += len(strings.Fields(text))
	// Stream raw text in real-time for better UX
	fmt.Print(text)
}

// EndAssistantResponse finishes response and shows metadata
func (d *EnhancedDisplay) EndAssistantResponse(sourceURLs []string) {
	duration := time.Since(d.startTime)

	fmt.Println()
	fmt.Println()

	// Render the complete response as markdown for final display
	if d.responseBuffer.Len() > 0 && d.renderer != nil {
		fmt.Printf("%s│ Rendered:%s\n", colorGray, colorReset)
		rendered, err := d.renderer.Render(d.responseBuffer.String())
		if err == nil {
			// Indent each line
			for _, line := range strings.Split(strings.TrimRight(rendered, "\n"), "\n") {
				fmt.Printf("%s│%s %s\n", colorGray, colorReset, line)
			}
		}
	}

	fmt.Println()

	// Show sources if available
	if len(sourceURLs) > 0 {
		fmt.Printf("%s│%s\n", colorGray, colorReset)
		fmt.Printf("%s│ 📚 Sources:%s\n", colorGray, colorReset)
		for _, url := range sourceURLs {
			shortened := truncate(url, 60)
			fmt.Printf("%s│    • %s%s\n", colorGray, shortened, colorReset)
		}
	}

	// Show metadata
	fmt.Printf("%s│%s\n", colorGray, colorReset)
	fmt.Printf("%s│ ⏱️  %s · 📝 ~%d words%s\n",
		colorGray,
		formatDuration(duration),
		d.tokenCount,
		colorReset)

	fmt.Printf("%s└%s\n", colorGray, colorReset)
}

// PrintSearchActivity shows search progress
func (d *EnhancedDisplay) PrintSearchActivity(message string) {
	fmt.Printf("%s%s🔍 %s...%s\n", colorDim, colorCyan, message, colorReset)
}

// PrintInfo displays info message
func (d *EnhancedDisplay) PrintInfo(msg string) {
	fmt.Printf("%sℹ %s%s\n", colorCyan, msg, colorReset)
}

// PrintWarning displays warning message
func (d *EnhancedDisplay) PrintWarning(msg string) {
	fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
}

// PrintError displays error message
func (d *EnhancedDisplay) PrintError(err error) {
	fmt.Printf("%s✗ Error: %v%s\n", colorRed, err, colorReset)
}

// PrintSuccess displays success message
func (d *EnhancedDisplay) PrintSuccess(msg string) {
	fmt.Printf("%s✓ %s%s\n", colorGreen, msg, colorReset)
}

// PrintGoodbye displays goodbye message
func (d *EnhancedDisplay) PrintGoodbye() {
	fmt.Printf("\n%s%sThank you for using web-ollama! 👋%s\n", colorBold, colorCyan, colorReset)
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func getTerminalSize() (width, height int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 80, 24 // defaults
	}

	fmt.Sscanf(string(out), "%d %d", &height, &width)
	return
}
