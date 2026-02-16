package session

import "time"

// Store defines the interface for session persistence
type Store interface {
	// Save persists a session
	Save(session *Session) error

	// Load retrieves a session by ID
	Load(id string) (*Session, error)

	// Delete removes a session by ID
	Delete(id string) error

	// List returns info about all stored sessions
	List() ([]*SessionInfo, error)
}

// SessionInfo contains summary information about a session
type SessionInfo struct {
	ID           string
	Name         string
	MessageCount int
	UpdatedAt    time.Time
}
