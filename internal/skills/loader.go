package skills

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Loader loads skills from a directory
type Loader struct {
	dir    string
	skills map[string]*Skill
	mu     sync.RWMutex
}

// NewLoader creates a new skill loader
func NewLoader(dir string) *Loader {
	return &Loader{
		dir:    dir,
		skills: make(map[string]*Skill),
	}
}

// LoadAll loads all skills from the directory
func (l *Loader) LoadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear existing skills
	l.skills = make(map[string]*Skill)

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip dot directories
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		skillPath := filepath.Join(l.dir, entry.Name(), "SKILL.md")
		skill, err := l.loadSkillFile(skillPath)
		if err != nil {
			continue // Skip invalid skills
		}

		l.skills[skill.Name] = skill
	}

	return nil
}

// loadSkillFile loads a single skill from a file
func (l *Loader) loadSkillFile(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseSkillFile(string(content), path)
}

// Get retrieves a skill by name
func (l *Loader) Get(name string) (*Skill, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	skill, ok := l.skills[name]
	return skill, ok
}

// All returns all loaded skills
func (l *Loader) All() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skills := make([]*Skill, 0, len(l.skills))
	for _, skill := range l.skills {
		skills = append(skills, skill)
	}
	return skills
}

// Count returns the number of loaded skills
func (l *Loader) Count() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.skills)
}
