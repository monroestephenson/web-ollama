package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"web-ollama/internal/analyzer"
	"web-ollama/internal/config"
	"web-ollama/internal/crawler"
	"web-ollama/internal/history"
	"web-ollama/internal/ollama"
	"web-ollama/internal/searxng"
	"web-ollama/internal/terminal"
	"web-ollama/internal/ui"
)

func main() {
	// Set the GetEnv function for config
	config.GetEnv = os.Getenv

	// Parse command-line flags
	cfg, showThinking := parseFlags()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize enhanced display
	display := ui.NewEnhancedDisplay(showThinking)

	// Initialize components
	historyMgr := history.NewManager(cfg.HistoryPath, cfg.MaxHistorySize)
	searxngClient := searxng.NewClient(cfg.SearXNGURL, cfg.SearchTimeout)
	webCrawler := crawler.NewCrawler(cfg.CrawlTimeout, cfg.MaxCrawlers, cfg.MaxContentSize, cfg.UserAgent)
	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaTimeout)

	// LLM-based query analyzer (uses same model)
	llmAnalyzer := analyzer.NewLLMAnalyzer(ollamaClient, cfg.ModelName)

	// Health checks
	if err := ollamaClient.HealthCheck(); err != nil {
		display.PrintError(err)
		display.PrintInfo("Make sure Ollama is running: ollama serve")
		os.Exit(1)
	}

	// Check if model exists
	if err := checkModel(ollamaClient, cfg.ModelName, display); err != nil {
		os.Exit(1)
	}

	// SearXNG health check (non-fatal)
	if err := searxngClient.HealthCheck(); err != nil {
		display.PrintWarning(fmt.Sprintf("SearXNG check failed: %v", err))
		display.PrintInfo("Web search will be disabled. Start SearXNG or use --no-search flag.")
		cfg.AutoSearch = false
	}

	// Load conversation history
	if err := historyMgr.Load(); err != nil {
		display.PrintWarning(fmt.Sprintf("Failed to load history: %v", err))
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		display.PrintInfo("\nShutting down gracefully...")
		display.PrintInfo("Stopping model to free up RAM...")
		if err := ollamaClient.StopModel(cfg.ModelName); err != nil {
			display.PrintWarning(fmt.Sprintf("Failed to stop model: %v", err))
		} else {
			display.PrintSuccess("Model stopped successfully")
		}
		cancel()
		os.Exit(0)
	}()

	// Print welcome message
	display.PrintWelcome(cfg.ModelName)

	// Main conversation loop
	for {
		// Show recent history
		recentMessages := historyMgr.GetRecentMessages(10)
		display.DrawHistoryPanel(recentMessages)

		// Get user input
		display.PrintPrompt()
		query, err := terminal.ReadUserInput()
		if err != nil {
			break
		}

		// Handle commands
		if query == "/exit" || query == "/quit" || query == "exit" || query == "quit" {
			break
		}
		if query == "/clear" {
			display.ClearScreen()
			display.PrintWelcome(cfg.ModelName)
			continue
		}
		if query == "/history" {
			displayFullHistory(historyMgr, display)
			continue
		}
		if query == "/files" {
			workingDir, _ := os.Getwd()
			if workingDir == "" {
				workingDir = "."
			}
			displayAvailableFiles(workingDir, display)
			continue
		}

		// Skip empty queries
		if strings.TrimSpace(query) == "" {
			continue
		}

		// Display user message with timestamp
		now := time.Now()
		display.PrintUserMessage(query, now)

		// DON'T save user message yet - wait until after LLM response
		// to avoid duplicate query in context

		// Extract and read file references from query
		fileRefs := extractFileReferences(query)
		var fileReferences []FileReference
		var fileContext string

		if len(fileRefs) > 0 {
			workingDir, err := os.Getwd()
			if err != nil {
				display.PrintWarning(fmt.Sprintf("Failed to get working directory: %v", err))
				workingDir = "."
			}

			fileReferences = readFileReferences(fileRefs, workingDir)
			fileContext = buildFileContext(fileReferences)

			// Display which files were loaded
			successCount := 0
			for _, ref := range fileReferences {
				if ref.Error == nil {
					successCount++
				}
			}

			if successCount > 0 {
				display.PrintSuccess(fmt.Sprintf("Loaded %d file(s): %s", successCount, strings.Join(fileRefs, ", ")))
				if cfg.Verbose {
					display.PrintInfo(fmt.Sprintf("File context size: %d characters", len(fileContext)))
				}
			}

			// Display errors for failed file loads with suggestions
			for _, ref := range fileReferences {
				if ref.Error != nil {
					display.PrintWarning(fmt.Sprintf("Failed to load @%s: %v", ref.Path, ref.Error))

					// Show suggestions for similar files
					matches := terminal.FindMatchingFiles(workingDir, ref.Path)
					if len(matches) > 0 {
						fmt.Printf("   💡 Did you mean:\n")
						for i, match := range matches {
							if i < 5 { // Show max 5 suggestions
								fmt.Printf("      @%s\n", match)
							}
						}
					}
				}
			}
		}

		// Analyze query for search trigger using LLM
		var searchContext string
		var sourceURLs []string

		if cfg.AutoSearch {
			// Strip file references from query before search analysis
			// to avoid confusing @filename with @username mentions
			queryForAnalysis := query
			for _, ref := range fileRefs {
				queryForAnalysis = strings.ReplaceAll(queryForAnalysis, "@"+ref, ref)
			}

			display.PrintInfo("Analyzing query...")
			decision, err := llmAnalyzer.AnalyzeWithLLM(ctx, queryForAnalysis)
			if err != nil {
				display.PrintWarning(fmt.Sprintf("Analysis failed: %v", err))
			} else {
				if decision.NeedsSearch {
					// Use the LLM's optimized search queries
					searchQueries := decision.SearchQueries
					if len(searchQueries) == 0 {
						searchQueries = []string{query} // Fallback to original
					}
					if cfg.Verbose {
						if len(searchQueries) == 1 {
							display.PrintInfo(fmt.Sprintf("Search query: \"%s\" (Reason: %s)", searchQueries[0], decision.Reason))
						} else {
							display.PrintInfo(fmt.Sprintf("Search queries: %v (Reason: %s)", searchQueries, decision.Reason))
						}
					}
					searchContext, sourceURLs = performMultiSearch(ctx, display, searxngClient, webCrawler, searchQueries, cfg)
				} else if cfg.Verbose {
					display.PrintInfo(fmt.Sprintf("No search needed: %s", decision.Reason))
				}
			}
		}

		// Build messages with context
		if cfg.Verbose && fileContext != "" {
			display.PrintInfo(fmt.Sprintf("Sending %d chars of file context to LLM", len(fileContext)))
		}
		messages := buildMessages(historyMgr, query, searchContext, fileContext)

		// Start assistant response
		display.StartAssistantResponse()

		// Stream response from Ollama with thinking support
		thinking, answer, err := ollamaClient.ChatWithCallbacks(ctx, ollama.ChatRequest{
			Model:    cfg.ModelName,
			Messages: messages,
			Options: map[string]interface{}{
				"num_ctx": 32768, // Set context window to 32K tokens (enough for file references)
			},
		}, ollama.StreamCallbacks{
			OnThinking: func(chunk string) {
				display.WriteThinking(chunk)
			},
			OnAnswer: func(chunk string) {
				display.WriteAnswer(chunk)
			},
			OnDone: func() {
				display.StartAnswer()
			},
		})

		if err != nil {
			display.PrintError(err)
			continue
		}

		// End response with metadata
		display.EndAssistantResponse(sourceURLs)

		// NOW save both user and assistant messages to history
		userMsg := history.Message{
			Role:      "user",
			Content:   query,
			Timestamp: now,
		}
		historyMgr.AddMessage(userMsg)

		assistantMsg := history.Message{
			Role:      "assistant",
			Content:   answer,
			Timestamp: time.Now(),
		}

		if len(sourceURLs) > 0 {
			assistantMsg.Metadata = &history.Metadata{
				SearchPerformed: true,
				SourceURLs:      sourceURLs,
			}
		}

		// Store thinking separately if available (for future reference)
		if thinking != "" && cfg.Verbose {
			// Could save thinking to a separate field in future
			_ = thinking
		}

		historyMgr.AddMessage(assistantMsg)
	}

	// Stop the model before exiting
	display.PrintInfo("Stopping model to free up RAM...")
	if err := ollamaClient.StopModel(cfg.ModelName); err != nil {
		display.PrintWarning(fmt.Sprintf("Failed to stop model: %v", err))
	}

	// Print goodbye message
	display.PrintGoodbye()
}

// parseFlags parses command-line flags with thinking option
func parseFlags() (*config.Config, bool) {
	cfg := config.NewConfig()

	flag.StringVar(&cfg.ModelName, "model", cfg.ModelName, "Ollama model name")
	flag.StringVar(&cfg.OllamaURL, "ollama-url", cfg.OllamaURL, "Ollama API URL")
	flag.StringVar(&cfg.SearXNGURL, "searxng-url", cfg.SearXNGURL, "SearXNG instance URL")
	flag.BoolVar(&cfg.AutoSearch, "auto-search", cfg.AutoSearch, "Enable automatic web search")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Enable verbose logging")
	flag.IntVar(&cfg.MaxResults, "max-results", cfg.MaxResults, "Maximum search results to crawl")

	// Timeout flag (in seconds)
	timeoutSeconds := flag.Int("timeout", 600, "Ollama request timeout in seconds (default: 600)")

	// New flags
	showThinking := flag.Bool("show-thinking", true, "Show model thinking process (default: true)")
	hideThinking := flag.Bool("hide-thinking", false, "Hide model thinking process")
	noSearch := flag.Bool("no-search", false, "Disable automatic web search")

	flag.Parse()

	// Apply timeout
	cfg.OllamaTimeout = time.Duration(*timeoutSeconds) * time.Second

	if *noSearch {
		cfg.AutoSearch = false
	}

	// Hide thinking takes precedence if specified
	if *hideThinking {
		return cfg, false
	}

	return cfg, *showThinking
}

// checkModel verifies that the specified model exists
func checkModel(client *ollama.Client, modelName string, display *ui.EnhancedDisplay) error {
	models, err := client.ListModels()
	if err != nil {
		display.PrintError(fmt.Errorf("failed to list models: %w", err))
		return err
	}

	for _, m := range models {
		if m == modelName {
			return nil
		}
	}

	display.PrintError(fmt.Errorf("model '%s' not found", modelName))
	display.PrintInfo("Available models:")
	for _, m := range models {
		fmt.Printf("  - %s\n", m)
	}
	display.PrintInfo(fmt.Sprintf("Pull the model with: ollama pull %s", modelName))

	return fmt.Errorf("model not found")
}

// performSearch executes web search with enhanced display
func performSearch(ctx context.Context, display *ui.EnhancedDisplay, searxngClient *searxng.Client, webCrawler *crawler.Crawler, query string, cfg *config.Config) (string, []string) {
	display.PrintSearchActivity("Searching the web")

	results, err := searxngClient.Search(ctx, query, cfg.MaxResults)
	if err != nil {
		display.PrintWarning(fmt.Sprintf("Search failed: %v", err))
		return "", nil
	}

	if len(results) == 0 {
		display.PrintInfo("No search results found")
		return "", nil
	}

	urls := make([]string, len(results))
	for i, result := range results {
		urls[i] = result.URL
	}

	display.PrintSearchActivity(fmt.Sprintf("Crawling %d URLs", len(urls)))

	crawlResults := webCrawler.CrawlURLs(ctx, urls)

	successCount := 0
	for _, result := range crawlResults {
		if result.Error == nil {
			successCount++
		}
	}

	if successCount > 0 {
		display.PrintSuccess(fmt.Sprintf("Gathered information from %d sources", successCount))
	}

	searchContext := buildSearchContext(crawlResults)
	return searchContext, urls
}

// performMultiSearch executes multiple web searches and aggregates results
func performMultiSearch(ctx context.Context, display *ui.EnhancedDisplay, searxngClient *searxng.Client, webCrawler *crawler.Crawler, queries []string, cfg *config.Config) (string, []string) {
	if len(queries) == 1 {
		return performSearch(ctx, display, searxngClient, webCrawler, queries[0], cfg)
	}

	display.PrintSearchActivity(fmt.Sprintf("Performing %d web searches", len(queries)))

	allCrawlResults := []crawler.CrawlResult{}
	allURLs := []string{}
	seenURLs := make(map[string]bool)

	// Perform each search
	for i, query := range queries {
		if cfg.Verbose {
			display.PrintSearchActivity(fmt.Sprintf("Search %d/%d: \"%s\"", i+1, len(queries), query))
		}

		results, err := searxngClient.Search(ctx, query, cfg.MaxResults)
		if err != nil {
			display.PrintWarning(fmt.Sprintf("Search %d failed: %v", i+1, err))
			continue
		}

		if len(results) == 0 {
			if cfg.Verbose {
				display.PrintInfo(fmt.Sprintf("Search %d: No results found", i+1))
			}
			continue
		}

		// Collect unique URLs
		urls := []string{}
		for _, result := range results {
			if !seenURLs[result.URL] {
				urls = append(urls, result.URL)
				seenURLs[result.URL] = true
				allURLs = append(allURLs, result.URL)
			}
		}

		if len(urls) > 0 {
			// Crawl URLs for this search
			crawlResults := webCrawler.CrawlURLs(ctx, urls)
			allCrawlResults = append(allCrawlResults, crawlResults...)
		}
	}

	// Count successful crawls
	successCount := 0
	for _, result := range allCrawlResults {
		if result.Error == nil {
			successCount++
		}
	}

	if successCount > 0 {
		display.PrintSuccess(fmt.Sprintf("Gathered information from %d sources across %d searches", successCount, len(queries)))
	} else {
		display.PrintWarning("No information gathered from searches")
	}

	searchContext := buildSearchContext(allCrawlResults)
	return searchContext, allURLs
}

// buildSearchContext formats crawled content for LLM
func buildSearchContext(results []crawler.CrawlResult) string {
	var sb strings.Builder

	sb.WriteString("# Web Search Results\n\n")
	sb.WriteString("The following information was retrieved from the web:\n\n")

	sourceNum := 1
	for _, result := range results {
		if result.Error != nil {
			continue // Skip failed crawls
		}

		if result.Content == "" {
			continue // Skip empty content
		}

		sb.WriteString(fmt.Sprintf("## Source %d: %s\n", sourceNum, result.Title))
		sb.WriteString(fmt.Sprintf("URL: %s\n\n", result.URL))
		sb.WriteString(result.Content)
		sb.WriteString("\n\n---\n\n")

		sourceNum++
	}

	return sb.String()
}

// buildMessages constructs the message array for Ollama
func buildMessages(historyMgr *history.Manager, currentQuery string, searchContext string, fileContext string) []ollama.Message {
	messages := []ollama.Message{}

	// Add system message
	systemPrompt := "You are a helpful AI assistant."
	if searchContext != "" {
		systemPrompt += " You have access to current web information to answer questions accurately. Cite sources when referencing specific information."
	}
	if fileContext != "" {
		systemPrompt += " The user has provided file contents that you MUST read and analyze carefully. Base your answer on the ACTUAL contents of the files provided, not on assumptions or general knowledge."
	}

	messages = append(messages, ollama.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// File context will be prepended to the current query instead of being a separate message

	// Add search context if available
	if searchContext != "" {
		messages = append(messages, ollama.Message{
			Role:    "user",
			Content: searchContext,
		})
		messages = append(messages, ollama.Message{
			Role:    "assistant",
			Content: "I've reviewed the web search results and I'm ready to answer your question based on this information.",
		})
	}

	// Add recent conversation history (last 10 messages, excluding current)
	recentMessages := historyMgr.GetRecentMessages(10)
	for _, msg := range recentMessages {
		messages = append(messages, ollama.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add current query (with file context prepended if available)
	finalQuery := currentQuery
	if fileContext != "" {
		// Prepend file contents directly to the query for better context
		finalQuery = fileContext + "\n\n" + currentQuery
		// Debug output would go here but we can't print from this function
	}

	messages = append(messages, ollama.Message{
		Role:    "user",
		Content: finalQuery,
	})

	return messages
}

// displayFullHistory shows all conversation history
func displayFullHistory(historyMgr *history.Manager, display *ui.EnhancedDisplay) {
	session := historyMgr.GetCurrentSession()
	if session == nil || len(session.Messages) == 0 {
		display.PrintInfo("No conversation history yet")
		return
	}

	display.PrintSeparator()
	fmt.Println("Full Conversation History")
	display.PrintSeparator()

	for _, msg := range session.Messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		if msg.Role == "user" {
			fmt.Printf("\n[%s] You:\n%s\n", timestamp, msg.Content)
		} else {
			fmt.Printf("\n[%s] Assistant:\n%s\n", timestamp, msg.Content)
			if msg.Metadata != nil && len(msg.Metadata.SourceURLs) > 0 {
				fmt.Println("Sources:")
				for _, url := range msg.Metadata.SourceURLs {
					fmt.Printf("  - %s\n", url)
				}
			}
		}
	}

	display.PrintSeparator()
}

// displayAvailableFiles shows all files that can be referenced with @
func displayAvailableFiles(workingDir string, display *ui.EnhancedDisplay) {
	display.PrintSeparator()
	fmt.Println("Available Files for @ Reference")
	display.PrintSeparator()

	matches := terminal.FindMatchingFiles(workingDir, "")
	if len(matches) == 0 {
		display.PrintInfo("No files found in current directory")
		return
	}

	fmt.Printf("\nFound %d files:\n\n", len(matches))
	for _, match := range matches {
		fmt.Printf("  @%s\n", match)
	}
	fmt.Println()
	display.PrintInfo("Usage: Include @filename in your query to reference a file")
	display.PrintInfo("Example: What does @main.go do?")
	display.PrintSeparator()
}

// FileReference represents a file mentioned in the query
type FileReference struct {
	Path    string
	Content string
	Error   error
}

// extractFileReferences finds all @filename mentions in the query
func extractFileReferences(query string) []string {
	// Match @filename or @path/to/filename pattern
	// Supports: @file.txt, @src/main.go, @"file with spaces.txt"
	re := regexp.MustCompile(`@"([^"]+)"|@([^\s]+)`)
	matches := re.FindAllStringSubmatch(query, -1)

	files := []string{}
	seen := make(map[string]bool)

	for _, match := range matches {
		var filename string
		if match[1] != "" {
			// Quoted filename (for files with spaces)
			filename = match[1]
		} else {
			// Unquoted filename
			filename = match[2]
		}

		if !seen[filename] {
			files = append(files, filename)
			seen[filename] = true
		}
	}

	return files
}

// readFileReferences reads the content of referenced files
func readFileReferences(files []string, workingDir string) []FileReference {
	references := make([]FileReference, 0, len(files))

	for _, file := range files {
		ref := FileReference{Path: file}

		// Resolve relative paths from working directory
		fullPath := file
		if !filepath.IsAbs(file) {
			fullPath = filepath.Join(workingDir, file)
		}

		// Read file content
		content, err := os.ReadFile(fullPath)
		if err != nil {
			ref.Error = err
		} else {
			ref.Content = string(content)
		}

		references = append(references, ref)
	}

	return references
}

// buildFileContext formats file references for LLM
func buildFileContext(references []FileReference) string {
	if len(references) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("=== FILE CONTENTS FOR ANALYSIS ===\n\n")

	for _, ref := range references {
		if ref.Error != nil {
			sb.WriteString(fmt.Sprintf("File: %s - Error: %v\n\n", ref.Path, ref.Error))
			continue
		}

		sb.WriteString(fmt.Sprintf("File path: %s\n", ref.Path))
		sb.WriteString("--- BEGIN FILE CONTENTS ---\n")
		sb.WriteString(ref.Content)
		sb.WriteString("\n--- END FILE CONTENTS ---\n\n")
	}

	sb.WriteString("=== END OF FILE CONTENTS ===\n\n")
	sb.WriteString("Please analyze the above file contents carefully and answer the user's question based on what you see in the actual file.\n\n")
	return sb.String()
}
