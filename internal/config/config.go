package config

import (
	"fmt"
	"time"
)

// Config holds all application configuration
type Config struct {
	// Ollama settings
	OllamaURL     string
	ModelName     string
	OllamaTimeout time.Duration

	// SearXNG settings
	SearXNGURL    string
	SearchTimeout time.Duration
	MaxResults    int

	// Crawler settings
	CrawlTimeout   time.Duration
	MaxCrawlers    int
	MaxContentSize int64
	UserAgent      string

	// History settings
	HistoryPath    string
	MaxHistorySize int

	// Feature flags
	AutoSearch bool
	Verbose    bool
}

// NewConfig creates a new configuration with default values
func NewConfig() *Config {
	return &Config{
		// Ollama defaults
		OllamaURL:     "http://localhost:11434",
		ModelName:     "deepseek-r1:8b",
		OllamaTimeout: 120 * time.Second,

		// SearXNG defaults
		SearXNGURL:    "http://localhost:9090",
		SearchTimeout: 10 * time.Second,
		MaxResults:    5,

		// Crawler defaults
		CrawlTimeout:   15 * time.Second,
		MaxCrawlers:    5,
		MaxContentSize: 5 * 1024 * 1024, // 5 MB
		UserAgent:      "web-ollama/1.0",

		// History defaults
		HistoryPath:    expandHome("~/.web-ollama/history.json"),
		MaxHistorySize: 10,

		// Feature flags
		AutoSearch: true,
		Verbose:    false,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.OllamaURL == "" {
		return fmt.Errorf("ollama URL cannot be empty")
	}
	if c.ModelName == "" {
		return fmt.Errorf("model name cannot be empty")
	}
	if c.MaxResults < 1 || c.MaxResults > 10 {
		return fmt.Errorf("max results must be between 1 and 10")
	}
	if c.MaxCrawlers < 1 {
		return fmt.Errorf("max crawlers must be at least 1")
	}
	return nil
}

// expandHome expands the ~ in file paths to the user's home directory
func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir := getHomeDir()
		return homeDir + path[1:]
	}
	return path
}

// getHomeDir returns the user's home directory
func getHomeDir() string {
	if home := GetEnv("HOME"); home != "" {
		return home
	}
	// Fallback for Windows
	if home := GetEnv("USERPROFILE"); home != "" {
		return home
	}
	return "."
}

// GetEnv is a wrapper around os.Getenv for easier testing
var GetEnv = func(key string) string {
	// Will be replaced with os.Getenv in main
	return ""
}
