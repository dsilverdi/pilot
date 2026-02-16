package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/skills"
)

// InvokeSkillTool allows the agent to invoke a skill
type InvokeSkillTool struct {
	loaders []*skills.Loader
}

// NewInvokeSkillTool creates a new invoke skill tool
func NewInvokeSkillTool(loaders []*skills.Loader) *InvokeSkillTool {
	return &InvokeSkillTool{loaders: loaders}
}

func (t *InvokeSkillTool) Name() string { return "invoke_skill" }

func (t *InvokeSkillTool) Description() string {
	// Build description with available skills
	var sb strings.Builder
	sb.WriteString("Invoke a skill to get specialized instructions for a task. ")
	sb.WriteString("Use this tool when the user's request matches one of the available skills. ")
	sb.WriteString("The skill will provide detailed instructions on how to complete the task.\n\n")
	sb.WriteString("Available skills:\n")

	skillList := t.getAllSkills()
	if len(skillList) == 0 {
		sb.WriteString("  (no skills loaded)")
	} else {
		for _, skill := range skillList {
			// Truncate description if too long
			desc := skill.Description
			if len(desc) > 150 {
				desc = desc[:147] + "..."
			}
			sb.WriteString(fmt.Sprintf("- %s: %s\n", skill.Name, desc))
		}
	}

	return sb.String()
}

func (t *InvokeSkillTool) InputSchema() anthropic.ToolInputSchemaParam {
	// Get skill names for enum
	var skillNames []any
	for _, skill := range t.getAllSkills() {
		skillNames = append(skillNames, skill.Name)
	}

	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"skill_name": map[string]any{
				"type":        "string",
				"description": "The name of the skill to invoke",
				"enum":        skillNames,
			},
			"context": map[string]any{
				"type":        "string",
				"description": "Additional context about what the user wants to accomplish (optional)",
			},
		},
		Required: []string{"skill_name"},
	}
}

type invokeSkillInput struct {
	SkillName string `json:"skill_name"`
	Context   string `json:"context,omitempty"`
}

func (t *InvokeSkillTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in invokeSkillInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if in.SkillName == "" {
		return "", fmt.Errorf("skill_name is required")
	}

	// Find the skill
	skill := t.getSkill(in.SkillName)
	if skill == nil {
		return "", fmt.Errorf("skill not found: %s", in.SkillName)
	}

	// Build response with skill instructions
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Skill: %s\n\n", skill.Name))

	if in.Context != "" {
		sb.WriteString(fmt.Sprintf("## User Context\n%s\n\n", in.Context))
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString(skill.Content)

	return sb.String(), nil
}

// getAllSkills returns all skills from all loaders
func (t *InvokeSkillTool) getAllSkills() []*skills.Skill {
	seen := make(map[string]bool)
	var result []*skills.Skill

	for _, loader := range t.loaders {
		for _, skill := range loader.All() {
			if !seen[skill.Name] {
				seen[skill.Name] = true
				result = append(result, skill)
			}
		}
	}

	return result
}

// getSkill finds a skill by name
func (t *InvokeSkillTool) getSkill(name string) *skills.Skill {
	for _, loader := range t.loaders {
		if skill, ok := loader.Get(name); ok {
			return skill
		}
	}
	return nil
}
