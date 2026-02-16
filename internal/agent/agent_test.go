package agent

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model != anthropic.ModelClaudeSonnet4_5_20250929 {
		t.Errorf("expected model %s, got %s", anthropic.ModelClaudeSonnet4_5_20250929, cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected max_tokens 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", cfg.Temperature)
	}
	if cfg.SystemPrompt == "" {
		t.Error("expected non-empty system prompt")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: nil,
		},
		{
			name: "invalid max_tokens zero",
			config: &Config{
				MaxTokens:   0,
				Temperature: 0.5,
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "invalid max_tokens negative",
			config: &Config{
				MaxTokens:   -1,
				Temperature: 0.5,
			},
			wantErr: ErrInvalidMaxTokens,
		},
		{
			name: "invalid temperature negative",
			config: &Config{
				MaxTokens:   1024,
				Temperature: -0.1,
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "invalid temperature too high",
			config: &Config{
				MaxTokens:   1024,
				Temperature: 1.5,
			},
			wantErr: ErrInvalidTemperature,
		},
		{
			name: "valid edge temperature 0",
			config: &Config{
				MaxTokens:   1024,
				Temperature: 0,
			},
			wantErr: nil,
		},
		{
			name: "valid edge temperature 1",
			config: &Config{
				MaxTokens:   1024,
				Temperature: 1,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAgent(t *testing.T) {
	cfg := DefaultConfig()
	registry := &MockToolRegistry{}

	agent, err := New(cfg, registry)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if agent == nil {
		t.Fatal("New() returned nil agent")
	}
	if agent.config != cfg {
		t.Error("agent config mismatch")
	}
}

func TestNewAgentValidation(t *testing.T) {
	registry := &MockToolRegistry{}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &Config{
				MaxTokens:   0,
				Temperature: 0.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config, registry)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAgentNilRegistry(t *testing.T) {
	cfg := DefaultConfig()
	_, err := New(cfg, nil)
	if err != ErrNoToolRegistry {
		t.Errorf("New() error = %v, want %v", err, ErrNoToolRegistry)
	}
}

// MockToolRegistry is a mock implementation for testing
type MockToolRegistry struct{}

func (m *MockToolRegistry) GetToolParams() []anthropic.ToolUnionParam {
	return nil
}

func (m *MockToolRegistry) Execute(ctx context.Context, name string, input json.RawMessage) (string, bool) {
	return "mock result", false
}

func TestResolveAuthOptions(t *testing.T) {
	// Save original env vars
	origOAuth := os.Getenv("ANTHROPIC_OAUTH_TOKEN")
	origAuth := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	origAPI := os.Getenv("ANTHROPIC_API_KEY")

	// Cleanup after test
	defer func() {
		os.Setenv("ANTHROPIC_OAUTH_TOKEN", origOAuth)
		os.Setenv("ANTHROPIC_AUTH_TOKEN", origAuth)
		os.Setenv("ANTHROPIC_API_KEY", origAPI)
	}()

	tests := []struct {
		name       string
		oauthToken string
		authToken  string
		apiKey     string
		wantLen    int
	}{
		{
			name:       "oauth token takes priority",
			oauthToken: "oauth-token",
			authToken:  "auth-token",
			apiKey:     "api-key",
			wantLen:    1, // Only OAuth token option added
		},
		{
			name:       "auth token when no oauth",
			oauthToken: "",
			authToken:  "auth-token",
			apiKey:     "api-key",
			wantLen:    1, // Auth token option added
		},
		{
			name:       "fallback to api key",
			oauthToken: "",
			authToken:  "",
			apiKey:     "api-key",
			wantLen:    0, // SDK handles API key by default
		},
		{
			name:       "whitespace oauth token ignored",
			oauthToken: "   ",
			authToken:  "auth-token",
			apiKey:     "",
			wantLen:    1, // Falls back to auth token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ANTHROPIC_OAUTH_TOKEN", tt.oauthToken)
			os.Setenv("ANTHROPIC_AUTH_TOKEN", tt.authToken)
			os.Setenv("ANTHROPIC_API_KEY", tt.apiKey)

			opts := resolveAuthOptions()
			if len(opts) != tt.wantLen {
				t.Errorf("resolveAuthOptions() returned %d options, want %d", len(opts), tt.wantLen)
			}
		})
	}
}
