package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

// MockTool is a mock implementation for testing
type MockTool struct {
	name        string
	description string
	result      string
	err         error
}

func (m *MockTool) Name() string        { return m.name }
func (m *MockTool) Description() string { return m.description }
func (m *MockTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"test": map[string]any{
				"type":        "string",
				"description": "A test parameter",
			},
		},
	}
}
func (m *MockTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	return m.result, m.err
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if r.tools == nil {
		t.Error("registry tools map is nil")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()
	tool := &MockTool{name: "test_tool", description: "A test tool"}

	r.Register(tool)

	if _, ok := r.tools["test_tool"]; !ok {
		t.Error("tool was not registered")
	}
}

func TestRegistryRegisterMultiple(t *testing.T) {
	r := NewRegistry()
	tool1 := &MockTool{name: "tool1", description: "Tool 1"}
	tool2 := &MockTool{name: "tool2", description: "Tool 2"}

	r.Register(tool1)
	r.Register(tool2)

	if len(r.tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(r.tools))
	}
}

func TestRegistryExecuteSuccess(t *testing.T) {
	r := NewRegistry()
	tool := &MockTool{
		name:   "test_tool",
		result: "success result",
		err:    nil,
	}
	r.Register(tool)

	result, isError := r.Execute(context.Background(), "test_tool", json.RawMessage(`{}`))

	if isError {
		t.Error("expected isError to be false")
	}
	if result != "success result" {
		t.Errorf("expected 'success result', got %q", result)
	}
}

func TestRegistryExecuteToolNotFound(t *testing.T) {
	r := NewRegistry()

	result, isError := r.Execute(context.Background(), "nonexistent", json.RawMessage(`{}`))

	if !isError {
		t.Error("expected isError to be true for nonexistent tool")
	}
	if result == "" {
		t.Error("expected error message, got empty string")
	}
}

func TestRegistryExecuteToolError(t *testing.T) {
	r := NewRegistry()
	tool := &MockTool{
		name:   "failing_tool",
		result: "",
		err:    context.DeadlineExceeded,
	}
	r.Register(tool)

	result, isError := r.Execute(context.Background(), "failing_tool", json.RawMessage(`{}`))

	if !isError {
		t.Error("expected isError to be true for tool error")
	}
	if result == "" {
		t.Error("expected error message, got empty string")
	}
}

func TestRegistryGetToolParams(t *testing.T) {
	r := NewRegistry()
	tool1 := &MockTool{name: "tool1", description: "Tool 1"}
	tool2 := &MockTool{name: "tool2", description: "Tool 2"}

	r.Register(tool1)
	r.Register(tool2)

	params := r.GetToolParams()

	if len(params) != 2 {
		t.Errorf("expected 2 tool params, got %d", len(params))
	}

	// Check that params have correct names
	names := make(map[string]bool)
	for _, p := range params {
		if p.OfTool != nil {
			names[p.OfTool.Name] = true
		}
	}
	if !names["tool1"] {
		t.Error("missing tool1 in params")
	}
	if !names["tool2"] {
		t.Error("missing tool2 in params")
	}
}

func TestRegistryGetToolParamsEmpty(t *testing.T) {
	r := NewRegistry()

	params := r.GetToolParams()

	if len(params) != 0 {
		t.Errorf("expected 0 tool params for empty registry, got %d", len(params))
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	// Register tools concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			tool := &MockTool{name: "tool_" + string(rune('a'+n)), description: "Tool"}
			r.Register(tool)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Execute tools concurrently
	for i := 0; i < 10; i++ {
		go func() {
			r.GetToolParams()
			r.Execute(context.Background(), "tool_a", json.RawMessage(`{}`))
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRegistryGet(t *testing.T) {
	r := NewRegistry()
	tool := &MockTool{name: "test_tool", description: "A test tool"}
	r.Register(tool)

	found, ok := r.Get("test_tool")
	if !ok {
		t.Error("expected to find tool")
	}
	if found.Name() != "test_tool" {
		t.Errorf("expected name 'test_tool', got %q", found.Name())
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(&MockTool{name: "tool_a", description: "Tool A"})
	r.Register(&MockTool{name: "tool_b", description: "Tool B"})

	tools := r.List()

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}
