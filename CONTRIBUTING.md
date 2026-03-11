# Contributing to Wunderpus

Thank you for your interest in contributing to Wunderpus. This document provides guidelines and best practices for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Making Changes](#making-changes)
- [Submitting Changes](#submitting-changes)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Commit Messages](#commit-messages)

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

## Getting Started

1. **Fork the Repository**: Click the "Fork" button on the GitHub page
2. **Clone Your Fork**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/wunderpus.git
   cd wunderpus
   ```
3. **Add Upstream Remote**:
   ```bash
   git remote add upstream https://github.com/wunderpus/wunderpus.git
   ```
4. **Create a Branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Environment

### Prerequisites

- Go 1.25 or later
- Make
- Git

### Setup

1. **Install Dependencies**:
   ```bash
   go mod download
   ```

2. **Verify Build**:
   ```bash
   make build
   ```

3. **Run Tests**:
   ```bash
   make test
   ```

4. **Run Linters**:
   ```bash
   make lint
   ```

### Running Locally

```bash
# Copy example configuration
cp config.example.yaml config.yaml

# Edit config.yaml with your API keys

# Run the application
go run cmd/wunderpus/main.go

# Or build and run
make build
./bin/wunderpus
```

## Making Changes

### Code Organization

The project follows a standard Go project layout:

```
internal/
  agent/       # Agent logic and session management
  app/         # Application bootstrap and wiring
  channel/     # Communication channel implementations
  config/      # Configuration loading and validation
  health/      # Health check server
  heartbeat/   # Periodic task scheduler
  memory/      # Session storage and context management
  provider/    # LLM provider adapters
  security/    # Encryption, audit logging, rate limiting
  skills/      # Skills system implementation
  subagent/    # Sub-agent spawning and management
  tool/        # Tool execution framework
  tui/         # Terminal user interface
  types/       # Shared type definitions
```

### Adding New Providers

To add a new LLM provider:

1. Create a new file in `internal/provider/` (e.g., `provider_new.go`)
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       Name() string
       Complete(ctx context.Context, req *Request) (*Response, error)
       Embed(ctx context.Context, text string) ([]float64, error)
   }
   ```
3. Register the provider in `internal/provider/providers.go`
4. Add configuration schema in `config.go`
5. Add documentation in `docs/providers.md`

### Adding New Channels

To add a new communication channel:

1. Create a new directory in `internal/channel/`
2. Implement the `Channel` interface:
   ```go
   type Channel interface {
       Name() string
       Start(ctx context.Context) error
       Stop() error
       Send(msg *Message) error
   }
   ```
3. Register the channel in the bootstrap process
4. Add configuration schema

### Adding New Skills

Skills are markdown-based extensions. To create a new skill:

1. Create a directory in `skills/your-skill-name/`
2. Add `SKILL.md` with the manifest header:
   ```markdown
   ---
   name: skill-name
   description: "Description of the skill"
   ---
   ```
3. Document usage and examples in the skill file

## Submitting Changes

### Pull Request Process

1. **Keep Your Branch Updated**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run All Checks**:
   ```bash
   make test
   make lint
   make build
   ```

3. **Write Meaningful Commit Messages**: See [Commit Messages](#commit-messages) guidelines

4. **Push Your Branch**:
   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request**: Fill out the PR template completely

6. **Respond to Feedback**: Address review comments promptly

### PR Title Convention

Use conventional commit format for PR titles:

- `feat: Add support for new LLM provider`
- `fix: Resolve rate limiting issue`
- `docs: Update installation guide`
- `refactor: Simplify provider interface`
- `test: Add provider fallback tests`

### What Happens Next

- Maintainers will review your PR
- You may be asked to make changes
- Once approved, your PR will be merged

## Coding Standards

### General Guidelines

- Write clear, readable, and maintainable code
- Follow Go idioms and best practices
- Keep functions focused and small (aim for < 50 lines)
- Use meaningful variable and function names
- Add comments for complex logic (explain *why*, not *what*)

### Code Style

This project uses:
- **gofmt** for code formatting
- **goimports** for import organization
- **gci** for import ordering
- **golangci-lint** for linting

Run formatting:
```bash
make format
make lint
```

### Complexity Limits

- Maximum cyclomatic complexity: 20
- Maximum function length: 100 lines
- Maximum file length: 500 lines
- Maximum parameters: 5

### Error Handling

- Always handle errors explicitly
- Return meaningful error messages
- Use the `errors` package for custom errors
- Avoid swallowing errors with `_`

### Logging

- Use structured logging with `zerolog`
- Log levels: debug, info, warn, error
- Include relevant context in log entries
- Never log sensitive information (API keys, tokens)

## Testing

### Writing Tests

- Write tests for all new functionality
- Follow the naming convention: `*_test.go`
- Use table-driven tests where appropriate
- Test both success and failure paths

```go
func TestProviderComplete(t *testing.T) {
    tests := []struct {
        name    string
        req     *Request
        want    *Response
        wantErr bool
    }{
        {
            name: "successful request",
            req: &Request{Prompt: "Hello"},
            want: &Response{Text: "Hi there"},
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run specific package
go test ./internal/provider/...

# Run with coverage
go test -cover ./...
```

### Test Coverage

- Aim for > 60% test coverage
- Focus on critical paths and edge cases
- Include integration tests for channel/provider interactions

## Documentation

### Code Documentation

- Add godoc comments for all exported functions and types
- Include usage examples where helpful
- Keep documentation in sync with code changes

```go
// Complete sends a completion request to the LLM provider.
//
// The ctx parameter controls request timeout and cancellation.
// Returns a Response containing the generated text or an error.
func (p *Provider) Complete(ctx context.Context, req *Request) (*Response, error) {
    // implementation
}
```

### Project Documentation

- Update README.md for user-facing changes
- Add or update docs/ files for technical details
- Include configuration examples for new features

## Commit Messages

### Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Build process, dependencies, tooling

### Examples

```
feat(provider): Add DeepSeek R1 model support

Implement support for DeepSeek's R1 reasoning model with
enhanced chain-of-thought capabilities.

Fixes #123
```

```
fix(security): Prevent SSRF via localhost binding

Add validation to block requests to localhost and private
IP ranges in HTTP tool execution.

Closes #456
```

### Rules

- Use imperative mood: "Add feature" not "Added feature"
- Subject line: max 50 characters
- Body: wrap at 72 characters
- Reference issues: "Fixes #123" or "Closes #456"

## Recognition

Contributors will be recognized in:
- README.md contributors section
- CHANGELOG.md
- Release notes for significant contributions

## Questions?

- Open an issue for bugs or feature requests
- Use GitHub Discussions for questions
- Join our Discord for real-time help
