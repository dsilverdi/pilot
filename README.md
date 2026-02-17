<div align="center">
  <img src="assets/banner.png" alt="Pilot" width="1024">

  <h1>Pilot: Lightweight Personal AI Assistant in Go</h1>

  <h3>Tool Calling · Session Persistence · Skills System</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/Channels-CLI%20%7C%20HTTP%20%7C%20Telegram-blue" alt="Channels">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
  </p>
</div>

Lightweight agentic runtime built with Go and the Anthropic SDK. Run your assistant from terminal, HTTP API, or Telegram with persistent sessions and tool integrations.

## Highlights

- Interactive CLI with streaming responses
- HTTP gateway for always-on agent access
- Optional Telegram channel on top of the gateway
- Tooling for file ops, shell commands, web search, and web fetch
- Session persistence and skill-based extensibility
- API key management for secured gateway access

## Quick Start

```bash
# 1) Build binaries
make build

# 2) Set one auth method
export ANTHROPIC_API_KEY="your-api-key"
# or
export ANTHROPIC_OAUTH_TOKEN="your-oauth-token"

# 3) Run CLI
./bin/pilot
```

Run the gateway instead:

```bash
./bin/pilot-gateway
```

## Install

System install (requires sudo):

```bash
make install
```

User-local install (no sudo):

```bash
make install-user
# If needed: export PATH="$HOME/bin:$PATH"
```

## Components

### CLI (`pilot`)

```bash
# Interactive mode
pilot

# Single prompt mode
pilot -p "What is Go?"

# API key management
pilot api-key generate --name my-app
pilot api-key list
pilot api-key revoke --name my-app
```

### Gateway (`pilot-gateway`)

```bash
# Default address
./bin/pilot-gateway

# Custom address
./bin/pilot-gateway --addr :3000
```

API endpoints:

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `GET` | `/health` | Health check |
| `POST` | `/chat` | Non-streaming chat response |
| `POST` | `/chat/stream` | Streaming chat response (SSE) |
| `DELETE` | `/session/{id}` | Delete a session |

See `docs/gateway.md` for full request/response details.

### Telegram Channel

Telegram support is embedded in `pilot-gateway`.

```bash
export TELEGRAM_BOT_TOKEN="123456:ABC-xyz..."
./bin/pilot-gateway
```

See `docs/telegram.md` for setup and access control.

## Configuration

```bash
# Authentication (set one)
ANTHROPIC_API_KEY=sk-ant-api-...
ANTHROPIC_OAUTH_TOKEN=sk-ant-oat-...

# Web search (optional)
SEARXNG_URL=http://localhost:8081

# Gateway (optional)
GATEWAY_ADDR=:8080

# Telegram (optional)
TELEGRAM_BOT_TOKEN=123456:ABC-xyz...
TELEGRAM_ALLOWED_USERS=123456,789012
```

## CLI Commands

| Command | Description |
| --- | --- |
| `/help` | Show available commands |
| `/session new [name]` | Create a session |
| `/session list` | List sessions |
| `/session switch <id>` | Switch active session |
| `/session delete <id>` | Delete a session |
| `/clear` | Clear active conversation |
| `/skill list` | List loaded skills |
| `/tool list` | List available tools |
| `/exit` | Exit CLI |

## Available Tools

| Tool | Description |
| --- | --- |
| `file_read` | Read file contents |
| `file_write` | Write file contents |
| `bash_exec` | Execute shell commands |
| `web_search` | Search the web via SearXNG |
| `web_fetch` | Fetch and extract web content |
| `invoke_skill` | Run specialized skills |

## Skills

Skills follow the [Agent Skills spec](https://agentskills.io) and can be loaded from `~/.pilot/skills/` or local `skills/`.

```text
skills/
└── my-skill/
    └── SKILL.md
```

Minimal example:

```markdown
---
name: my-skill
description: What this skill does and when to use it.
---

# My Skill
Instructions for the skill...
```

## Data Directory

```text
~/.pilot/
├── sessions/       # Persisted chat sessions
├── skills/         # User-installed skills
└── api-keys.json   # Hashed API keys for gateway auth
```

## Project Layout

```text
pilot/
├── cmd/
│   ├── pilot/
│   └── pilot-gateway/
├── internal/
│   ├── agent/
│   ├── channel/telegram/
│   ├── cli/
│   ├── gateway/
│   ├── session/
│   ├── skills/
│   ├── browser/
│   └── tools/
├── docs/
└── skills/
```

## Development

```bash
# Build
make build

# Test
make test
make test-cover

# Run
make run
make run-gateway

# Discover more targets
make help
```

## Documentation

- Gateway setup and API usage: `docs/gateway.md`
- Telegram setup: `docs/telegram.md`

## License

MIT. See `LICENSE`.
