package crawler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CrawlResult represents the result of crawling a single URL
type CrawlResult struct {
	URL      string
	Title    string
	Content  string
	Error    error
	Duration time.Duration
}

// Crawler handles web page crawling
type Crawler struct {
	httpClient     *http.Client
	timeout        time.Duration
	maxSize        int64
	userAgent      string
	maxWorkers     int
}

// NewCrawler creates a new crawler instance
func NewCrawler(timeout time.Duration, maxWorkers int, maxSize int64, userAgent string) *Crawler {
	return &Crawler{
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		timeout:    timeout,
		maxSize:    maxSize,
		userAgent:  userAgent,
		maxWorkers: maxWorkers,
	}
}

// CrawlURLs crawls multiple URLs in parallel and returns results
func (c *Crawler) CrawlURLs(ctx context.Context, urls []string) []CrawlResult {
	if len(urls) == 0 {
		return []CrawlResult{}
	}

	// Create channels for job distribution and result collection
	jobs := make(chan string, len(urls))
	results := make(chan CrawlResult, len(urls))

	// Determine number of workers (don't exceed number of URLs)
	numWorkers := c.maxWorkers
	if len(urls) < numWorkers {
		numWorkers = len(urls)
	}

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range jobs {
				results <- c.crawlSingle(ctx, url)
			}
		}()
	}

	// Send jobs
	for _, url := range urls {
		jobs <- url
	}
	close(jobs)

	// Wait for all workers to finish and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var crawlResults []CrawlResult
	for result := range results {
		crawlResults = append(crawlResults, result)
	}

	return crawlResults
}

// crawlSingle crawls a single URL and returns the result
func (c *Crawler) crawlSingle(ctx context.Context, urlStr string) CrawlResult {
	start := time.Now()

	result := CrawlResult{
		URL:      urlStr,
		Duration: 0,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Set headers
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("request failed: %w", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != 200 {
		result.Error = fmt.Errorf("HTTP %d", resp.StatusCode)
		result.Duration = time.Since(start)
		return result
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !contains(contentType, "text/html") && !contains(contentType, "application/xhtml") {
		result.Error = fmt.Errorf("non-HTML content type: %s", contentType)
		result.Duration = time.Since(start)
		return result
	}

	// Read body with size limit
	body, err := ReadLimitedBody(resp.Body, c.maxSize)
	if err != nil {
		result.Error = fmt.Errorf("failed to read body: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	// Extract text from HTML
	title, text, err := ExtractText(body, urlStr)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract text: %w", err)
		result.Duration = time.Since(start)
		return result
	}

	result.Title = title
	result.Content = text
	result.Duration = time.Since(start)

	return result
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr ||
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}
