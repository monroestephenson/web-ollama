package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// StreamCallbacks defines callbacks for different parts of the response
type StreamCallbacks struct {
	OnThinking func(string) // Called for thinking tokens
	OnAnswer   func(string) // Called for answer tokens
	OnDone     func()       // Called when thinking transitions to answer
}

// ChatWithCallbacks sends a chat request with separate callbacks for thinking/answer
func (c *Client) ChatWithCallbacks(ctx context.Context, req ChatRequest, callbacks StreamCallbacks) (thinking string, answer string, err error) {
	// Force streaming
	req.Stream = true

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request with streaming client (no timeout)
	resp, err := c.streamingClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream response with thinking detection
	thinking, answer, err = c.streamWithThinking(resp.Body, callbacks)
	if err != nil {
		return thinking, answer, fmt.Errorf("failed to stream response: %w", err)
	}

	return thinking, answer, nil
}

// streamWithThinking reads response and separates thinking from answer
func (c *Client) streamWithThinking(body io.Reader, callbacks StreamCallbacks) (thinking string, answer string, err error) {
	scanner := bufio.NewScanner(body)
	var thinkingBuf strings.Builder
	var answerBuf strings.Builder
	wasThinking := false
	isFirstAnswer := true

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Parse JSON chunk
		var chunk ChatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		// Check for thinking field (deepseek-r1 style)
		thinkingContent := chunk.Message.Thinking
		if thinkingContent != "" {
			wasThinking = true
			thinkingBuf.WriteString(thinkingContent)
			if callbacks.OnThinking != nil {
				callbacks.OnThinking(thinkingContent)
			}
		}

		// Check for answer content
		answerContent := chunk.Message.Content
		if answerContent != "" {
			// If this is the first answer after thinking, call OnDone
			if wasThinking && isFirstAnswer && callbacks.OnDone != nil {
				callbacks.OnDone()
				isFirstAnswer = false
			}

			answerBuf.WriteString(answerContent)
			if callbacks.OnAnswer != nil {
				callbacks.OnAnswer(answerContent)
			}
		}

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return thinkingBuf.String(), answerBuf.String(), fmt.Errorf("scanner error: %w", err)
	}

	return thinkingBuf.String(), answerBuf.String(), nil
}

// processThinkingTags handles <think> and </think> tags
func processThinkingTags(content string, inThinking *bool, thinkingDone *bool, callbacks StreamCallbacks) string {
	// Remove <think> tags
	if strings.Contains(content, "<think>") {
		*inThinking = true
		content = strings.ReplaceAll(content, "<think>", "")
	}

	// Remove </think> tags and mark transition
	if strings.Contains(content, "</think>") {
		*inThinking = false
		*thinkingDone = true
		content = strings.ReplaceAll(content, "</think>", "")
	}

	return content
}
