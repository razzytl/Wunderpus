# Provider Reference

Complete reference for all supported LLM providers.

## Supported Providers

### OpenAI

```yaml
model_list:
  - model_name: "gpt4o"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    max_tokens: 4096
```

**Models:** gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-3.5-turbo

### Anthropic

```yaml
model_list:
  - model_name: "claude"
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: "${ANTHROPIC_API_KEY}"
    max_tokens: 4096
```

**Models:** claude-sonnet-4-20250514, claude-opus-4-20250514, claude-3-5-sonnet, claude-3-haiku

### Google Gemini

```yaml
model_list:
  - model_name: "gemini"
    model: "gemini/gemini-2.0-flash"
    api_key: "${GEMINI_API_KEY}"
    max_tokens: 4096
```

**Models:** gemini-2.0-flash, gemini-2.0-flash-exp, gemini-1.5-pro, gemini-1.5-flash

### Ollama (Local)

```yaml
model_list:
  - model_name: "local"
    model: "ollama/llama3.2"
    api_base: "http://localhost:11434"
    max_tokens: 4096
```

**Models:** Any model pulled in Ollama

### OpenRouter

```yaml
model_list:
  - model_name: "deepseek"
    model: "openrouter/deepseek/deepseek-r1"
    api_key: "${OPENROUTER_API_KEY}"
    api_base: "https://openrouter.ai/api/v1"
    max_tokens: 4096
```

### Groq

```yaml
model_list:
  - model_name: "groq"
    model: "groq/llama-3.3-70b-versatile"
    api_key: "${GROQ_API_KEY}"
    api_base: "https://api.groq.com/openai/v1"
    max_tokens: 4096
```

### DeepSeek

```yaml
model_list:
  - model_name: "deepseek"
    model: "deepseek/deepseek-chat"
    api_key: "${DEEPSEEK_API_KEY}"
    api_base: "https://api.deepseek.com/v1"
    max_tokens: 4096
```

### Cerebras

```yaml
model_list:
  - model_name: "cerebras"
    model: "cerebras/llama-3.3-70b"
    api_key: "${CEREBRAS_API_KEY}"
    api_base: "https://api.cerebras.ai/v1"
    max_tokens: 4096
```

### NVIDIA NIM

```yaml
model_list:
  - model_name: "nvidia"
    model: "nvidia/llama-3.1-nemotron-70b-instruct"
    api_key: "${NVIDIA_API_KEY}"
    api_base: "https://integrate.api.nvidia.com/v1"
    max_tokens: 4096
```

### Mistral

```yaml
model_list:
  - model_name: "mistral"
    model: "mistral/mistral-large-latest"
    api_key: "${MISTRAL_API_KEY}"
    api_base: "https://api.mistral.ai/v1"
    max_tokens: 4096
```

### Additional Providers

| Provider | Model Prefix | API Base | Env Variable |
|---|---|---|---|
| Zhipu (GLM) | `zhipu/` | `open.bigmodel.cn` | `ZHIPU_API_KEY` |
| Moonshot | `moonshot/` | `api.moonshot.cn` | `MOONSHOT_API_KEY` |
| Qwen | `qwen/` | `dashscope.aliyuncs.com` | `QWEN_API_KEY` |
| Volcanic | `volcanic/` | `ark.cn-beijing.volces.com` | `VOLCANIC_API_KEY` |
| vLLM | `vllm/` | `localhost:8000/v1` | — |
| LiteLLM | `litellm/` | `localhost:4000/v1` | — |

## Free Tier Providers

| Provider | Free Tier | Rate Limit | Models |
|---|---|---|---|
| OpenRouter | Yes | 20 req/min | gemma-2-9b, llama-3.1-8b |
| Groq | Yes | 30 req/min | llama3-8b, mixtral-8x7b |
| DeepSeek | Yes | 60 req/min | deepseek-chat |

## Fallback Configuration

```yaml
model_list:
  - model_name: "primary"
    model: "openai/gpt-4o"
    api_key: "${OPENAI_API_KEY}"
    fallback_models:
      - "anthropic/claude-sonnet-4-20250514"
      - "openrouter/deepseek/deepseek-r1"
```

## Model Selection Guide

| Use Case | Recommended Model |
|---|---|
| General conversation | gpt-4o, claude-sonnet-4 |
| Fast/lightweight | gpt-4o-mini, claude-3-haiku |
| Reasoning/complex | deepseek-r1, claude-opus-4 |
| Code generation | gpt-4o, claude-sonnet-4 |
| Cost optimization | groq/llama-3.3-70b |
| Local/offline | ollama/llama3.2 |
