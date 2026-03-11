# Monitoring and Observability

Wunderpus provides comprehensive monitoring capabilities through Prometheus metrics, health checks, and structured logging. This guide covers configuration and usage.

## Overview

The monitoring system consists of:

- **Prometheus Metrics**: Quantitative measurements for alerting and dashboards
- **Health Checks**: Liveness and readiness probes
- **Structured Logging**: JSON-formatted logs for analysis

## Prometheus Metrics

### Enabling Metrics

```yaml
monitoring:
  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"
```

### Metrics Endpoint

Once enabled, metrics are available at:

```
http://localhost:9090/metrics
```

### Available Metrics

#### Agent Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `wunderpus_agent_requests_total` | Counter | Total agent requests |
| `wunderpus_agent_requests_duration_seconds` | Histogram | Request duration |
| `wunderpus_agent_messages_total` | Counter | Total messages processed |
| `wunderpus_agent_sessions_active` | Gauge | Active sessions |
| `wunderpus_agent_sessions_total` | Counter | Total sessions created |

#### Provider Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `wunderpus_provider_requests_total` | Counter | Requests per provider |
| `wunderpus_provider_errors_total` | Counter | Errors per provider |
| `wunderpus_provider_duration_seconds` | Histogram | Provider response time |
| `wunderpus_provider_tokens_total` | Counter | Tokens used per provider |
| `wunderpus_provider_cost_total` | Gauge | Cumulative cost |

#### Channel Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `wunderpus_channel_messages_total` | Counter | Messages per channel |
| `wunderpus_channel_errors_total` | Counter | Channel errors |
| `wunderpus_channel_connections` | Gauge | Active connections |

#### Tool Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `wunderpus_tool_executions_total` | Counter | Tool executions |
| `wunderpus_tool_execution_duration_seconds` | Histogram | Tool execution time |
| `wunderpus_tool_errors_total` | Counter | Tool errors |

#### System Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `wunderpus_build_info` | Gauge | Build version info |
| `wunderpus_uptime_seconds` | Gauge | Application uptime |
| `wunderpus_memory_usage_bytes` | Gauge | Memory usage |
| `wunderpus_goroutines` | Gauge | Number of goroutines |

### Example Queries

#### Request Rate

```promql
rate(wunderpus_agent_requests_total[5m])
```

#### Error Rate

```promql
rate(wunderpus_provider_errors_total[5m])
```

#### Token Usage

```promql
sum by (provider) (wunderpus_provider_tokens_total)
```

#### P95 Latency

```promql
histogram_quantile(0.95, wunderpus_provider_duration_seconds_bucket)
```

## Prometheus Configuration

### Basic Scrape Config

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'wunderpus'
    static_configs:
      - targets: ['localhost:9090']
```

### Advanced Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'wunderpus'
    static_configs:
      - targets: ['wunderpus:9090']
    metrics_path: '/metrics'
    scrape_interval: 10s
    basic_auth:
      username: 'monitoring'
      password: '${PROMETHEUS_PASSWORD}'
```

## Health Checks

### Configuration

```yaml
server:
  health_port: 8080
```

### Endpoints

#### Liveness Probe

```bash
GET /health
```

Response:
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": "24h0m0s"
}
```

Used to determine if the container should be restarted.

#### Readiness Probe

```bash
GET /ready
```

Response:
```json
{
  "status": "ready",
  "providers": {
    "openai": "connected",
    "anthropic": "connected"
  },
  "channels": {
    "telegram": "connected",
    "discord": "connected"
  }
}
```

Used to determine if the service should receive traffic.

### Kubernetes Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  failureThreshold: 3
  successThreshold: 1

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  failureThreshold: 3
  successThreshold: 1
```

## Logging

### Configuration

```yaml
logging:
  level: "info"      # debug, info, warn, error
  format: "json"     # json, text
  output: "stderr"   # stderr, stdout, or file path
```

### JSON Format

When using JSON format, logs include:

```json
{
  "level": "info",
  "time": "2024-01-15T10:30:00Z",
  "caller": "agent/manager.go:123",
  "msg": "Processing message",
  "session_id": "abc123",
  "provider": "openai"
}
```

### Log Levels

| Level | Usage |
|-------|-------|
| `debug` | Detailed debugging info |
| `info` | General informational messages |
| `warn` | Warning conditions |
| `error` | Error conditions |

### Structured Logging Fields

Key fields included in logs:

| Field | Description |
|-------|-------------|
| `timestamp` | ISO8601 timestamp |
| `level` | Log level |
| `message` | Log message |
| `session_id` | Session identifier |
| `provider` | LLM provider |
| `channel` | Communication channel |
| `error` | Error message (if applicable) |

### Log Aggregation

#### File Output

```yaml
logging:
  level: "info"
  format: "json"
  output: "/var/log/wunderpus/app.log"
```

#### Docker Logging

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  }
}
```

#### Fluent Bit Integration

```yaml
logging:
  output: "stdout"
  format: "json"
```

```yaml
# fluent-bit.conf
[INPUT]
    Name tail
    Path /var/lib/docker/containers/*/*.log
    Parser docker
    Tag wunderpus.*

[FILTER]
    Name parser
    Match wunderpus.*
    Key_Name log
    Parser json

[OUTPUT]
    Name loki
    Match *
    Host loki
    Port 3100
```

## Alerting Rules

### Example Alert Rules

```yaml
# alerts.yml
groups:
  - name: wunderpus
    rules:
      # High Error Rate
      - alert: HighErrorRate
        expr: rate(wunderpus_provider_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate on {{ $labels.provider }}"
          
      # High Latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, wunderpus_provider_duration_seconds_bucket) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P95 latency above 5s"
          
      # Provider Down
      - alert: ProviderDown
        expr: wunderpus_provider_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Provider {{ $labels.provider }} is down"
          
      # Rate Limit Hit
      - alert: RateLimitExceeded
        expr: rate(wunderpus_security_rate_limits_exceeded_total[5m]) > 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Rate limits being exceeded"
          
      # High Memory Usage
      - alert: HighMemoryUsage
        expr: wunderpus_memory_usage_bytes / wunderpus_memory_limit_bytes > 0.9
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Memory usage above 90%"
```

## Dashboards

### Grafana Dashboard

Example Grafana dashboard JSON:

```json
{
  "title": "Wunderpus Overview",
  "panels": [
    {
      "title": "Requests per Second",
      "type": "graph",
      "targets": [
        {
          "expr": "rate(wunderpus_agent_requests_total[1m])"
        }
      ]
    },
    {
      "title": "Provider Response Time (P95)",
      "type": "graph",
      "targets": [
        {
          "expr": "histogram_quantile(0.95, rate(wunderpus_provider_duration_seconds_bucket[5m]))",
          "legendFormat": "{{provider}}"
        }
      ]
    },
    {
      "title": "Token Usage",
      "type": "graph",
      "targets": [
        {
          "expr": "sum by (provider) (wunderpus_provider_tokens_total)"
        }
      ]
    },
    {
      "title": "Active Sessions",
      "type": "stat",
      "targets": [
        {
          "expr": "wunderpus_agent_sessions_active"
        }
      ]
    },
    {
      "title": "Error Rate",
      "type": "graph",
      "targets": [
        {
          "expr": "rate(wunderpus_provider_errors_total[5m])"
        }
      ]
    }
  ]
}
```

## Integration Examples

### Prometheus + Grafana Stack

```yaml
# docker-compose.yml
version: '3.8'

services:
  wunderpus:
    image: wunderpus/wunderpus:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    volumes:
      - ./config.yaml:/app/config.yaml

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  prometheus-data:
  grafana-data:
```

### Alertmanager Integration

```yaml
# alertmanager.yml
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'default'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://alert-webhook:5000/alerts'
```

## Best Practices

1. **Enable Metrics in Production**: Always enable Prometheus metrics for production deployments
2. **Set Appropriate Retention**: Configure Prometheus retention based on storage (default: 15d)
3. **Monitor Both Success and Failure**: Track both successful requests and errors
4. **Set Up Alerts**: Configure alerts for critical conditions before they become outages
5. **Use Distributed Tracing**: For complex deployments, consider adding trace IDs to logs
6. **Log Correlation**: Include session_id, request_id in all logs for debugging
7. **Capacity Planning**: Use metrics to plan capacity and scaling

## Troubleshooting

### Metrics Not Appearing

1. Check metrics endpoint is enabled:
   ```bash
   curl http://localhost:9090/metrics
   ```

2. Verify Prometheus is scraping the target:
   ```bash
   curl http://localhost:9090/api/v1/targets
   ```

### High Memory Usage

1. Check memory metrics:
   ```bash
   wunderpus_memory_usage_bytes
   ```

2. Review session limits:
   ```yaml
   agent:
     max_sessions: 100
   ```

### Health Check Failures

1. Check health endpoint:
   ```bash
   curl http://localhost:8080/health
   ```

2. Review provider status:
   ```bash
   curl http://localhost:8080/ready
   ```

3. Check logs for errors:
   ```bash
   wunderpus gateway -v 2>&1 | grep -i error
   ```
