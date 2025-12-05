package crawler

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// ExtractText extracts clean text from HTML content
func ExtractText(htmlContent []byte, sourceURL string) (title string, text string, err error) {
	doc, err := html.Parse(bytes.NewReader(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Extract title
	title = extractTitle(doc)

	// Extract body text
	text = extractBodyText(doc)

	// Clean up whitespace
	text = cleanText(text)

	// Truncate to reasonable size (approximately 500 words)
	text = truncateWords(text, 500)

	return title, text, nil
}

// extractTitle finds and returns the page title
func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		return getNodeText(n)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}

	return ""
}

// extractBodyText extracts text from the body, excluding unwanted elements
func extractBodyText(n *html.Node) string {
	// Skip unwanted elements
	if n.Type == html.ElementNode {
		tag := n.Data
		if tag == "script" || tag == "style" || tag == "nav" || tag == "footer" || tag == "header" || tag == "aside" {
			return ""
		}
	}

	var text strings.Builder

	// Add text from this node
	if n.Type == html.TextNode {
		text.WriteString(n.Data)
		text.WriteString(" ")
	}

	// Recursively process children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(extractBodyText(c))
	}

	return text.String()
}

// getNodeText extracts all text from a node and its children
func getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text.WriteString(getNodeText(c))
	}

	return text.String()
}

// cleanText removes excessive whitespace and normalizes text
func cleanText(text string) string {
	// Replace multiple spaces with single space
	text = strings.Join(strings.Fields(text), " ")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// truncateWords truncates text to approximately N words
func truncateWords(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}

	return strings.Join(words[:maxWords], " ") + "..."
}

// ReadLimitedBody reads up to maxBytes from a reader
func ReadLimitedBody(body io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(body, maxBytes)
	return io.ReadAll(limited)
}
