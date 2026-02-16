package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchToolName(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	if tool.Name() != "web_search" {
		t.Errorf("expected name 'web_search', got %q", tool.Name())
	}
}

func TestWebSearchToolDescription(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "search") {
		t.Error("description should mention search")
	}
}

func TestWebSearchToolInputSchema(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	schema := tool.InputSchema()

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if _, ok := props["query"]; !ok {
		t.Error("schema should have 'query' property")
	}
}

func TestWebSearchToolExecuteSuccess(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("X-Subscription-Token") != "test-api-key" {
			t.Error("missing or incorrect API key header")
		}

		query := r.URL.Query().Get("q")
		if query != "golang concurrency" {
			t.Errorf("expected query 'golang concurrency', got %q", query)
		}

		// Return mock response
		response := BraveSearchResponse{
			Web: WebResults{
				Results: []WebResult{
					{
						Title:       "Go Concurrency Patterns",
						URL:         "https://go.dev/blog/pipelines",
						Description: "Concurrency is the key to designing high performance network services.",
					},
					{
						Title:       "Effective Go - Concurrency",
						URL:         "https://go.dev/doc/effective_go#concurrency",
						Description: "Go encourages a different approach to concurrent programming.",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = server.URL // Override for testing

	input := json.RawMessage(`{"query": "golang concurrency"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "Go Concurrency Patterns") {
		t.Error("result should contain first result title")
	}
	if !strings.Contains(result, "https://go.dev/blog/pipelines") {
		t.Error("result should contain first result URL")
	}
}

func TestWebSearchToolExecuteEmptyResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := BraveSearchResponse{
			Web: WebResults{
				Results: []WebResult{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = server.URL

	input := json.RawMessage(`{"query": "asdfghjklzxcvbnm123456"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(strings.ToLower(result), "no results") {
		t.Error("result should indicate no results found")
	}
}

func TestWebSearchToolExecuteInvalidJSON(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestWebSearchToolExecuteMissingQuery(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	input := json.RawMessage(`{}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for missing query")
	}
}

func TestWebSearchToolExecuteEmptyQuery(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")

	input := json.RawMessage(`{"query": ""}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestWebSearchToolExecuteAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error": "Invalid API key"}`)
	}))
	defer server.Close()

	tool := NewWebSearchTool("invalid-key")
	tool.baseURL = server.URL

	input := json.RawMessage(`{"query": "test"}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for API error response")
	}
}

func TestWebSearchToolExecuteNetworkError(t *testing.T) {
	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = "http://localhost:99999" // Invalid port

	input := json.RawMessage(`{"query": "test"}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for network failure")
	}
}

func TestWebSearchToolExecuteWithCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := r.URL.Query().Get("count")
		if count != "5" {
			t.Errorf("expected count '5', got %q", count)
		}

		response := BraveSearchResponse{
			Web: WebResults{
				Results: []WebResult{
					{Title: "Result 1", URL: "https://example.com/1", Description: "Desc 1"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = server.URL

	input := json.RawMessage(`{"query": "test", "count": 5}`)
	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestWebSearchToolContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-r.Context().Done()
	}))
	defer server.Close()

	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	input := json.RawMessage(`{"query": "test"}`)
	_, err := tool.Execute(ctx, input)

	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestWebSearchToolFormatsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := BraveSearchResponse{
			Web: WebResults{
				Results: []WebResult{
					{
						Title:       "Test Title",
						URL:         "https://example.com",
						Description: "Test description with details.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	tool := NewWebSearchTool("test-api-key")
	tool.baseURL = server.URL

	input := json.RawMessage(`{"query": "test"}`)
	result, _ := tool.Execute(context.Background(), input)

	// Check formatting
	if !strings.Contains(result, "Test Title") {
		t.Error("result should contain title")
	}
	if !strings.Contains(result, "https://example.com") {
		t.Error("result should contain URL")
	}
	if !strings.Contains(result, "Test description") {
		t.Error("result should contain description")
	}
}
