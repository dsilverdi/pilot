package cli

import (
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		isCommand bool
		command   string
		args      []string
	}{
		{
			name:      "help command",
			input:     "/help",
			isCommand: true,
			command:   "help",
			args:      nil,
		},
		{
			name:      "session new with name",
			input:     "/session new test-session",
			isCommand: true,
			command:   "session",
			args:      []string{"new", "test-session"},
		},
		{
			name:      "exit command",
			input:     "/exit",
			isCommand: true,
			command:   "exit",
			args:      nil,
		},
		{
			name:      "regular text",
			input:     "Hello, how are you?",
			isCommand: false,
			command:   "",
			args:      nil,
		},
		{
			name:      "text starting with slash but not command",
			input:     "/ this is not a command",
			isCommand: false,
			command:   "",
			args:      nil,
		},
		{
			name:      "skill list command",
			input:     "/skill list",
			isCommand: true,
			command:   "skill",
			args:      []string{"list"},
		},
		{
			name:      "tool list command",
			input:     "/tool list",
			isCommand: true,
			command:   "tool",
			args:      []string{"list"},
		},
		{
			name:      "clear command",
			input:     "/clear",
			isCommand: true,
			command:   "clear",
			args:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isCmd, cmd, args := ParseCommand(tt.input)

			if isCmd != tt.isCommand {
				t.Errorf("ParseCommand() isCommand = %v, want %v", isCmd, tt.isCommand)
			}
			if cmd != tt.command {
				t.Errorf("ParseCommand() command = %q, want %q", cmd, tt.command)
			}
			if len(args) != len(tt.args) {
				t.Errorf("ParseCommand() args length = %d, want %d", len(args), len(tt.args))
			}
			for i := range args {
				if i < len(tt.args) && args[i] != tt.args[i] {
					t.Errorf("ParseCommand() args[%d] = %q, want %q", i, args[i], tt.args[i])
				}
			}
		})
	}
}

func TestBuiltinCommands(t *testing.T) {
	// Test that all expected commands are registered
	expectedCommands := []string{"help", "session", "clear", "skill", "tool", "exit"}

	cli := &CLI{
		commands: make(map[string]Command),
	}
	cli.registerBuiltinCommands()

	for _, cmdName := range expectedCommands {
		if _, ok := cli.commands[cmdName]; !ok {
			t.Errorf("expected command %q to be registered", cmdName)
		}
	}
}

func TestHelpCommand(t *testing.T) {
	cmd := &HelpCommand{
		commands: map[string]Command{
			"help": &HelpCommand{},
			"exit": &ExitCommand{},
		},
	}

	if cmd.Name() != "help" {
		t.Errorf("HelpCommand.Name() = %q, want %q", cmd.Name(), "help")
	}

	desc := cmd.Description()
	if desc == "" {
		t.Error("HelpCommand.Description() should not be empty")
	}
}

func TestExitCommand(t *testing.T) {
	cmd := &ExitCommand{}

	if cmd.Name() != "exit" {
		t.Errorf("ExitCommand.Name() = %q, want %q", cmd.Name(), "exit")
	}

	desc := cmd.Description()
	if desc == "" {
		t.Error("ExitCommand.Description() should not be empty")
	}
}

func TestSessionCommand(t *testing.T) {
	cmd := &SessionCommand{}

	if cmd.Name() != "session" {
		t.Errorf("SessionCommand.Name() = %q, want %q", cmd.Name(), "session")
	}

	desc := cmd.Description()
	if !strings.Contains(desc, "new") || !strings.Contains(desc, "list") {
		t.Error("SessionCommand.Description() should mention subcommands")
	}
}

func TestClearCommand(t *testing.T) {
	cmd := &ClearCommand{}

	if cmd.Name() != "clear" {
		t.Errorf("ClearCommand.Name() = %q, want %q", cmd.Name(), "clear")
	}
}

func TestSkillCommand(t *testing.T) {
	cmd := &SkillCommand{}

	if cmd.Name() != "skill" {
		t.Errorf("SkillCommand.Name() = %q, want %q", cmd.Name(), "skill")
	}

	desc := cmd.Description()
	if !strings.Contains(desc, "list") {
		t.Error("SkillCommand.Description() should mention list subcommand")
	}
}

func TestToolCommand(t *testing.T) {
	cmd := &ToolCommand{}

	if cmd.Name() != "tool" {
		t.Errorf("ToolCommand.Name() = %q, want %q", cmd.Name(), "tool")
	}

	desc := cmd.Description()
	if !strings.Contains(desc, "list") {
		t.Error("ToolCommand.Description() should mention list subcommand")
	}
}
