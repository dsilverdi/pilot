package session

import (
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// Session represents a conversation session
type Session struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Messages  []anthropic.MessageParam `json:"messages"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`

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
