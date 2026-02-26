# Inline Git Diff Viewer — Design Document

> Generated: 2026-02-25

## Summary

Add an inline git diff viewer to Hangar. A diffstat summary appears in the preview pane for any session in a git repo. Pressing `D` opens a full-screen scrollable overlay showing the full diff, rendered in oasis_lagoon_dark colors. Pressing `e` in the overlay opens the file under the cursor in `$EDITOR` at the first changed line.

## Acceptance Criteria

- [ ] Preview pane shows a single-line diffstat ("3 files, +47 -12") for sessions in a git repo
- [ ] `D` key opens/closes a full-screen diff overlay
- [ ] Overlay is scrollable (j/k, ↑/↓, PgUp/PgDn)
- [ ] `e` opens file under cursor in `$EDITOR` at first changed line, closes overlay
- [ ] `q`, `esc`, or `D` closes the overlay
- [ ] Worktree sessions diff branch vs. base (`git diff <base>...HEAD`)
- [ ] Regular sessions diff working tree vs. HEAD (`git diff HEAD`)
- [ ] Sessions not in a git repo: `D` is a no-op, no diffstat in preview pane
- [ ] Clean working tree: preview shows "no changes", overlay shows "nothing to diff"
- [ ] Binary files rendered as `Binary files a/... b/... differ` in amber
- [ ] All diff colors use oasis_lagoon_dark / oasis_dawn palette (theme-aware)

## Architecture

```
git diff (subprocess)
        │
        ▼
go-diff parser  →  []FileDiff  →  DiffView renderer (lipgloss)
                                         │
                          ┌──────────────┴──────────────────┐
                          ▼                                  ▼
               Preview pane diffstat              Full-screen overlay
               (always visible when              (D key, scrollable,
                session has a git repo)           e opens $EDITOR)
```

The diff is fetched once per session focus change (debounced 300ms), parsed, and cached on the home model. Both the preview pane and the full-screen overlay share the same parsed `[]FileDiff` slice.

## Components

### New files

**`internal/git/diff.go`**
- `FetchDiff(dir string) (string, error)` — runs git subprocess, returns raw unified diff
  - Worktree sessions: `git diff <base>...HEAD` (branch vs. upstream)
  - Regular sessions: `git diff HEAD` (working tree vs. last commit)
  - Returns `("", nil)` for clean trees, `("", ErrNotGitRepo)` for non-git dirs

**`internal/ui/diff_view.go`**
- `DiffView` struct — owns parsed diff state and scroll position
- `Parse(raw string) error` — parses via `go-diff`, stores `[]FileDiff`
- `Summary() string` — returns "N files, +X -Y" or "no changes"
- `View(width, height int) string` — renders full overlay with lipgloss
- `HandleKey(key string) (bool, tea.Cmd)` — scroll, open editor, close
- `IsVisible() / Show() / Hide() / SetSize()` — standard dialog interface
- `FileUnderCursor() (path string, line int)` — returns file + first changed line at current scroll

### Modified files

**`internal/git/diff.go`** (new — see above)

**`go.mod` / `go.sum`** — add `github.com/sourcegraph/go-diff`

**`internal/ui/home.go`** — 7 wiring points:
1. Add `diffView *DiffView` field to `homeModel`
2. Init: `h.diffView = NewDiffView()`
3. Key routing: route `D` to `diffView` when no other overlay is visible
4. Mouse guard: add `h.diffView.IsVisible()` to the existing overlay guard chain
5. Trigger: on session focus change, dispatch `fetchDiffMsg` (debounced 300ms)
6. `SetSize`: call `h.diffView.SetSize(width, height)`
7. View check: render `h.diffView.View()` when `h.diffView.IsVisible()`

**`internal/ui/preview.go`** — add diffstat line to preview rendering

## Rendering (theme-aware)

| Element | Dark color | Light color |
|---|---|---|
| File header | `ColorAccent` (#58B8FD) bold | `ColorAccent` (#1670AD) bold |
| Hunk header `@@ -L +L @@` | `ColorComment` dim | `ColorComment` dim |
| Addition line `+` | `ColorGreen` (#53D390) | `ColorGreen` (#1b491d) |
| Deletion line `-` | `ColorRed` (#FF7979) | `ColorRed` (#663021) |
| Context line | `ColorTextDim` | `ColorTextDim` |
| Binary / rename notice | `ColorYellow` italic | `ColorYellow` italic |
| Diffstat (preview pane) | `ColorTextDim` | `ColorTextDim` |

Colors use the existing `ColorXxx` package-level variables from `styles.go`, which are already theme-aware.

## Key Bindings

| Key | Context | Action |
|---|---|---|
| `D` | Session list (any git session focused) | Toggle diff overlay |
| `j` / `↓` | Diff overlay | Scroll down one line |
| `k` / `↑` | Diff overlay | Scroll up one line |
| `PgDn` / `ctrl+d` | Diff overlay | Scroll down half page |
| `PgUp` / `ctrl+u` | Diff overlay | Scroll up half page |
| `e` | Diff overlay | Open file under cursor in `$EDITOR`, close overlay |
| `q` / `esc` / `D` | Diff overlay | Close overlay |

## Error Handling

| Scenario | Behavior |
|---|---|
| Session not in git repo | `D` no-op; no diffstat line in preview |
| Clean working tree | Preview: "no changes"; overlay: "nothing to diff" |
| `git diff` fails | Error logged; preview: "diff unavailable" |
| Binary file | Rendered as amber italic notice |
| `$EDITOR` not set | Falls back to `vi` |
| Very large diff | Rendered in full; scroll handles it (no truncation in v1) |

## Non-Goals (v1)

- Staging / discarding hunks
- Side-by-side diff view
- Syntax highlighting within diff lines (file content coloring)
- Configurable diff base per session
