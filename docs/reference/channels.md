# Channel Reference

Complete reference for all supported messaging channels.

## Supported Channels

| Channel | Protocol | Status | Dependencies |
|---|---|---|---|
| TUI | Local | Stable | None |
| Telegram | Bot API | Stable | Bot Token |
| Discord | Gateway API | Stable | Bot Token |
| Slack | Socket Mode | Stable | App + Bot Token |
| WhatsApp | whatsmeow | Stable | Phone pairing |
| WebSocket | HTTP/WS | Stable | None |

### Optional Channels (contrib/)

These channels are available in `contrib/channels/` but are not loaded by default:

| Channel | Location | Notes |
|---|---|---|
| Feishu/Lark | `contrib/channels/feishu/` | Chinese market |
| QQ | `contrib/channels/qq/` | Chinese market |
| WeCom | `contrib/channels/wecom/` | Chinese market |
| DingTalk | `contrib/channels/dingtalk/` | Chinese market |

## Configuration

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"

  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"

  slack:
    enabled: false
    app_token: "xapp-..."
    bot_token: "xoxb-..."

  whatsapp:
    enabled: false

  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 9090
```

## Telegram

### Setup

1. Create bot via [@BotFather](https://t.me/BotFather)
2. Get bot token
3. Configure:

```yaml
channels:
  telegram:
    enabled: true
    bot_token: "${TELEGRAM_BOT_TOKEN}"
```

## Discord

### Setup

1. Go to [Discord Developer Portal](https://discord.com/developers/applications)
2. Create application → Bot → Get token
3. Enable Message Content Intent
4. Invite bot to server

```yaml
channels:
  discord:
    enabled: true
    bot_token: "${DISCORD_BOT_TOKEN}"
```

### Required Intents

- MESSAGE CONTENT INTENT
- GUILD MESSAGES

## Slack

### Setup

1. Create Slack app at [api.slack.com](https://api.slack.com)
2. Enable Socket Mode
3. Get App Token (`xapp-...`) and Bot Token (`xoxb-...`)

```yaml
channels:
  slack:
    enabled: true
    app_token: "xapp-..."
    bot_token: "xoxb-..."
```

## WhatsApp

### Setup

Uses whatsmeow library — requires phone pairing:

```yaml
channels:
  whatsapp:
    enabled: true
```

First run will generate a QR code to scan with WhatsApp.

## WebSocket

Real-time bidirectional communication for custom clients:

```yaml
channels:
  websocket:
    enabled: true
    host: "0.0.0.0"
    port: 9090
```

### Client Connection

```javascript
const ws = new WebSocket('ws://localhost:9090');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'user_message',
    payload: { content: 'Hello!' }
  }));
};

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(msg.type, msg.payload);
};
```

### Message Types

| Type | Direction | Description |
|---|---|---|
| `user_message` | Client → Server | User input |
| `branch_switch` | Client → Server | Switch conversation branch |
| `list_branches` | Client → Server | List all branches for session |
| `chat_token` | Server → Client | Streaming token |
| `chat_complete` | Server → Client | Response finished |
| `tool_execution_start` | Server → Client | Tool starting |
| `tool_execution_result` | Server → Client | Tool result |
| `system_log` | Server → Client | System message |
| `error` | Server → Client | Error message |

## Gateway Mode

Start all enabled channels:

```bash
wunderpus gateway
```

## File Sending

The ChannelAggregator provides unified file sending across channels:

```go
aggregator.SendFile(sessionID, filePath, caption)
```
