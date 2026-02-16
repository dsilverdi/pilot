package skills

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ParseSkill parses a SKILL.md file content into a Skill struct
func ParseSkill(content string) (*Skill, error) {
	// Check for frontmatter delimiters
	if !strings.HasPrefix(content, "---") {
		return nil, ErrNoFrontmatter
	}

	// Find the end of frontmatter
	endIndex := strings.Index(content[3:], "---")
	if endIndex == -1 {
		return nil, ErrNoFrontmatter
	}

	// Extract frontmatter and body
	frontmatter := content[3 : endIndex+3]
	body := strings.TrimSpace(content[endIndex+6:])

	// Parse YAML frontmatter
	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFrontmatter, err)
	}

	// Set content and metadata
	skill.Content = body
	skill.LoadedAt = time.Now()

	// Validate
	if err := skill.Validate(); err != nil {
		return nil, err
	}

	return &skill, nil
}

// ParseSkillFile parses a SKILL.md file and sets the path
func ParseSkillFile(content, path string) (*Skill, error) {
	skill, err := ParseSkill(content)
	if err != nil {
		return nil, err
	}
	skill.Path = path
	return skill, nil
}
