package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/dsilverdi/pilot/internal/session"
	"github.com/dsilverdi/pilot/internal/skills"
	"github.com/dsilverdi/pilot/internal/tools"
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
	fmt.Println("  /help                    - Show this help message")
	fmt.Println("  /session new [name]      - Create a new session")
	fmt.Println("  /session list            - List all sessions")
	fmt.Println("  /session switch <id>     - Switch to a session")
	fmt.Println("  /session delete <id>     - Delete a session")
	fmt.Println("  /clear                   - Clear conversation history")
	fmt.Println("  /compact                 - Compact conversation history (summarize old messages)")
	fmt.Println("  /skill list              - List available skills")
	fmt.Println("  /skill install <name>    - Install skill to ~/.pilot/skills/")
	fmt.Println("  /skill install --all     - Install all local skills globally")
	fmt.Println("  /skill reload            - Reload skills from directories")
	fmt.Println("  /tool list               - List available tools")
	fmt.Println("  /exit                    - Exit the CLI")
	fmt.Println()
	return nil
}

// SessionCommand manages sessions
type SessionCommand struct {
	manager *session.Manager
	cli     *CLI
}

func (c *SessionCommand) Name() string { return "session" }

func (c *SessionCommand) Description() string {
	return "Manage sessions (new, list, switch, delete)"
}

func (c *SessionCommand) Execute(args []string) error {
	if c.manager == nil {
		return fmt.Errorf("session management not configured")
	}

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
		sess, err := c.manager.Create(name)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		// Clear current messages for new session
		if c.cli != nil {
			c.cli.ClearMessages()
		}
		fmt.Printf("Created new session: %s (%s)\n", sess.Name, sess.ID)
		return nil

	case "list":
		list, err := c.manager.List()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		if len(list) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}
		fmt.Println("\nSessions:")
		current := c.manager.Current()
		for _, info := range list {
			marker := "  "
			if current != nil && current.ID == info.ID {
				marker = "* "
			}
			fmt.Printf("%s%s  %s  (%d messages)\n", marker, info.ID, info.Name, info.MessageCount)
		}
		fmt.Println()
		return nil

	case "switch":
		if len(subargs) == 0 {
			return fmt.Errorf("session switch requires an ID")
		}
		if err := c.manager.Switch(subargs[0]); err != nil {
			return err
		}
		sess := c.manager.Current()
		// Load messages from session
		if c.cli != nil && sess != nil {
			c.cli.LoadMessagesFromSession(sess)
		}
		fmt.Printf("Switched to session: %s (%s)\n", sess.Name, sess.ID)
		return nil

	case "delete":
		if len(subargs) == 0 {
			return fmt.Errorf("session delete requires an ID")
		}
		if err := c.manager.Delete(subargs[0]); err != nil {
			return fmt.Errorf("failed to delete session: %w", err)
		}
		fmt.Printf("Deleted session: %s\n", subargs[0])
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

// CompactCommand compacts conversation history
type CompactCommand struct {
	cli *CLI
}

func (c *CompactCommand) Name() string { return "compact" }

func (c *CompactCommand) Description() string {
	return "Compact conversation history by summarizing old messages"
}

func (c *CompactCommand) Execute(args []string) error {
	if c.cli == nil {
		return fmt.Errorf("CLI not configured")
	}
	return c.cli.CompactMessages()
}

// SkillCommand manages skills
type SkillCommand struct {
	loaders        []*skills.Loader
	globalSkillDir string
}

func (c *SkillCommand) Name() string { return "skill" }

func (c *SkillCommand) Description() string {
	return "Manage skills (list, install, reload)"
}

func (c *SkillCommand) Execute(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: /skill <list|install|reload>")
		return nil
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		if len(c.loaders) == 0 {
			fmt.Println("No skill loaders configured.")
			return nil
		}

		var allSkills []*skills.Skill
		for _, loader := range c.loaders {
			allSkills = append(allSkills, loader.All()...)
		}

		if len(allSkills) == 0 {
			fmt.Println("No skills loaded.")
			return nil
		}

		// Sort skills by name
		sort.Slice(allSkills, func(i, j int) bool {
			return allSkills[i].Name < allSkills[j].Name
		})

		fmt.Println("\nAvailable skills:")
		for _, skill := range allSkills {
			fmt.Printf("  %s - %s\n", skill.Name, skill.Description)
		}
		fmt.Println()
		return nil

	case "install":
		if len(subargs) == 0 {
			fmt.Println("Usage: /skill install <name>")
			fmt.Println("       /skill install --all")
			return nil
		}

		if c.globalSkillDir == "" {
			return fmt.Errorf("global skill directory not configured")
		}

		// Handle --all flag
		if subargs[0] == "--all" {
			return c.installAllSkills()
		}

		return c.installSkill(subargs[0])

	case "reload":
		if len(c.loaders) == 0 {
			fmt.Println("No skill loaders configured.")
			return nil
		}

		totalCount := 0
		for _, loader := range c.loaders {
			if err := loader.LoadAll(); err != nil {
				fmt.Printf("Warning: failed to reload skills: %v\n", err)
				continue
			}
			totalCount += loader.Count()
		}
		fmt.Printf("Reloaded %d skills.\n", totalCount)
		return nil

	default:
		return fmt.Errorf("unknown skill subcommand: %s", subcommand)
	}
}

// installSkill copies a skill from local to global directory
func (c *SkillCommand) installSkill(name string) error {
	// Find the skill in loaders
	var skill *skills.Skill
	for _, loader := range c.loaders {
		if s, ok := loader.Get(name); ok {
			skill = s
			break
		}
	}

	if skill == nil {
		return fmt.Errorf("skill not found: %s", name)
	}

	// Get skill source directory from SourcePath
	srcDir := filepath.Dir(skill.Path)
	dstDir := filepath.Join(c.globalSkillDir, name)

	// Check if already installed
	if _, err := os.Stat(dstDir); err == nil {
		return fmt.Errorf("skill already installed: %s (use /skill reload after manual update)", name)
	}

	// Copy the skill directory
	if err := copyDir(srcDir, dstDir); err != nil {
		return fmt.Errorf("failed to install skill: %w", err)
	}

	fmt.Printf("Installed skill '%s' to %s\n", name, dstDir)

	// Reload the first loader (global) to pick up the new skill
	if len(c.loaders) > 0 {
		c.loaders[0].LoadAll()
	}

	return nil
}

// installAllSkills installs all local skills to global directory
func (c *SkillCommand) installAllSkills() error {
	var installed int
	var skipped int

	for _, loader := range c.loaders[1:] { // Skip first loader (global)
		for _, skill := range loader.All() {
			dstDir := filepath.Join(c.globalSkillDir, skill.Name)

			// Skip if already installed
			if _, err := os.Stat(dstDir); err == nil {
				skipped++
				continue
			}

			srcDir := filepath.Dir(skill.Path)
			if err := copyDir(srcDir, dstDir); err != nil {
				fmt.Printf("Warning: failed to install %s: %v\n", skill.Name, err)
				continue
			}
			installed++
		}
	}

	// Reload global loader
	if len(c.loaders) > 0 {
		c.loaders[0].LoadAll()
	}

	fmt.Printf("Installed %d skills (%d already installed)\n", installed, skipped)
	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file info for permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// ToolCommand lists available tools
type ToolCommand struct {
	registry *tools.Registry
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
		if c.registry == nil {
			fmt.Println("No tool registry configured.")
			return nil
		}

		toolList := c.registry.ListTools()
		if len(toolList) == 0 {
			fmt.Println("No tools registered.")
			return nil
		}

		fmt.Println("\nAvailable tools:")
		for _, tool := range toolList {
			fmt.Printf("  %s - %s\n", tool.Name, tool.Description)
		}
		fmt.Println()
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
