package cli

import (
	"errors"
	"fmt"
)

// ErrExit signals that the CLI should exit
var ErrExit = errors.New("exit")

// Command defines the interface for CLI commands
type Command interface {
	Name() string
	Description() string
	Execute(args []string) error
}

// HelpCommand shows available commands
type HelpCommand struct {
	commands map[string]Command
}

func (c *HelpCommand) Name() string { return "help" }

func (c *HelpCommand) Description() string {
	return "Show available commands"
}

func (c *HelpCommand) Execute(args []string) error {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  /help                - Show this help message")
	fmt.Println("  /session new [name]  - Create a new session")
	fmt.Println("  /session list        - List all sessions")
	fmt.Println("  /session switch <id> - Switch to a session")
	fmt.Println("  /session delete <id> - Delete a session")
	fmt.Println("  /clear               - Clear conversation history")
	fmt.Println("  /skill list          - List available skills")
	fmt.Println("  /skill reload        - Reload skills from directory")
	fmt.Println("  /tool list           - List available tools")
	fmt.Println("  /exit                - Exit the CLI")
	fmt.Println()
	return nil
}

// SessionCommand manages sessions
type SessionCommand struct {
	// manager will be set when session management is implemented
}

func (c *SessionCommand) Name() string { return "session" }

func (c *SessionCommand) Description() string {
	return "Manage sessions (new, list, switch, delete)"
}

func (c *SessionCommand) Execute(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: /session <new|list|switch|delete> [args]")
		return nil
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "new":
		name := "default"
		if len(subargs) > 0 {
			name = subargs[0]
		}
		fmt.Printf("Creating new session: %s\n", name)
		// TODO: Implement when session manager is ready
		return nil

	case "list":
		fmt.Println("Sessions:")
		fmt.Println("  (session management not yet implemented)")
		// TODO: Implement when session manager is ready
		return nil

	case "switch":
		if len(subargs) == 0 {
			return fmt.Errorf("session switch requires an ID")
		}
		fmt.Printf("Switching to session: %s\n", subargs[0])
		// TODO: Implement when session manager is ready
		return nil

	case "delete":
		if len(subargs) == 0 {
			return fmt.Errorf("session delete requires an ID")
		}
		fmt.Printf("Deleting session: %s\n", subargs[0])
		// TODO: Implement when session manager is ready
		return nil

	default:
		return fmt.Errorf("unknown session subcommand: %s", subcommand)
	}
}

// ClearCommand clears the conversation history
type ClearCommand struct {
	cli *CLI
}

func (c *ClearCommand) Name() string { return "clear" }

func (c *ClearCommand) Description() string {
	return "Clear conversation history"
}

func (c *ClearCommand) Execute(args []string) error {
	if c.cli != nil {
		c.cli.ClearMessages()
	}
	fmt.Println("Conversation cleared.")
	return nil
}

// SkillCommand manages skills
type SkillCommand struct {
	// loader will be set when skills are implemented
}

func (c *SkillCommand) Name() string { return "skill" }

func (c *SkillCommand) Description() string {
	return "Manage skills (list, reload)"
}

func (c *SkillCommand) Execute(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: /skill <list|reload>")
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		fmt.Println("Skills:")
		fmt.Println("  (skills not yet loaded)")
		// TODO: Implement when skill loader is ready
		return nil

	case "reload":
		fmt.Println("Reloading skills...")
		// TODO: Implement when skill loader is ready
		return nil

	default:
		return fmt.Errorf("unknown skill subcommand: %s", subcommand)
	}
}

// ToolCommand lists available tools
type ToolCommand struct {
	// registry will be set when tools are implemented
}

func (c *ToolCommand) Name() string { return "tool" }

func (c *ToolCommand) Description() string {
	return "Manage tools (list)"
}

func (c *ToolCommand) Execute(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: /tool <list>")
		return nil
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		fmt.Println("Tools:")
		fmt.Println("  (tools not yet registered)")
		// TODO: Implement when tool registry is ready
		return nil

	default:
		return fmt.Errorf("unknown tool subcommand: %s", subcommand)
	}
}

// ExitCommand exits the CLI
type ExitCommand struct{}

func (c *ExitCommand) Name() string { return "exit" }

func (c *ExitCommand) Description() string {
	return "Exit the CLI"
}

func (c *ExitCommand) Execute(args []string) error {
	fmt.Println("Goodbye!")
	return ErrExit
}
