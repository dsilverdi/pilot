package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReadToolName(t *testing.T) {
	tool := NewFileReadTool("/tmp")

	if tool.Name() != "file_read" {
		t.Errorf("expected name 'file_read', got %q", tool.Name())
	}
}

func TestFileReadToolDescription(t *testing.T) {
	tool := NewFileReadTool("/tmp")

	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "read") {
		t.Error("description should mention reading files")
	}
}

func TestFileReadToolInputSchema(t *testing.T) {
	tool := NewFileReadTool("/tmp")

	schema := tool.InputSchema()

	// Check that path property exists
	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if _, ok := props["path"]; !ok {
		t.Error("schema should have 'path' property")
	}
}

func TestFileReadToolExecuteSuccess(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewFileReadTool(tmpDir)

	input := json.RawMessage(`{"path": "test.txt"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != testContent {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestFileReadToolExecuteAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Absolute path test"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewFileReadTool(tmpDir)

	input := json.RawMessage(`{"path": "` + testFile + `"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != testContent {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestFileReadToolExecuteFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileReadTool(tmpDir)

	input := json.RawMessage(`{"path": "nonexistent.txt"}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFileReadToolExecuteInvalidJSON(t *testing.T) {
	tool := NewFileReadTool("/tmp")

	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFileReadToolExecuteMissingPath(t *testing.T) {
	tool := NewFileReadTool("/tmp")

	input := json.RawMessage(`{}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFileReadToolExecuteLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// Create a file larger than the truncation limit
	largeContent := strings.Repeat("x", 150000)
	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewFileReadTool(tmpDir)

	input := json.RawMessage(`{"path": "large.txt"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should be truncated
	if len(result) > 110000 { // 100000 + some overhead for truncation message
		t.Errorf("result should be truncated, got length %d", len(result))
	}
	if !strings.Contains(result, "truncated") {
		t.Error("result should mention truncation")
	}
}

func TestFileReadToolExecuteNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	testFile := filepath.Join(nestedDir, "nested.txt")
	testContent := "Nested content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tool := NewFileReadTool(tmpDir)

	input := json.RawMessage(`{"path": "subdir/nested.txt"}`)
	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != testContent {
		t.Errorf("expected %q, got %q", testContent, result)
	}
}

func TestFileReadToolExecuteDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileReadTool(tmpDir)

	// Try to read a directory
	input := json.RawMessage(`{"path": "."}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error when trying to read a directory")
	}
}
