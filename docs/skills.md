# Skills System

Wunderpus features an extensible skills system that allows you to add new capabilities to the agent. Skills are modular packages that define agent behavior, tools, and documentation.

## Overview

Skills provide:

- **Behavior Templates**: Structured prompts that guide agent actions
- **Tool Integration**: Custom tools specific to the skill
- **Dependency Management**: Declaration of required tools or other skills
- **Version Control**: Semantic versioning for compatibility

## Built-in Skills

Wunderpus ships with several built-in skills:

### GitHub Skill

Interact with GitHub using the `gh` CLI.

**Features:**
- Issue management (create, list, view, close)
- Pull request operations (create, review, merge)
- CI/CD monitoring (view runs, logs, status)
- Repository queries via GitHub API

**Requirements:**
- [GitHub CLI](https://cli.github.com) installed

**Installation:**
```bash
# macOS
brew install gh

# Linux
# See https://github.com/cli/cli#installation
```

**Usage:**
```bash
# List open issues
gh issue list --repo owner/repo

# View PR status
gh pr checks 55 --repo owner/repo

# List recent workflow runs
gh run list --repo owner/repo --limit 10
```

### Tmux Skill

Automate terminal multiplexer sessions.

**Features:**
- Create and manage tmux sessions
- Send commands to sessions
- Window and pane management

**Requirements:**
- tmux installed

**Installation:**
```bash
# macOS
brew install tmux

# Linux
sudo apt install tmux
```

### Weather Skill

Get current weather and forecasts.

**Features:**
- Current conditions
- Multi-day forecasts
- Location-based queries

**Usage:**
```
What's the weather in Tokyo?
Weather for New York tomorrow
```

### Summarize Skill

Automatic content summarization.

**Features:**
- URL content summarization
- Text summarization
- Configurable summary length

### Skill Creator

Generate new skills from natural language descriptions.

**Usage:**
```
Create a skill that searches Stack Overflow
Make a skill for managing Redis
```

## Skill Structure

A skill is a directory containing:

```
skills/my-skill/
  SKILL.md          # Required: Manifest and documentation
  handler.go        # Optional: Custom Go handler
  tools/            # Optional: Custom tools
  scripts/          # Optional: External scripts
```

### SKILL.md Format

The manifest file uses YAML frontmatter:

```markdown
---
name: my-skill
description: "A description of what the skill does"
metadata:
  version: "1.0.0"
  author: "Your Name"
  dependencies:
    tools: ["gh", "git"]
    skills: ["github"]
---

# My Skill

## Overview

Detailed description of the skill's purpose and capabilities.

## Usage

### Basic Usage

How to use the skill in conversation.

### Examples

Example interactions.
```

## Creating Custom Skills

### Step 1: Create the Skill Directory

```bash
mkdir -p skills/my-awesome-skill
```

### Step 2: Create the Manifest

Create `SKILL.md`:

```markdown
---
name: awesome
description: "Does awesome things"
metadata:
  version: "1.0.0"
  author: "Your Name"
  dependencies:
    tools: []
    skills: []
---

# Awesome Skill

This skill enables the agent to perform awesome tasks.

## Usage

The agent can use natural language to invoke skill capabilities.

## Examples

User: "Can you do something awesome?"
Agent: [uses skill to accomplish task]
```

### Step 3: Define Behavior

Add detailed prompts to guide the agent:

```markdown
## Behavior Guidelines

When invoked, the skill should:

1. Understand the user's request
2. Break down the task into steps
3. Execute using available tools
4. Report results clearly

## Error Handling

- If a step fails, explain the error
- Suggest alternatives when possible
- Never make assumptions about user intent
```

### Step 4: Add Custom Tools (Optional)

For advanced functionality, implement custom Go tools:

```go
// skills/awesome/tools.go
package awesome

import (
    "context"
    "fmt"
)

func init() {
    // Register custom tool
    // Tool implementation
}

type AwesomeTool struct{}

func (t *AwesomeTool) Name() string {
    return "awesome_do"
}

func (t *AwesomeTool) Description() string {
    return "Does something awesome"
}

func (t *AwesomeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    // Implementation
    return "Awesome result!", nil
}
```

## Installing External Skills

### From GitHub

```bash
# Install from GitHub repository
wunderpus skills install https://github.com/user/repo

# Shorthand (assumes github.com)
wunderpus skills install user/repo
```

### From Local Path

```bash
# Install from local directory
wunderpus skills install ./path/to/skill
```

### From Registry

Configure custom skill registries in `config.yaml`:

```yaml
tools:
  skills:
    registries:
      clawhub:
        enabled: true
        base_url: "https://clawhub.ai"
        auth_token: "${CLAWHUB_TOKEN}"  # Optional
```

Install from registry:
```bash
wunderpus skills install clawhub:skill-name
```

## Skill Management

### Listing Skills

```bash
wunderpus skills list
```

Output:
```
Installed Skills (5):
- github (builtin): GitHub CLI integration
- tmux (builtin): Terminal multiplexer automation
- weather (builtin): Weather information
- summarize (builtin): Content summarization
- awesome (local): Does awesome things
```

### Skill Discovery

Skills are discovered from configured directories:

```yaml
tools:
  skills:
    # Built-in skills (shipped with wunderpus)
    builtin_skills_path: "./skills"
    
    # User skills (~/.wunderpus/skills)
    global_skills_path: "~/.wunderpus/skills"
```

### Skill Versioning

Skills use semantic versioning:

```
MAJOR.MINOR.PATCH
1.0.0      - Initial release
1.1.0      - New features
2.0.0      - Breaking changes
```

Version constraints can be specified:

```yaml
tools:
  skills:
    constraints:
      github:
        min_version: "1.0.0"
        max_version: "2.0.0"
```

## Advanced Topics

### Skill Dependencies

Skills can declare dependencies:

```yaml
metadata:
  dependencies:
    tools: ["gh", "docker"]
    skills: ["github"]
```

The system ensures dependencies are available before loading the skill.

### Skill Composition

Skills can compose other skills:

```yaml
metadata:
  compose:
    - "github"
    - "summarize"
```

### Custom Handlers

For complex integrations, implement Go handlers:

```go
package myskill

import (
    "context"
    "github.com/wunderpus/wunderpus/internal/skills"
)

type MySkill struct{}

func (s *MySkill) Name() string {
    return "my-skill"
}

func (s *MySkill) Load(ctx context.Context, loader *skills.Loader) error {
    // Custom initialization
    return nil
}

func (s *MySkill) Execute(ctx context.Context, req *skills.Request) (*skills.Response, error) {
    // Custom execution logic
    return &skills.Response{
        Content: "Result",
    }, nil
}
```

### Skill Events

Skills can respond to system events:

```yaml
metadata:
  events:
    - name: "on_message"
      handler: "handle_message"
    - name: "on_startup"
      handler: "initialize"
```

## Best Practices

### Skill Design

1. **Single Responsibility**: Each skill should have a clear purpose
2. **Clear Documentation**: Document usage, examples, and limitations
3. **Error Handling**: Provide meaningful error messages
4. **Graceful Degradation**: Work around missing dependencies

### Naming Conventions

- Use lowercase, hyphenated names: `my-skill`
- Prefix organization: `github-enterprise`
- Version appropriately: `skill-v2`

### Testing

Test skills thoroughly:

```bash
# Test skill loading
wunderpus skills list

# Test skill execution
wunderpus agent -m "Use the awesome skill to do X"
```

## Configuration Reference

```yaml
tools:
  skills:
    enabled: true
    
    # Directories
    global_skills_path: "~/.wunderpus/skills"
    builtin_skills_path: "./skills"
    
    # Registry configuration
    registries:
      clawhub:
        enabled: false
        base_url: "https://clawhub.ai"
        auth_token: ""
    
    # Version constraints
    constraints:
      skill-name:
        min_version: "1.0.0"
        max_version: "2.0.0"
```

## Troubleshooting

### Skill Not Loading

1. Check skill directory structure
2. Verify SKILL.md format
3. Ensure dependencies are available

```bash
# Debug skill loading
wunderpus skills list -v
```

### Skill Execution Fails

1. Check required tools are installed
2. Verify permissions
3. Check logs for error details

### Version Conflicts

Use version constraints to resolve conflicts:

```yaml
tools:
  skills:
    constraints:
      my-skill:
        min_version: "1.1.0"
```
