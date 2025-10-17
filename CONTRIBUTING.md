# Contributing to ircd

First off, thank you for considering contributing to this IRC server project! 🎉

## Code of Conduct

Be respectful, inclusive, and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues. When you create a bug report, include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples** (commands, configuration, etc.)
- **Describe the behavior you observed** and what you expected
- **Include logs** if relevant
- **Specify your environment** (Go version, OS, etc.)

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:

- **Use a clear and descriptive title**
- **Provide a detailed description** of the suggested enhancement
- **Explain why this enhancement would be useful**
- **List any similar features** in other IRC servers if applicable

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Follow the project structure** and coding conventions
3. **Add tests** for any new functionality
4. **Ensure tests pass**: `go test ./...`
5. **Update documentation** as needed
6. **Write clear commit messages**

## Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/ircd.git
cd ircd

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run the server
./bin/ircd
```

## Coding Conventions

### Go Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` to format code
- Run `go vet` before committing
- Keep functions focused and small
- Add comments for exported functions

### Project Structure
```
internal/           # Private application code
├── server/        # Server core logic
├── client/        # Client management
├── channel/       # Channel operations
├── commands/      # IRC command handlers
├── parser/        # Protocol parsing
├── security/      # Security features
└── logger/        # Logging

cmd/               # Application entry points
docs/              # Documentation
tests/             # Integration tests
```

### Naming Conventions
- **Files**: lowercase with underscores (e.g., `rate_limiter.go`)
- **Packages**: short, lowercase, single word
- **Types**: PascalCase (e.g., `RateLimiter`)
- **Functions**: camelCase for private, PascalCase for exported
- **Constants**: PascalCase (e.g., `MaxNickLength`)

### Testing
- Write unit tests for all new functionality
- Aim for >70% code coverage
- Use table-driven tests when appropriate
- Mock external dependencies

Example test:
```go
func TestHandleCommand(t *testing.T) {
    tests := []struct {
        name string
        input string
        expected string
    }{
        {"valid nick", "NICK alice", "alice"},
        {"empty nick", "NICK", ""},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### IRC Protocol
- Follow RFC 1459 specifications
- Use proper numeric reply codes
- Include error handling for all edge cases
- Document any deviations from the RFC

### Commit Messages
```
type(scope): subject

body (optional)

footer (optional)
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions/changes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes

Examples:
```
feat(commands): add WHO command implementation

Implements the WHO command per RFC 1459 specification.
Supports channel queries and wildcard patterns.

Closes #42

---

fix(parser): handle empty parameter lists

Fixes parsing error when messages have no parameters.

Fixes #38

---

docs(readme): update installation instructions
```

## Pull Request Process

1. **Update documentation** for any changed functionality
2. **Add tests** that prove your fix/feature works
3. **Run the full test suite** and ensure all tests pass
4. **Update CHANGELOG.md** with your changes
5. **Request review** from maintainers

### PR Checklist
- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Commit messages are clear
- [ ] No merge conflicts

## Project Phases

We organize development into phases. Current focus:

- ✅ **Phase 5**: Testing & Deployment (Complete)
- 🚧 **Phase 6**: Advanced Features (In Progress)

See [docs/PHASE6_ADVANCED_FEATURES.md](docs/PHASE6_ADVANCED_FEATURES.md) for details.

## Areas for Contribution

### High Priority
- WebSocket support for browser clients
- Channel keys (+k mode)
- Voice mode (+v)
- OPER command with authentication
- Additional IRC commands (AWAY, USERHOST, ISON)

### Medium Priority
- Improve test coverage (client, server packages)
- Performance optimization
- Load testing and benchmarking
- Additional channel modes (+l limit, +e exception, +I invite exception)

### Nice to Have
- REST API for statistics
- Web-based admin dashboard
- Server-to-server federation
- SASL authentication
- Plugin/extension system

## Questions?

Feel free to open an issue labeled "question" or reach out to the maintainers.

## Thank You!

Your contributions make this project better for everyone. Thank you for your time and effort! 🙏
