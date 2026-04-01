# Skills Reference

Wunderpus features an extensible skills system — markdown-based capability extensions that guide agent behavior.

## Overview

Skills are loaded from three priority levels:

| Priority | Location | Description |
|---|---|---|
| 1 (Highest) | `./skills/` (workspace) | Project-specific skills |
| 2 | `~/.wunderpus/skills/` (global) | User-wide skills |
| 3 (Lowest) | `./skills/` (builtin) | Framework-default skills |

## Skill Structure

```
skills/my-skill/
└── SKILL.md          # Required: Manifest + documentation
```

### SKILL.md Format

```markdown
---
name: my-skill
description: "What this skill does"
---

# My Skill

Detailed instructions for the agent on how to use this skill.

## Usage

How to invoke and use the skill.

## Examples

Example interactions.
```

## Built-in Skills

| Skill | Description |
|---|---|
| `github` | Interact with GitHub using `gh` CLI |
| `tmux` | Automate terminal multiplexer sessions |
| `weather` | Get current weather and forecasts |
| `summarize` | Automatic content summarization |
| `social-post` | Generate social media posts |
| `skill-creator` | Generate new skills from descriptions |

## Skill Management

### List Skills

```bash
wunderpus skills list
```

### Install Skills

```bash
# From GitHub
wunderpus skills install https://github.com/user/repo

# Shorthand
wunderpus skills install user/repo

# From local path
wunderpus skills install ./path/to/skill
```

### Skill Registry

Configure remote skill registries:

```yaml
tools:
  skills:
    registries:
      clawhub:
        enabled: false
        base_url: "https://clawhub.ai"
        auth_token: ""
```

## Creating Custom Skills

### Step 1: Create Directory

```bash
mkdir -p skills/my-awesome-skill
```

### Step 2: Create SKILL.md

```markdown
---
name: awesome
description: "Does awesome things"
---

# Awesome Skill

This skill enables the agent to perform awesome tasks.

## Usage

When the user asks to do something awesome:
1. Understand the request
2. Use available tools to accomplish it
3. Report results clearly
```

### Step 3: Test

```bash
wunderpus skills list  # Verify skill loads
wunderpus agent -m "Use the awesome skill to do X"
```

## Configuration

```yaml
tools:
  skills:
    enabled: true
    global_skills_path: "~/.wunderpus/skills"
    builtin_skills_path: "./skills"
```

## Skill Loading Flow

```
Agent starts
    │
    ▼
Scan skill directories (workspace > global > builtin)
    │
    ▼
Parse SKILL.md manifests
    │
    ▼
Register metadata (name, description)
    │
    ▼
Inject into system prompt when relevant
```
