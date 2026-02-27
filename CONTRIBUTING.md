# Contributing to Hangar

Thank you for your interest in contributing to Hangar! This document provides guidelines and information for contributors.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/hangar.git
   cd hangar
   ```
3. **Add upstream remote**:
   ```bash
   git remote add upstream https://github.com/sjoeboo/hangar.git
   ```

## Development Setup

### Prerequisites

- Go 1.24 or later (or Go 1.21+ with automatic toolchain download)
- tmux
- Make

### Building

```bash
make build      # Build binary to ./build/hangar
make test       # Run tests
make lint       # Run linter (requires golangci-lint)
make fmt        # Format code
```

### Running Locally

```bash
make dev        # Run with auto-reload (requires 'air')
# or
make run        # Run directly
```

## Making Changes

### Branch Naming

- `feature/description` - New features
- `fix/description` - Bug fixes
- `perf/description` - Performance optimizations
- `docs/description` - Documentation changes
- `refactor/description` - Code refactoring

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add support for custom commands
fix: resolve status detection for Claude
docs: update installation instructions
refactor: simplify project management logic
```

### Code Style

- Run `make fmt` before committing
- Run `make lint` to check for issues
- Follow existing code patterns
- Add tests for new functionality

## Pull Request Process

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes** and commit them

3. **Keep your branch updated**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

4. **Push to your fork**:
   ```bash
   git push origin feature/my-feature
   ```

5. **Open a Pull Request** on GitHub

### PR Guidelines

- Provide a clear description of the changes
- Reference any related issues
- Ensure all tests pass
- Update documentation if needed

## Reporting Issues

### Bug Reports

Include:
- Hangar version (`hangar version`)
- Operating system and version
- tmux version (`tmux -V`)
- Steps to reproduce
- Expected vs actual behavior
- Any error messages or logs

### Feature Requests

- Describe the use case
- Explain why existing features don't solve it
- Provide examples if possible

## Project Structure

```
hangar/
├── cmd/hangar/         # CLI entry point
├── internal/
│   ├── ui/             # TUI components (Bubble Tea)
│   ├── session/        # Session & project management
│   └── tmux/           # tmux integration, status detection
├── .github/workflows/  # CI/CD
├── Makefile           # Build automation
└── README.md
```

## Testing

- Add tests for new functionality
- Run the full test suite: `make test`
- Tests should be deterministic and not depend on external state

### Debug Mode

Enable debug logging:
```bash
HANGAR_DEBUG=1 hangar
```

## Questions?

- **Issues:** [Open an issue](https://github.com/sjoeboo/hangar/issues) for bugs or feature requests

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
