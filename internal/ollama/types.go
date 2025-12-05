package ollama

// ChatRequest represents a chat request to Ollama
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message represents a chat message
type Message struct {
	Role     string `json:"role"`    // "user", "assistant", or "system"
	Content  string `json:"content"`
	Thinking string `json:"thinking"` // For reasoning models like deepseek-r1
}

// ChatResponse represents a streaming response chunk from Ollama
type ChatResponse struct {
	Model     string  `json:"model"`
	CreatedAt string  `json:"created_at"`
	Message   Message `json:"message"`
	Done      bool    `json:"done"`
}
