# Deployment Guide

Production deployment strategies for Wunderpus.

## Deployment Options

| Method | Use Case | Complexity |
|---|---|---|
| Binary | Direct installation | Low |
| Docker | Containerized | Low |
| systemd | Linux servers | Medium |
| Kubernetes | Cloud-native, scalable | High |

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
# docker-compose.yml
services:
  agent:
    build: .
    image: wunderpus:agent
    container_name: wunderpus-agent
    profiles: [agent]
    environment:
      - WONDERPUS_CONFIG=/app/config.yaml
    volumes:
      - wunderpus-data:/app/data
      - ./skills:/app/skills:ro
    stdin_open: true
    tty: true
    restart: unless-stopped

  gateway:
    build: .
    image: wunderpus:gateway
    container_name: wonderpus-gateway
    profiles: [gateway]
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - WONDERPUS_CONFIG=/app/config.yaml
    volumes:
      - wunderpus-data:/app/data
      - ./skills:/app/skills:ro
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/live"]
      interval: 30s
      timeout: 10s
      retries: 3

  dev:
    build: .
    image: wunderpus:dev
    container_name: wunderpus-dev
    profiles: [dev]
    environment:
      - WONDERPUS_CONFIG=/app/config.yaml
    volumes:
      - wunderpus-data:/app/data
      - ./skills:/app/skills
      - ./workspace:/app/workspace
    stdin_open: true
    tty: true

volumes:
  wunderpus-data:
```

### Run with Profiles

```bash
# Gateway mode
docker compose --profile gateway up -d

# Agent mode
docker compose --profile agent up

# Development
docker compose --profile dev up
```

### Dockerfile

The production Dockerfile uses a multi-stage build with `scratch` as the final stage:

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download -x
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wonderpus \
    -ldflags="-s -w -buildid=" \
    -trimpath \
    ./cmd/wonderpus

# Production stage - Ultra minimal
FROM scratch
LABEL org.opencontainers.image.source="https://github.com/wunderpus/wunderpus"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/wonderpus /usr/local/bin/wunderpus
COPY --from=builder /app/config.example.yaml /app/config.yaml
WORKDIR /app
EXPOSE 8080 9090
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/wunderpus"]
```

## Binary Installation

### Linux

```bash
# Build
make build

# Install to ~/.local/bin
make install

# Or manually
sudo cp build/wunderpus /usr/local/bin/
sudo chmod +x /usr/local/bin/wunderpus
```

### macOS

```bash
make build
cp build/wunderpus /usr/local/bin/
```

### Windows

```bash
go build -o wunderpus.exe ./cmd/wunderpus
```

## systemd (Linux)

### Create Service

```ini
# /etc/systemd/system/wunderpus.service
[Unit]
Description=Wunderpus AI Agent
After=network.target

[Service]
Type=simple
User=wunderpus
Group=wunderpus
WorkingDirectory=/var/lib/wunderpus
ExecStart=/usr/local/bin/wunderpus gateway --config /etc/wunderpus/config.yaml
Restart=always
RestartSec=10
Environment=WUNDERPUS_CONFIG=/etc/wunderpus/config.yaml
Environment=WUNDERPUS_LOG_LEVEL=info

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/wunderpus /var/log/wunderpus

# Resources
MemoryMax=512M
CPUQuota=50%

[Install]
WantedBy=multi-user.target
```

### Start Service

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now wunderpus
sudo systemctl status wunderpus
sudo journalctl -u wunderpus -f
```

## Kubernetes

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wunderpus
spec:
  replicas: 2
  selector:
    matchLabels:
      app: wunderpus
  template:
    metadata:
      labels:
        app: wunderpus
    spec:
      containers:
      - name: wunderpus
        image: wunderpus:latest
        ports:
        - containerPort: 8080
          name: health
        - containerPort: 9090
          name: websocket
        env:
        - name: WUNDERPUS_CONFIG
          value: "/app/config/config.yaml"
        volumeMounts:
        - name: config
          mountPath: /app/config
          readOnly: true
        - name: data
          mountPath: /app/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        secret:
          secretName: wunderpus-config
      - name: data
        emptyDir: {}
```

## Reverse Proxy

### Nginx

```nginx
server {
    listen 443 ssl http2;
    server_name wunderpus.example.com;

    ssl_certificate /etc/ssl/certs/wunderpus.crt;
    ssl_certificate_key /etc/ssl/private/wunderpus.key;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Caddy

```Caddyfile
wunderpus.example.com {
    reverse_proxy localhost:8080
}
```

## Health Checks

| Endpoint | Purpose |
|---|---|
| `/health` | Liveness — returns uptime JSON |
| `/live` | Liveness probe |
| `/ready` | Readiness — checks provider/channel connectivity |

```bash
curl http://localhost:8080/health
# {"status":"ok","uptime":"2h30m"}
```

## Backup

```bash
# Backup databases
cp ~/.wunderpus/*.db /backup/wunderpus-$(date +%Y%m%d)/

# Or with Docker
docker cp wunderpus-agent:/app/data /backup/
```
