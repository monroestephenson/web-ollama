package history

import (
	"time"
)

// History represents all conversation sessions
type History struct {
	Sessions []Session `json:"sessions"`
}

// Session represents a single conversation session
type Session struct {
	ID        string    `json:"id"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Messages  []Message `json:"messages"`
}

// Message represents a single message in a conversation
type Message struct {
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  *Metadata `json:"metadata,omitempty"`
}

// Metadata contains additional information about a message
type Metadata struct {
	SearchPerformed bool     `json:"search_performed"`
	SourceURLs      []string `json:"source_urls,omitempty"`
}
