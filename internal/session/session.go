package session

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// Compaction holds information about compacted conversation history
type Compaction struct {
	Summary        string    `json:"summary"`         // Summary of compacted messages
	CompactedCount int       `json:"compacted_count"` // Number of messages that were compacted
	CompactedAt    time.Time `json:"compacted_at"`    // When compaction occurred
}

// Session represents a conversation session
type Session struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Messages   []anthropic.MessageParam `json:"messages"`
	FilesDir   string                   `json:"files_dir,omitempty"`  // Directory for session shell files
	Compaction *Compaction              `json:"compaction,omitempty"` // Compacted history summary
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`

	mu sync.RWMutex
}

// New creates a new session with the given ID and name
func New(id, name string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		Name:      name,
		Messages:  make([]anthropic.MessageParam, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage adds a message to the session
func (s *Session) AddMessage(msg anthropic.MessageParam) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

// GetMessages returns a copy of the session's messages
func (s *Session) GetMessages() []anthropic.MessageParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent race conditions
	msgs := make([]anthropic.MessageParam, len(s.Messages))
	copy(msgs, s.Messages)
	return msgs
}

// Clear removes all messages from the session
func (s *Session) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = make([]anthropic.MessageParam, 0)
	s.UpdatedAt = time.Now()
}

// MessageCount returns the number of messages in the session
func (s *Session) MessageCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.Messages)
}

// SetMessages replaces all messages (used when loading from storage)
func (s *Session) SetMessages(msgs []anthropic.MessageParam) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Messages = make([]anthropic.MessageParam, len(msgs))
	copy(s.Messages, msgs)
}

// EstimateTokens estimates the token count for all messages
// Uses rough approximation: ~4 characters per token for English text
func (s *Session) EstimateTokens() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return estimateMessagesTokens(s.Messages)
}

// estimateMessagesTokens estimates tokens for a slice of messages
func estimateMessagesTokens(messages []anthropic.MessageParam) int {
	totalChars := 0
	for _, msg := range messages {
		// Marshal content to get text representation
		data, err := json.Marshal(msg.Content)
		if err != nil {
			continue
		}
		totalChars += len(data)
	}
	// Rough estimate: ~4 chars per token
	return totalChars / 4
}

// NeedsCompaction checks if the session needs compaction based on token threshold
func (s *Session) NeedsCompaction(threshold int) bool {
	return s.EstimateTokens() > threshold
}

// SetCompaction sets the compaction summary
func (s *Session) SetCompaction(summary string, compactedCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Compaction = &Compaction{
		Summary:        summary,
		CompactedCount: compactedCount,
		CompactedAt:    time.Now(),
	}
	s.UpdatedAt = time.Now()
}

// GetCompaction returns the current compaction info
func (s *Session) GetCompaction() *Compaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Compaction
}

// HasCompaction returns true if the session has been compacted
func (s *Session) HasCompaction() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Compaction != nil && s.Compaction.Summary != ""
}

// GetMessagesWithContext returns messages prefixed with compaction summary if available
func (s *Session) GetMessagesWithContext() []anthropic.MessageParam {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.Compaction == nil || s.Compaction.Summary == "" {
		msgs := make([]anthropic.MessageParam, len(s.Messages))
		copy(msgs, s.Messages)
		return msgs
	}

	// Prepend compaction summary as a system context message
	contextMsg := anthropic.NewUserMessage(anthropic.NewTextBlock(
		"[Previous conversation summary]\n" + s.Compaction.Summary + "\n[End of summary - conversation continues below]",
	))

	msgs := make([]anthropic.MessageParam, 0, len(s.Messages)+1)
	msgs = append(msgs, contextMsg)
	msgs = append(msgs, s.Messages...)
	return msgs
}
