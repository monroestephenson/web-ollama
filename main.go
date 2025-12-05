package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
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
	queryAnalyzer := analyzer.NewAnalyzer()
	searxngClient := searxng.NewClient(cfg.SearXNGURL, cfg.SearchTimeout)
	webCrawler := crawler.NewCrawler(cfg.CrawlTimeout, cfg.MaxCrawlers, cfg.MaxContentSize, cfg.UserAgent)
	ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaTimeout)

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

		// Skip empty queries
		if strings.TrimSpace(query) == "" {
			continue
		}

		// Display user message with timestamp
		now := time.Now()
		display.PrintUserMessage(query, now)

		// DON'T save user message yet - wait until after LLM response
		// to avoid duplicate query in context

		// Analyze query for search trigger
		var searchContext string
		var sourceURLs []string

		if cfg.AutoSearch {
			trigger := queryAnalyzer.AnalyzeQuery(query)

			if cfg.Verbose {
				display.PrintInfo(fmt.Sprintf("Search analysis: score=%d, reason=%s", trigger.Confidence, trigger.Reason))
			}

			if trigger.NeedsSearch {
				searchContext, sourceURLs = performSearch(ctx, display, searxngClient, webCrawler, query, cfg)
			}
		}

		// Build messages with context
		messages := buildMessages(historyMgr, query, searchContext)

		// Start assistant response
		display.StartAssistantResponse()

		// Stream response from Ollama with thinking support
		thinking, answer, err := ollamaClient.ChatWithCallbacks(ctx, ollama.ChatRequest{
			Model:    cfg.ModelName,
			Messages: messages,
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

	// New flags
	showThinking := flag.Bool("show-thinking", true, "Show model thinking process (default: true)")
	hideThinking := flag.Bool("hide-thinking", false, "Hide model thinking process")
	noSearch := flag.Bool("no-search", false, "Disable automatic web search")

	flag.Parse()

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
func buildMessages(historyMgr *history.Manager, currentQuery string, searchContext string) []ollama.Message {
	messages := []ollama.Message{}

	// Add system message
	systemPrompt := "You are a helpful AI assistant."
	if searchContext != "" {
		systemPrompt += " You have access to current web information to answer questions accurately. Cite sources when referencing specific information."
	}

	messages = append(messages, ollama.Message{
		Role:    "system",
		Content: systemPrompt,
	})

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

	// Add current query
	messages = append(messages, ollama.Message{
		Role:    "user",
		Content: currentQuery,
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
