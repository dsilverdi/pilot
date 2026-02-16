package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
)

// FileWriteTool writes content to a file
type FileWriteTool struct {
	workDir string
}

// NewFileWriteTool creates a new file write tool
func NewFileWriteTool(workDir string) *FileWriteTool {
	return &FileWriteTool{workDir: workDir}
}

func (t *FileWriteTool) Name() string { return "file_write" }

func (t *FileWriteTool) Description() string {
	return "Write content to a file at the specified path. Creates parent directories if needed."
}

func (t *FileWriteTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to write",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	}
}

type fileWriteInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *FileWriteTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in fileWriteInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if in.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	path := in.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(t.workDir, path)
	}

	// Clean the path
	path = filepath.Clean(path)

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(path, []byte(in.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(in.Content), path), nil
}
