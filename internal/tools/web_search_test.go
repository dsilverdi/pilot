package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchToolName(t *testing.T) {
	tool := NewWebSearchTool()

	if tool.Name() != "web_search" {
		t.Errorf("expected name 'web_search', got %q", tool.Name())
	}
}

func TestWebSearchToolDescription(t *testing.T) {
	tool := NewWebSearchTool()

	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "search") {
		t.Error("description should mention search")
	}
	if !strings.Contains(strings.ToLower(desc), "searxng") {
		t.Error("description should mention SearXNG")
	}
}

func TestWebSearchToolInputSchema(t *testing.T) {
	tool := NewWebSearchTool()

	schema := tool.InputSchema()

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if _, ok := props["query"]; !ok {
		t.Error("schema should have 'query' property")
	}
	if _, ok := props["count"]; !ok {
		t.Error("schema should have 'count' property")
	}
	if _, ok := props["category"]; !ok {
		t.Error("schema should have 'category' property")
	}
}

func TestWebSearchToolExecuteInvalidJSON(t *testing.T) {
	tool := NewWebSearchTool()

	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWebSearchToolExecuteMissingQuery(t *testing.T) {
	tool := NewWebSearchTool()

	input := json.RawMessage(`{}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for missing query")
	}
	if err != nil && !strings.Contains(err.Error(), "query is required") {
		t.Errorf("error should mention query is required, got: %v", err)
	}
}

func TestWebSearchToolExecuteEmptyQuery(t *testing.T) {
	tool := NewWebSearchTool()

	input := json.RawMessage(`{"query": ""}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestWebSearchToolWithMockServer(t *testing.T) {
	// Create mock SearXNG server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := SearXNGResponse{
			Query: r.URL.Query().Get("q"),
			Results: []SearXNGResult{
				{
					Title:   "Go Concurrency",
					URL:     "https://go.dev/doc",
					Content: "Learn about Go concurrency.",
					Engines: []string{"google", "brave"},
					Score:   9.0,
				},
				{
					Title:   "Go Tutorial",
					URL:     "https://go.dev/tour",
					Content: "Interactive Go tutorial.",
					Engines: []string{"startpage"},
					Score:   5.0,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create tool with mock server URL
	tool := &WebSearchTool{
		searxngURL: server.URL,
		httpClient: server.Client(),
	}

	input := json.RawMessage(`{"query": "golang concurrency"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "golang concurrency") {
		t.Error("result should contain query")
	}
	if !strings.Contains(result, "Go Concurrency") {
		t.Error("result should contain first title")
	}
	if !strings.Contains(result, "https://go.dev/doc") {
		t.Error("result should contain first URL")
	}
	if !strings.Contains(result, "google, brave") {
		t.Error("result should contain engines")
	}
}

func TestWebSearchToolFormatResults(t *testing.T) {
	tool := NewWebSearchTool()

	results := []SearXNGResult{
		{
			Title:   "Go Concurrency",
			URL:     "https://go.dev/doc",
			Content: "Learn about Go concurrency.",
			Engines: []string{"google", "brave"},
		},
		{
			Title:   "Go Tutorial",
			URL:     "https://go.dev/tour",
			Content: "Interactive Go tutorial.",
			Engines: []string{"startpage"},
		},
	}

	formatted := tool.formatResults("golang", results)

	if !strings.Contains(formatted, "golang") {
		t.Error("formatted result should contain query")
	}
	if !strings.Contains(formatted, "Go Concurrency") {
		t.Error("formatted result should contain first title")
	}
	if !strings.Contains(formatted, "https://go.dev/doc") {
		t.Error("formatted result should contain first URL")
	}
	if !strings.Contains(formatted, "Go Tutorial") {
		t.Error("formatted result should contain second title")
	}
}

func TestWebSearchToolFormatResultsEmpty(t *testing.T) {
	tool := NewWebSearchTool()

	formatted := tool.formatResults("nonexistent", []SearXNGResult{})

	if !strings.Contains(strings.ToLower(formatted), "no results") {
		t.Error("formatted result should indicate no results found")
	}
}

func TestWebSearchToolContextCancellation(t *testing.T) {
	tool := NewWebSearchTool()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := json.RawMessage(`{"query": "test"}`)
	_, err := tool.Execute(ctx, input)

	// Should error due to cancelled context
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
