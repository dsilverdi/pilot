# Telegram Bot Setup

The pilot-gateway includes an optional Telegram bot that allows you to chat with the AI agent via Telegram.

## Prerequisites

- pilot-gateway built and configured
- Telegram account
- Anthropic API key or OAuth token

## Step 1: Create a Telegram Bot

1. Open Telegram and search for **@BotFather**
2. Start a chat and send `/newbot`
3. Follow the prompts:
   - Enter a name for your bot (e.g., "My Pilot Bot")
   - Enter a username (must end with "bot", e.g., "my_pilot_bot")
4. BotFather will give you a token like:
   ```
   123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```
5. Save this token securely

## Step 2: Configure the Gateway

Add the bot token to your environment:

```bash
# Add to .env file
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz

# Or export directly
export TELEGRAM_BOT_TOKEN="123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
```

## Step 3: Start the Gateway

```bash
# Using make (loads .env automatically)
make run-gateway

# Or directly
source .env && ./bin/pilot-gateway
```

You should see:
```
2024/01/15 10:30:00 Starting pilot-gateway on :8080
2024/01/15 10:30:00 Telegram bot enabled: @my_pilot_bot
```

## Step 4: Start Chatting

1. Open Telegram
2. Search for your bot by username (e.g., @my_pilot_bot)
3. Click "Start" or send `/start`
4. Send any message to chat with the AI

## Bot Commands

| Command | Description |
|---------|-------------|
| `/start` | Welcome message and introduction |
| `/help` | Show available commands |
| `/clear` | Clear your conversation history |
| `/status` | Show bot status and info |

## Configuration Options

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `TELEGRAM_ALLOWED_USERS` | No | Restrict to specific users |

### Restricting Access

To limit who can use the bot, set `TELEGRAM_ALLOWED_USERS` with comma-separated user IDs:

```bash
TELEGRAM_ALLOWED_USERS=123456789,987654321
```

**Finding your user ID:**
1. Search for @userinfobot on Telegram
2. Start a chat with it
3. It will show your user ID

## How It Works

### Session Management

- Each Telegram user gets a unique session: `tg_{user_id}`
- Sessions persist across messages (conversation memory)
- Use `/clear` to start a fresh conversation
- Sessions are stored in `~/.pilot/sessions/`

### Message Flow

```
User sends message
       ↓
Bot receives via long-polling
       ↓
Bot gets/creates session for user
       ↓
Agent processes message (may use tools)
       ↓
Bot sends response to user
```

### Tool Usage

When the agent uses tools (web search, file operations, etc.), the bot shows:
```
🔧 Using web_search...
```

Then sends the final response.

## Examples

### Basic Conversation

```
User: What is the capital of France?
Bot: The capital of France is Paris...

User: Tell me more about it
Bot: Paris is the capital and largest city of France...
```

### Using Tools

```
User: Search for the latest Go release
Bot: 🔧 Using web_search...
Bot: The latest Go release is Go 1.22, released in February 2024...
```

### Clearing History

```
User: /clear
Bot: 🗑️ Conversation history cleared. Send a message to start fresh!
```

## Customizing the Bot

### Bot Profile

Use @BotFather to customize:
- `/setname` - Change display name
- `/setdescription` - Set bot description
- `/setabouttext` - Set "About" text
- `/setuserpic` - Set profile picture
- `/setcommands` - Set command menu

### Suggested Commands

Send to @BotFather:
```
/setcommands
```

Then send:
```
start - Welcome message
help - Show commands
clear - Clear conversation
status - Bot status
```

## Running 24/7

### Using systemd (Linux)

Create `/etc/systemd/system/pilot-gateway.service`:

```ini
[Unit]
Description=Pilot Gateway with Telegram
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

Enable and start:

```bash
sudo systemctl enable pilot-gateway
sudo systemctl start pilot-gateway
sudo systemctl status pilot-gateway
```

### Using screen/tmux

```bash
# Start in screen
screen -S pilot
make run-gateway
# Detach: Ctrl+A, D

# Reattach later
screen -r pilot
```

### Using Docker

```yaml
version: '3.8'

services:
  pilot-gateway:
    build: .
    environment:
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_ALLOWED_USERS=${TELEGRAM_ALLOWED_USERS}
    volumes:
      - pilot-data:/root/.pilot
    restart: unless-stopped

volumes:
  pilot-data:
```

## Troubleshooting

### Bot not responding

1. Check gateway is running:
   ```bash
   curl http://localhost:8080/health
   ```

2. Verify bot token:
   ```bash
   curl "https://api.telegram.org/bot<YOUR_TOKEN>/getMe"
   ```

3. Check logs for errors

### "You are not authorized"

- Your user ID is not in `TELEGRAM_ALLOWED_USERS`
- Find your user ID and add it to the list
- Restart the gateway

### Slow responses

- Agent is processing (normal for complex queries)
- Check Anthropic API status
- Tool execution (web search) may take time

### Messages not sending

- Check Telegram API limits (30 messages/second)
- Message may be too long (4096 char limit)
- Bot will automatically split long messages

## Security Best Practices

1. **Use TELEGRAM_ALLOWED_USERS** in production
2. **Keep bot token secret** - don't commit to git
3. **Monitor usage** - check logs regularly
4. **Rotate tokens** if compromised via @BotFather `/revoke`
5. **Use separate bot** for testing vs production

## Multi-User Support

The bot naturally supports multiple users:

- Each user has isolated session (`tg_{user_id}`)
- Conversations don't interfere with each other
- All users share the same agent configuration
- Use `TELEGRAM_ALLOWED_USERS` to control access

## Limitations

- **Text only**: Images and files not supported (yet)
- **No group chats**: Bot only works in private chats
- **No inline mode**: Can't use @bot_name in other chats
- **Long-polling only**: No webhook support (yet)

## Next Steps

- Set up [SearXNG](https://docs.searxng.org/) for web search
- Create custom [Skills](../README.md#skills) for specialized tasks
- Configure [Gateway API](gateway.md) for programmatic access
