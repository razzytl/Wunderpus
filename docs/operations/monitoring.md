# Monitoring & Observability

Wunderpus provides comprehensive monitoring through Prometheus metrics, health checks, OpenTelemetry tracing, and structured logging.

## Health Checks

### Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Overall health with component-level status |
| `/live` | GET | Liveness probe — always returns `{"status": "ok"}` |
| `/ready` | GET | Readiness — returns unhealthy if any critical component is down |

### Response Format

```json
// GET /health
{
  "status": "healthy",
  "uptime": "2h30m15s",
  "components": {
    "core_db": {"status": "healthy"},
    "audit_db": {"status": "healthy"},
    "provider": {"status": "healthy"},
    "memory": {"status": "healthy", "details": "42 records"},
    "channel:telegram": {"status": "healthy"},
    "channel:discord": {"status": "healthy"}
  }
}
```

Status values: `healthy`, `degraded`, `unhealthy`

The overall status is the worst status across all components.

### Health Aggregator

Components are registered during bootstrap:

```go
healthAgg := health.NewAggregator()
health.RegisterDBCheck(healthAgg, "core_db", dbManager.CoreDB)
health.RegisterDBCheck(healthAgg, "audit_db", dbManager.AuditDB)
health.RegisterProviderCheck(healthAgg, "provider", func() bool {
    return router.Active() != nil
})
health.RegisterMemoryCheck(healthAgg, "memory", func() int {
    sessions, _ := memStore.GetSessions()
    return len(sessions)
})
health.RegisterChannelCheck(healthAgg, "channel:telegram", func() bool {
    return telegramChannel != nil
})
```

## OpenTelemetry Tracing

Wunderpus instruments key operations with OpenTelemetry spans:

### Span Hierarchy

```
agent.handle_message
├── agent.loop_iteration (iteration 1)
│   ├── provider.complete
│   └── tool.execute (×N)
├── agent.loop_iteration (iteration 2)
│   ├── provider.complete
│   └── tool.execute (×N)
└── ...
```

### Span Attributes

| Span | Attributes |
|---|---|
| `agent.handle_message` | `session_id`, `input_length` |
| `agent.loop_iteration` | `iteration.count`, `provider.name`, `tool_call_count` |
| `provider.complete` | `provider.name`, `model`, `message_count`, `prompt_tokens`, `completion_tokens`, `finish_reason` |
| `provider.stream` | `provider.name`, `model` |
| `tool.execute` | `tool.name`, `duration_ms`, `has_error`, `error.message` |

### Configuration

The tracer attempts to connect to an OTLP HTTP collector at `localhost:4318`. If no collector is available, it falls back to a no-op tracer (spans are created but not exported).

To collect traces, run Jaeger or Tempo:

```bash
# Jaeger all-in-one
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 4318:4318 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Then visit `http://localhost:16686` to view traces.

## Prometheus Metrics

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

## Alerting

### Recommended Alerts

| Alert | Condition | Severity |
|---|---|---|
| High Error Rate | `rate(wunderpus_agent_errors_total[5m]) > 0.1` | Critical |
| High Latency | `histogram_quantile(0.95, ...) > 5s` | Warning |
| Provider Down | Provider not responding | Critical |
| Rate Limit Hit | Rate limits exceeded frequently | Warning |
| High Memory | Memory usage > 90% | Warning |
| Health Unhealthy | `/health` returns non-200 | Critical |
