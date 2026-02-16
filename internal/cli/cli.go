package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/dsilverdi/pilot/internal/agent"
)

// CLI handles the interactive command-line interface
type CLI struct {
	agent    *agent.Agent
	renderer *Renderer
	commands map[string]Command
	messages []anthropic.MessageParam
}

// New creates a new CLI instance
func New(ag *agent.Agent) *CLI {
	cli := &CLI{
		agent:    ag,
		renderer: NewRenderer(),
		commands: make(map[string]Command),
		messages: make([]anthropic.MessageParam, 0),
	}
	cli.registerBuiltinCommands()
	return cli
}

// registerBuiltinCommands registers all built-in CLI commands
func (c *CLI) registerBuiltinCommands() {
	helpCmd := &HelpCommand{commands: c.commands}
	c.commands["help"] = helpCmd
	c.commands["session"] = &SessionCommand{}
	c.commands["clear"] = &ClearCommand{cli: c}
	c.commands["skill"] = &SkillCommand{}
	c.commands["tool"] = &ToolCommand{}
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
}

// ExecutePrompt executes a single prompt and exits (non-interactive mode)
func (c *CLI) ExecutePrompt(ctx context.Context, prompt string) error {
	// Add user message
	c.messages = append(c.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))

	// Execute and stream response
	_, err := c.agent.Chat(ctx, c.messages, c.renderer.HandleEvent)
	if err != nil {
		return err
	}

	// Print newline for clean output
	fmt.Println()
	return nil
}
