# Configuration

## Config File

Config lives at `~/.hangar/config.toml`. Projects are stored in `~/.hangar/projects.toml`.

```toml
[notifications]
enabled = true
minimal = true    # Show ⚡ ● N │ ◐ N format (default)

[worktree]
auto_update_base = true
default_location = "subdirectory"   # repo/.worktrees/<branch>

[claude]
hooks_enabled = true                # instant status via lifecycle hooks
allow_dangerous_mode = false        # --allow-dangerously-skip-permissions

[tmux]
mouse_mode = true
inject_status_line = true           # set false to keep your own status bar

[api]
port = 47437                        # HTTP port for API + web UI
bind_address = "0.0.0.0"           # "127.0.0.1" for localhost-only, "0.0.0.0" for all interfaces (Tailscale)
```

## Key Settings

### `[worktree]`

| Key | Default | Description |
|-----|---------|-------------|
| `auto_update_base` | `true` | Pull base branch before creating a new worktree |
| `default_location` | `"subdirectory"` | Places worktrees at `repo/.worktrees/<branch>` |

### `[claude]`

| Key | Default | Description |
|-----|---------|-------------|
| `hooks_enabled` | `true` | Use lifecycle hooks for instant status detection |
| `allow_dangerous_mode` | `false` | Pass `--allow-dangerously-skip-permissions` to Claude |

### `[tmux]`

| Key | Default | Description |
|-----|---------|-------------|
| `mouse_mode` | `true` | Enable tmux mouse support |
| `inject_status_line` | `true` | Apply oasis_lagoon_dark status bar theme |

Set `inject_status_line = false` to keep your own tmux status bar configuration.

### `[api]`

| Key | Default | Description |
|-----|---------|-------------|
| `port` | `47437` | HTTP port for the REST API and embedded web UI |
| `bind_address` | `"0.0.0.0"` | Network interface to bind. Use `"127.0.0.1"` for localhost-only access, `"0.0.0.0"` to allow connections from other devices (e.g., over Tailscale) |

The API server starts automatically with the TUI and can also be run standalone with `hangar web start`. The web UI is served at `/ui/` on the same port.

### `[notifications]`

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Show session status in tmux status bar |
| `minimal` | `true` | Show compact `⚡ ● N │ ◐ N` format |

## Projects File

Projects are stored in `~/.hangar/projects.toml`. Managed via CLI:

```bash
hangar project add myrepo ~/code/myrepo        # add project
hangar project add myrepo ~/code/myrepo main   # with explicit base branch
hangar project list                            # list all
hangar project remove myrepo                  # remove
```

## State Database

Todos are stored in `~/.hangar/state.db` (SQLite). This file is managed automatically — no manual editing required.

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `HANGAR_INSTANCE_ID` | Current session ID (set by Hangar) |
| `HANGAR_TITLE` | Current session title (set by Hangar) |
| `HANGAR_TOOL` | Tool in use — `claude`, `shell`, etc. (set by Hangar) |
| `HANGAR_PROFILE` | Active profile (set by Hangar) |
