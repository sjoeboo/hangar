# Screenshots

These screenshots are referenced from the main README. Capture at **~160 columns × ~40 rows** terminal size using `screencapture -x` (macOS, no drop shadow).

## Checklist

### `session-list.png` — Hero shot
- [ ] 2+ projects visible in sidebar
- [ ] Mix of statuses: at least one `●` green, one `◐` yellow, one `○` gray
- [ ] 2+ sessions with PR badges (`[#123]` style) — green and purple
- [ ] Detail/preview panel visible on the right showing session content
- [ ] tmux status bar visible (oasis_lagoon_dark theme with fleet pill)

```bash
screencapture -x assets/screenshots/session-list.png
```

### `pr-overview.png` — PR Overview (`P` key)
- [ ] Full-screen PR overview (press `P` from session list)
- [ ] 3+ sessions listed with PR numbers
- [ ] Mix of states: open (green) and merged (purple)
- [ ] CI check counts visible (`✓N ●N` etc.)
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/pr-overview.png
```

### `diff-view.png` — Inline Diff (`D` key)
- [ ] Diff overlay open over the session list (press `D` on a worktree session)
- [ ] 2+ hunks visible with real file paths
- [ ] Green `+` additions and red `-` deletions clearly visible
- [ ] File name header and change count (`N files, +N -N`)
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/diff-view.png
```

### `todo-board.png` — Todo Kanban (`t` key)
- [ ] All 4 columns: todo, in progress, in review, done
- [ ] Cards in at least 3 columns (doesn't need to be all 4)
- [ ] One card visibly selected/highlighted
- [ ] Description panel below showing the selected card's text
- [ ] Key hint bar at the bottom

```bash
screencapture -x assets/screenshots/todo-board.png
```

### `worktree-finish.png` — Worktree Finish Dialog (`W` key)
- [ ] Finish dialog open (press `W` on a worktree session)
- [ ] PR state visible in the dialog
- [ ] Options/checkboxes for merge + cleanup visible

```bash
screencapture -x assets/screenshots/worktree-finish.png
```

## Tips

- Use a clean terminal with a readable font (JetBrains Mono, Fira Code, etc.)
- Use real project/branch names — placeholder text looks less authentic
- If using iTerm2, disable transparency before capturing
- Crop to the terminal content only (no window chrome needed)
