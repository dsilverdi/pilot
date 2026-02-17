package session

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// Manager handles session lifecycle and switching
type Manager struct {
	store    Store
	current  *Session
	baseDir  string // Base directory for session files (e.g., ~/.pilot/sessions)
	mu       sync.RWMutex
}

// NewManager creates a new session manager
func NewManager(store Store, baseDir string) *Manager {
	return &Manager{
		store:   store,
		baseDir: baseDir,
	}
}

// Create creates a new session and sets it as current
func (m *Manager) Create(name string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := uuid.New().String()[:8]
	s := New(id, name)

	// Set files directory for shell-snapshot
	if m.baseDir != "" {
		s.FilesDir = filepath.Join(m.baseDir, id, "files")
	}

	if err := m.store.Save(s); err != nil {
		return nil, err
	}

	m.current = s
	return s, nil
}

// Switch changes the current session to the one with the given ID
func (m *Manager) Switch(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, err := m.store.Load(id)
	if err != nil {
		return fmt.Errorf("session not found: %s", id)
	}

	m.current = s
	return nil
}

// Current returns the current session
func (m *Manager) Current() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// List returns info about all sessions
func (m *Manager) List() ([]*SessionInfo, error) {
	return m.store.List()
}

// Delete removes a session
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.Delete(id); err != nil {
		return err
	}

	// Clear current if it was deleted
	if m.current != nil && m.current.ID == id {
		m.current = nil
	}

	return nil
}

// SaveCurrent persists the current session
func (m *Manager) SaveCurrent() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return nil
	}
	return m.store.Save(m.current)
}

// Save persists a session
func (m *Manager) Save(s *Session) error {
	return m.store.Save(s)
}

// GetOrCreate gets an existing session by name or creates a new one
func (m *Manager) GetOrCreate(name string) (*Session, error) {
	// Check if session with this name exists
	list, err := m.store.List()
	if err != nil {
		return nil, err
	}

	for _, info := range list {
		if info.Name == name {
			return m.store.Load(info.ID)
		}
	}

	// Create new session
	return m.Create(name)
}

// CurrentFilesDir returns the files directory for the current session
// Returns empty string if no session is active
func (m *Manager) CurrentFilesDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.current == nil {
		return ""
	}

	// If FilesDir not set (old session), compute it
	if m.current.FilesDir == "" && m.baseDir != "" {
		return filepath.Join(m.baseDir, m.current.ID, "files")
	}

	return m.current.FilesDir
}
