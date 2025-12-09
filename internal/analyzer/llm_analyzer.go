package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LLMAnalyzer uses the LLM to decide if search is needed
type LLMAnalyzer struct {
	ollamaClient OllamaClient
	model        string
}

// OllamaClient interface for making LLM calls
type OllamaClient interface {
	ChatSync(ctx context.Context, model string, messages interface{}) (string, error)
}

// OllamaMessage represents a chat message (matches ollama package)
type OllamaMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Thinking string `json:"thinking,omitempty"`
}

// SearchDecision represents the LLM's decision
type SearchDecision struct {
	NeedsSearch   bool     `json:"needs_search"`
	SearchQueries []string `json:"search_queries,omitempty"` // Support multiple searches
	Reason        string   `json:"reason"`
}

// NewLLMAnalyzer creates a new LLM-based analyzer
func NewLLMAnalyzer(client OllamaClient, model string) *LLMAnalyzer {
	return &LLMAnalyzer{
		ollamaClient: client,
		model:        model,
	}
}

// AnalyzeWithLLM asks the LLM if search is needed and what to search for
func (a *LLMAnalyzer) AnalyzeWithLLM(ctx context.Context, userQuery string) (SearchDecision, error) {
	prompt := fmt.Sprintf(`You are a search decision system. Analyze if the user's query requires web search.

User query: "%s"

Decide if this query needs current web information. Respond ONLY with valid JSON in this exact format:
{
  "needs_search": true/false,
  "search_queries": ["query 1", "query 2"],
  "reason": "brief reason"
}

Guidelines:
- needs_search=true for: current events, recent news, prices, weather, facts that change
- needs_search=false for: coding help, explanations, math, creative writing, general knowledge
- If needs_search=true, provide search_queries as an array (each query: concise, 2-5 words)
- You can provide multiple queries to gather comprehensive information (e.g., "iPhone 16 specs" and "Samsung S24 specs" for comparison)
- Keep reason under 10 words

Respond with JSON only, no other text.`, userQuery)

	messages := []OllamaMessage{
		{Role: "user", Content: prompt},
	}

	response, err := a.ollamaClient.ChatSync(ctx, a.model, messages)
	if err != nil {
		return SearchDecision{}, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var decision SearchDecision

	// Clean response - extract JSON if wrapped in markdown
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	if err := json.Unmarshal([]byte(response), &decision); err != nil {
		return SearchDecision{}, fmt.Errorf("failed to parse LLM response: %w\nResponse: %s", err, response)
	}

	return decision, nil
}
