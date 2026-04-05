# Provider System

Wunderpus connects to 15+ LLM providers through a protocol-based routing system with automatic fallback, caching, and cooldown management.

## Supported Protocols

| Protocol | Providers |
|---|---|
| `openai` | OpenAI, Groq, OpenRouter, DeepSeek, Cerebras, NVIDIA, LiteLLM, vLLM, Mistral, Qwen, Zhipu, Moonshot |
| `anthropic` | Anthropic Claude |
| `ollama` | Local Ollama instances |
| `gemini` | Google Gemini |

## Configuration

### Recommended: model_list Format

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "sk-your-key"
    max_tokens: 4096

  - model_name: "claude"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "sk-ant-your-key"
    max_tokens: 4096

  - model_name: "local"
    model: "ollama/llama3.2"
    api_base: "http://localhost:11434"
    max_tokens: 4096
```

The `model` field format is `protocol/model-id`. The protocol is auto-detected from the prefix.

### Legacy: Provider-Specific Format

```yaml
default_provider: "openai"

providers:
  openai:
    api_key: "sk-your-key"
    model: "gpt-4o"
    max_tokens: 4096
```

## Provider Router

The router manages all registered providers and handles:

### Fallback Chain

```
Request
    │
    ▼
Active Provider (primary)
    │  (fails)
    ▼
Fallback Models (from config)
    │  (all fail)
    ▼
All Other Healthy Providers
    │  (those not in cooldown)
    ▼
Error (if all exhausted)
```

### Cooldown System

Failed providers enter cooldown:
- Tracked per-provider failure count
- Providers with ≥5 recent failures are skipped
- Cooldown clears after successful request

### Response Cache

5-minute TTL cache keyed by request content:
- Identical requests return cached responses
- Reduces API costs for repeated queries
- Automatically invalidated on TTL expiry

## OpenAI-Compatible Providers

Works with any OpenAI-compatible API:

```yaml
- model_name: "gpt4o"
  model: "openai/gpt-4o"
  api_key: "sk-..."
  api_base: "https://api.openai.com/v1"  # Custom base for proxies
  max_tokens: 4096
```

**Supported providers via this protocol:**

| Provider | API Base | Model Prefix |
|---|---|---|
| OpenAI | `api.openai.com` | `openai/` |
| Groq | `api.groq.com/openai/v1` | `groq/` |
| OpenRouter | `openrouter.ai/api/v1` | `openrouter/` |
| DeepSeek | `api.deepseek.com/v1` | `deepseek/` |
| Cerebras | `api.cerebras.ai/v1` | `cerebras/` |
| NVIDIA NIM | `integrate.api.nvidia.com/v1` | `nvidia/` |
| Mistral | `api.mistral.ai/v1` | `mistral/` |
| vLLM | `localhost:8000/v1` | `vllm/` |
| LiteLLM | `localhost:4000/v1` | `litellm/` |
| Qwen | `dashscope.aliyuncs.com` | `qwen/` |
| Zhipu | `open.bigmodel.cn` | `zhipu/` |
| Moonshot | `api.moonshot.cn` | `moonshot/` |

## Embeddings

For RAG vector search, configure an embedding-capable provider:

```yaml
# The router auto-selects the first provider that supports embeddings
model_list:
  - model_name: "embedder"
    model: "openai/text-embedding-3-small"
    api_key: "sk-..."
```

## Free Tiers

Wunderpus tracks free-tier providers in `free_tiers.yaml`:

| Provider | Free Tier | Rate Limit | Models |
|---|---|---|---|
| OpenRouter | Yes | 20 req/min | gemma-2-9b, llama-3.1-8b |
| Groq | Yes | 30 req/min | llama3-8b, mixtral-8x7b |
| DeepSeek | Yes | 60 req/min | deepseek-chat |

## Best Practices

1. **Always configure fallbacks** — At least one backup provider
2. **Use environment variables** for API keys
3. **Set appropriate max_tokens** — Match your use case
4. **Monitor costs** — Enable cost tracking
5. **Choose models wisely** — Use cheaper models for simple tasks
