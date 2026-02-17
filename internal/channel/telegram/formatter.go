package telegram

import (
	"fmt"
	"html"
	"math/rand"
	"regexp"
	"strings"
)

const (
	// Telegram message limit
	maxMessageLength = 4096
)

// Friendly tool messages - more conversational than technical names
var toolMessages = map[string][]string{
	"web_search": {
		"🔍 Searching the web...",
		"🌐 Let me look that up...",
		"🔎 Finding info for you...",
	},
	"web_fetch": {
		"📄 Reading that page...",
		"🌐 Grabbing the details...",
		"📖 Checking the source...",
	},
	"file_read": {
		"📂 Reading the file...",
		"📄 Looking at that...",
	},
	"file_write": {
		"✍️ Writing that down...",
		"💾 Saving the file...",
	},
	"bash_exec": {
		"⚡ Running a command...",
		"🖥️ Working on it...",
	},
	"invoke_skill": {
		"🎯 Getting specialized help...",
		"✨ Using a special skill...",
	},
}

// FormatResponse formats the agent response for Telegram with HTML
func FormatResponse(text string) string {
	// Convert markdown to Telegram HTML
	formatted := ConvertToTelegramHTML(text)
	return TruncateIfNeeded(formatted, maxMessageLength)
}

// FormatToolCall returns a friendly tool notification
func FormatToolCall(name string) string {
	if messages, ok := toolMessages[name]; ok {
		return messages[rand.Intn(len(messages))]
	}
	return "⚡ Working on something..."
}

// FormatToolResult formats a tool result (abbreviated)
func FormatToolResult(name, result string) string {
	// Truncate long results
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	return "✓ Done!"
}

// FormatError formats an error message
func FormatError(err error) string {
	return fmt.Sprintf("😅 Oops, something went wrong: %s", err.Error())
}

// FormatWelcome returns the welcome message
func FormatWelcome(botName string) string {
	_ = botName // Bot name available for customization
	return `👋 Hey there! Welcome!

I'm here to help you with all sorts of things:
• Answering questions
• Searching the web
• Working with files
• And much more!

Just send me a message to get started.

Commands:
/help - Show available commands
/clear - Clear conversation history
/status - Check bot status`
}

// FormatHelp returns the help message
func FormatHelp() string {
	return `📖 Here's what I can do:

/start - Welcome message
/help - Show this help
/clear - Clear your conversation history
/status - Check bot status

Just send any message and I'll do my best to help!`
}

// FormatStatus returns the status message
func FormatStatus(botUsername string, sessionCount int) string {
	return fmt.Sprintf(`🤖 Bot Status

Bot: @%s
Status: Online and ready!
Active sessions: %d`, botUsername, sessionCount)
}

// FormatSessionCleared returns the session cleared message
func FormatSessionCleared() string {
	return "🗑️ All cleared! Send a message to start fresh."
}

// FormatUnauthorized returns the unauthorized message
func FormatUnauthorized() string {
	return "🚫 Sorry, you're not on the guest list for this bot."
}

// FormatProcessing returns a processing indicator
func FormatProcessing() string {
	return "🤔 Thinking..."
}

// ConvertToTelegramHTML converts Markdown to Telegram HTML format
func ConvertToTelegramHTML(text string) string {
	// First escape HTML entities
	text = html.EscapeString(text)

	// Convert markdown headers (## Header) to bold text
	// Remove the ## symbols but keep the text bold
	headerRegex := regexp.MustCompile(`(?m)^#{1,6}\s*(.+)$`)
	text = headerRegex.ReplaceAllString(text, "<b>$1</b>")

	// Convert **bold** to <b>bold</b>
	boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*`)
	text = boldRegex.ReplaceAllString(text, "<b>$1</b>")

	// Convert __bold__ to <b>bold</b>
	boldUnderscoreRegex := regexp.MustCompile(`__(.+?)__`)
	text = boldUnderscoreRegex.ReplaceAllString(text, "<b>$1</b>")

	// Convert *italic* to <i>italic</i> (but not if it's a bullet point)
	// Only match *word* patterns that aren't at line start
	italicRegex := regexp.MustCompile(`(?:^|[^*])\*([^*\n]+?)\*(?:[^*]|$)`)
	text = italicRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract the content between asterisks
		inner := regexp.MustCompile(`\*([^*]+)\*`).FindStringSubmatch(match)
		if len(inner) > 1 {
			// Preserve surrounding characters
			prefix := ""
			suffix := ""
			if len(match) > 0 && match[0] != '*' {
				prefix = string(match[0])
			}
			if len(match) > 0 && match[len(match)-1] != '*' {
				suffix = string(match[len(match)-1])
			}
			return prefix + "<i>" + inner[1] + "</i>" + suffix
		}
		return match
	})

	// Convert `code` to <code>code</code>
	codeRegex := regexp.MustCompile("`([^`\n]+)`")
	text = codeRegex.ReplaceAllString(text, "<code>$1</code>")

	// Convert ```code blocks``` to <pre>code</pre>
	codeBlockRegex := regexp.MustCompile("```[a-zA-Z]*\n?([\\s\\S]*?)```")
	text = codeBlockRegex.ReplaceAllString(text, "<pre>$1</pre>")

	// Convert [link](url) to <a href="url">link</a>
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	text = linkRegex.ReplaceAllString(text, `<a href="$2">$1</a>`)

	// Convert bullet points: - item or * item to • item
	bulletRegex := regexp.MustCompile(`(?m)^[\-\*]\s+`)
	text = bulletRegex.ReplaceAllString(text, "• ")

	return text
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

// SplitLongMessageHTML splits HTML-formatted messages safely
func SplitLongMessageHTML(text string) []string {
	// For HTML, we need to be more careful about not breaking tags
	// Use the same logic but ensure we don't split inside a tag
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

		// Find a good split point
		splitAt := maxMessageLength

		// Try to find a newline first
		if idx := strings.LastIndex(remaining[:maxMessageLength], "\n"); idx > maxMessageLength/2 {
			splitAt = idx + 1
		} else if idx := strings.LastIndex(remaining[:maxMessageLength], " "); idx > maxMessageLength/2 {
			// Make sure we're not inside an HTML tag
			lastOpenTag := strings.LastIndex(remaining[:idx], "<")
			lastCloseTag := strings.LastIndex(remaining[:idx], ">")
			if lastOpenTag <= lastCloseTag {
				splitAt = idx + 1
			}
		}

		parts = append(parts, remaining[:splitAt])
		remaining = remaining[splitAt:]
	}

	return parts
}
