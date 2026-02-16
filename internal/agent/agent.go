package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	// OAuth beta header required for OAuth tokens
	oauthBetaHeader = "oauth-2025-04-20"
	// User agent for CLI
	pilotUserAgent = "pilot-cli/1.0.0"
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

	// Resolve authentication
	opts := resolveAuthOptions()
	client := anthropic.NewClient(opts...)

	return &Agent{
		client:   &client,
		config:   config,
		registry: registry,
	}, nil
}

// resolveAuthOptions determines the authentication method based on environment variables
// Priority: ANTHROPIC_OAUTH_TOKEN > ANTHROPIC_AUTH_TOKEN > ANTHROPIC_API_KEY
func resolveAuthOptions() []option.RequestOption {
	var opts []option.RequestOption

	// Check for OAuth token first (highest priority)
	if token := strings.TrimSpace(os.Getenv("ANTHROPIC_OAUTH_TOKEN")); token != "" {
		// OAuth tokens require special headers
		opts = append(opts, option.WithAuthToken(token))
		opts = append(opts, option.WithHeader("anthropic-beta", oauthBetaHeader))
		opts = append(opts, option.WithHeader("anthropic-dangerous-direct-browser-access", "true"))
		opts = append(opts, option.WithHeader("user-agent", pilotUserAgent))
		return opts
	}

	// Check for auth token (used by SDK default)
	if token := strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")); token != "" {
		opts = append(opts, option.WithAuthToken(token))
		return opts
	}

	// Fall back to API key (SDK reads ANTHROPIC_API_KEY by default)
	// No need to add explicit option, SDK handles it
	return opts
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

// CreateMessage creates a non-streaming message (used for compaction summaries)
func (a *Agent) CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error) {
	return a.client.Messages.New(ctx, params)
}

// Model returns the agent's configured model
func (a *Agent) Model() anthropic.Model {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Model
}
