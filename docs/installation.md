# Installation Guide

This guide covers all supported methods for installing Wunderpus, from pre-built binaries to building from source.

## Prerequisites

Before installing Wunderpus, ensure you have:

- **Go 1.25 or later** (for building from source)
- **Git** (for cloning and version control)
- **SQLite** (included in the binary)
- **Network access** to LLM provider APIs (depending on providers used)

## Installation Methods

### Method 1: Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/wunderpus/wunderpus/releases) page.

#### Linux

```bash
# Download
curl -sL https://github.com/wunderpus/wunderpus/releases/latest/download/wunderpus-linux-amd64.tar.gz | tar xz

# Install to PATH
sudo mv wunderpus /usr/local/bin/

# Verify
wunderpus --version
```

#### macOS

```bash
# Download
curl -sL https://github.com/wunderpus/wunderpus/releases/latest/download/wunderpus-darwin-amd64.tar.gz | tar xz

# Install to PATH
sudo mv wunderpus /usr/local/bin/

# Verify
wunderpus --version
```

#### Windows

```bash
# Download and extract using PowerShell
Invoke-WebRequest -Uri "https://github.com/wunderpus/wunderpus/releases/latest/download/wunderpus-windows-amd64.zip" -OutFile "wunderpus.zip"
Expand-Archive -Path "wunderpus.zip" -DestinationPath "."
```

Or use [Scoop](https://scoop.sh):
```powershell
scoop bucket add wunderpus https://github.com/wunderpus/scoop
scoop install wunderpus
```

### Method 2: Homebrew (macOS/Linux)

```bash
# Add the tap
brew tap wunderpus/wunderpus

# Install
brew install wunderpus

# Verify
wunderpus --version
```

To upgrade:
```bash
brew upgrade wunderpus
```

### Method 3: Docker

#### Basic Usage

```bash
# Pull the image
docker pull wunderpus/wunderpus:latest

# Run interactively
docker run -it wunderpus/wunderpus:latest

# Run with config file
docker run -it -v $(pwd)/config.yaml:/app/config.yaml wunderpus/wunderpus:latest
```

#### Production Docker Compose

Create a `docker-compose.yaml`:

```yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data
      - ./logs:/app/logs
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
    restart: unless-stopped
    ports:
      - "8080:8080"  # Health check port
```

Run:
```bash
docker-compose up -d
```

#### Building Your Own Image

```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o wunderpus ./cmd/wunderpus

FROM alpine:3.19
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/wunderpus .
COPY config.example.yaml .

EXPOSE 8080
CMD ["./wunderpus", "gateway"]
```

Build and run:
```bash
docker build -t my-wunderpus .
docker run -it -v $(pwd)/config.yaml:/app/config.yaml my-wunderpus
```

#### WhatsApp Support

For WhatsApp integration, use the build tag:

```bash
# Build with WhatsApp support
docker build -t wunderpus:whatsapp --build-tags=whatsapp .

# Run
docker run -it -v $(pwd)/config.yaml:/app/config.yaml wunderpus:whatsapp
```

### Method 4: From Source

#### Prerequisites

1. **Go 1.25 or later**:
   ```bash
   # Check version
   go version

   # Install (Linux)
   wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

2. **Git**: Install via package manager or [git-scm.com](https://git-scm.com)

#### Clone and Build

```bash
# Clone the repository
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus

# Download dependencies
go mod download

# Build
make build

# Or build manually
go build -o wunderpus ./cmd/wunderpus
```

The binary will be created at `./wunderpus` or `./bin/wunderpus` (via make).

#### Build Options

The Makefile provides additional build targets:

```bash
make build           # Build for current platform
make build-all      # Build for all platforms (Linux, macOS, Windows)
make build-linux    # Build for Linux
make build-darwin   # Build for macOS
make build-windows  # Build for Windows
make clean          # Clean build artifacts
```

#### Run Without Building

```bash
# Run directly from source
go run cmd/wunderpus/main.go

# With configuration
go run cmd/wunderpus/main.go --config /path/to/config.yaml
```

## Post-Installation

### 1. Create Configuration

```bash
# Copy example configuration
cp config.example.yaml config.yaml

# Edit with your preferred editor
vim config.yaml
# or
nano config.yaml
# or
code config.yaml
```

### 2. Add API Keys

At minimum, you need one LLM provider. Add your API key:

```yaml
# Option 1: In config.yaml
providers:
  openai:
    api_key: "sk-your-key-here"
    model: "gpt-4o"

# Option 2: Environment variables (recommended)
# Export before running:
export OPENAI_API_KEY="sk-your-key-here"
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

See the [Configuration Guide](configuration.md) for complete options.

### 3. Verify Installation

```bash
# Check version
wunderpus --version

# Check status
wunderpus status

# Or start interactive mode
wunderpus
```

## Installation Verification

Run the following to verify your installation:

```bash
# 1. Version check
$ wunderpus --version
wunderpus version 0.1.0

# 2. Authentication status
$ wunderpus auth status
Authentication Status:
- openai: Authenticated

# 3. Provider test (one-shot message)
$ wunderpus agent -m "Say hello"
Hello! How can I help you today?

# 4. Health check (if running gateway)
$ curl http://localhost:8080/health
{"status":"ok"}
```

## Optional Dependencies

Depending on your use case, you may need:

### For Specific Channels

| Channel | Dependency | Installation |
|---------|------------|--------------|
| Telegram | Bot Token | Create via @BotFather |
| Discord | Bot Token | Create in Discord Developer Portal |
| QQ | None | Use OneBot protocol |
| WeCom | Corp ID + Agent ID | Configure in WeCom admin |

### For Tools

| Tool | Dependency | Installation |
|------|------------|--------------|
| GitHub | `gh` CLI | `brew install gh` or see [cli.github.com](https://cli.github.com) |
| Tmux | `tmux` | `brew install tmux` or `apt install tmux` |

### For Development

```bash
# Install development tools
make setup-dev

# This includes:
# - golangci-lint
# - goimports
# - gci
# - gofumpt
```

## Upgrading

### Binary Installation

```bash
# Linux/macOS - re-download
curl -sL https://github.com/wunderpus/wunderpus/releases/latest/download/... | tar xz

# Homebrew
brew upgrade wunderpus

# Scoop
scoop update wunderpus
```

### Docker

```bash
docker pull wunderpus/wunderpus:latest
```

### From Source

```bash
git pull origin main
go mod download
make build
```

### Configuration Migration

When upgrading, check the [CHANGELOG](CHANGELOG.md) for configuration changes. You may need to:

1. Update deprecated configuration keys
2. Add new required settings
3. Migrate to new configuration format

## Uninstallation

### Binary

```bash
# Linux/macOS
sudo rm /usr/local/bin/wunderpus

# macOS Homebrew
brew uninstall wunderpus

# Windows Scoop
scoop uninstall wunderpus
```

### Docker

```bash
docker rm wunderpus
docker rmi wunderpus/wunderpus:latest
```

### Source

```bash
rm -rf /path/to/wunderpus
```

## Troubleshooting

### "command not found" after installation

Ensure the binary is in your PATH:
```bash
# Add to PATH temporarily
export PATH=$PATH:/usr/local/bin

# Or add permanently (Linux)
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

### Permission denied

```bash
# Make executable
chmod +x wunderpus

# Or install to directory with write permissions
./wunderpus
```

### Connection errors

- Check firewall/proxy settings
- Verify network access to LLM providers
- Ensure API keys are correct
- Check provider status pages

See [Troubleshooting Guide](troubleshooting.md) for more help.
