# Pilot

A lightweight agentic CLI system built with Go and the Anthropic SDK.

## Features

- **Interactive CLI** - Conversational interface with streaming responses
- **File Operations** - Read and write files via tool calls
- **Web Search** - Search the web using Brave Search API
- **Session Management** - Persist and switch between conversation sessions
- **Skills System** - Extensible markdown-based skills (Agent Skills spec)

## Installation

### Quick Install (requires sudo)

```bash
make install
```

### User Install (no sudo)

```bash
make install-user
# Then add ~/bin to PATH if not already:
# export PATH="$HOME/bin:$PATH"
```

### Build Only

```bash
make build
```

## Usage

```bash
# Set Anthropic authentication (one of the following)
export ANTHROPIC_API_KEY="your-anthropic-api-key"
# OR use OAuth token (takes priority over API key)
export ANTHROPIC_OAUTH_TOKEN="your-oauth-token"

# Optional: Enable web search
export BRAVE_API_KEY="your-brave-api-key"

# Start interactive mode
pilot

# Or execute a prompt directly
pilot -p "What is the capital of France?"
pilot -p "Read the file main.go and explain it"
```

### Command Line Options

| Flag | Description |
|------|-------------|
| `-p <prompt>` | Execute a prompt directly without entering interactive mode |
| `-v, --version` | Show version |
| `-h, --help` | Show help |

## Authentication

Pilot supports multiple authentication methods with the following priority:

1. **`ANTHROPIC_OAUTH_TOKEN`** - OAuth token (highest priority)
2. **`ANTHROPIC_AUTH_TOKEN`** - Auth token
3. **`ANTHROPIC_API_KEY`** - API key (lowest priority)

OAuth tokens are useful when authenticating via Claude's OAuth flow (tokens prefixed with `sk-ant-oat-`).

## CLI Commands

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/session new [name]` | Create a new session |
| `/session list` | List all sessions |
| `/session switch <id>` | Switch to a session |
| `/session delete <id>` | Delete a session |
| `/clear` | Clear conversation history |
| `/skill list` | List available skills |
| `/tool list` | List available tools |
| `/exit` | Exit the CLI |

## Available Tools

- **file_read** - Read file contents
- **file_write** - Write content to files
- **web_search** - Search the web (requires `BRAVE_API_KEY`)

## Skills

Skills are markdown files following the [Agent Skills spec](https://agentskills.io). Place them in `~/.pilot/skills/` or the local `skills/` directory.

```
skills/
└── my-skill/
    └── SKILL.md
```

Example SKILL.md:
```markdown
---
name: my-skill
description: What this skill does and when to use it.
---

# My Skill

Instructions for the skill...
```

## Project Structure

```
pilot/
├── cmd/pilot/          # Entry point
├── internal/
│   ├── agent/          # Core agent with streaming loop
│   ├── cli/            # Interactive REPL
│   ├── session/        # Session management
│   ├── skills/         # Skill parser and loader
│   └── tools/          # Tool interface and implementations
└── skills/             # Local skills directory
```

## Development

```bash
# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Build
go build -o pilot ./cmd/pilot
```

## License

MIT
