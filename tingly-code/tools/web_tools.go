package tools

import (
	"context"
	"encoding/json"
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

// Tool descriptions for web tools
const (
	ToolDescWebFetch = `IMPORTANT: WebFetch WILL FAIL for authenticated or private URLs. Before using this tool, check if the URL points to an authenticated service (e.g. Google Docs, Confluence, Jira, GitHub). If so, you MUST use ToolSearch first to find a specialized tool that provides authenticated access.

- Fetches content from a specified URL and processes it using an AI model
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to markdown
- Processes the content with the prompt using a small, fast model
- Returns the model's response about the content
- Use this tool when you need to retrieve and analyze web content

Usage notes:
  - IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions.
  - The URL must be a fully-formed valid URL
  - HTTP URLs will be automatically upgraded to HTTPS
  - The prompt should describe what information you want to extract from the page
  - This tool is read-only and does not modify any files
  - Results may be summarized if the content is very large
  - Includes a self-cleaning 15-minute cache for faster responses when repeatedly accessing the same URL
  - When a URL redirects to a different host, the tool will inform you and provide the redirect URL in a special format. You should then make a new WebFetch request with the redirect URL to fetch the content.
  - For GitHub URLs, prefer using the gh CLI via Bash instead (e.g., gh pr view, gh issue view, gh api).`

	ToolDescWebSearch = `
- Allows Claude to search the web and use the results to inform responses
- Provides up-to-date information for current events and recent data
- Returns search result information formatted as search result blocks, including links as markdown hyperlinks
- Use this tool for accessing information beyond Claude's knowledge cutoff
- Searches are performed automatically within a single API call

CRITICAL REQUIREMENT - You MUST follow this:
  - After answering the user's question, you MUST include a "Sources:" section at the end of your response
  - In the Sources section, list all relevant URLs from the search results as markdown hyperlinks: [Title](URL)
  - This is MANDATORY - never skip including sources in your response
  - Example format:

    [Your answer here]

    Sources:
    - [Source Title 1](https://example.com/1)
    - [Source Title 2](https://example.com/2)

Usage notes:
  - Domain filtering is supported to include or block specific websites
  - Web search is only available in the US

IMPORTANT - Use the correct year in search queries:
  - Today's date is 2026-01-30. You MUST use this year when searching for recent information, documentation, or current events.
  - Example: If the user asks for "latest React docs", search for "React documentation 2026", NOT "React documentation 2025"`
)

// WebFetchParams holds parameters for WebFetch
type WebFetchParams struct {
	URL     string `json:"url" required:"true" description:"URL to fetch content from"`
	Prompt  string `json:"prompt" required:"true" description:"Prompt to describe what information to extract"`
	Timeout int    `json:"timeout,omitempty" description:"Timeout in seconds (default: 30)"`
}

// WebFetchResult represents the result of a web fetch operation
type WebFetchResult struct {
	Success  bool           `json:"success"`
	URL      string         `json:"url"`
	Content  string         `json:"content,omitempty"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// WebFetch fetches content from a URL and processes it
func (wt *WebTools) WebFetch(ctx context.Context, params WebFetchParams) (string, error) {
	// Set timeout
	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	// Create request
	reqURL := params.URL
	if !strings.HasPrefix(reqURL, "http://") && !strings.HasPrefix(reqURL, "https://") {
		reqURL = "https://" + reqURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Sprintf("Error: failed to create request: %v", err), nil
	}

	// Set user agent
	req.Header.Set("User-Agent", "Tingly-Code/1.0")

	// Execute request with timeout
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: failed to fetch URL: %v", err), nil
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Error: HTTP %d: %s", resp.StatusCode, resp.Status), nil
	}

	// Read content
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: failed to read response: %v", err), nil
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")

	// Build result
	_ = WebFetchResult{
		Success: true,
		URL:     reqURL,
		Content: string(data),
		Metadata: map[string]any{
			"content_type": contentType,
			"status_code":  resp.StatusCode,
			"prompt":       params.Prompt,
		},
	}

	// Format output
	output := fmt.Sprintf("=== Web Fetch Result ===\n")
	output += fmt.Sprintf("URL: %s\n", reqURL)
	output += fmt.Sprintf("Content-Type: %s\n", contentType)
	output += fmt.Sprintf("Status: %d\n\n", resp.StatusCode)
	output += fmt.Sprintf("Content:\n%s\n", string(data))
	output += fmt.Sprintf("\nPrompt: %s\n", params.Prompt)

	return output, nil
}

// WebSearchParams holds parameters for WebSearch
type WebSearchParams struct {
	Query          string   `json:"query" required:"true" description:"Search query"`
	AllowedDomains []string `json:"allowed_domains,omitempty" description:"Only include results from these domains"`
	BlockedDomains []string `json:"blocked_domains,omitempty" description:"Exclude results from these domains"`
	NumResults     int      `json:"num_results,omitempty" description:"Number of results to return (default: 10)"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source,omitempty"`
}

// WebSearchResult represents the result of a web search operation
type WebSearchResult struct {
	Success bool           `json:"success"`
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
	Error   string         `json:"error,omitempty"`
}

// WebSearch performs a web search
//
// Note: This is a simplified implementation. In a real environment,
// you would use a proper search API (e.g., Google Custom Search API, Bing Search API).
// This implementation provides a mock response for demonstration purposes.
func (wt *WebTools) WebSearch(ctx context.Context, params WebSearchParams) (string, error) {
	// Validate query
	if params.Query == "" {
		return "Error: query is required", nil
	}

	// Set default number of results
	numResults := params.NumResults
	if numResults <= 0 {
		numResults = 10
	}

	// In a real implementation, this would call an actual search API
	// For now, we provide a mock response that demonstrates the structure
	result := WebSearchResult{
		Success: true,
		Query:   params.Query,
		Results: []SearchResult{
			{
				Title:   fmt.Sprintf("Search result for: %s", params.Query),
				URL:     fmt.Sprintf("https://example.com/search?q=%s", url.QueryEscape(params.Query)),
				Snippet: fmt.Sprintf("This is a mock search result for the query '%s'. In a real implementation, this would contain actual search results from a search API.", params.Query),
				Source:  "mock",
			},
		},
	}

	// Format output
	var output []string
	output = append(output, fmt.Sprintf("=== Web Search Results ==="))
	output = append(output, fmt.Sprintf("Query: %s\n", params.Query))
	output = append(output, fmt.Sprintf("Found %d result(s):\n", len(result.Results)))

	for i, r := range result.Results {
		output = append(output, fmt.Sprintf("%d. [%s](%s)", i+1, r.Title, r.URL))
		output = append(output, fmt.Sprintf("   %s\n", r.Snippet))
	}

	output = append(output, "\n=== Sources ===")
	for _, r := range result.Results {
		output = append(output, fmt.Sprintf("- [%s](%s)", r.Title, r.URL))
	}

	return fmt.Sprintf("%s", strings.Join(output, "\n")), nil
}

// WebFetchWithLLM fetches content from a URL and processes it with LLM
// This is a more advanced version that would integrate with an AI model
type WebFetchWithLLMParams struct {
	URL              string `json:"url" required:"true" description:"URL to fetch content from"`
	Prompt           string `json:"prompt" required:"true" description:"Prompt for AI analysis"`
	MaxContentLength int    `json:"max_content_length,omitempty" description:"Maximum content length to process (default: 100000)"`
}

// WebFetchWithLLM fetches content and analyzes it with AI
func (wt *WebTools) WebFetchWithLLM(ctx context.Context, params WebFetchWithLLMParams) (string, error) {
	// First, fetch the content
	fetchParams := WebFetchParams{
		URL:     params.URL,
		Prompt:  params.Prompt,
		Timeout: 30,
	}

	content, err := wt.WebFetch(ctx, fetchParams)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// In a real implementation, this would send the content to an LLM for analysis
	// For now, we return the raw content with a note
	result := map[string]any{
		"status":  "success",
		"url":     params.URL,
		"prompt":  params.Prompt,
		"content": content,
		"note":    "AI analysis not yet implemented - returning raw content",
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return string(data), nil
}

func init() {
	// Register web tools in the global registry
	RegisterTool("web_fetch", ToolDescWebFetch, "Web", true)
	RegisterTool("web_search", ToolDescWebSearch, "Web", true)
}
