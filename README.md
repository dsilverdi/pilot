# Pilot

A lightweight agentic CLI system built with Go and the Anthropic SDK. Run AI agents via CLI, HTTP API, or Telegram bot.

## Features

- **Interactive CLI** - Conversational interface with streaming responses
- **HTTP Gateway** - REST API for 24/7 agent access (`pilot-gateway`)
- **Telegram Bot** - Chat with your agent via Telegram
- **File Operations** - Read and write files via tool calls
- **Web Search** - Search the web using SearXNG (self-hosted)
- **Web Fetch** - Fetch and extract content from web pages
- **Bash Execution** - Run shell commands
- **Session Management** - Persist and switch between conversation sessions
- **Skills System** - Extensible markdown-based skills (Agent Skills spec)
- **API Key Management** - Secure API key generation for gateway access

## Quick Start

```bash
# Build
make build

# Set authentication
export ANTHROPIC_API_KEY="your-api-key"
# OR use OAuth token
export ANTHROPIC_OAUTH_TOKEN="your-oauth-token"

# Run CLI
./bin/pilot

# Or run gateway (HTTP + optional Telegram)
./bin/pilot-gateway
```

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
# Binaries are created in bin/
```

## Components

### 1. CLI (`pilot`)

Interactive command-line interface for chatting with the agent.

```bash
# Interactive mode
pilot

# Execute a prompt directly
pilot -p "What is Go?"

# Manage API keys for gateway
pilot api-key generate --name my-app
pilot api-key list
pilot api-key revoke --name my-app
```

### 2. Gateway (`pilot-gateway`)

HTTP service that exposes the agent via REST API. Optionally includes Telegram bot.

```bash
# Start gateway
./bin/pilot-gateway

# With custom address
./bin/pilot-gateway --addr :3000
```

**API Endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/chat` | Send message, get response |
| POST | `/chat/stream` | Streaming response (SSE) |
| DELETE | `/session/{id}` | Delete a session |

See [Gateway Documentation](docs/gateway.md) for details.

### 3. Telegram Bot

Built into `pilot-gateway`. Enable by setting `TELEGRAM_BOT_TOKEN`.

```bash
export TELEGRAM_BOT_TOKEN="123456:ABC-xyz..."
./bin/pilot-gateway
```

See [Telegram Setup Guide](docs/telegram.md) for details.

## Environment Variables

```bash
# Authentication (one required)
ANTHROPIC_API_KEY=sk-ant-api-...
ANTHROPIC_OAUTH_TOKEN=sk-ant-oat-...

# Web Search (optional)
SEARXNG_URL=http://localhost:8081

# Gateway (optional)
GATEWAY_ADDR=:8080

# Telegram (optional)
TELEGRAM_BOT_TOKEN=123456:ABC-xyz...
TELEGRAM_ALLOWED_USERS=123456,789012
```

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

## API Key Subcommand

```bash
# Generate a new API key for gateway
pilot api-key generate --name telegram-bot

# List all keys
pilot api-key list

# Revoke a key
pilot api-key revoke --name telegram-bot
```

## Available Tools

| Tool | Description |
|------|-------------|
| `file_read` | Read file contents |
| `file_write` | Write content to files |
| `bash_exec` | Execute shell commands |
| `web_search` | Search the web (via SearXNG) |
| `web_fetch` | Fetch and extract web page content |
| `invoke_skill` | Invoke specialized skills |

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

## Data Directory

Pilot stores data in `~/.pilot/`:

```
~/.pilot/
├── sessions/       # Persisted chat sessions
├── skills/         # User-installed skills
└── api-keys.json   # API keys for gateway (hashed)
```

## Project Structure

```
pilot/
├── cmd/
│   ├── pilot/           # CLI entry point
│   └── pilot-gateway/   # Gateway entry point
├── internal/
│   ├── agent/           # Core agent with streaming loop
│   ├── channel/
│   │   └── telegram/    # Telegram bot adapter
│   ├── cli/             # Interactive REPL
│   ├── gateway/         # HTTP server and handlers
│   │   └── apikey/      # API key management
│   ├── session/         # Session management
│   ├── skills/          # Skill parser and loader
│   ├── browser/         # Headless browser for web fetch
│   └── tools/           # Tool implementations
├── docs/                # Documentation
└── skills/              # Local skills directory
```

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-cover

# Build all binaries
make build

# Run gateway with .env
make run-gateway

# Show all targets
make help
```

## Documentation

- [Gateway Setup](docs/gateway.md) - HTTP API setup and usage
- [Telegram Setup](docs/telegram.md) - Telegram bot configuration

## License

MIT
