# LLM Provider Configuration

Wunderpus provides vendor-agnostic access to multiple LLM providers. This guide covers configuration options, supported models, and best practices.

## Configuration Format

Wunderpus supports two configuration formats: the legacy provider-specific format and the new recommended `model_list` format.

### Recommended: model_list Format

The `model_list` format provides a unified, provider-agnostic configuration:

```yaml
model_list:
  - model_name: "gpt-primary"
    model: "openai/gpt-4o"
    api_key: "sk-your-key"
    max_tokens: 4096
    temperature: 0.7

  - model_name: "claude-fast"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "sk-ant-your-key"
    max_tokens: 4096

  - model_name: "local-llama"
    model: "ollama/llama3.2"
    api_base: "http://localhost:11434"
```

The model identifier uses the format `provider/model-name`. The protocol is auto-detected from the provider prefix.

### Legacy: Provider-Specific Format

For backward compatibility, individual provider configurations are still supported:

```yaml
default_provider: "openai"

providers:
  openai:
    api_key: "sk-your-key"
    model: "gpt-4o"
    max_tokens: 4096
```

## Supported Providers

### OpenAI

**Protocol:** `openai`

**Configuration:**
```yaml
- model_name: "gpt4o"
  model: "openai/gpt-4o"
  api_key: "sk-your-key"
  api_base: "https://api.openai.com/v1"  # optional, for proxies
  max_tokens: 4096
  temperature: 0.7
```

**Supported Models:**
- `gpt-4o` - Latest GPT-4 Omni
- `gpt-4o-mini` - Fast, cost-effective
- `gpt-4-turbo` - Previous generation
- `gpt-3.5-turbo` - Budget option

**Environment Variables:**
- `OPENAI_API_KEY`

### Anthropic

**Protocol:** `anthropic`

**Configuration:**
```yaml
- model_name: "claude-sonnet"
  model: "anthropic/claude-sonnet-4-20250514"
  api_key: "sk-ant-your-key"
  max_tokens: 4096
  temperature: 0.7
```

**Supported Models:**
- `claude-sonnet-4-20250514` - Latest Sonnet 4
- `claude-opus-4-20250514` - Most capable
- `claude-3-5-sonnet-20241022` - Previous generation
- `claude-3-haiku-20240307` - Fast, cost-effective

**Environment Variables:**
- `ANTHROPIC_API_KEY`

### Google Gemini

**Protocol:** `gemini`

**Configuration:**
```yaml
- model_name: "gemini-flash"
  model: "gemini/gemini-2.0-flash"
  api_key: "your-api-key"
  max_tokens: 4096
```

**Supported Models:**
- `gemini-2.0-flash` - Latest, fast
- `gemini-2.0-flash-exp` - Experimental
- `gemini-1.5-pro` - Large context
- `gemini-1.5-flash` - Cost-effective

**Environment Variables:**
- `GEMINI_API_KEY`

### Ollama (Local)

**Protocol:** `ollama`

**Configuration:**
```yaml
- model_name: "llama-local"
  model: "ollama/llama3.2"
  api_base: "http://localhost:11434"
  max_tokens: 4096
```

**Setup:**
```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2
```

**Supported Models:**
- Llama 3.2 variants
- Mistral
- Codestral
- Any model available in Ollama library

### OpenRouter

**Protocol:** `openai` (compatible)

**Configuration:**
```yaml
- model_name: "deepseek-r1"
  model: "openrouter/deepseek/deepseek-r1"
  api_key: "sk-or-your-key"
  api_base: "https://openrouter.ai/api/v1"
  max_tokens: 4096
```

**Access 100+ models** including:
- DeepSeek R1
- OpenAI models
- Anthropic models
- Meta Llama models
- And many more

**Environment Variables:**
- `OPENROUTER_API_KEY`

### Groq

**Protocol:** `openai` (compatible)

**Configuration:**
```yaml
- model_name: "groq-llama"
  model: "groq/llama-3.3-70b-versatile"
  api_key: "gsk_your-key"
  api_base: "https://api.groq.com/openai/v1"
  max_tokens: 4096
```

**Supported Models:**
- `llama-3.3-70b-versatile`
- `llama-3.1-70b-versatile`
- `mixtral-8x7b-32768`
- `gemma2-9b-it`

**Environment Variables:**
- `GROQ_API_KEY`

### DeepSeek

**Protocol:** `openai` (compatible)

**Configuration:**
```yaml
- model_name: "deepseek-r1"
  model: "deepseek/deepseek-chat"
  api_key: "your-key"
  api_base: "https://api.deepseek.com/v1"
  max_tokens: 4096
```

**Supported Models:**
- `deepseek-r1` - Reasoning model
- `deepseek-chat` - Chat model
- `deepseek-coder` - Code-focused

**Environment Variables:**
- `DEEPSEEK_API_KEY`

### Cerebras

**Protocol:** `openai` (compatible)

**Configuration:**
```yaml
- model_name: "cerebras-llama"
  model: "cerebras/llama-3.3-70b"
  api_key: "cbrs_your-key"
  api_base: "https://api.cerebras.ai/v1"
```

**Environment Variables:**
- `CEREBRAS_API_KEY`

### NVIDIA NIM

**Protocol:** `openai` (compatible)

**Configuration:**
```yaml
- model_name: "nvidia-nemotron"
  model: "nvidia/llama-3.1-nemotron-70b-instruct"
  api_key: "nvapi_your-key"
  api_base: "https://integrate.api.nvidia.com/v1"
```

**Environment Variables:**
- `NVIDIA_API_KEY`

### Additional Providers

| Provider | Model Prefix | API Base |
|----------|--------------|----------|
| Zhipu (GLM) | `zhipu/` | `https://open.bigmodel.cn/api/paas/v4` |
| Moonshot | `moonshot/` | `https://api.moonshot.cn/v1` |
| Mistral | `mistral/` | `https://api.mistral.ai/v1` |
| vLLM | `vllm/` | `http://localhost:8000/v1` |
| LiteLLM | `litellm/` | `http://localhost:4000/v1` |
| Qwen | `qwen/` | `https://dashscope.aliyuncs.com/api/v1` |
| Volcanic | `volcanic/` | `https://ark.cn-beijing.volces.com/api/v3` |

## Advanced Configuration

### Provider Fallback

Configure automatic fallback when a provider fails:

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "sk-primary"
    fallback_models:
      - "anthropic/claude-sonnet-4-20250514"
      - "openrouter/deepseek/deepseek-r1"
```

When `gpt-4o` is unavailable, the system automatically tries the fallback models in order.

### Parallel Probing

Enable parallel provider probing for fastest response:

```yaml
agent:
  parallel_probe: true
  probe_timeout: 5s
```

This sends requests to multiple providers simultaneously and uses the first response.

### Per-Provider Settings

```yaml
model_list:
  - model_name: "gpt4o"
    model: "openai/gpt-4o"
    api_key: "sk-your-key"
    
    # Generation settings
    max_tokens: 4096
    temperature: 0.7
    top_p: 0.9
    
    # Timeout
    timeout: 120s
    
    # Custom headers
    headers:
      "X-Custom-Header": "value"
```

### Organization API Keys

For OpenAI:
```yaml-Level
providers:
  openai:
    api_key: "sk-your-key"
    organization: "org-your-org-id"
```

### Proxy Configuration

```yaml
providers:
  openai:
    api_key: "sk-your-key"
    api_base: "https://api.openai.com/v1"
    http_proxy: "http://proxy:8080"
    https_proxy: "http://proxy:8080"
```

## Cost Management

### Token Tracking

Enable usage tracking:

```yaml
stats:
  enabled: true
```

This tracks:
- Tokens used per request
- Cost per provider
- Daily/monthly totals

### Context Window Management

Control token usage:

```yaml
agent:
  max_context_tokens: 8000
  max_response_tokens: 4096
```

When context approaches the limit, older messages are automatically pruned.

## Best Practices

### 1. Use Environment Variables

Store API keys in environment variables rather than config files:

```yaml
# config.yaml
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
```

```bash
# shell
export OPENAI_API_KEY="sk-your-key"
```

### 2. Configure Multiple Providers

Always have at least one backup provider:

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    
  - model_name: "backup"
    model: "anthropic/claude-sonnet-4-20250514"
```

### 3. Choose Appropriate Models

| Use Case | Recommended Model |
|----------|-------------------|
| General conversation | gpt-4o, claude-sonnet-4 |
| Fast/lightweight | gpt-4o-mini, claude-3-haiku |
| Reasoning/complex | deepseek-r1, claude-opus-4 |
| Code generation | gpt-4o, claude-sonnet-4 |
| Cost optimization | groq/llama-3.3-70b |
| Local/offline | ollama/llama3.2 |

### 4. Set Appropriate Limits

```yaml
agent:
  temperature: 0.7    # 0-2, lower = more focused
  max_tokens: 4096    # Adjust based on expected response
```

## Troubleshooting

### "model not found" Error

- Verify the model identifier is correct
- Check the provider prefix matches the API
- For OpenRouter, use the full model path: `openrouter/deepseek/deepseek-r1`

### Authentication Failures

- Confirm API key is correct and active
- Check environment variables aren't conflicting
- Verify the API key has sufficient credits/quota

### Rate Limiting

- Implement exponential backoff
- Use fallback providers
- Reduce request frequency

### Timeout Errors

- Increase timeout in configuration
- Check network connectivity
- Consider using regional API endpoints

## API Key Environment Variables

| Provider | Environment Variable |
|----------|---------------------|
| OpenAI | `OPENAI_API_KEY` |
| Anthropic | `ANTHROPIC_API_KEY` |
| Google Gemini | `GEMINI_API_KEY` |
| OpenRouter | `OPENROUTER_API_KEY` |
| Groq | `GROQ_API_KEY` |
| DeepSeek | `DEEPSEEK_API_KEY` |
| Cerebras | `CEREBRAS_API_KEY` |
| NVIDIA | `NVIDIA_API_KEY` |
| Zhipu | `ZHIPU_API_KEY` |
| Moonshot | `MOONSHOT_API_KEY` |
| Mistral | `MISTRAL_API_KEY` |
| Qwen | `QWEN_API_KEY` |
