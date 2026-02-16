package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	// DefaultTokenThreshold is the default threshold for triggering compaction (~100k tokens)
	DefaultTokenThreshold = 100000

	// RecentMessagesToKeep is the number of recent messages to keep after compaction
	RecentMessagesToKeep = 10

	// CompactionMaxTokens is the max tokens for the summary response
	CompactionMaxTokens = 2048
)

// CompactionClient interface for making API calls
type CompactionClient interface {
	CreateMessage(ctx context.Context, params anthropic.MessageNewParams) (*anthropic.Message, error)
}

// Compactor handles context compaction for sessions
type Compactor struct {
	client    CompactionClient
	model     anthropic.Model
	threshold int
}

// NewCompactor creates a new Compactor
func NewCompactor(client CompactionClient, model anthropic.Model, threshold int) *Compactor {
	if threshold <= 0 {
		threshold = DefaultTokenThreshold
	}
	return &Compactor{
		client:    client,
		model:     model,
		threshold: threshold,
	}
}

// ShouldCompact checks if a session needs compaction
func (c *Compactor) ShouldCompact(sess *Session) bool {
	return sess.NeedsCompaction(c.threshold)
}

// Compact compacts the session's message history
// It summarizes older messages and keeps only recent ones
func (c *Compactor) Compact(ctx context.Context, sess *Session) error {
	sess.mu.Lock()
	messages := sess.Messages
	sess.mu.Unlock()

	if len(messages) <= RecentMessagesToKeep {
		// Not enough messages to compact
		return nil
	}

	// Split into messages to compact and recent messages to keep
	toCompact := messages[:len(messages)-RecentMessagesToKeep]
	toKeep := messages[len(messages)-RecentMessagesToKeep:]

	// Generate summary of old messages
	summary, err := c.generateSummary(ctx, toCompact, sess.Compaction)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Update session
	sess.mu.Lock()
	defer sess.mu.Unlock()

	// Calculate total compacted count (previous + new)
	compactedCount := len(toCompact)
	if sess.Compaction != nil {
		compactedCount += sess.Compaction.CompactedCount
	}

	sess.Compaction = &Compaction{
		Summary:        summary,
		CompactedCount: compactedCount,
		CompactedAt:    sess.UpdatedAt,
	}
	sess.Messages = toKeep

	return nil
}

// generateSummary uses Claude to summarize the conversation history
func (c *Compactor) generateSummary(ctx context.Context, messages []anthropic.MessageParam, existingCompaction *Compaction) (string, error) {
	// Build conversation text for summarization
	var conversationText strings.Builder

	// Include existing summary if present
	if existingCompaction != nil && existingCompaction.Summary != "" {
		conversationText.WriteString("Previous conversation summary:\n")
		conversationText.WriteString(existingCompaction.Summary)
		conversationText.WriteString("\n\n---\n\nContinued conversation:\n")
	}

	for _, msg := range messages {
		role := "User"
		if msg.Role == anthropic.MessageParamRoleAssistant {
			role = "Assistant"
		}

		// Extract text content
		content := extractTextContent(msg.Content)
		conversationText.WriteString(fmt.Sprintf("%s: %s\n\n", role, content))
	}

	// Create summarization prompt
	summaryPrompt := fmt.Sprintf(`Please provide a concise but comprehensive summary of the following conversation.
Focus on:
1. Key topics discussed
2. Important decisions or conclusions reached
3. Any tasks completed or pending
4. User preferences or context that should be remembered

Keep the summary under 1000 words but capture all essential information needed to continue the conversation meaningfully.

Conversation to summarize:
%s

Summary:`, conversationText.String())

	// Call Claude to generate summary
	resp, err := c.client.CreateMessage(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: CompactionMaxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(summaryPrompt)),
		},
	})
	if err != nil {
		return "", err
	}

	// Extract text from response
	var summary strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			summary.WriteString(text.Text)
		}
	}

	return summary.String(), nil
}

// extractTextContent extracts text from message content blocks
func extractTextContent(content []anthropic.ContentBlockParamUnion) string {
	// Marshal to JSON and extract text fields
	data, err := json.Marshal(content)
	if err != nil {
		return ""
	}

	var blocks []map[string]any
	if err := json.Unmarshal(data, &blocks); err != nil {
		return string(data) // Fallback to raw JSON
	}

	var texts []string
	for _, block := range blocks {
		if text, ok := block["text"].(string); ok {
			texts = append(texts, text)
		}
	}

	if len(texts) == 0 {
		return string(data)
	}
	return strings.Join(texts, " ")
}
