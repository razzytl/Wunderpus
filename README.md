# 🐙 Wunderpus — Universal AI Agent

Autonomous, multi-channel AI agent built in Go.

## Features
- **Multi-Provider**: Support for OpenAI, Anthropic, Gemini, and Ollama.
- **Multi-Channel**: Interface via TUI, Telegram, Discord, or WebSocket.
- **Tool System**: Extensible tool execution environment.
- **Secure**: Built-in sanitization, encryption, and audit logging.

## Getting Started

1. **Clone the repository**:
   ```bash
   git clone https://github.com/wonderpus/wonderpus.git
   ```

2. **Configure**:
   Copy `config.example.yaml` to `config.yaml` and add your API keys.

3. **Run**:
   ```bash
   go run cmd/wonderpus/main.go
   ```

## TUI Commands
- `/help` - Show command list.
- `Tab` - Cycle through AI providers.
- `Ctrl+P` - Open Command Palette.
