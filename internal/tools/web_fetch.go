package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/browser"
)

const (
	defaultMaxContentLength = 50000 // Characters
	fetchTimeout            = 30 * time.Second
)

// WebFetchTool fetches and extracts content from web pages
type WebFetchTool struct {
	browserMgr *browser.Manager
}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool(browserMgr *browser.Manager) *WebFetchTool {
	return &WebFetchTool{
		browserMgr: browserMgr,
	}
}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return `Fetch and extract content from a web page URL.

Use this tool to:
- Read article or blog post content
- Extract documentation from a specific URL
- Get details from a page found via web_search
- Scrape text content from any web page

Returns the page title and main text content (scripts/styles/navigation removed).`
}

func (t *WebFetchTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch content from",
			},
			"selector": map[string]any{
				"type":        "string",
				"description": "Optional CSS selector to extract specific content (e.g., 'article', '.content', '#main')",
			},
			"max_length": map[string]any{
				"type":        "integer",
				"description": "Maximum content length in characters (default: 50000)",
			},
		},
		Required: []string{"url"},
	}
}

type webFetchInput struct {
	URL       string `json:"url"`
	Selector  string `json:"selector,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
}

func (t *WebFetchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in webFetchInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if in.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Normalize URL
	url := in.URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	maxLength := in.MaxLength
	if maxLength <= 0 {
		maxLength = defaultMaxContentLength
	}

	// Create page with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	page, err := t.browserMgr.NewPage(timeoutCtx)
	if err != nil {
		return "", fmt.Errorf("failed to create browser page: %w", err)
	}
	defer page.Close()

	// Set headers to get English/US content when available
	_, _ = page.SetExtraHeaders([]string{
		"Accept-Language", "en-US,en;q=0.9",
		"X-Forwarded-For", "8.8.8.8",
	})

	// Navigate to URL
	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("failed to load page: %w", err)
	}

	// Wait for page to load and stabilize (important for JS-rendered content)
	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("page load timeout: %w", err)
	}
	// Wait for network idle / DOM stability
	page.WaitStable(1000)

	// Extract content
	content, err := browser.ExtractPageContent(page, maxLength)
	if err != nil {
		return "", fmt.Errorf("failed to extract content: %w", err)
	}

	// If selector specified, try to extract just that element
	if in.Selector != "" {
		if el, err := page.Element(in.Selector); err == nil && el != nil {
			if text, err := el.Text(); err == nil {
				content.Content = text
				if len(content.Content) > maxLength {
					content.Content = content.Content[:maxLength] + "\n\n... [content truncated]"
				}
			}
		}
	}

	return t.formatContent(content), nil
}

func (t *WebFetchTool) formatContent(content *browser.PageContent) string {
	var sb strings.Builder

	if content.Title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", content.Title))
	}
	sb.WriteString(fmt.Sprintf("URL: %s\n\n", content.URL))
	sb.WriteString("---\n\n")
	sb.WriteString(content.Content)

	return sb.String()
}
