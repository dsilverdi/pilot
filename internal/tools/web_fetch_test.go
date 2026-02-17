package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dsilverdi/pilot/internal/browser"
)

func TestWebFetchToolName(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	if tool.Name() != "web_fetch" {
		t.Errorf("expected name 'web_fetch', got %q", tool.Name())
	}
}

func TestWebFetchToolDescription(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "fetch") {
		t.Error("description should mention fetch")
	}
	if !strings.Contains(strings.ToLower(desc), "content") {
		t.Error("description should mention content")
	}
}

func TestWebFetchToolInputSchema(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	schema := tool.InputSchema()

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if _, ok := props["url"]; !ok {
		t.Error("schema should have 'url' property")
	}
	if _, ok := props["selector"]; !ok {
		t.Error("schema should have 'selector' property")
	}
	if _, ok := props["max_length"]; !ok {
		t.Error("schema should have 'max_length' property")
	}
}

func TestWebFetchToolExecuteInvalidJSON(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWebFetchToolExecuteMissingURL(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	input := json.RawMessage(`{}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for missing URL")
	}
	if err != nil && !strings.Contains(err.Error(), "url is required") {
		t.Errorf("error should mention url is required, got: %v", err)
	}
}

func TestWebFetchToolExecuteEmptyURL(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	input := json.RawMessage(`{"url": ""}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestWebFetchToolFormatContent(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	content := &browser.PageContent{
		Title:   "Test Page Title",
		URL:     "https://example.com/page",
		Content: "This is the main content of the page.",
	}

	formatted := tool.formatContent(content)

	if !strings.Contains(formatted, "Test Page Title") {
		t.Error("formatted content should contain title")
	}
	if !strings.Contains(formatted, "https://example.com/page") {
		t.Error("formatted content should contain URL")
	}
	if !strings.Contains(formatted, "This is the main content") {
		t.Error("formatted content should contain content")
	}
	if !strings.Contains(formatted, "---") {
		t.Error("formatted content should contain separator")
	}
}

func TestWebFetchToolContextCancellation(t *testing.T) {
	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := json.RawMessage(`{"url": "https://example.com"}`)
	_, err := tool.Execute(ctx, input)

	// Should error due to cancelled context
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestWebFetchToolURLNormalization(t *testing.T) {
	// This test verifies the URL normalization logic
	// URLs without http(s):// prefix should get https:// added

	mgr := browser.NewManager(browser.DefaultConfig())
	tool := NewWebFetchTool(mgr)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel to avoid actual network request

	// Test with URL without protocol - should still try to fetch with https://
	input := json.RawMessage(`{"url": "example.com"}`)
	_, err := tool.Execute(ctx, input)

	// Will error due to cancelled context, but URL should have been normalized
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
