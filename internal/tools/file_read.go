package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
)

// FileReadTool reads the contents of a file
type FileReadTool struct {
	workDir string
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool(workDir string) *FileReadTool {
	return &FileReadTool{workDir: workDir}
}

func (t *FileReadTool) Name() string { return "file_read" }

func (t *FileReadTool) Description() string {
	return "Read the contents of a file at the specified path. Use this to examine file contents."
}

func (t *FileReadTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The path to the file to read (relative to working directory or absolute)",
			},
		},
		Required: []string{"path"},
	}
}

type fileReadInput struct {
	Path string `json:"path"`
}

func (t *FileReadTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in fileReadInput
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

	// Check if it's a directory
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Truncate large files
	const maxSize = 100000
	if len(content) > maxSize {
		content = content[:maxSize]
		return string(content) + "\n\n... [truncated - file exceeds 100KB]", nil
	}

	return string(content), nil
}
