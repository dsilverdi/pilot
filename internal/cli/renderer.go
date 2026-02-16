package cli

import (
	"fmt"
	"os"

	"github.com/dsilverdi/pilot/internal/agent"
)

// Renderer handles output formatting and streaming display
type Renderer struct {
	isStreaming bool
}

// NewRenderer creates a new Renderer instance
func NewRenderer() *Renderer {
	return &Renderer{}
}

// HandleEvent processes streaming events from the agent
func (r *Renderer) HandleEvent(e agent.Event) {
	switch e.Type {
	case agent.EventText:
		if !r.isStreaming {
			r.isStreaming = true
			fmt.Print("\n")
		}
		fmt.Print(e.Text)

	case agent.EventToolCall:
		if r.isStreaming {
			fmt.Println()
			r.isStreaming = false
		}
		fmt.Printf("\n[Tool: %s]\n", e.ToolName)
		if e.ToolInput != "" {
			fmt.Printf("  Input: %s\n", truncate(e.ToolInput, 200))
		}

	case agent.EventToolResult:
		fmt.Printf("  Result: %s\n", truncate(e.ToolResult, 500))

	case agent.EventDone:
		if r.isStreaming {
			fmt.Println()
			r.isStreaming = false
		}
		fmt.Println()

	case agent.EventError:
		if r.isStreaming {
			fmt.Println()
			r.isStreaming = false
		}
		fmt.Fprintf(os.Stderr, "\n[Error]: %v\n", e.Error)
	}
}

// PrintWelcome displays the welcome message
func (r *Renderer) PrintWelcome() {
	fmt.Println("┌─────────────────────────────────────┐")
	fmt.Println("│  Pilot - AI Agent CLI               │")
	fmt.Println("│  Type /help for commands            │")
	fmt.Println("└─────────────────────────────────────┘")
	fmt.Println()
}

// PrintPrompt displays the input prompt
func (r *Renderer) PrintPrompt(sessionName string) {
	fmt.Printf("[%s] > ", sessionName)
}

// PrintError displays an error message
func (r *Renderer) PrintError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

// truncate shortens a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
