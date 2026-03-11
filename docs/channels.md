# Channel Configuration

Wunderpus supports multiple communication channels, enabling integration with popular messaging platforms. This guide covers configuration for each supported channel.

## Channel Overview

| Channel | Protocol | Status | Dependencies |
|---------|----------|--------|--------------|
| TUI | Local | Stable | None |
| Telegram | Bot API | Stable | Bot Token |
| Discord | Gateway API | Stable | Bot Token |
| WebSocket | HTTP/WS | Stable | None |
| QQ | HTTP API | Stable | None |
| WeCom | Webhook | Stable | Corp credentials |
| DingTalk | Webhook | Stable | App credentials |
| OneBot | WebSocket | Stable | OneBot server |

## Configuration Structure

Channels are configured in `config.yaml`:

```yaml
# Enable/disable channels
channels:
  telegram:
    enabled: true
    # channel-specific settings
    
  discord:
    enabled: true
    # channel-specific settings
```

## Terminal User Interface (TUI)

The TUI is the default interface when running `wunderpus` without arguments.

### Features

- Interactive command input with history
- Command palette (`Ctrl+P`)
- Provider switching (`Tab`)
- Markdown rendering
- Syntax highlighting for code

### Configuration

```yaml
tui:
  # TUI-specific settings (future expansion)
  theme: "default"  # terminal, dark, light
```

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Send message |
| `Ctrl+C` | Exit |
| `Tab` | Cycle providers |
| `Ctrl+P` | Command palette |
| `Ctrl+L` | Clear screen |

## Telegram

### Prerequisites

1. Create a bot via [@BotFather](https://t.me/BotFather)
2. Get the bot token
3. Optionally set bot commands

### Configuration

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
    
    # Optional settings
    # parse_mode: "MarkdownV2"  # Markdown, MarkdownV2, or HTML
    # allowed_users: []         # Restrict to specific user IDs
    # allowed_chats: []        # Restrict to specific chat IDs
```

### Environment Variable

```bash
export TELEGRAM_BOT_TOKEN="your-token"
```

### Bot Commands

Set up commands via BotFather:
```
help - Show help information
status - Show Wunderpus status
clear - Clear conversation history
```

### Webhook vs Polling

By default, Telegram uses webhooks. For development, you can use long polling:

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    use_polling: true  # For development
```

## Discord

### Prerequisites

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create a new application
3. Create a bot and get the token
4. Enable required intents (Message Content)
5. Invite the bot to your server

### Configuration

```yaml
channels:
  discord:
    enabled: true
    bot_token: "your-discord-bot-token"
    
    # Optional settings
    # guild_id: "123456789"     # For guild-specific commands
    # channel_id: "123456789"   # Default channel
    # prefix: "!"               # Command prefix (for text commands)
```

### Environment Variable

```bash
export DISCORD_BOT_TOKEN="your-token"
```

### Slash Commands

Discord uses slash commands. Common commands:
- `/chat [message]` - Send a message to the agent
- `/clear` - Clear conversation history
- `/status` - Show bot status

### Required Intents

Ensure these Gateway Intents are enabled in Discord Developer Portal:
- `MESSAGE CONTENT INTENT` - Required to read messages
- `GUILD MESSAGES` - For server messages

## WebSocket

WebSocket provides real-time bidirectional communication for custom clients.

### Configuration

```yaml
channels:
  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 8081
    
    # Optional settings
    # path: "/ws"              # WebSocket path
    # auth_token: ""           # Token for authentication
    # max_connections: 100    # Max concurrent connections
```

### Client Connection

```javascript
// JavaScript example
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onopen = () => {
  console.log('Connected to Wunderpus');
  ws.send(JSON.stringify({
    type: 'message',
    content: 'Hello!'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Response:', data.content);
};
```

### Protocol

Messages are JSON formatted:

```json
// Send message
{
  "type": "message",
  "content": "Your message here"
}

// Receive response
{
  "type": "response",
  "content": "Agent response",
  "session_id": "abc123"
}

// Events
{
  "type": "event",
  "event": "message",
  "data": { ... }
}
```

## QQ (via NoneBot)

### Prerequisites

1. Install NoneBot2
2. Configure the QQ adapter
3. Get the account credentials

### Configuration

```yaml
channels:
  qq:
    enabled: true
    account: 123456789
    
    # NoneBot2 adapter settings
    # adapter: "nonebot2"
    # host: "127.0.0.1"
    # port: 8082
```

## WeChat Work (WeCom)

### Prerequisites

1. Log in to [WeCom Admin](https://work.weixin.qq.com/)
2. Create a custom application
3. Get Corp ID, Agent ID, and Secret

### Configuration

```yaml
channels:
  wecom:
    enabled: true
    corp_id: "your-corp-id"
    agent_id: "1000001"
    secret: "your-agent-secret"
    
    # Optional
    # token: "verification-token"  # For callback verification
    # encoding_aes_key: ""         # For encrypted messages
```

### Environment Variables

```bash
export WECOM_CORP_ID="your-corp-id"
export WECOM_AGENT_ID="1000001"
export WECOM_SECRET="your-secret"
```

## DingTalk

### Prerequisites

1. Create an application in DingTalk Open Platform
2. Get App ID and App Secret
3. Configure callback URL

### Configuration

```yaml
channels:
  dingtalk:
    enabled: true
    app_id: "your-app-id"
    app_secret: "your-app-secret"
    
    # Optional
    # token: "verification-token"
    # encoding_aes_key: ""
```

## OneBot

OneBot is a standardized bot protocol used by various chat platforms.

### Configuration

```yaml
channels:
  onebot:
    enabled: true
    
    # OneBot v11 settings
    # protocol: "http"  # http, websocket
    # host: "0.0.0.0"
    # port: 8082
    
    # For HTTP callback
    # callback_url: "http://your-server/onebot/callback"
```

## Channel Management

### Starting Channels

```bash
# Start gateway with all enabled channels
wunderpus gateway

# With verbose logging
wunderpus gateway -v
```

### Checking Channel Status

```bash
# Check status
wunderpus status
```

Output:
```
Channel Status:
- telegram: Connected
- discord: Connected
- websocket: Listening on :8081
```

### Channel-Specific Commands

Each channel may support special commands:

```yaml
# Global commands (all channels)
commands:
  help: "Show available commands"
  clear: "Clear conversation history"
  status: "Show system status"
```

## Advanced Configuration

### Message Queue

For high-traffic deployments, configure message queuing:

```yaml
channels:
  telegram:
    enabled: true
    queue_size: 100
    
    # Message processing
    # concurrent_workers: 5
    # retry_attempts: 3
```

### Rate Limiting

Per-channel rate limiting:

```yaml
channels:
  telegram:
    enabled: true
    rate_limit:
      messages_per_minute: 20
      burst: 10
```

### Message Formatting

Configure how messages are formatted per channel:

```yaml
channels:
  telegram:
    enabled: true
    parse_mode: "MarkdownV2"
    # escape_html: true
    
  discord:
    enabled: true
    embed: true
    # use_slash_commands: true
```

## Security Considerations

### Token Storage

Store tokens in environment variables:

```yaml
channels:
  telegram:
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    
  discord:
    bot_token: "${DISCORD_BOT_TOKEN}"
```

### Access Control

Restrict access to specific users or channels:

```yaml
channels:
  telegram:
    enabled: true
    allowed_users:
      - 123456789
      - 987654321
    allowed_chats:
      - -100123456789  # Supergroup ID
```

### Webhook Security

For webhook-based channels, verify signatures:

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
    secret_token: "your-secret-token"  # For webhook verification
```

## Troubleshooting

### Channel Fails to Start

Check logs:
```bash
wunderpus gateway -v 2>&1 | grep -i telegram
```

Common issues:
- Invalid bot token
- Network connectivity
- Insufficient bot permissions

### Messages Not Delivered

1. Check channel is connected: `wunderpus status`
2. Verify rate limits not exceeded
3. Check bot has necessary permissions

### Webhook Issues

For Telegram/Discord webhooks:
- Verify public URL is reachable
- Check SSL certificate is valid
- Ensure callback URL is correctly configured
