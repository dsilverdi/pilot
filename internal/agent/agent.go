package agent

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// ToolRegistry defines the interface for tool management
type ToolRegistry interface {
	GetToolParams() []anthropic.ToolUnionParam
	Execute(ctx context.Context, name string, input json.RawMessage) (result string, isError bool)
}

// Agent is the core orchestrator for the agentic system
type Agent struct {
	client   *anthropic.Client
	config   *Config
	registry ToolRegistry

	mu sync.RWMutex
}

// New creates a new Agent with the given configuration and tool registry
func New(config *Config, registry ToolRegistry) (*Agent, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if registry == nil {
		return nil, ErrNoToolRegistry
	}

	client := anthropic.NewClient()

	return &Agent{
		client:   &client,
		config:   config,
		registry: registry,
	}, nil
}

// Config returns the agent's configuration
func (a *Agent) Config() *Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// EventType represents the type of streaming event
type EventType int

const (
	EventText EventType = iota
	EventToolCall
	EventToolResult
	EventError
	EventDone
)

// Event represents a streaming event from the agent
type Event struct {
	Type       EventType
	Text       string
	ToolName   string
	ToolInput  string
	ToolResult string
	Error      error
}

// EventHandler is a callback for handling streaming events
type EventHandler func(Event)

// Chat starts a conversation with the given messages and streams responses
func (a *Agent) Chat(ctx context.Context, messages []anthropic.MessageParam, onEvent EventHandler) ([]anthropic.MessageParam, error) {
	return a.runAgenticLoop(ctx, messages, onEvent)
}
