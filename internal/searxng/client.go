package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// Client handles communication with SearXNG
type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new SearXNG client
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Search performs a web search and returns the top N results
func (c *Client) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	// Build URL with query parameters
	searchURL := fmt.Sprintf("%s/search", c.baseURL)
	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")

	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "web-ollama/1.0")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("SearXNG returned 403 Forbidden. JSON API may not be enabled. Check settings.yml for 'formats: [html, json]'")
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SearXNG returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Sort by score (highest first)
	sort.Slice(searchResp.Results, func(i, j int) bool {
		return searchResp.Results[i].Score > searchResp.Results[j].Score
	})

	// Return top N results
	if len(searchResp.Results) > maxResults {
		return searchResp.Results[:maxResults], nil
	}

	return searchResp.Results, nil
}

// HealthCheck verifies that SearXNG is accessible
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testURL := fmt.Sprintf("%s/search?q=test&format=json", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("User-Agent", "web-ollama/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SearXNG is unreachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 {
		return fmt.Errorf("SearXNG API access forbidden. Check settings.yml to enable JSON format")
	}

	if resp.StatusCode >= 500 {
		return fmt.Errorf("SearXNG returned server error: %d", resp.StatusCode)
	}

	return nil
}
