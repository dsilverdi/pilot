package telegram

import (
	"fmt"
	"strings"
)

const (
	// Telegram message limit
	maxMessageLength = 4096
)

// FormatResponse formats the agent response for Telegram
func FormatResponse(text string) string {
	// Escape special Markdown characters for MarkdownV2
	// For now, use plain text to avoid escaping complexity
	return TruncateIfNeeded(text, maxMessageLength)
}

// FormatToolCall formats a tool call notification
func FormatToolCall(name string) string {
	return fmt.Sprintf("🔧 Using %s...", name)
}

// FormatToolResult formats a tool result (abbreviated)
func FormatToolResult(name, result string) string {
	// Truncate long results
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	return fmt.Sprintf("✓ %s complete", name)
}

// FormatError formats an error message
func FormatError(err error) string {
	return fmt.Sprintf("❌ Error: %s", err.Error())
}

// FormatWelcome returns the welcome message
func FormatWelcome(botName string) string {
	return fmt.Sprintf(`👋 Welcome to %s!

I'm an AI assistant powered by Claude. I can help you with:
• Answering questions
• Writing and explaining code
• Research and analysis
• And much more!

Just send me a message to get started.

Commands:
/help - Show available commands
/clear - Clear conversation history
/status - Check bot status`, botName)
}

// FormatHelp returns the help message
func FormatHelp() string {
	return `📖 Available Commands:

/start - Welcome message
/help - Show this help
/clear - Clear your conversation history
/status - Check bot status

Just send any message to chat with the AI assistant.`
}

// FormatStatus returns the status message
func FormatStatus(botUsername string, sessionCount int) string {
	return fmt.Sprintf(`🤖 Bot Status

Bot: @%s
Status: Online
Active sessions: %d`, botUsername, sessionCount)
}

// FormatSessionCleared returns the session cleared message
func FormatSessionCleared() string {
	return "🗑️ Conversation history cleared. Send a message to start fresh!"
}

// FormatUnauthorized returns the unauthorized message
func FormatUnauthorized() string {
	return "🚫 Sorry, you are not authorized to use this bot."
}

// FormatProcessing returns a processing indicator
func FormatProcessing() string {
	return "🤔 Thinking..."
}

// TruncateIfNeeded truncates text if it exceeds max length
func TruncateIfNeeded(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max-3] + "..."
}

// SplitLongMessage splits a message that exceeds Telegram's limit
func SplitLongMessage(text string) []string {
	if len(text) <= maxMessageLength {
		return []string{text}
	}

	var parts []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxMessageLength {
			parts = append(parts, remaining)
			break
		}

		// Find a good split point (newline or space)
		splitAt := maxMessageLength
		if idx := strings.LastIndex(remaining[:maxMessageLength], "\n"); idx > maxMessageLength/2 {
			splitAt = idx + 1
		} else if idx := strings.LastIndex(remaining[:maxMessageLength], " "); idx > maxMessageLength/2 {
			splitAt = idx + 1
		}

		parts = append(parts, remaining[:splitAt])
		remaining = remaining[splitAt:]
	}

	return parts
}
