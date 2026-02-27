# Development

## Build & Test

```bash
go build ./...                  # Build all packages
go test ./...                   # Run all tests
go test ./internal/ui/...       # UI tests only
go test ./internal/session/...  # Session tests
go run ./cmd/hangar              # Run locally without installing
make install-user               # Build + install to ~/.local/bin
```

> **Note:** Two tests in `internal/ui/` have pre-existing failures:
> - `TestNewDialog_WorktreeToggle_ViaKeyPress`
> - `TestNewDialog_TypingResetsSuggestionNavigation`

## Architecture

```
cmd/hangar/          # CLI entry point and subcommands
internal/
  session/           # Core session model, config, hooks, status detection, todos
  statedb/           # SQLite state (todos table)
  tmux/              # tmux interaction layer (status, capture, send-keys)
  ui/                # Bubble Tea TUI (home.go is the main model)
  git/               # Git worktree operations + diff fetching
  update/            # Self-update logic
  profile/           # Multi-profile support
  logging/           # Structured logging
```

Key files: `internal/ui/home.go` (main TUI model), `internal/ui/styles.go` (theme), `internal/session/instance.go` (session model).

## Release

Releases are built via [GoReleaser](https://goreleaser.com):

```bash
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser release --snapshot --clean   # Test build (no publish)
goreleaser release                       # Full release (requires GITHUB_TOKEN)
```

Release artifacts: `hangar_{version}_{os}_{arch}.tar.gz`

Homebrew formula: [sjoeboo/homebrew-tap](https://github.com/sjoeboo/homebrew-tap)
