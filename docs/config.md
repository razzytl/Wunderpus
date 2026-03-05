# 🐙 Wunderpus Configuration Guide

Wunderpus uses a YAML-based configuration system. By default, it looks for `config.yaml` in the current directory.

## Provider Configuration

### OpenAI
```yaml
providers:
  openai:
    api_key: "sk-..."
    model: "gpt-4o"
    max_tokens: 4000
```

### Anthropic
```yaml
providers:
  anthropic:
    api_key: "sk-ant-..."
    model: "claude-3-opus-20240229"
```

## Security Settings

### Encryption
Sensitive API keys can be encrypted at rest. Set `security.encryption.enabled` to true and provide a base64 encoded 32-byte key.

```yaml
security:
  encryption:
    enabled: true
    key: "base64-encoded-key-here"
```

## Advanced Settings
- `agent.system_prompt`: Global instruction tool for the agent.
- `agent.max_context_tokens`: Maximum history to maintain.
