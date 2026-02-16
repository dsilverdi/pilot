package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	defaultBraveSearchURL = "https://api.search.brave.com/res/v1/web/search"
	defaultResultCount    = 5
	maxResultCount        = 20
	requestTimeout        = 30 * time.Second
)

// WebSearchTool searches the web using Brave Search API
type WebSearchTool struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool(apiKey string) *WebSearchTool {
	return &WebSearchTool{
		apiKey:  apiKey,
		baseURL: defaultBraveSearchURL,
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Search the web for information using Brave Search. Returns relevant search results with titles, URLs, and descriptions."
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
		},
		Required: []string{"query"},
	}
}

type webSearchInput struct {
	Query string `json:"query"`
	Count int    `json:"count,omitempty"`
}

// BraveSearchResponse represents the Brave Search API response
type BraveSearchResponse struct {
	Web WebResults `json:"web"`
}

// WebResults contains the web search results
type WebResults struct {
	Results []WebResult `json:"results"`
}

// WebResult represents a single search result
type WebResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
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

	// Build request URL
	reqURL, err := url.Parse(t.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	q := reqURL.Query()
	q.Set("q", in.Query)
	q.Set("count", strconv.Itoa(count))
	reqURL.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search API error: status %d", resp.StatusCode)
	}

	// Parse response
	var searchResp BraveSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Format results
	return t.formatResults(in.Query, searchResp.Web.Results), nil
}

func (t *WebSearchTool) formatResults(query string, results []WebResult) string {
	if len(results) == 0 {
		return fmt.Sprintf("No results found for query: %q", query)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %q\n\n", query))

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		if result.Description != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", result.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
