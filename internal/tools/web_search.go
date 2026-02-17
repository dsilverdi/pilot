package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	defaultResultCount    = 5
	maxResultCount        = 20
	searchTimeout         = 30 * time.Second
	defaultSearXNGURL     = "http://localhost:8081"
)

// SearXNGResult represents a single search result from SearXNG
type SearXNGResult struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Content string   `json:"content"`
	Engines []string `json:"engines"`
	Score   float64  `json:"score"`
}

// SearXNGResponse represents the response from SearXNG API
type SearXNGResponse struct {
	Results []SearXNGResult `json:"results"`
	Query   string          `json:"query"`
}

// WebSearchTool searches the web using SearXNG
type WebSearchTool struct {
	searxngURL string
	httpClient *http.Client
}

// NewWebSearchTool creates a new web search tool using SearXNG
func NewWebSearchTool() *WebSearchTool {
	searxngURL := os.Getenv("SEARXNG_URL")
	if searxngURL == "" {
		searxngURL = defaultSearXNGURL
	}

	return &WebSearchTool{
		searxngURL: searxngURL,
		httpClient: &http.Client{
			Timeout: searchTimeout,
		},
	}
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return `Search the web for information using SearXNG (aggregates Google, Brave, Startpage, and more).

Use this tool when you need to:
- Find current information about a topic
- Research recent news or events
- Look up documentation or tutorials
- Verify facts or find sources`
}

func (t *WebSearchTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query",
			},
			"count": map[string]any{
				"type":        "integer",
				"description": "Number of results to return (1-20, default 5)",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Search category: general, images, news, videos, it, science (default: general)",
			},
		},
		Required: []string{"query"},
	}
}

type webSearchInput struct {
	Query    string `json:"query"`
	Count    int    `json:"count,omitempty"`
	Category string `json:"category,omitempty"`
}

func (t *WebSearchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in webSearchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if in.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	// Set default count
	count := in.Count
	if count <= 0 {
		count = defaultResultCount
	}
	if count > maxResultCount {
		count = maxResultCount
	}

	// Set default category
	category := in.Category
	if category == "" {
		category = "general"
	}

	// Build SearXNG URL
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&categories=%s&language=en",
		t.searxngURL,
		url.QueryEscape(in.Query),
		url.QueryEscape(category),
	)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searxResp SearXNGResponse
	if err := json.NewDecoder(resp.Body).Decode(&searxResp); err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	// Limit results
	results := searxResp.Results
	if len(results) > count {
		results = results[:count]
	}

	return t.formatResults(in.Query, results), nil
}

func (t *WebSearchTool) formatResults(query string, results []SearXNGResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No results found for query: %q", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %q\n\n", query))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		if result.Content != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", result.Content))
		}
		if len(result.Engines) > 0 {
			sb.WriteString(fmt.Sprintf("   Sources: %s\n", strings.Join(result.Engines, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
