package tools

import (
	"context"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
)

// Tool defines the interface for all tools
type Tool interface {
	// Name returns the tool's name as it will appear to the model
	Name() string

	// Description returns a description of what the tool does
	Description() string

	// InputSchema returns the JSON schema for the tool's input parameters
	InputSchema() anthropic.ToolInputSchemaParam

	// Execute runs the tool with the given input and returns the result
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}
