package skills

import "errors"

var (
	ErrSkillNoName             = errors.New("skill name is required")
	ErrSkillNameTooLong        = errors.New("skill name must be 64 characters or less")
	ErrSkillNoDescription      = errors.New("skill description is required")
	ErrSkillDescriptionTooLong = errors.New("skill description must be 1024 characters or less")
	ErrSkillInvalidName        = errors.New("skill name must be lowercase alphanumeric with hyphens, cannot start/end with hyphen")
	ErrSkillNotFound           = errors.New("skill not found")
	ErrMaxDepthExceeded        = errors.New("maximum skill nesting depth exceeded")
	ErrInvalidFrontmatter      = errors.New("invalid YAML frontmatter")
	ErrNoFrontmatter           = errors.New("SKILL.md must have YAML frontmatter")
)
