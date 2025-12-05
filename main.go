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
)

func main() {
	// Set the GetEnv function for config
	config.GetEnv = os.Getenv

	// Parse command-line flags
	cfg := parseFlags()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize display
	display := terminal.NewDisplay()
	defer display.Cleanup()

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
		display.Cleanup()
		os.Exit(0)
	}()

	// Print welcome message
	display.PrintWelcome(cfg.ModelName)

	// Main conversation loop
	for {
		// Get user input
		display.PrintPrompt()
		query, err := terminal.ReadUserInput()
		if err != nil {
			break
		}

		// Check for exit command
		if query == "/exit" || query == "/quit" || query == "exit" || query == "quit" {
			break
		}

		// Skip empty queries
		if strings.TrimSpace(query) == "" {
			continue
		}

		// Save user message
		userMsg := history.Message{
			Role:      "user",
			Content:   query,
			Timestamp: time.Now(),
		}
		historyMgr.AddMessage(userMsg)

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

		// Stream response from Ollama
		display.PrintAssistantPrefix()
		response, err := ollamaClient.Chat(ctx, ollama.ChatRequest{
			Model:    cfg.ModelName,
			Messages: messages,
		}, func(chunk string) {
			display.WriteChunk(chunk)
		})

		if err != nil {
			display.WriteNewline()
			display.PrintError(err)
			continue
		}

		display.WriteNewline()

		// Save assistant message
		assistantMsg := history.Message{
			Role:      "assistant",
			Content:   response,
			Timestamp: time.Now(),
		}

		if len(sourceURLs) > 0 {
			assistantMsg.Metadata = &history.Metadata{
				SearchPerformed: true,
				SourceURLs:      sourceURLs,
			}
		}

		historyMgr.AddMessage(assistantMsg)
	}

	// Print goodbye message
	display.PrintGoodbye()
}

// parseFlags parses command-line flags and returns a config
func parseFlags() *config.Config {
	cfg := config.NewConfig()

	flag.StringVar(&cfg.ModelName, "model", cfg.ModelName, "Ollama model name")
	flag.StringVar(&cfg.OllamaURL, "ollama-url", cfg.OllamaURL, "Ollama API URL")
	flag.StringVar(&cfg.SearXNGURL, "searxng-url", cfg.SearXNGURL, "SearXNG instance URL")
	flag.BoolVar(&cfg.AutoSearch, "auto-search", cfg.AutoSearch, "Enable automatic web search")
	flag.BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "Enable verbose logging")
	flag.IntVar(&cfg.MaxResults, "max-results", cfg.MaxResults, "Maximum search results to crawl")

	flag.Parse()

	// Handle --no-search flag
	noSearch := flag.Bool("no-search", false, "Disable automatic web search")
	if *noSearch {
		cfg.AutoSearch = false
	}

	return cfg
}

// checkModel verifies that the specified model exists
func checkModel(client *ollama.Client, modelName string, display *terminal.Display) error {
	models, err := client.ListModels()
	if err != nil {
		display.PrintError(fmt.Errorf("failed to list models: %w", err))
		return err
	}

	// Check if model exists
	for _, m := range models {
		if m == modelName {
			return nil
		}
	}

	// Model not found
	display.PrintError(fmt.Errorf("model '%s' not found", modelName))
	display.PrintInfo("Available models:")
	for _, m := range models {
		fmt.Printf("  - %s\n", m)
	}
	display.PrintInfo(fmt.Sprintf("Pull the model with: ollama pull %s", modelName))

	return fmt.Errorf("model not found")
}

// performSearch executes web search and crawling
func performSearch(ctx context.Context, display *terminal.Display, searxngClient *searxng.Client, webCrawler *crawler.Crawler, query string, cfg *config.Config) (string, []string) {
	// Show searching spinner
	display.ShowSpinner("Searching the web...")

	// Query SearXNG
	results, err := searxngClient.Search(ctx, query, cfg.MaxResults)
	if err != nil {
		display.StopSpinner()
		display.PrintWarning(fmt.Sprintf("Search failed: %v", err))
		return "", nil
	}

	if len(results) == 0 {
		display.StopSpinner()
		display.PrintInfo("No search results found")
		return "", nil
	}

	// Extract URLs
	urls := make([]string, len(results))
	for i, result := range results {
		urls[i] = result.URL
	}

	// Update spinner for crawling
	display.ShowSpinner(fmt.Sprintf("Crawling %d URLs...", len(urls)))

	// Crawl URLs
	crawlResults := webCrawler.CrawlURLs(ctx, urls)

	display.StopSpinner()

	// Count successful crawls
	successCount := 0
	for _, result := range crawlResults {
		if result.Error == nil {
			successCount++
		}
	}

	if successCount > 0 {
		display.PrintSearchSources(successCount)
	}

	// Build context
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
