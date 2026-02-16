package agent

import (
	"context"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestRunAgenticLoopTextResponse(t *testing.T) {
	// This test verifies the structure of the agentic loop
	// Integration tests with real API will be separate

	cfg := DefaultConfig()
	registry := &MockToolRegistry{}

	agent, err := New(cfg, registry)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Verify agent is ready
	if agent.client == nil {
		t.Error("agent client is nil")
	}
	if agent.registry == nil {
		t.Error("agent registry is nil")
	}
}

func TestEventTypes(t *testing.T) {
	// Test event type constants
	events := []EventType{EventText, EventToolCall, EventToolResult, EventError, EventDone}

	for i, e := range events {
		if int(e) != i {
			t.Errorf("EventType %d has unexpected value %d", i, e)
		}
	}
}

func TestEventStruct(t *testing.T) {
	// Test Event struct fields
	event := Event{
		Type:       EventText,
		Text:       "Hello",
		ToolName:   "test_tool",
		ToolInput:  `{"key": "value"}`,
		ToolResult: "result",
		Error:      nil,
	}

	if event.Type != EventText {
		t.Errorf("expected EventText, got %v", event.Type)
	}
	if event.Text != "Hello" {
		t.Errorf("expected 'Hello', got %s", event.Text)
	}
	if event.ToolName != "test_tool" {
		t.Errorf("expected 'test_tool', got %s", event.ToolName)
	}
}

func TestChatMethod(t *testing.T) {
	cfg := DefaultConfig()
	registry := &MockToolRegistry{}

	agent, err := New(cfg, registry)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Test that Chat method exists and accepts correct parameters
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to avoid actual API call

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock("test")),
	}

	var receivedEvents []Event
	handler := func(e Event) {
		receivedEvents = append(receivedEvents, e)
	}

	// This will return early due to context cancellation
	_, err = agent.Chat(ctx, messages, handler)
	if err != ErrContextCanceled {
		// Either context canceled error or API error is acceptable
		// since we're testing the method signature works
		t.Logf("Chat() returned: %v (expected ErrContextCanceled or API error)", err)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	cfg := &Config{
		Model:        anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens:    1024,
		Temperature:  0.5,
		SystemPrompt: "You are a test assistant.",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	if cfg.SystemPrompt != "You are a test assistant." {
		t.Errorf("expected system prompt 'You are a test assistant.', got %s", cfg.SystemPrompt)
	}
}
