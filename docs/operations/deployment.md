# Deployment Guide

Production deployment strategies for Wunderpus.

## Deployment Options

| Method | Use Case | Complexity |
|---|---|---|
| Binary | Direct installation | Low |
| Docker | Containerized | Low |
| systemd | Linux servers | Medium |

## Docker

### Quick Start

```bash
docker build -t wunderpus:latest .
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  -v wunderpus-data:/app/data \
  wunderpus:latest
```

### Docker Compose

```yaml
services:
  wunderpus:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    volumes:
      - wunderpus-data:/app/data
      - ./config.yaml:/app/config.yaml:ro
      - ./skills:/app/skills:ro
    restart: unless-stopped

volumes:
  wunderpus-data:
```

## Binary Deployment

### Build

```bash
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus
go build -o build/wunderpus ./cmd/wunderpus
```

### Run

```bash
./build/wunderpus gateway &
```

## systemd Deployment

### Unit File

Create `/etc/systemd/system/wunderpus.service`:

```ini
[Unit]
Description=Wunderpus AI Agent
After=network.target

[Service]
Type=simple
User=wunderpus
Group=wunderpus
WorkingDirectory=/opt/wunderpus
ExecStart=/opt/wunderpus/build/wunderpus gateway
Restart=on-failure
RestartSec=10

Environment="WUNDERPUS_CONFIG=/opt/wunderpus/config.yaml"
Environment="OPENAI_API_KEY=sk-..."

[Install]
WantedBy=multi-user.target
```

### Enable and Start

```bash
sudo systemctl daemon-reload
sudo systemctl enable wunderpus
sudo systemctl start wunderpus
sudo systemctl status wunderpus
```

## Production Checklist

- [ ] Config file permissions: `chmod 600 config.yaml`
- [ ] API keys in environment variables, not config files
- [ ] Database files on persistent storage
- [ ] Health check endpoint monitored (`/health`)
- [ ] Log output captured (systemd journal or file)
- [ ] Workspace directory isolated and backed up
- [ ] Rate limiting enabled
- [ ] Encryption enabled for sensitive data
