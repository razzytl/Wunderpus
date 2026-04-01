# Monitoring & Observability

Wunderpus provides comprehensive monitoring through Prometheus metrics, health checks, and structured logging.

## Prometheus Metrics

### Enable Metrics

Metrics are exposed on the health server port at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

### Available Metrics

| Metric | Type | Labels | Description |
|---|---|---|---|
| `wunderpus_messages_total` | Counter | `channel`, `provider` | Total messages processed |
| `wunderpus_tool_execution_seconds` | Histogram | `tool` | Tool execution duration |
| `wunderpus_agent_errors_total` | Counter | `type` | Agent errors by type |
| `wunderpus_tokens_total` | Counter | `model`, `type` | Token usage (input/output) |
| `wunderpus_cost_usd_total` | Counter | `model`, `session` | Cumulative cost in USD |
| `wunderpus_provider_latency_seconds` | Histogram | `provider`, `model` | Provider response latency |

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'wunderpus'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana

Pre-built Grafana dashboards are available in the `grafana/` directory.

## Health Checks

### Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Overall health with uptime |
| `/live` | GET | Liveness probe |
| `/ready` | GET | Readiness — provider/channel status |

### Response Format

```json
// /health
{
  "status": "ok",
  "uptime": "2h30m15s"
}

// /ready
{
  "status": "ready",
  "providers": {
    "openai": "connected",
    "anthropic": "connected"
  },
  "channels": {
    "telegram": "connected"
  }
}
```

## Structured Logging

### Configuration

```yaml
logging:
  level: "info"       # debug, info, warn, error
  format: "json"      # json, text
  output: "stderr"    # stderr, stdout, or file path
```

### JSON Log Format

```json
{
  "level": "info",
  "time": "2026-04-01T10:30:00Z",
  "caller": "agent/manager.go:123",
  "msg": "Processing message",
  "session_id": "abc123",
  "provider": "openai"
}
```

### Correlation IDs

Each request gets a correlation ID for tracing:

```go
logger := logging.WithCorrelation(requestID)
```

## Log Aggregation

### Docker

```bash
docker logs -f wunderpus-agent
```

### File Output

```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/wunderpus/app.log"
```

### Fluent Bit / Loki

```yaml
# fluent-bit.conf
[INPUT]
    Name tail
    Path /var/log/wunderpus/*.log
    Parser json

[OUTPUT]
    Name loki
    Match *
    Host loki
    Port 3100
```

## Alerting

### Recommended Alerts

| Alert | Condition | Severity |
|---|---|---|
| High Error Rate | `rate(wunderpus_agent_errors_total[5m]) > 0.1` | Critical |
| High Latency | `histogram_quantile(0.95, ...) > 5s` | Warning |
| Provider Down | Provider not responding | Critical |
| Rate Limit Hit | Rate limits exceeded frequently | Warning |
| High Memory | Memory usage > 90% | Warning |

## Observability Stack

### Quick Start with Docker Compose

```yaml
services:
  wunderpus:
    image: wunderpus:latest
    ports:
      - "8080:8080"
      - "9090:9090"

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  grafana-data:
```
