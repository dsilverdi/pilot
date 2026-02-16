package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// BashExecTool executes bash commands
type BashExecTool struct {
	workDir        string
	sessionDir     string // Directory for session-specific files
	defaultTimeout time.Duration
}

// NewBashExecTool creates a new bash execution tool
func NewBashExecTool(workDir string, sessionDir string) *BashExecTool {
	return &BashExecTool{
		workDir:        workDir,
		sessionDir:     sessionDir,
		defaultTimeout: 120 * time.Second, // 2 minute default timeout
	}
}

func (t *BashExecTool) Name() string { return "bash_exec" }

func (t *BashExecTool) Description() string {
	return `Execute a bash command and return the output. Use this to:
- Run scripts (node script.js, python script.py)
- Install dependencies (npm install, pip install)
- Check system state (which node, pip show package)
- File operations (ls, mkdir, mv, cp)

Commands run in the current working directory. Use 'cd' within the command if needed.
Timeout: 2 minutes. For long-running commands, consider background execution.

Session files directory: ` + t.sessionDir
}

func (t *BashExecTool) InputSchema() anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
			"timeout_seconds": map[string]any{
				"type":        "integer",
				"description": "Optional timeout in seconds (default: 120, max: 300)",
			},
			"working_dir": map[string]any{
				"type":        "string",
				"description": "Optional working directory (default: current directory). Use 'session' for session files directory.",
			},
		},
		Required: []string{"command"},
	}
}

type bashExecInput struct {
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
	WorkingDir     string `json:"working_dir,omitempty"`
}

func (t *BashExecTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var in bashExecInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if in.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Determine timeout
	timeout := t.defaultTimeout
	if in.TimeoutSeconds > 0 {
		if in.TimeoutSeconds > 300 {
			in.TimeoutSeconds = 300 // Max 5 minutes
		}
		timeout = time.Duration(in.TimeoutSeconds) * time.Second
	}

	// Determine working directory
	workDir := t.workDir
	if in.WorkingDir != "" {
		if in.WorkingDir == "session" {
			workDir = t.sessionDir
			// Ensure session directory exists
			if err := os.MkdirAll(workDir, 0755); err != nil {
				return "", fmt.Errorf("failed to create session directory: %w", err)
			}
		} else if filepath.IsAbs(in.WorkingDir) {
			workDir = in.WorkingDir
		} else {
			workDir = filepath.Join(t.workDir, in.WorkingDir)
		}
	}

	// Create command with timeout context
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", in.Command)
	cmd.Dir = workDir

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set environment
	cmd.Env = append(os.Environ(),
		"SESSION_DIR="+t.sessionDir,
	)

	// Run command
	err := cmd.Run()

	// Build result
	var result strings.Builder

	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}

	if stderr.Len() > 0 {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString("[stderr]\n")
		result.WriteString(stderr.String())
	}

	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			result.WriteString(fmt.Sprintf("\n[error] Command timed out after %v", timeout))
		} else {
			result.WriteString(fmt.Sprintf("\n[error] %v", err))
		}
	}

	// Truncate if too long
	output := result.String()
	if len(output) > 50000 {
		output = output[:50000] + "\n... (output truncated)"
	}

	if output == "" {
		output = "(no output)"
	}

	return output, nil
}

// SetSessionDir updates the session directory (called when session changes)
func (t *BashExecTool) SetSessionDir(dir string) {
	t.sessionDir = dir
}

// GetSessionDir returns the current session directory
func (t *BashExecTool) GetSessionDir() string {
	return t.sessionDir
}
