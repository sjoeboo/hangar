# Install Script & Release Script Design

**Date:** 2026-02-27
**Status:** Approved

## Context

Hangar needs two improvements to its release tooling:

1. A `curl | bash` install script that downloads pre-built binaries from GitHub releases
2. A local release script for cutting releases with safety checks

The existing `install.sh` builds from source (requires Go + repo checkout). It will be
renamed to `install-from-source.sh`. The `.goreleaser.yml` already references `install.sh`
as the curl-pipe target.

## 1. `install.sh` — Binary Downloader

### Purpose

Enables `curl -fsSL https://github.com/sjoeboo/hangar/raw/master/install.sh | bash`

### Behavior

- Detect OS (`darwin`/`linux`) and arch (`amd64`/`arm64`)
- Fetch latest release from GitHub API (`/releases/latest`), or use `--version X.Y.Z`
- Find matching asset: `hangar_{version}_{os}_{arch}.tar.gz`
- Download → extract binary → install to `~/.local/bin` (or `--dir`)
- Offer tmux config + Claude hooks prompts (same as current source-build script)
- Flags: `--non-interactive`, `--skip-tmux-config`, `--skip-hooks`, `--version`, `--dir`

### Install Location

`~/.local/bin` — no `sudo` required.

## 2. `scripts/release.sh` — Release Automation

### Purpose

Single script a maintainer runs locally to cut a release.

### Steps

1. Verify clean git working tree
2. Verify on `master` branch
3. Read current version from latest git tag
4. Prompt for next version (with suggested patch/minor/major)
5. Validate `CHANGELOG.md` has an entry for the new version
6. Confirm with user before tagging
7. Create signed git tag `vX.Y.Z`
8. Push tag to origin (triggers CI goreleaser workflow)
9. Print GitHub releases URL

### Safety

- Refuses if working tree is dirty
- Refuses if not on master
- Refuses if CHANGELOG.md has no `## [X.Y.Z]` entry for the new version
- Confirms before pushing (destructive action)
