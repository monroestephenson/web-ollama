package searxng

// SearchResponse represents the JSON response from SearXNG
type SearchResponse struct {
	Query           string         `json:"query"`
	NumberOfResults int            `json:"number_of_results"`
	Results         []SearchResult `json:"results"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"` // Snippet
	Engine  string  `json:"engine"`
	Score   float64 `json:"score"`
}
