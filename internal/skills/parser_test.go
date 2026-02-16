package skills

import (
	"strings"
	"testing"
)

func TestParseSkillValid(t *testing.T) {
	input := `---
name: test-skill
description: A test skill for testing purposes.
version: 1.0.0
---

# Test Skill

This is the skill content.
`

	skill, err := ParseSkill(input)
	if err != nil {
		t.Fatalf("ParseSkill() error = %v", err)
	}

	if skill.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", skill.Name)
	}
	if skill.Description != "A test skill for testing purposes." {
		t.Errorf("expected description 'A test skill for testing purposes.', got %q", skill.Description)
	}
	if skill.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", skill.Version)
	}
	if !strings.Contains(skill.Content, "# Test Skill") {
		t.Error("content should contain markdown body")
	}
}

func TestParseSkillMinimal(t *testing.T) {
	input := `---
name: minimal
description: Minimal skill
---

Content here.
`

	skill, err := ParseSkill(input)
	if err != nil {
		t.Fatalf("ParseSkill() error = %v", err)
	}

	if skill.Name != "minimal" {
		t.Errorf("expected name 'minimal', got %q", skill.Name)
	}
	if skill.Version != "" {
		t.Errorf("expected empty version, got %q", skill.Version)
	}
}

func TestParseSkillNoFrontmatter(t *testing.T) {
	input := `# Just markdown

No frontmatter here.
`

	_, err := ParseSkill(input)
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseSkillInvalidYAML(t *testing.T) {
	input := `---
name: test
description: [invalid yaml
---

Content.
`

	_, err := ParseSkill(input)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseSkillMissingName(t *testing.T) {
	input := `---
description: Missing name field
---

Content.
`

	_, err := ParseSkill(input)
	if err == nil {
		t.Error("expected error for missing name")
	}
}

func TestParseSkillMissingDescription(t *testing.T) {
	input := `---
name: test-skill
---

Content.
`

	_, err := ParseSkill(input)
	if err == nil {
		t.Error("expected error for missing description")
	}
}

func TestParseSkillInvalidName(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name: "uppercase",
			input: `---
name: Test-Skill
description: Test
---
`,
		},
		{
			name: "starts with hyphen",
			input: `---
name: -test
description: Test
---
`,
		},
		{
			name: "ends with hyphen",
			input: `---
name: test-
description: Test
---
`,
		},
		{
			name: "contains space",
			input: `---
name: test skill
description: Test
---
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSkill(tt.input)
			if err == nil {
				t.Error("expected error for invalid name")
			}
		})
	}
}

func TestParseSkillWithMetadata(t *testing.T) {
	input := `---
name: with-meta
description: Skill with metadata
metadata:
  author: test
  license: MIT
---

Content.
`

	skill, err := ParseSkill(input)
	if err != nil {
		t.Fatalf("ParseSkill() error = %v", err)
	}

	if skill.Metadata == nil {
		t.Fatal("metadata should not be nil")
	}
	if skill.Metadata["author"] != "test" {
		t.Errorf("expected author 'test', got %q", skill.Metadata["author"])
	}
}

func TestParseSkillLongDescription(t *testing.T) {
	longDesc := strings.Repeat("x", 1025)
	input := `---
name: test
description: ` + longDesc + `
---

Content.
`

	_, err := ParseSkill(input)
	if err == nil {
		t.Error("expected error for description too long")
	}
}

func TestParseSkillContentPreservesFormatting(t *testing.T) {
	input := `---
name: test
description: Test
---

# Heading

- Item 1
- Item 2

` + "```go" + `
code here
` + "```" + `
`

	skill, err := ParseSkill(input)
	if err != nil {
		t.Fatalf("ParseSkill() error = %v", err)
	}

	if !strings.Contains(skill.Content, "# Heading") {
		t.Error("content should preserve headings")
	}
	if !strings.Contains(skill.Content, "- Item 1") {
		t.Error("content should preserve lists")
	}
	if !strings.Contains(skill.Content, "code here") {
		t.Error("content should preserve code blocks")
	}
}
