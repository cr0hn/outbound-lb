# Contributing to Outbound LB

First off, thank you for considering contributing to Outbound LB! It's people like you that make Outbound LB such a great tool.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Guidelines](#coding-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)
- [Issue Guidelines](#issue-guidelines)

## Code of Conduct

This project and everyone participating in it is governed by our commitment to providing a welcoming and inclusive environment. By participating, you are expected to uphold this standard. Please be respectful and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/outbound-lb.git
   cd outbound-lb
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/cr0hn/outbound-lb.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or later
- Make
- Docker (optional, for containerized testing)
- golangci-lint (for linting)

### Setup

```bash
# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Verify setup
make test
```

### IDE Setup

For VSCode, we recommend the Go extension with these settings:

```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.formatTool": "goimports"
}
```

## Making Changes

### Branch Naming

Use descriptive branch names:

- `feature/add-socks5-support`
- `fix/race-condition-in-limiter`
- `docs/improve-readme`
- `refactor/simplify-balancer`

### Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

**Examples:**
```
feat(balancer): add weighted IP selection

fix(limiter): prevent race condition in Acquire()

docs(readme): add Kubernetes deployment example
```

## Pull Request Process

1. **Create a feature branch** from `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/your-feature
   ```

2. **Make your changes** following the [Coding Guidelines](#coding-guidelines)

3. **Run tests and linting**:
   ```bash
   make lint
   make test
   ```

4. **Commit your changes** with a clear commit message

5. **Push to your fork**:
   ```bash
   git push origin feature/your-feature
   ```

6. **Create a Pull Request** on GitHub with:
   - Clear description of the changes
   - Link to related issue(s) if applicable
   - Screenshots/examples if UI/output changes

7. **Address review feedback** promptly

### PR Checklist

Before submitting your PR, ensure:

- [ ] Tests pass locally (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code coverage hasn't decreased significantly
- [ ] Documentation is updated if needed
- [ ] Commit messages follow conventions
- [ ] PR description is clear and complete

## Coding Guidelines

### Go Style

- Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` and `goimports` for formatting
- Keep functions focused and reasonably sized
- Prefer explicit error handling over panics

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to connect: %w", err)
}

// Avoid
if err != nil {
    panic(err)
}
```

### Naming

```go
// Good
func (l *Limiter) Acquire(ip string) error

// Avoid
func (l *Limiter) AcquireConnectionSlotForIP(ip string) error
```

### Comments

- Write comments for exported functions and types
- Focus on "why" not "what"
- Keep comments up to date with code changes

```go
// Acquire attempts to acquire a connection slot for the given IP.
// It uses CAS operations to prevent race conditions.
// Returns ErrIPLimitReached if the per-IP limit is exceeded.
func (l *Limiter) Acquire(ip string) error {
```

### Concurrency

- Use `sync/atomic` for counters
- Prefer channels for coordination
- Document any goroutines spawned
- Always handle cleanup (defer, context cancellation)

## Testing

### Running Tests

```bash
# All tests
make test

# With race detector
go test -race ./...

# Specific package
go test -v ./internal/limiter/...

# Specific test
go test -v -run TestLimiter_Acquire ./internal/limiter/...
```

### Writing Tests

- Table-driven tests are preferred
- Test edge cases and error conditions
- Use meaningful test names

```go
func TestLimiter_Acquire(t *testing.T) {
    tests := []struct {
        name    string
        maxIP   int
        maxTotal int
        acquires int
        wantErr error
    }{
        {"under limit", 10, 100, 5, nil},
        {"at limit", 10, 100, 10, nil},
        {"over limit", 10, 100, 11, ErrIPLimitReached},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

### Coverage

Aim for >70% test coverage. Check coverage with:

```bash
make coverage
```

## Documentation

### Code Documentation

- All exported functions, types, and constants must have comments
- Follow [Effective Go](https://golang.org/doc/effective_go.html#commentary) guidelines

### README Updates

If your change affects:
- CLI flags or configuration
- New features
- Breaking changes
- Examples

Please update the README accordingly.

### Changelog

For notable changes, add an entry to CHANGELOG.md following the existing format.

## Issue Guidelines

### Bug Reports

Include:
- Go version (`go version`)
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs/error messages

### Feature Requests

Include:
- Use case description
- Proposed solution (if any)
- Alternative solutions considered
- Impact on existing functionality

### Questions

For questions about usage, please check the README first. If your question isn't answered there, feel free to open an issue with the "question" label.

---

Thank you for contributing!
