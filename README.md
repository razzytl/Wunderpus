<!-- Banner Header -->
<div align="center">
  <h1>🐙 Wunderpus</h1>
  <h3>Universal Autonomous AI Agent Framework</h3>
  <p><em>Production-grade, vendor-agnostic, and written in Go</em></p>
</div>

<br/>

<!-- Badges -->
<div align="center">

[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/doc/devel/release.html#policy)
[![License](https://img.shields.io/badge/license-MIT-blue?style=for-the-badge)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/wunderpus/wunderpus/ci.yml?style=for-the-badge&logo=github)](https://github.com/wunderpus/wunderpus/actions)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen?style=for-the-badge)](https://goreportcard.com/report/wunderpus/wunderpus)

[![Stars](https://img.shields.io/github/stars/wunderpus/wunderpus?style=for-the-badge&color=gold)](https://github.com/wunderpus/wunderpus/stargazers)
[![Forks](https://img.shields.io/github/forks/wunderpus/wunderpus?style=for-the-badge&color=orange)](https://github.com/wunderpus/wunderpus/network/members)
[![Issues](https://img.shields.io/github/issues/wunderpus/wunderpus?style=for-the-badge&color=purple)](https://github.com/wunderpus/wunderpus/issues)
[![Contributors](https://img.shields.io/github/contributors/wunderpus/wunderpus?style=for-the-badge&color=cyan)](https://github.com/wunderpus/wunderpus/graphs/contributors)

</div>

<br/>

<!-- Hero Image / Diagram -->
<div align="center">
  <img src="https://img.shields.io/badge/🚀-Autonomous_AI_Agents-black?style=flat-square" alt="Hero"/>
</div>

---

## 📖 Table of Contents

<details open>
<summary><b>Click to expand navigation</b></summary>

- [✨ Features](#-features)
- [🏗️ Architecture](#️-architecture)
- [⚡ Quick Start](#-quick-start)
- [🔌 Supported Providers](#-supported-providers)
- [🛡️ Security Model](#️-security-model)
- [📁 Project Structure](#-project-structure)
- [📚 Documentation](#-documentation)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

</details>

---

## ✨ Features

<div align="center">

| 🧠 **Intelligence** | 🔗 **Integration** | ⚙️ **Operations** |
|:---:|:---:|:---:|
| Multi-Provider LLM Routing | Multi-Channel Messaging | Tool Synthesis Engine |
| RAG-Powered Memory | 5+ Platform Connectors | World Model Knowledge Graph |
| Conversation Branching | Real-time WebSocket | Computer Vision (Playwright) |
| Self-Improvement Loop | Unified Multi-Modal Input | Cost Prediction & Budgeting |
| Structured JSON Output | Event-Driven Webhooks | Checkpoint & Resume |
| Autonomous Goal Setting | Health Monitoring | OpenTelemetry Tracing |

</div>

### 🎯 Core Capabilities Deep Dive

<details>
<summary><b>🤖 Multi-Provider LLM Intelligence</b></summary>

Connect to **15+ LLM providers** with intelligent fallback and load balancing:

- ✅ **Enterprise**: OpenAI GPT-4o, Anthropic Claude, Google Gemini
- ✅ **High-Speed**: Groq, Cerebras, NVIDIA NIM
- ✅ **Open Source**: Ollama, vLLM, DeepSeek, Qwen, Mistral
- ✅ **Aggregators**: OpenRouter (100+ models), LiteLLM proxy

</details>

<details>
<summary><b>💬 Multi-Channel Communication</b></summary>

Deploy agents across all major platforms **simultaneously**:

<div align="center">

| Platform | Status | Features |
|:--------:|:------:|:---------|
| ![Telegram](https://img.shields.io/badge/Telegram-26A5E4?style=flat&logo=telegram&logoColor=white) | ✅ Ready | Groups, Bots, Inline |
| ![Discord](https://img.shields.io/badge/Discord-5865F2?style=flat&logo=discord&logoColor=white) | ✅ Ready | Servers, DMs, Threads |
| ![Slack](https://img.shields.io/badge/Slack-4A154B?style=flat&logo=slack&logoColor=white) | ✅ Ready | Channels, Workspaces |
| ![WhatsApp](https://img.shields.io/badge/WhatsApp-25D366?style=flat&logo=whatsapp&logoColor=white) | ✅ Ready | Business API |
| ![WebSocket](https://img.shields.io/badge/WebSocket-010101?style=flat&logo=websocket&logoColor=white) | ✅ Ready | Real-time Streaming |

</div>

</details>

<details>
<summary><b>🔧 Advanced Tool System</b></summary>

**15+ built-in tools** with enterprise-grade security:

```yaml
tools:
  - filesystem: read, write, search (sandboxed)
  - shell: command execution (policy-gated)
  - http: REST API calls with auth
  - browser: Playwright automation
  - calculator: math expressions
  - code: multi-language execution
  - database: SQL operations
  - email: SMTP/IMAP integration
  - calendar: scheduling & reminders
  - search: web & local indexing
  - vision: image analysis
  - audio: transcription & synthesis
  - document: PDF/DOCX parsing
  - crypto: encryption/decryption
  - network: port scanning & discovery
```

</details>

<details>
<summary><b>🧠 Memory & RAG System</b></summary>

- **Persistent Sessions**: SQLite-backed with AES-256-GCM encryption
- **Vector Search**: Semantic similarity matching
- **SOP Retrieval**: Standard Operating Procedures auto-loading
- **Context Windows**: Intelligent summarization & compression
- **Long-Term Memory**: Cross-session knowledge retention

</details>

<details>
<summary><b>🕸️ World Model Knowledge Graph</b></summary>

Build and query a **persistent knowledge graph**:

- Entity & Relation Tracking
- Confidence Scoring
- Cypher-like Query Language
- Automatic Inference
- Temporal Reasoning

</details>

---

## 🏗️ Architecture

<div align="center">
  <img src="https://img.shields.io/badge/📐-System_Architecture-black?style=flat-square" alt="Architecture"/>
</div>

```mermaid
graph TB
    subgraph CLI["🖥️ CLI Layer"]
        A[wunderpus]
        B[agent]
        C[gateway]
        D[skills]
    end
    
    subgraph APP["⚙️ Application Bootstrap"]
        E[Config]
        F[Logging]
        G[Database]
        H[Security]
    end
    
    subgraph CORE["🧠 Core Systems"]
        I[Agent Loop]
        J[Tool Engine]
        K[Memory/RAG]
        L[World Model]
        M[Perception]
        N[Tool Synth]
    end
    
    subgraph CHANNELS["📡 Channels"]
        O[Telegram]
        P[Discord]
        Q[Slack]
        R[WhatsApp]
        S[WebSocket]
    end
    
    subgraph PROVIDERS["🤖 LLM Providers"]
        T[OpenAI]
        U[Anthropic]
        V[Gemini]
        W[Ollama]
        X[Groq]
        Y[Others...]
    end
    
    subgraph DATA["💾 Persistence"]
        Z[wunderpus.db]
        AA[audit.db]
    end
    
    CLI --> APP
    APP --> CORE
    CORE --> CHANNELS
    CORE --> PROVIDERS
    CORE --> DATA
    
    style CLI fill:#e1f5fe,stroke:#01579b,stroke-width:2px
    style APP fill:#fff3e0,stroke:#e65100,stroke-width:2px
    style CORE fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    style CHANNELS fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px
    style PROVIDERS fill:#ffebee,stroke:#b71c1c,stroke-width:2px
    style DATA fill:#eceff1,stroke:#37474f,stroke-width:2px
```

### 🔄 Data Flow

```mermaid
sequenceDiagram
    participant User
    participant Channel
    participant Agent
    participant Tools
    participant Memory
    participant Provider
    
    User->>Channel: Send Message
    Channel->>Agent: Route Input
    Agent->>Memory: Load Context (RAG)
    Agent->>Provider: Generate Response
    Provider-->>Agent: LLM Output
    Agent->>Tools: Execute Actions
    Tools-->>Agent: Results
    Agent->>Memory: Store New Knowledge
    Agent->>Channel: Stream Response
    Channel-->>User: Final Output
```

---

## ⚡ Quick Start

<div align="center">

[![Get Started](https://img.shields.io/badge/🚀-Get_Started-green?style=for-the-badge)](#quick-start)
[![Docker](https://img.shields.io/badge/🐳-Docker_Hub-blue?style=for-the-badge&logo=docker)](https://hub.docker.com/r/wunderpus/wunderpus)
[![Docs](https://img.shields.io/badge/📚-Full_Docs-orange?style=for-the-badge)](docs/guides/getting-started.md)

</div>

### 📋 Prerequisites

<div align="center">

| Requirement | Version | Install |
|:-----------:|:-------:|:-------:|
| ![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go) | 1.25+ | [golang.org](https://golang.org/dl/) |
| ![Docker](https://img.shields.io/badge/Docker-Latest-2496ED?style=flat&logo=docker) | Latest | [docker.com](https://www.docker.com/get-started) |
| ![Node](https://img.shields.io/badge/Node.js-Optional-339933?style=flat&logo=node.js) | 18+ | [nodejs.org](https://nodejs.org/) |

</div>

### 🚀 Installation

#### Option 1: Build from Source

```bash
# 📥 Clone the repository
git clone https://github.com/wunderpus/wunderpus.git
cd wunderpus

# 🔨 Build the binary
go build -o build/wunderpus ./cmd/wunderpus

# ⚙️ Configure your environment
cp config.example.yaml config.yaml
# Edit config.yaml with your API keys

# ▶️ Run interactive TUI
./build/wunderpus
```

#### Option 2: One-Shot Command

```bash
# Run a single agent command
./build/wunderpus agent -m "What can you do?"
```

#### Option 3: Gateway Mode (Background Services)

```bash
# Start all configured channels and services
./build/wunderpus gateway
```

#### Option 4: Docker Deployment

```bash
# Build the image
docker build -t wunderpus:latest .

# Run with mounted config
docker run -d \
  -p 8080:8080 \
  -p 9090:9090 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  wunderpus:latest
```

#### Option 5: Docker Compose

```bash
# Start full stack (agent + monitoring + database)
docker-compose up -d
```

---

## 🔌 Supported Providers

<div align="center">
  <img src="https://img.shields.io/badge/🔗-15+_LLM_Providers-black?style=flat-square" alt="Providers"/>
</div>

| Provider | Protocol | Popular Models | Free Tier | Latency |
|:--------:|:--------:|:--------------:|:---------:|:-------:|
| ![OpenAI](https://img.shields.io/badge/OpenAI-412991?style=flat&logo=openai&logoColor=white) | `openai` | GPT-4o, GPT-4o-mini | ❌ | ⚡⚡⚡ |
| ![Anthropic](https://img.shields.io/badge/Anthropic-191919?style=flat&logo=anthropic&logoColor=white) | `anthropic` | Claude Sonnet-4, Opus-4 | ❌ | ⚡⚡⚡ |
| ![Google](https://img.shields.io/badge/Google-4285F4?style=flat&logo=google&logoColor=white) | `gemini` | Gemini 2.0 Flash, 1.5 Pro | ✅ | ⚡⚡⚡ |
| ![Ollama](https://img.shields.io/badge/Ollama-000000?style=flat&logo=ollama&logoColor=white) | `ollama` | Any Local Model | ✅ | ⚡⚡⚡⚡⚡ |
| ![Groq](https://img.shields.io/badge/Groq-F5A623?style=flat&logo=groq&logoColor=white) | `openai` | Llama-3.3-70B, Mixtral | ✅ | ⚡⚡⚡⚡⚡ |
| ![DeepSeek](https://img.shields.io/badge/DeepSeek-3B82F6?style=flat) | `openai` | DeepSeek Chat, R1 | ✅ | ⚡⚡⚡⚡ |
| ![OpenRouter](https://img.shields.io/badge/OpenRouter-7C3AED?style=flat) | `openai` | 100+ Models | ✅ | ⚡⚡⚡ |
| ![NVIDIA](https://img.shields.io/badge/NVIDIA-76B900?style=flat&logo=nvidia&logoColor=white) | `openai` | Nemotron-70B | ❌ | ⚡⚡⚡⚡ |
| ![Mistral](https://img.shields.io/badge/Mistral-FF7000?style=flat) | `openai` | Mistral Large | ❌ | ⚡⚡⚡⚡ |
| ![Cerebras](https://img.shields.io/badge/Cerebras-E1171E?style=flat) | `openai` | Llama-3.3-70B | ❌ | ⚡⚡⚡⚡⚡ |

> 💡 **Pro Tip**: Wunderpus automatically routes to the best available provider based on latency, cost, and availability!

---

## 🛡️ Security Model

<div align="center">
  <img src="https://img.shields.io/badge/🔒-Defense_in_Depth-black?style=flat-square" alt="Security"/>
</div>

Wunderpus implements **five layers of defense** to ensure safe autonomous operation:

<div align="center">

```
┌─────────────────────────────────────────────────────────────┐
│  🔍 INPUT LAYER: Unicode Normalization + Injection Detection │
├─────────────────────────────────────────────────────────────┤
│  📦 EXECUTION LAYER: Workspace Sandbox + Command Prevention  │
├─────────────────────────────────────────────────────────────┤
│  🌐 NETWORK LAYER: SSRF Blocklist + Private IP Filtering     │
├─────────────────────────────────────────────────────────────┤
│  ✅ APPROVAL LAYER: Policy-Based Tool Classification         │
├─────────────────────────────────────────────────────────────┤
│  🔐 STORAGE LAYER: AES-256-GCM + Hash-Chained Audit Log      │
└─────────────────────────────────────────────────────────────┘
```

</div>

### Policy Classifications

| Classification | Auto-Execute | Notification | Use Case |
|:--------------:|:------------:|:------------:|:---------|
| `AutoExecute` | ✅ Yes | ❌ No | Safe read-only ops |
| `NotifyOnly` | ✅ Yes | ✅ Yes | Low-risk actions |
| `RequiresApproval` | ❌ No | ✅ Yes | Sensitive operations |
| `Blocked` | ❌ No | ✅ Yes | Dangerous commands |

---

## 📁 Project Structure

<details open>
<summary><b>📂 Click to explore the codebase</b></summary>

```
wunderpus/
│
├── 📱 cmd/wunderpus/          # CLI entry point (Cobra commands)
│
├── 🏛️ internal/               # Core application logic
│   ├── app/                   # Bootstrap & dependency injection
│   ├── agent/                 # Agent loop, context, branching
│   ├── agents/                # Sub-agent orchestration
│   ├── provider/              # LLM provider adapters
│   ├── channel/               # Messaging integrations
│   ├── tool/                  # Tool system & implementations
│   ├── toolsynth/             # Self-improvement engine
│   ├── skills/                # Extensibility registry
│   ├── memory/                # RAG, sessions, vector search
│   ├── worldmodel/            # Knowledge graph
│   ├── perception/            # Computer vision & multi-modal
│   ├── security/              # Sanitization & encryption
│   ├── audit/                 # Tamper-evident logging
│   ├── health/                # Health checks & monitoring
│   ├── telemetry/             # OpenTelemetry integration
│   └── tui/                   # Terminal UI (Bubbletea)
│
├── 🔌 contrib/channels/       # Community channel plugins
├── 🌐 web/                    # HTTP + WebSocket server
├── 🎯 skills/                 # Built-in skill definitions
├── 📚 docs/                   # Comprehensive documentation
│
├── ⚙️ config.example.yaml     # Configuration template
├── 🆓 free_tiers.yaml         # Free-tier provider configs
├── 💓 HEARTBEAT.md            # Scheduled task definitions
│
├── 🐳 Dockerfile              # Production container
├── 🐙 docker-compose.yml      # Multi-service orchestration
├── 🛠️ Makefile                # Build automation
└── 📦 go.mod                  # Module dependencies
```

</details>

---

## 📚 Documentation

<div align="center">

| 📘 Guide | 📖 Reference | 🔧 Operations |
|:--------:|:------------:|:-------------:|
| [Getting Started](docs/guides/getting-started.md) | [Providers](docs/reference/providers.md) | [Deployment](docs/operations/deployment.md) |
| [Security Best Practices](docs/guides/security.md) | [Channels](docs/reference/channels.md) | [Monitoring](docs/operations/monitoring.md) |
| [Building Agents](docs/guides/building-agents.md) | [Tools](docs/reference/tools.md) | [Troubleshooting](docs/operations/troubleshooting.md) |
| [Creating Skills](docs/guides/creating-skills.md) | [Skills](docs/reference/skills.md) | [Performance Tuning](docs/operations/performance.md) |
| [Advanced Patterns](docs/guides/advanced-patterns.md) | [CLI Reference](docs/reference/cli.md) | [Scaling Strategies](docs/operations/scaling.md) |

</div>

### 🎓 Learning Path

```mermaid
graph LR
    A[Installation] --> B[Basic Agent]
    B --> C[Tool Usage]
    C --> D[Memory & RAG]
    D --> E[Multi-Channel]
    E --> F[Custom Skills]
    F --> G[Production Deploy]
    
    style A fill:#bbdefb,stroke:#1565c0
    style B fill:#bbdefb,stroke:#1565c0
    style C fill:#e1f5fe,stroke:#0277bd
    style D fill:#e1f5fe,stroke:#0277bd
    style E fill:#e8f5e9,stroke:#2e7d32
    style F fill:#e8f5e9,stroke:#2e7d32
    style G fill:#c8e6c9,stroke:#1b5e20
```

---

## 🤝 Contributing

<div align="center">

[![Contributors](https://img.shields.io/github/contributors/wunderpus/wunderpus?style=for-the-badge)](https://github.com/wunderpus/wunderpus/graphs/contributors)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen?style=for-the-badge)](CONTRIBUTING.md)

</div>

We welcome contributions! Here's how you can help:

1. 🍴 **Fork** the repository
2. 🌿 **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. 💾 **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. 📤 **Push** to the branch (`git push origin feature/amazing-feature`)
5. 🔄 **Open** a Pull Request

Please read our [Contributing Guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) first.

---

## 📄 License

<div align="center">

[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

**MIT License** — See [LICENSE](LICENSE) for details.

<br/>

Made with ❤️ by the Wunderpus Team

[Website](https://wunderpus.github.io) · [Twitter](https://twitter.com/wunderpus) · [Discord](https://discord.gg/wunderpus)

</div>

---

<div align="center">

⭐ **Star this repo if you find it useful!** ⭐

[![Star History](https://api.star-history.com/svg?repos=wunderpus/wunderpus&type=Date)](https://star-history.com/#wunderpus/wunderpus&Date)

</div>
