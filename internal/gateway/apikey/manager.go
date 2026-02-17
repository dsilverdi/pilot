package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	keyPrefix = "psk_"
	keyLength = 32 // 32 bytes = 256 bits
)

// KeyInfo represents stored API key information
type KeyInfo struct {
	Name      string    `json:"name"`
	KeyHash   string    `json:"key_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// KeyStore represents the storage format for API keys
type KeyStore struct {
	Keys []KeyInfo `json:"keys"`
}

// Manager handles API key generation, validation, and storage
type Manager struct {
	keysFile string
	mu       sync.RWMutex
}

// NewManager creates a new API key manager
func NewManager(pilotDir string) *Manager {
	return &Manager{
		keysFile: filepath.Join(pilotDir, "api-keys.json"),
	}
}

// Generate creates a new API key with the given name
func (m *Manager) Generate(name string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		return "", fmt.Errorf("key name is required")
	}

	// Load existing keys
	store, err := m.loadStore()
	if err != nil {
		return "", err
	}

	// Check if name already exists
	for _, k := range store.Keys {
		if k.Name == name {
			return "", fmt.Errorf("key with name %q already exists", name)
		}
	}

	// Generate random key
	randomBytes := make([]byte, keyLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Create the plain key
	plainKey := keyPrefix + base64.URLEncoding.EncodeToString(randomBytes)

	// Hash the key for storage
	hash := sha256.Sum256([]byte(plainKey))
	keyHash := "sha256:" + hex.EncodeToString(hash[:])

	// Add to store
	store.Keys = append(store.Keys, KeyInfo{
		Name:      name,
		KeyHash:   keyHash,
		CreatedAt: time.Now().UTC(),
	})

	// Save store
	if err := m.saveStore(store); err != nil {
		return "", err
	}

	return plainKey, nil
}

// Validate checks if the given key is valid
func (m *Manager) Validate(key string) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if key == "" {
		return false, "", nil
	}

	// Hash the provided key
	hash := sha256.Sum256([]byte(key))
	keyHash := "sha256:" + hex.EncodeToString(hash[:])

	// Load and check against stored keys
	store, err := m.loadStore()
	if err != nil {
		return false, "", err
	}

	for _, k := range store.Keys {
		if k.KeyHash == keyHash {
			return true, k.Name, nil
		}
	}

	return false, "", nil
}

// List returns all stored keys (with masked key values)
func (m *Manager) List() ([]KeyInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store, err := m.loadStore()
	if err != nil {
		return nil, err
	}

	return store.Keys, nil
}

// Revoke removes a key by name
func (m *Manager) Revoke(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, err := m.loadStore()
	if err != nil {
		return err
	}

	// Find and remove the key
	found := false
	newKeys := make([]KeyInfo, 0, len(store.Keys))
	for _, k := range store.Keys {
		if k.Name == name {
			found = true
			continue
		}
		newKeys = append(newKeys, k)
	}

	if !found {
		return fmt.Errorf("key with name %q not found", name)
	}

	store.Keys = newKeys
	return m.saveStore(store)
}

// HasKeys returns true if any API keys are configured
func (m *Manager) HasKeys() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store, err := m.loadStore()
	if err != nil {
		return false
	}

	return len(store.Keys) > 0
}

func (m *Manager) loadStore() (*KeyStore, error) {
	data, err := os.ReadFile(m.keysFile)
	if os.IsNotExist(err) {
		return &KeyStore{Keys: []KeyInfo{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read keys file: %w", err)
	}

	var store KeyStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("failed to parse keys file: %w", err)
	}

	return &store, nil
}

func (m *Manager) saveStore(store *KeyStore) error {
	// Ensure directory exists
	dir := filepath.Dir(m.keysFile)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize keys: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(m.keysFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write keys file: %w", err)
	}

	return nil
}
