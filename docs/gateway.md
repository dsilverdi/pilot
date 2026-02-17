# Pilot Gateway Setup

The pilot-gateway is an HTTP service that exposes the pilot agent via REST API. It allows external applications to interact with the agent 24/7.

## Prerequisites

- Go 1.21 or later
- Anthropic API key or OAuth token
- (Optional) SearXNG for web search
- (Optional) Telegram bot token

## Quick Start

```bash
# Build
make build

# Set authentication
export ANTHROPIC_API_KEY="your-api-key"

# Start gateway
./bin/pilot-gateway
```

The gateway will start on `http://localhost:8080`.

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | Yes* | - | Anthropic API key |
| `ANTHROPIC_OAUTH_TOKEN` | Yes* | - | OAuth token (higher priority) |
| `PILOT_HOME` | No | `~/.pilot` | Data directory (sessions, keys) |
| `GATEWAY_ADDR` | No | `:8080` | Listen address |
| `SEARXNG_URL` | No | `http://localhost:8081` | SearXNG URL for web search |
| `TELEGRAM_BOT_TOKEN` | No | - | Enables Telegram bot |
| `TELEGRAM_ALLOWED_USERS` | No | - | Comma-separated user IDs |

*One of `ANTHROPIC_API_KEY` or `ANTHROPIC_OAUTH_TOKEN` is required.

**Note:** Set `PILOT_HOME` when running as a systemd service to use a shared data directory (e.g., `/var/lib/pilot`).

### Command Line Options

```bash
./bin/pilot-gateway [options]

Options:
  --addr <address>    Listen address (default: :8080)
  --workdir <path>    Working directory for tools
  --version           Show version
  --help              Show help
```

## API Endpoints

### Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "version": "dev"
}
```

### Send Message

```bash
POST /chat
```

**Request:**
```json
{
  "session_id": "user_123",
  "message": "What is Go?"
}
```

- `session_id` (optional): Session identifier. Auto-generated if not provided.
- `message` (required): The user's message.

**Response:**
```json
{
  "session_id": "user_123",
  "response": "Go is a statically typed, compiled programming language...",
  "tool_calls": [
    {
      "name": "web_search",
      "input": "{\"query\": \"Go programming language\"}",
      "result": "..."
    }
  ]
}
```

### Streaming Response (SSE)

```bash
POST /chat/stream
```

Same request format as `/chat`. Returns Server-Sent Events:

```
event: text
data: {"content": "Go is "}

event: text
data: {"content": "a programming language"}

event: tool_call
data: {"name": "web_search", "input": "..."}

event: tool_result
data: {"name": "web_search", "result": "..."}

event: done
data: {"session_id": "user_123"}
```

### Delete Session

```bash
DELETE /session/{id}
```

**Response:**
```json
{
  "status": "deleted",
  "session_id": "user_123"
}
```

## Authentication

The gateway uses API keys for authentication. Keys are managed via the `pilot` CLI.

### Generate API Key

```bash
./bin/pilot api-key generate --name my-app
```

Output:
```
Generated API key:
  psk_abc123xyz...

Name: my-app

Store this key securely - it won't be shown again.
```

### List API Keys

```bash
./bin/pilot api-key list
```

### Revoke API Key

```bash
./bin/pilot api-key revoke --name my-app
```

### Using API Keys

Include the key in requests:

```bash
# Via X-API-Key header
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: psk_abc123xyz..." \
  -d '{"message": "Hello"}'

# Via Authorization header
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer psk_abc123xyz..." \
  -d '{"message": "Hello"}'
```

### Open Mode

If no API keys are configured, the gateway runs in "open mode" and accepts all requests. A warning is logged:

```
WARNING: No API keys configured - server is open to all requests
```

## Usage Examples

### Simple Chat

```bash
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: psk_..." \
  -d '{"message": "What is the capital of France?"}'
```

### With Session

```bash
# First message
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: psk_..." \
  -d '{"session_id": "user_123", "message": "My name is Alice"}'

# Follow-up (same session)
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -H "X-API-Key: psk_..." \
  -d '{"session_id": "user_123", "message": "What is my name?"}'
```

### Streaming

```bash
curl -X POST http://localhost:8080/chat/stream \
  -H "Content-Type: application/json" \
  -H "X-API-Key: psk_..." \
  -d '{"message": "Explain quantum computing"}'
```

### Python Client Example

```python
import requests

API_KEY = "psk_abc123xyz..."
GATEWAY_URL = "http://localhost:8080"

def chat(message, session_id=None):
    response = requests.post(
        f"{GATEWAY_URL}/chat",
        headers={"X-API-Key": API_KEY},
        json={
            "session_id": session_id,
            "message": message
        }
    )
    return response.json()

# Usage
result = chat("What is Python?")
print(result["response"])
```

## Running with Docker

### Using docker-compose

Create a `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pilot-gateway:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - GATEWAY_ADDR=:8080
    volumes:
      - pilot-data:/root/.pilot

volumes:
  pilot-data:
```

### With SearXNG

```yaml
version: '3.8'

services:
  pilot-gateway:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - SEARXNG_URL=http://searxng:8080
    depends_on:
      - searxng

  searxng:
    image: searxng/searxng:latest
    ports:
      - "8081:8080"
    volumes:
      - ./searxng:/etc/searxng
```

## Running as a Service

### Automated Setup (Recommended)

Use the provided setup script:

```bash
# Build first
make build

# Run setup (requires sudo)
make setup-systemd
```

This will:
- Create a `pilot` system user
- Install binaries to `/opt/pilot`
- Create data directory at `/var/lib/pilot`
- Configure systemd service
- Generate initial API key

After setup, edit the config:
```bash
sudo nano /opt/pilot/.env
# Add your ANTHROPIC_API_KEY
sudo systemctl restart pilot-gateway
```

### Manual systemd Setup (Linux)

Create `/etc/systemd/system/pilot-gateway.service`:

```ini
[Unit]
Description=Pilot Gateway
After=network.target

[Service]
Type=simple
User=pilot
WorkingDirectory=/opt/pilot
ExecStart=/opt/pilot/bin/pilot-gateway
Restart=always
RestartSec=5
EnvironmentFile=/opt/pilot/.env

[Install]
WantedBy=multi-user.target
```

Create `/opt/pilot/.env`:
```bash
ANTHROPIC_API_KEY=your-key
PILOT_HOME=/var/lib/pilot
GATEWAY_ADDR=:8080
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable pilot-gateway
sudo systemctl start pilot-gateway
```

### Managing API Keys with systemd

When running as a systemd service, use the service user and `PILOT_HOME`:

```bash
# Generate API key
sudo -u pilot PILOT_HOME=/var/lib/pilot /opt/pilot/bin/pilot api-key generate --name myapp

# List keys
sudo -u pilot PILOT_HOME=/var/lib/pilot /opt/pilot/bin/pilot api-key list
```

### launchd (macOS)

Create `~/Library/LaunchAgents/com.pilot.gateway.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.pilot.gateway</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/pilot-gateway</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>ANTHROPIC_API_KEY</key>
        <string>your-key</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load:

```bash
launchctl load ~/Library/LaunchAgents/com.pilot.gateway.plist
```

## Troubleshooting

### 401 Unauthorized

- Check that you're including the API key in requests
- Verify the key is valid: `./bin/pilot api-key list`
- Generate a new key if needed

### 500 Agent Error

- Check Anthropic API key is set correctly
- Verify network connectivity to api.anthropic.com
- Check gateway logs for details

### Connection Refused

- Verify gateway is running: `curl http://localhost:8080/health`
- Check the port isn't in use: `lsof -i :8080`
- Try a different port: `--addr :3000`

## Security Recommendations

1. **Always use API keys** in production
2. **Use HTTPS** with a reverse proxy (nginx, Caddy)
3. **Restrict network access** via firewall
4. **Rotate keys** periodically
5. **Monitor logs** for unauthorized access attempts
