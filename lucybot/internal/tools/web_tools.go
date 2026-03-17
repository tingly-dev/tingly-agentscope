package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebTools holds tools for web operations
type WebTools struct {
	client *http.Client
}

// NewWebTools creates a new WebTools instance
func NewWebTools() *WebTools {
	return &WebTools{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WebFetch fetches content from a URL
func (wt *WebTools) WebFetch(ctx context.Context, urlStr string) (string, error) {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "LucyBot/1.0")

	resp, err := wt.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(data), nil
}

// WebSearch performs a web search (mock implementation)
func (wt *WebTools) WebSearch(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Mock implementation - in production, use a real search API
	result := fmt.Sprintf(`Web Search Results for: %s

Note: This is a mock implementation. In production, integrate with:
- Google Custom Search API
- Bing Search API
- Brave Search API
- SerpAPI

To search, use the query: %s
`, query, url.QueryEscape(query))

	return result, nil
}
