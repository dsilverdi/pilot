package gateway

import (
	"os"
	"time"
)

// Config holds the gateway server configuration
type Config struct {
	Addr           string        // Listen address (e.g., ":8080")
	APIKey         string        // Optional API key for authentication
	AllowedOrigins []string      // CORS allowed origins
	ReadTimeout    time.Duration // HTTP read timeout
	WriteTimeout   time.Duration // HTTP write timeout
	Version        string        // Version string for health endpoint
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Addr:           getEnvOrDefault("GATEWAY_ADDR", ":8080"),
		APIKey:         os.Getenv("GATEWAY_API_KEY"),
		AllowedOrigins: []string{"*"},
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   120 * time.Second, // Longer for streaming responses
		Version:        "dev",
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
