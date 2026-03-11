---
name: social-media-poster
description: Use when posting updates, announcements, or content to social media platforms (Discord, Telegram, Slack).
---

# Social Media Poster

Post updates to Discord, Telegram, or Slack using the built-in channel system.

## Supported Platforms

| Platform | Config Key | Required |
|----------|-----------|----------|
| Discord | `channels.discord.enabled` + token | Token |
| Telegram | `channels.telegram.enabled` + token | Bot Token |
| Slack | channels.slack.enabled + token | Bot Token |

## Quick Reference

```bash
# Discord - send message to channel
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"content": "Your message here"}' \
  "https://discord.com/api/v10/channels/$CHANNEL_ID/messages"

# Telegram - send message
curl -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
  -d '{"chat_id": "$CHAT_ID", "text": "Your message here"}'

# Slack - post to channel
curl -X POST "$WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d '{"text": "Your message here"}'
```

## Using the Agent's Channel System

The wunderpus agent has built-in channel support. Use the gateway command to start channels:

```bash
# Start the gateway with channels enabled
wunderpus gateway
```

Then send messages through the agent's message processing:
```go
// In internal/channel implementations
// Discord: internal/channel/discord
// Telegram: internal/channel/telegram
// Slack: internal/channel/slack
```

## Discord

**Setup:** Get bot token from Discord Developer Portal, add to config.yaml:
```yaml
channels:
  discord:
    enabled: true
    token: "YOUR_BOT_TOKEN"
```

**Post via curl:**
```bash
# Get channel ID from Discord (enable Developer Mode -> right-click channel -> Copy ID)
CHANNEL_ID="123456789"
BOT_TOKEN="YOUR_TOKEN"
curl -X POST \
  -H "Authorization: Bot $BOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "Hello from wunderpus!"}' \
  "https://discord.com/api/v10/channels/$CHANNEL_ID/messages"
```

## Telegram

**Setup:** Get bot token from @BotFather, add to config.yaml:
```yaml
channels:
  telegram:
    enabled: true
    token: "YOUR_BOT_TOKEN"
```

**Post via curl:**
```bash
TOKEN="YOUR_BOT_TOKEN"
CHAT_ID="your_chat_id"
curl -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
  -d "chat_id=$CHAT_ID&text=Hello from wunderpus!"
```

**Get chat ID:** Start a chat with your bot, then:
```bash
curl "https://api.telegram.org/bot$TOKEN/getUpdates"
```

## Slack

**Setup:** Create Slack App with `chat:write` scope, install to workspace, get webhook URL:
```yaml
channels:
  slack:
    enabled: true
    token: "xoxb-..."
```

**Post via curl:**
```bash
WEBHOOK="https://hooks.slack.com/services/XXX/YYY/ZZZ"
curl -X POST "$WEBHOOK" \
  -H 'Content-Type: application/json' \
  -d '{"text": "Hello from wunderpus!"}'
```

## Rich Formatting

### Discord Embeds
```json
{
  "embeds": [{
    "title": "Title",
    "description": "Description",
    "color": 5763714,
    "fields": [
      {"name": "Field 1", "value": "Value", "inline": true}
    ]
  }]
}
```

### Slack Blocks
```json
{
  "blocks": [
    {
      "type": "section",
      "text": {"type": "mrkdwn", "text": "Hello *world*"}
    }
  ]
}
```

## Best Practices

1. **Never hardcode tokens** - Use environment variables or config
2. **Rate limits** - Respect platform limits (Discord: 5msg/5sec, Telegram: 30msg/sec)
3. **Escape special chars** - JSON requires proper escaping
4. **Test first** - Use dry-run or echo before posting
5. **Parse responses** - Check for errors in API responses

## Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `401 Unauthorized` | Invalid token | Regenerate token |
| `403 Forbidden` | Missing permissions | Add scopes to bot |
| `404 Not Found` | Wrong channel ID | Verify channel exists |
| `429 Too Many Requests` | Rate limited | Add delay between requests |
