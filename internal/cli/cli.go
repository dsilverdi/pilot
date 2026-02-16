package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/agent"
	"github.com/dsilverdi/pilot/internal/session"
	"github.com/dsilverdi/pilot/internal/skills"
	"github.com/dsilverdi/pilot/internal/tools"
)

// CLI handles the interactive command-line interface
type CLI struct {
	agent          *agent.Agent
	renderer       *Renderer
	commands       map[string]Command
	messages       []anthropic.MessageParam
	sessionManager *session.Manager
	skillLoaders   []*skills.Loader
	toolRegistry   *tools.Registry
	globalSkillDir string
}

// Options configures the CLI
type Options struct {
	SessionManager *session.Manager
	SkillLoaders   []*skills.Loader
	ToolRegistry   *tools.Registry
	GlobalSkillDir string
}

// New creates a new CLI instance
func New(ag *agent.Agent, opts *Options) *CLI {
	cli := &CLI{
		agent:    ag,
		renderer: NewRenderer(),
		commands: make(map[string]Command),
		messages: make([]anthropic.MessageParam, 0),
	}

	if opts != nil {
		cli.sessionManager = opts.SessionManager
		cli.skillLoaders = opts.SkillLoaders
		cli.toolRegistry = opts.ToolRegistry
		cli.globalSkillDir = opts.GlobalSkillDir
	}

	cli.registerBuiltinCommands()
	return cli
}

// registerBuiltinCommands registers all built-in CLI commands
func (c *CLI) registerBuiltinCommands() {
	helpCmd := &HelpCommand{commands: c.commands}
	c.commands["help"] = helpCmd
	c.commands["session"] = &SessionCommand{manager: c.sessionManager, cli: c}
	c.commands["clear"] = &ClearCommand{cli: c}
	c.commands["skill"] = &SkillCommand{loaders: c.skillLoaders, globalSkillDir: c.globalSkillDir}
	c.commands["tool"] = &ToolCommand{registry: c.toolRegistry}
	c.commands["exit"] = &ExitCommand{}
}

// Run starts the interactive CLI loop
func (c *CLI) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	c.renderer.PrintWelcome()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		c.renderer.PrintPrompt("default")

		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Check for commands
		if isCmd, cmd, args := ParseCommand(input); isCmd {
			if err := c.handleCommand(cmd, args); err != nil {
				if err == ErrExit {
					return nil
				}
				c.renderer.PrintError(err)
			}
			continue
		}

		// Send to agent
		c.messages = append(c.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(input)))

		newMessages, err := c.agent.Chat(ctx, c.messages, c.renderer.HandleEvent)
		if err != nil {
			c.renderer.PrintError(err)
			continue
		}

		c.messages = newMessages
		c.saveToSession()
	}
}

// handleCommand executes a CLI command
func (c *CLI) handleCommand(name string, args []string) error {
	cmd, ok := c.commands[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	return cmd.Execute(args)
}

// ParseCommand parses user input for commands
// Commands start with "/" followed immediately by a word (no space)
func ParseCommand(input string) (isCommand bool, command string, args []string) {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, "/") {
		return false, "", nil
	}

	// Commands must have "/" immediately followed by a letter (no space)
	if len(input) < 2 || !isLetter(input[1]) {
		return false, "", nil
	}

	// Remove the leading "/"
	input = strings.TrimPrefix(input, "/")

	// Split into parts
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false, "", nil
	}

	// First part is the command
	command = parts[0]

	// Rest are arguments
	if len(parts) > 1 {
		args = parts[1:]
	}

	return true, command, args
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// ClearMessages clears the conversation history
func (c *CLI) ClearMessages() {
	c.messages = make([]anthropic.MessageParam, 0)
	// Also clear from session if we have one
	if c.sessionManager != nil {
		if sess := c.sessionManager.Current(); sess != nil {
			sess.Clear()
			c.sessionManager.SaveCurrent()
		}
	}
}

// LoadMessagesFromSession loads messages from a session into the CLI
func (c *CLI) LoadMessagesFromSession(sess *session.Session) {
	c.messages = sess.GetMessages()
}

// saveToSession saves current messages to the session
func (c *CLI) saveToSession() {
	if c.sessionManager != nil {
		if sess := c.sessionManager.Current(); sess != nil {
			sess.SetMessages(c.messages)
			c.sessionManager.SaveCurrent()
		}
	}
}

// ExecutePrompt executes a single prompt and exits (non-interactive mode)
func (c *CLI) ExecutePrompt(ctx context.Context, prompt string) error {
	// Add user message
	c.messages = append(c.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))

	// Execute and stream response
	newMessages, err := c.agent.Chat(ctx, c.messages, c.renderer.HandleEvent)
	if err != nil {
		return err
	}

	c.messages = newMessages
	c.saveToSession()

	// Print newline for clean output
	fmt.Println()
	return nil
}
