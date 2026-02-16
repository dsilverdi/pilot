package agent

import (
	"context"
	"encoding/json"
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
