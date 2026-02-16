package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir)
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestLoaderLoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill directory
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	// Create SKILL.md
	skillContent := `---
name: test-skill
description: A test skill
---

# Test Skill

Content here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}

	loader := NewLoader(tmpDir)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(loader.skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(loader.skills))
	}
}

func TestLoaderLoadAllEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	loader := NewLoader(tmpDir)
	if err := loader.LoadAll(); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	if len(loader.skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(loader.skills))
	}
}

func TestLoaderLoadAllSkipsInvalid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid skill
	validDir := filepath.Join(tmpDir, "valid-skill")
	os.MkdirAll(validDir, 0755)
	os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte(`---
name: valid-skill
description: Valid
---
Content.
`), 0644)

	// Create invalid skill (no frontmatter)
	invalidDir := filepath.Join(tmpDir, "invalid-skill")
	os.MkdirAll(invalidDir, 0755)
	os.WriteFile(filepath.Join(invalidDir, "SKILL.md"), []byte(`Just content, no frontmatter.`), 0644)

	loader := NewLoader(tmpDir)
	loader.LoadAll()

	// Should have 1 valid skill
	if len(loader.skills) != 1 {
		t.Errorf("expected 1 valid skill, got %d", len(loader.skills))
	}
}

func TestLoaderGet(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "my-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: my-skill
description: My skill
---
Content.
`), 0644)

	loader := NewLoader(tmpDir)
	loader.LoadAll()

	skill, ok := loader.Get("my-skill")
	if !ok {
		t.Fatal("expected to find skill")
	}
	if skill.Name != "my-skill" {
		t.Errorf("expected name 'my-skill', got %q", skill.Name)
	}

	_, ok = loader.Get("nonexistent")
	if ok {
		t.Error("expected not to find nonexistent skill")
	}
}

func TestLoaderAll(t *testing.T) {
	tmpDir := t.TempDir()

	for _, name := range []string{"skill-a", "skill-b"} {
		dir := filepath.Join(tmpDir, name)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(`---
name: `+name+`
description: Test skill
---
Content.
`), 0644)
	}

	loader := NewLoader(tmpDir)
	loader.LoadAll()

	skills := loader.All()
	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(skills))
	}
}

func TestLoaderReload(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial skill
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: Original description
---
Content.
`), 0644)

	loader := NewLoader(tmpDir)
	loader.LoadAll()

	skill, _ := loader.Get("test-skill")
	if skill.Description != "Original description" {
		t.Errorf("expected original description")
	}

	// Update skill
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: Updated description
---
Content.
`), 0644)

	// Reload
	loader.LoadAll()

	skill, _ = loader.Get("test-skill")
	if skill.Description != "Updated description" {
		t.Errorf("expected updated description")
	}
}

func TestLoaderSkipsDotDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .hidden skill directory
	hiddenDir := filepath.Join(tmpDir, ".hidden-skill")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte(`---
name: hidden-skill
description: Hidden
---
`), 0644)

	loader := NewLoader(tmpDir)
	loader.LoadAll()

	if len(loader.skills) != 0 {
		t.Error("should skip dot directories")
	}
}
