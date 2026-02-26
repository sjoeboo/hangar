# Installation

## Requirements

| Tool | Version | Required? |
|------|---------|-----------|
| Go | 1.24+ | Required |
| tmux | any | Required |
| git | any | Required |
| `gh` CLI | any | Optional — PR status, PR overview, diff view |
| lazygit | any | Optional — `G` key integration |

## Install from Source

```bash
git clone git@ghe.spotify.net:mnicholson/hangar
cd hangar
./install.sh
```

The installer will:
1. Check for Go (1.24+) and tmux
2. Build from source with version embedding
3. Install to `~/.local/bin/hangar`
4. Configure tmux (mouse, clipboard, 256-color, 50k history)
5. Optionally install Claude Code lifecycle hooks

## Install Options

```bash
./install.sh --dir /usr/local/bin       # custom install dir
./install.sh --skip-tmux-config         # skip tmux setup
./install.sh --skip-hooks               # skip Claude hooks prompt
./install.sh --non-interactive          # CI / unattended install
```

## Post-Install Setup

### Register a project

```bash
hangar project add myrepo ~/code/myrepo        # uses default branch
hangar project add myrepo ~/code/myrepo main   # explicit base branch
```

### Install Claude Code hooks

Hooks enable instant status detection (vs. periodic polling):

```bash
hangar hooks install
hangar hooks status   # verify
```

This writes a hook command to `~/.claude/settings.json`. The hook sends lifecycle events (SessionStart, Stop, UserPromptSubmit, etc.) to Hangar via `~/.hangar/hooks/{id}.json`.

### Launch

```bash
hangar
```
