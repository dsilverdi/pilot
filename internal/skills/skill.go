package skills

import "time"

// Skill represents a loaded skill from SKILL.md
type Skill struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
	Content     string            `yaml:"-"` // Markdown body (not from frontmatter)
	Path        string            `yaml:"-"` // Path to SKILL.md file
	LoadedAt    time.Time         `yaml:"-"`
}

// Validate checks if the skill has required fields
func (s *Skill) Validate() error {
	if s.Name == "" {
		return ErrSkillNoName
	}
	if len(s.Name) > 64 {
		return ErrSkillNameTooLong
	}
	if s.Description == "" {
		return ErrSkillNoDescription
	}
	if len(s.Description) > 1024 {
		return ErrSkillDescriptionTooLong
	}
	// Validate name format: lowercase, numbers, hyphens
	for i, c := range s.Name {
		isLower := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		isHyphen := c == '-'

		if !isLower && !isDigit && !isHyphen {
			return ErrSkillInvalidName
		}
		// Cannot start or end with hyphen
		if isHyphen && (i == 0 || i == len(s.Name)-1) {
			return ErrSkillInvalidName
		}
	}
	return nil
}
