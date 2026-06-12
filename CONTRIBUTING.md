# Contributing to Musketeers

Thank you for your interest in contributing to Musketeers! This document provides guidelines and information for contributors.

## Code of Conduct

Please be respectful and constructive in all interactions. We are building a welcoming community.

## How to Contribute

### Reporting Bugs

1. Check if the bug is already reported in [Issues](https://github.com/MortalArena/Musketeers/issues)
2. If not, create a new issue with:
   - Clear description
   - Steps to reproduce
   - Expected vs actual behavior
   - Environment details (Go version, OS)

### Suggesting Features

1. Open a [Discussion](https://github.com/MortalArena/Musketeers/discussions) first
2. Describe the use case and proposed solution
3. Wait for community feedback before implementing

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature` 
3. Make your changes
4. Write tests for new code
5. Ensure all tests pass: `go test ./...` 
6. Run linters: `golangci-lint run` 
7. Commit with clear messages
8. Push to your fork
9. Open a Pull Request

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/Musketeers.git
cd Musketeers

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run tests
go test ./...

# Build
make build
```

## Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Use `goimports` for import organization
- Maximum line length: 100 characters
- Use meaningful variable names
- Document all public APIs with godoc comments

## Testing Requirements

- Unit tests for all new functions/methods
- Integration tests for new features
- Target: ≥70% code coverage
- Run with race detection: `go test -race ./...` 

## Commit Message Format

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Formatting
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance

Examples:
```
feat(workflow): add conditional edge support
fix(runtime): prevent goroutine leak in scheduler
docs(readme): update installation instructions
```

## Review Process

1. All PRs require at least one review
2. CI must pass (tests, linting, build)
3. Address all review comments
4. Squash commits before merging

## Questions?

Open a [Discussion](https://github.com/MortalArena/Musketeers/discussions) or join our [Discord](https://discord.gg/musketeers).
