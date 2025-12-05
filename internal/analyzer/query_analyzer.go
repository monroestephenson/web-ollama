package analyzer

import (
	"strings"
)

// SearchTrigger represents the decision whether to perform a search
type SearchTrigger struct {
	NeedsSearch bool
	Confidence  int
	Reason      string
}

// Analyzer handles query analysis
type Analyzer struct {
	timeSensitive  []string
	factualQueries []string
	researchQueries []string
	explanationQueries []string
	codeQueries []string
}

// NewAnalyzer creates a new query analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		timeSensitive: []string{
			"latest", "current", "today", "now", "recent",
			"2024", "2025", "this year", "this month", "this week",
			"breaking", "news", "updated", "new", "yesterday",
			"live", "ongoing", "happening",
		},
		factualQueries: []string{
			"what is", "what are", "who is", "who are",
			"when did", "when was", "where is", "where are",
			"how many", "how much", "which", "price of", "cost of",
			"weather", "stock", "score", "result",
		},
		researchQueries: []string{
			"compare", "comparison", "best", "top", "review", "reviews",
			"vs", "versus", "difference between", "pros and cons",
			"alternatives", "options", "recommend",
		},
		explanationQueries: []string{
			"explain", "how does", "how do", "why does", "why do",
			"concept of", "teach me", "tutorial", "guide",
			"understanding", "learn",
		},
		codeQueries: []string{
			"code", "function", "algorithm", "implement",
			"debug", "error", "syntax", "program",
			"variable", "class", "method", "api",
		},
	}
}

// AnalyzeQuery determines if a query needs web search
func (a *Analyzer) AnalyzeQuery(query string) SearchTrigger {
	query = strings.ToLower(strings.TrimSpace(query))

	if query == "" {
		return SearchTrigger{
			NeedsSearch: false,
			Confidence:  0,
			Reason:      "empty query",
		}
	}

	score := 0
	reasons := []string{}

	// Check for time-sensitive keywords (+40 points)
	if count := a.countMatches(query, a.timeSensitive); count > 0 {
		score += 40
		reasons = append(reasons, "time-sensitive")
	}

	// Check for factual queries (+30 points)
	if count := a.countMatches(query, a.factualQueries); count > 0 {
		score += 30
		reasons = append(reasons, "factual")
	}

	// Check for research queries (+20 points)
	if count := a.countMatches(query, a.researchQueries); count > 0 {
		score += 20
		reasons = append(reasons, "research")
	}

	// Check for explanation queries (-30 points)
	if count := a.countMatches(query, a.explanationQueries); count > 0 {
		score -= 30
		reasons = append(reasons, "explanation (LLM better)")
	}

	// Check for code queries (-40 points)
	if count := a.countMatches(query, a.codeQueries); count > 0 {
		score -= 40
		reasons = append(reasons, "code (LLM better)")
	}

	// Threshold: score > 40 triggers search
	needsSearch := score > 40

	reason := strings.Join(reasons, ", ")
	if reason == "" {
		reason = "general query"
	}

	return SearchTrigger{
		NeedsSearch: needsSearch,
		Confidence:  score,
		Reason:      reason,
	}
}

// countMatches counts how many patterns match in the query
func (a *Analyzer) countMatches(query string, patterns []string) int {
	count := 0
	for _, pattern := range patterns {
		if strings.Contains(query, pattern) {
			count++
		}
	}
	return count
}
