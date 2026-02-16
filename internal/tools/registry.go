package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// Registry manages tool registration and execution
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// Execute runs a tool by name and returns the result
func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (result string, isError bool) {
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()

	if !exists {
		return fmt.Sprintf("tool not found: %s", name), true
	}

	res, err := tool.Execute(ctx, input)
	if err != nil {
		return err.Error(), true
	}
	return res, false
}

// GetToolParams returns tool parameters for the Anthropic API
func (r *Registry) GetToolParams() []anthropic.ToolUnionParam {
	r.mu.RLock()
	defer r.mu.RUnlock()

	params := make([]anthropic.ToolUnionParam, 0, len(r.tools))
	for _, tool := range r.tools {
		param := anthropic.ToolParam{
			Name:        tool.Name(),
			Description: anthropic.String(tool.Description()),
			InputSchema: tool.InputSchema(),
		}
		params = append(params, anthropic.ToolUnionParam{OfTool: &param})
	}
	return params
}
