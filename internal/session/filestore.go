package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// FileStore implements Store using the filesystem
type FileStore struct {
	dir string
}

// NewFileStore creates a new file-based session store
func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &FileStore{dir: dir}, nil
}

// Save persists a session to a JSON file
func (fs *FileStore) Save(s *Session) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(fs.dir, s.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// Load retrieves a session from a JSON file
func (fs *FileStore) Load(id string) (*Session, error) {
	path := filepath.Join(fs.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete removes a session file
func (fs *FileStore) Delete(id string) error {
	path := filepath.Join(fs.dir, id+".json")
	return os.Remove(path)
}

// List returns info about all stored sessions
func (fs *FileStore) List() ([]*SessionInfo, error) {
	entries, err := os.ReadDir(fs.dir)
	if err != nil {
		return nil, err
	}

	var infos []*SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		id := strings.TrimSuffix(entry.Name(), ".json")
		s, err := fs.Load(id)
		if err != nil {
			continue // Skip corrupted files
		}

		infos = append(infos, &SessionInfo{
			ID:           s.ID,
			Name:         s.Name,
			MessageCount: len(s.Messages),
			UpdatedAt:    s.UpdatedAt,
		})
	}

	return infos, nil
}
