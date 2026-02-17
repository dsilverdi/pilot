package telegram

import (
	"os"
	"strconv"
	"strings"
)

// Config holds Telegram bot configuration
type Config struct {
	Token        string  // Bot token from @BotFather
	AllowedUsers []int64 // Optional: restrict to user IDs (empty = allow all)
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Token:        os.Getenv("TELEGRAM_BOT_TOKEN"),
		AllowedUsers: parseUserList(os.Getenv("TELEGRAM_ALLOWED_USERS")),
	}
}

// Enabled returns true if Telegram bot is configured
func (c *Config) Enabled() bool {
	return c.Token != ""
}

// IsUserAllowed checks if a user ID is allowed to use the bot
func (c *Config) IsUserAllowed(userID int64) bool {
	// If no allowlist, allow everyone
	if len(c.AllowedUsers) == 0 {
		return true
	}

	for _, id := range c.AllowedUsers {
		if id == userID {
			return true
		}
	}
	return false
}

// parseUserList parses comma-separated user IDs
func parseUserList(s string) []int64 {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	users := make([]int64, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			users = append(users, id)
		}
	}

	return users
}
