# CLI Reference

Complete reference for all Wunderpus CLI commands.

## Root Command

```bash
wunderpus [flags]
```

Launches the interactive Terminal UI (TUI).

### Flags

| Flag | Short | Description |
|---|---|---|
| `--config` | | Path to config file (or set `WUNDERPUS_CONFIG`) |
| `--verbose` | `-v` | Enable debug logging |
| `--ui` | | Start the web UI server on port 8080 |

## Subcommands

### agent

Run the Wunderpus agent.

```bash
wunderpus agent [flags]
```

| Flag | Short | Description |
|---|---|---|
| `--message` | `-m` | One-shot message to agent |

**Examples:**

```bash
# Interactive mode
wunderpus agent

# One-shot
wunderpus agent -m "What's the weather in Tokyo?"

# With verbose logging
wunderpus agent -v -m "Explain quantum computing"
```

### gateway

Start background services (channels + heartbeat).

```bash
wunderpus gateway
```

Runs until interrupted (Ctrl+C). Starts all enabled channels and the heartbeat scheduler.

### status

Show system status.

```bash
wunderpus status
```

**Output:**
```
Wunderpus Status
Workspace: /path/to/workspace
Providers: [openai, anthropic]
DefaultProvider: openai
Uptime: 2h30m15s
```

### onboard

Interactive configuration wizard.

```bash
wunderpus onboard
```

Guides through:
1. Provider selection and API key entry
2. Workspace configuration
3. Channel setup
4. Security settings

### cron

Manage periodic (heartbeat) tasks.

```bash
wunderpus cron [subcommand]
```

#### cron list

List all scheduled tasks.

```bash
wunderpus cron list
```

#### cron add

Add a new periodic task.

```bash
wunderpus cron add "Review open issues in my GitHub repos"
```

Adds task to `HEARTBEAT.md` in the workspace directory.

### skills

Manage agent skills.

```bash
wunderpus skills [subcommand]
```

#### skills list

List installed skills.

```bash
wunderpus skills list
```

#### skills install

Install a skill from GitHub or local path.

```bash
# From GitHub
wunderpus skills install https://github.com/user/repo

# Shorthand
wunderpus skills install user/repo

# From local path
wunderpus skills install ./path/to/skill
```

### auth

Manage authentication.

```bash
wunderpus auth [subcommand]
```

#### auth status

Show authentication status.

```bash
wunderpus auth status
```

**Output:**
```
Authentication Status:
- openai: Authenticated
- anthropic: Authenticated
```

#### auth login

Login to a provider.

```bash
wunderpus auth login openai
```

## TUI Keybindings

When running the interactive TUI:

| Key | Action |
|---|---|
| `Enter` | Send message |
| `Ctrl+C` | Exit |
| `Tab` | Cycle providers |
| `Ctrl+P` | Command palette |
| `Ctrl+L` | Clear screen |
| `Ctrl+S` | Switch provider |
| `Up/Down` | Input history |

## TUI Commands

Type these in the TUI input:

| Command | Description |
|---|---|
| `/provider <name>` | Switch LLM provider |
| `/orchestrate <task>` | Multi-agent task decomposition |
| `/clear` | Clear conversation |
| `/help` | Show available commands |

## Environment Variables

| Variable | Purpose |
|---|---|
| `WUNDERPUS_CONFIG` | Config file path (overrides `--config`) |
| `WUNDERPUS_HOME` | Base directory for workspace (default: `~/.wunderpus`) |
| `WORKSPACE_DIR` | Workspace directory override |
