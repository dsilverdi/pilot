package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileWriteToolName(t *testing.T) {
	tool := NewFileWriteTool("/tmp")

	if tool.Name() != "file_write" {
		t.Errorf("expected name 'file_write', got %q", tool.Name())
	}
}

func TestFileWriteToolDescription(t *testing.T) {
	tool := NewFileWriteTool("/tmp")

	desc := tool.Description()
	if desc == "" {
		t.Error("description should not be empty")
	}
	if !strings.Contains(strings.ToLower(desc), "write") {
		t.Error("description should mention writing files")
	}
}

func TestFileWriteToolInputSchema(t *testing.T) {
	tool := NewFileWriteTool("/tmp")

	schema := tool.InputSchema()

	props, ok := schema.Properties.(map[string]any)
	if !ok {
		t.Fatal("Properties should be a map")
	}

	if _, ok := props["path"]; !ok {
		t.Error("schema should have 'path' property")
	}
	if _, ok := props["content"]; !ok {
		t.Error("schema should have 'content' property")
	}
}

func TestFileWriteToolExecuteSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileWriteTool(tmpDir)

	testContent := "Hello, World!"
	input := json.RawMessage(`{"path": "test.txt", "content": "` + testContent + `"}`)

	result, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(result, "Successfully") {
		t.Errorf("expected success message, got %q", result)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(content))
	}
}

func TestFileWriteToolExecuteOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Write initial content
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	tool := NewFileWriteTool(tmpDir)

	newContent := "overwritten"
	input := json.RawMessage(`{"path": "test.txt", "content": "` + newContent + `"}`)

	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify file was overwritten
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != newContent {
		t.Errorf("expected %q, got %q", newContent, string(content))
	}
}

func TestFileWriteToolExecuteCreateParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileWriteTool(tmpDir)

	testContent := "nested content"
	input := json.RawMessage(`{"path": "a/b/c/test.txt", "content": "` + testContent + `"}`)

	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify file was created with parent dirs
	content, err := os.ReadFile(filepath.Join(tmpDir, "a", "b", "c", "test.txt"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(content))
	}
}

func TestFileWriteToolExecuteInvalidJSON(t *testing.T) {
	tool := NewFileWriteTool("/tmp")

	input := json.RawMessage(`{invalid json}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFileWriteToolExecuteMissingPath(t *testing.T) {
	tool := NewFileWriteTool("/tmp")

	input := json.RawMessage(`{"content": "test"}`)
	_, err := tool.Execute(context.Background(), input)

	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFileWriteToolExecuteMissingContent(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileWriteTool(tmpDir)

	// Empty content is allowed (creates empty file)
	input := json.RawMessage(`{"path": "empty.txt", "content": ""}`)
	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify empty file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "empty.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(content))
	}
}

func TestFileWriteToolExecuteAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileWriteTool(tmpDir)

	testFile := filepath.Join(tmpDir, "absolute.txt")
	testContent := "absolute path test"

	input := json.RawMessage(`{"path": "` + testFile + `", "content": "` + testContent + `"}`)

	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(content))
	}
}

func TestFileWriteToolExecuteWithNewlines(t *testing.T) {
	tmpDir := t.TempDir()
	tool := NewFileWriteTool(tmpDir)

	// Use JSON with escaped newlines
	input := json.RawMessage(`{"path": "multiline.txt", "content": "line1\nline2\nline3"}`)

	_, err := tool.Execute(context.Background(), input)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tmpDir, "multiline.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	expected := "line1\nline2\nline3"
	if string(content) != expected {
		t.Errorf("expected %q, got %q", expected, string(content))
	}
}
