# Deployment Guide

This guide covers production deployment strategies for Wunderpus, including Docker, systemd, and cloud-native configurations.

## Deployment Options

| Method | Use Case | Complexity |
|--------|----------|------------|
| Docker | Containerized deployments | Low |
| Binary | Direct installation | Low |
| systemd | Linux servers | Medium |
| Kubernetes | Cloud-native, scalable | High |

## Docker Deployment

### Basic Docker Setup

```yaml
# docker-compose.yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    container_name: wunderpus
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./data:/app/data
      - ./logs:/app/logs
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
    ports:
      - "8080:8080"  # Health check
      - "8081:8081"  # WebSocket
    networks:
      - wunderpus-network

networks:
  wunderpus-network:
    driver: bridge
```

Run:
```bash
docker-compose up -d
```

### Production Docker Configuration

```yaml
# docker-compose.prod.yaml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    container_name: wunderpus
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - wunderpus-data:/app/data
      - wunderpus-logs:/app/logs
    environment:
      - WUNDERPUS_CONFIG=/app/config.yaml
      - WUNDERPUS_LOG_LEVEL=info
    ports:
      - "127.0.0.1:8080:8080"
      - "127.0.0.1:8081:8081"
    healthcheck:
      test: ["CMD", "wunderpus", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    mem_limit: 512m
    cpus: 1.0
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs:
      - /tmp:size=100m,mode=1777

volumes:
  wunderpus-data:
  wunderpus-logs:
```

### Building Custom Image

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o wunderpus ./cmd/wunderpus

# Production image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite

# Create non-root user
RUN adduser -D -u 1000 wunderpus
USER wunderpus

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=wunderpus:wunderpus /app/wunderpus .
COPY --from=builder --chown=wunderpus:wunderpus /app/config.example.yaml .

# Create required directories
RUN mkdir -p data logs

# Expose ports
EXPOSE 8080 8081

# Run
ENTRYPOINT ["./wunderpus"]
CMD ["gateway"]
```

Build and run:
```bash
docker build -t my-wunderpus .
docker run -d \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  -p 8080:8080 \
  my-wunderpus gateway
```

## Systemd Deployment (Linux)

### Install Binary

```bash
# Download
curl -sL https://github.com/wunderpus/wunderpus/releases/latest/download/wunderpus-linux-amd64.tar.gz | tar xz

# Install
sudo mv wunderpus /usr/local/bin/
sudo chmod +x /usr/local/bin/wunderpus

# Verify
wunderpus --version
```

### Create Service User

```bash
# Create dedicated user
sudo useradd -r -s /bin/false wunderpus

# Create directories
sudo mkdir -p /etc/wunderpus
sudo mkdir -p /var/lib/wunderpus
sudo mkdir -p /var/log/wunderpus

# Set ownership
sudo chown -R wunderpus:wunderpus /etc/wunderpus
sudo chown -R wunderpus:wunderpus /var/lib/wunderpus
sudo chown -R wunderpus:wunderpus /var/log/wunderpus
```

### Create Configuration

```bash
# Copy example config
sudo cp config.example.yaml /etc/wunderpus/config.yaml
sudo chown wunderpus:wunderpus /etc/wunderpus/config.yaml
sudo chmod 600 /etc/wunderpus/config.yaml

# Edit configuration
sudo vim /etc/wunderpus/config.yaml
```

### Create Systemd Service

```ini
# /etc/systemd/system/wunderpus.service
[Unit]
Description=Wunderpus AI Agent
After=network.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
User=wunderpus
Group=wunderpus
WorkingDirectory=/var/lib/wunderpus
ExecStart=/usr/local/bin/wunderpus gateway --config /etc/wunderpus/config.yaml
Restart=always
RestartSec=10

# Environment
Environment=WUNDERPUS_CONFIG=/etc/wunderpus/config.yaml
Environment=WUNDERPUS_LOG_LEVEL=info

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/wunderpus /var/log/wunderpus

# Resource limits
MemoryMax=512M
CPUQuota=50%

[Install]
WantedBy=multi-user.target
```

### Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable and start
sudo systemctl enable --now wunderpus

# Check status
sudo systemctl status wunderpus

# View logs
sudo journalctl -u wunderpus -f
```

## Kubernetes Deployment

### Deployment Manifest

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wunderpus
  labels:
    app: wunderpus
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
        image: wunderpus/wunderpus:latest
        ports:
        - containerPort: 8080
          name: health
        - containerPort: 8081
          name: websocket
        env:
        - name: WUNDERPUS_CONFIG
          value: "/app/config/config.yaml"
        - name: WUNDERPUS_LOG_LEVEL
          value: "info"
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
      - name: data:
        emptyDir: {}

---
apiVersion: v1
kind: Service
metadata:
  name: wunderpus
spec:
  selector:
    app: wunderpus
  ports:
  - name: health
    port: 8080
    targetPort: 8080
  - name: websocket
    port: 8081
    targetPort: 8081
```

### Secrets

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: wunderpus-config
type: Opaque
stringData:
  config.yaml: |
    providers:
      openai:
        api_key: "${OPENAI_API_KEY}"
    # ... rest of config
```

### Horizontal Pod Autoscaler

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: wunderpus-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: wunderpus
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

Deploy:
```bash
kubectl apply -f k8s/
```

## Reverse Proxy Configuration

### Nginx

```nginx
# /etc/nginx/sites-available/wunderpus
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

    location /ws {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Caddy

```Caddyfile
wunderpus.example.com {
    reverse_proxy localhost:8080
	
    # WebSocket support
    @ws {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @ws localhost:8081
}
```

## Environment-Specific Configuration

### Development

```yaml
# config.dev.yaml
logging:
  level: debug
  format: text

server:
  health_port: 8080

tools:
  enabled: true
  shell_whitelist:
    - git
    - go
    - npm
```

### Staging

```yaml
# config.staging.yaml
logging:
  level: info
  format: json

server:
  health_port: 8080

security:
  audit_enabled: true
  rate_limit:
    requests_per_minute: 30
```

### Production

```yaml
# config.prod.yaml
logging:
  level: warn
  format: json
  output: /var/log/wunderpus/app.log

server:
  health_port: 8080
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/wunderpus.crt
    key_file: /etc/ssl/private/wunderpus.key

security:
  encryption:
    enabled: true
    key: "${ENCRYPTION_KEY}"
  audit_enabled: true
  audit_db_path: /var/lib/wunderpus/audit.db
  rate_limit:
    requests_per_minute: 60

tools:
  enabled: true
  shell_whitelist:
    - git
    - go
```

## Health Checks

### Built-in Health Endpoint

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": "24h0m0s"
}
```

### Readiness Check

```bash
curl http://localhost:8080/ready
```

### Kubernetes Probe Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
```

## Monitoring Integration

### Prometheus Metrics

Enable metrics endpoint:

```yaml
monitoring:
  prometheus:
    enabled: true
    port: 9090
    path: /metrics
```

Scrape configuration:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'wunderpus'
    static_configs:
      - targets: ['wunderpus:9090']
```

### Log Aggregation

#### Fluent Bit

```yaml
# fluent-bit.conf
[INPUT]
    Name tail
    Path /var/log/wunderpus/*.log

[OUTPUT]
    Name stdout
    Match *
```

#### Loki

```yaml
# promtail.yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: wunderpus
    static_configs:
      - targets: [localhost]
        labels:
          job: wunderpus
          __path__: /var/log/wunderpus/*.log
```

## Backup and Recovery

### Database Backup

```bash
# Backup audit database
cp /var/lib/wunderpus/wunderpus_audit.db /backup/audit-$(date +%Y%m%d).db

# Backup session database
cp /var/lib/wunderpus/sessions.db /backup/sessions-$(date +%Y%m%d).db
```

### Automated Backup

```bash
# /etc/cron.daily/wunderpus-backup
#!/bin/bash
BACKUP_DIR="/backup/wunderpus"
DATE=$(date +%Y%m%d)

mkdir -p "$BACKUP_DIR"

# Backup databases
cp /var/lib/wunderpus/*.db "$BACKUP_DIR/"

# Compress
tar -czf "$BACKUP_DIR-$DATE.tar.gz" -C "$BACKUP_DIR" .

# Cleanup old backups (keep 7 days)
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +7 -delete
```

## Performance Tuning

### Connection Pooling

```yaml
provider:
  openai:
    http_client:
      max_idle_connections: 10
      max_idle_connections_per_host: 5
      idle_conn_timeout: 30s
```

### Memory Management

```yaml
agent:
  max_context_tokens: 8000
  max_sessions: 100
```

### Database Optimization

```bash
# Analyze database
sqlite3 wunderpus_audit.db "ANALYZE;"

# Vacuum to reclaim space
sqlite3 wunderpus_audit.db "VACUUM;"
```

## Troubleshooting Deployment

### Container Won't Start

1. Check logs:
   ```bash
   docker logs wunderpus
   ```

2. Verify configuration:
   ```bash
   docker run --rm -v $(pwd)/config.yaml:/app/config.yaml:ro wunderpus/wunderpus:latest wunderpus status
   ```

### High Memory Usage

1. Check container metrics
2. Reduce `max_context_tokens`
3. Limit concurrent sessions

### Connection Issues

1. Verify network configuration
2. Check firewall rules
3. Test provider connectivity
