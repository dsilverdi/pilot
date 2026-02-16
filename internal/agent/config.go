package agent

import "github.com/anthropics/anthropic-sdk-go"

// Config holds agent configuration
type Config struct {
	Model        anthropic.Model
	MaxTokens    int64
	Temperature  float64
	SystemPrompt string
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		Model:        anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens:    4096,
		Temperature:  0.7,
		SystemPrompt: "You are a helpful AI assistant with access to tools for file operations and web search. Be concise but thorough.",
	}
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	if c.MaxTokens <= 0 {
		return ErrInvalidMaxTokens
	}
	if c.Temperature < 0 || c.Temperature > 1 {
		return ErrInvalidTemperature
	}
	return nil
}
