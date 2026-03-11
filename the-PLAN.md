# Wunderpus - Project Plan

*Last Updated: 2026-03-10*

---

## Completed ✅

### Phase 1: Security
- Encryption salt fix
- Shell regex patterns
- Rate limiter cleanup

### Phase 2: Providers & Channels
- 10+ LLM providers (OpenAI, Anthropic, Ollama, Gemini, OpenRouter, Groq, Zhipu, DeepSeek, Moonshot, Cerebras, NVIDIA, LiteLLM, vLLM)
- 4+ messaging channels (QQ, WeCom, DingTalk, OneBot)
- 3 search providers (Brave, Tavily, DuckDuckGo)

### Phase 3: Tools
- MCP client & server implementation
- Message, cron, edit tools
- Heartbeat with cron expression support ("at 9am daily")
- Natural language → cron conversion

### Phase 4: Architecture
- SQLite session persistence
- Provider fallback with cooldown
- Parallel provider probing option
- Async tool execution

### Phase 5: Skills
- GitHub, Tmux, Weather, Summarize, Skill Creator skills
- Skill versioning

### Phase 6: Build
- Multi-platform Makefile
- Docker optimization
- WhatsApp native build tag

### Phase 7: Quality Tools
- Cyclomatic complexity threshold: 20
- Formatters: gci, gofmt, goimports
- Go generate directives
- Generate target in Makefile
- Generate step in CI

### Phase 8: Documentation
- Godoc comments
- ADRs
- Config documentation
- Troubleshooting guide
- All CLI commands

---

## Pending

### MCP Testing
- [ ] Test MCP implementation with common MCP servers

---

## Test & Validation (For Later Session)

### Test Coverage
- [ ] Fix broken test packages (provider, memory, security, tool, subagent)
- [ ] Expand working test coverage to 50+ tests

### Benchmark Tests
- [ ] Complete benchmark: agent message handling
- [ ] Complete benchmark: provider completion  
- [ ] Complete benchmark: tool execution

### Test Execution
- [ ] Run full test suite: `go test ./internal/...`

---

*Core functionality complete. Next phase: test validation.*
