# Projects-as-Source-of-Truth Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `projects.toml` the sole source of truth for what projects appear in the sidebar — replacing the disconnected SQLite groups table as the authority for group existence.

**Architecture:** Add `NewGroupTreeFromProjects` that builds the group tree from `projects.toml` (authoritative) + session data (membership) + stored groups (expanded/order state). Wire all TUI project operations (create/rename/delete) to write through to `projects.toml`. Remove the project-picker step from the new-session dialog since the project is always determined by cursor context.

**Tech Stack:** Go, Bubble Tea TUI, SQLite (modernc), TOML (BurntSushi/toml)

---

### Task 1: Add `RenameProject` to `projects.go`

**Files:**
- Modify: `internal/session/projects.go`
- Test: `internal/session/projects_test.go` (create if it doesn't exist)

**Step 1: Write the failing test**

Create `internal/session/projects_test.go`:

```go
package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameProject(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "projects.toml")

	// Seed with two projects
	initial := []*Project{
		{Name: "Alpha", BaseDir: "/tmp/alpha", BaseBranch: "main", Order: 0},
		{Name: "Beta", BaseDir: "/tmp/beta", BaseBranch: "master", Order: 1},
	}
	// Override path for testing
	oldGetDir := getHangarDirFn
	getHangarDirFn = func() (string, error) { return dir, nil }
	defer func() { getHangarDirFn = oldGetDir }()

	if err := SaveProjects(initial); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}

	if err := RenameProject("Alpha", "AlphaRenamed"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	projects, err := LoadProjects()
	if err != nil {
		t.Fatalf("LoadProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("want 2 projects, got %d", len(projects))
	}

	var found bool
	for _, p := range projects {
		if p.Name == "AlphaRenamed" {
			found = true
			if p.BaseDir != "/tmp/alpha" {
				t.Errorf("BaseDir changed: got %q", p.BaseDir)
			}
		}
		if p.Name == "Alpha" {
			t.Error("old name 'Alpha' still present")
		}
	}
	if !found {
		t.Error("renamed project 'AlphaRenamed' not found")
	}
}

func TestRenameProject_NotFound(t *testing.T) {
	dir := t.TempDir()
	getHangarDirFn = func() (string, error) { return dir, nil }
	defer func() { getHangarDirFn = nil }()

	_ = os.WriteFile(filepath.Join(dir, "projects.toml"), []byte(""), 0644)

	err := RenameProject("NoSuch", "NewName")
	if err == nil {
		t.Error("expected error for missing project")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd /Users/mnicholson/code/github/hangar/.worktrees/feature-project-mgmt
go test ./internal/session/... -run TestRenameProject -v
```
Expected: compile error (RenameProject undefined) or test failure about getHangarDirFn

**Note on test design:** The existing code calls `GetHangarDir()` directly. To make it testable, we need `GetHangarDir` to be injectable. Check whether the tests already use a pattern for this — look at how other tests in `session` package mock file paths. If they use `t.TempDir()` with `os.Setenv("HOME", ...)` or a function variable, use the same approach.

Actually, the simpler approach: just check that `RenameProject("nosuchproject", "x")` returns an error, and test the happy path by writing a real toml file to a temp dir using the existing pattern in the package.

Revised test approach (using `os.Setenv("HOME")`):

```go
func TestRenameProject(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	hangarDir := filepath.Join(dir, ".hangar")
	if err := os.MkdirAll(hangarDir, 0755); err != nil {
		t.Fatal(err)
	}

	initial := []*Project{
		{Name: "Alpha", BaseDir: "/tmp/alpha", BaseBranch: "main", Order: 0},
		{Name: "Beta", BaseDir: "/tmp/beta", BaseBranch: "master", Order: 1},
	}
	if err := SaveProjects(initial); err != nil {
		t.Fatalf("SaveProjects: %v", err)
	}

	if err := RenameProject("Alpha", "AlphaRenamed"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}

	projects, err := LoadProjects()
	if err != nil {
		t.Fatalf("LoadProjects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("want 2 projects, got %d", len(projects))
	}
	names := make(map[string]bool)
	for _, p := range projects {
		names[p.Name] = true
	}
	if !names["AlphaRenamed"] {
		t.Error("renamed project 'AlphaRenamed' not found")
	}
	if names["Alpha"] {
		t.Error("old name 'Alpha' still present")
	}
}
```

**Step 3: Implement `RenameProject`**

Add to `internal/session/projects.go` after `RemoveProject`:

```go
// RenameProject renames a project in projects.toml. Only the Name field changes;
// BaseDir and BaseBranch are preserved.
func RenameProject(oldName, newName string) error {
	projects, err := LoadProjects()
	if err != nil {
		return err
	}

	found := false
	for _, p := range projects {
		if strings.EqualFold(p.Name, oldName) {
			p.Name = newName
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("project %q not found", oldName)
	}

	return SaveProjects(projects)
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/session/... -run TestRenameProject -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/session/projects.go internal/session/projects_test.go
git commit -m "feat(session): add RenameProject function"
```

---

### Task 2: Add `NewGroupTreeFromProjects` to `groups.go`

This is the core change. It builds the sidebar group list from `projects.toml` rather than from the SQLite groups table.

**Files:**
- Modify: `internal/session/groups.go`
- Test: `internal/session/groups_test.go`

**Step 1: Write the failing tests**

Add to `internal/session/groups_test.go`:

```go
func TestNewGroupTreeFromProjects_EmptyProjects(t *testing.T) {
	tree := NewGroupTreeFromProjects(nil, nil, nil)
	if len(tree.GroupList) != 0 {
		t.Errorf("expected 0 groups, got %d", len(tree.GroupList))
	}
}

func TestNewGroupTreeFromProjects_ShowsEmptyProjects(t *testing.T) {
	projects := []*Project{
		{Name: "Alpha", BaseDir: "/tmp/alpha", Order: 0},
		{Name: "Beta", BaseDir: "/tmp/beta", Order: 1},
	}
	// No sessions
	tree := NewGroupTreeFromProjects(nil, projects, nil)

	if len(tree.GroupList) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(tree.GroupList))
	}
	// Alpha should be first (Order=0)
	if tree.GroupList[0].Name != "Alpha" {
		t.Errorf("expected Alpha first, got %s", tree.GroupList[0].Name)
	}
	if tree.GroupList[0].DefaultPath != "/tmp/alpha" {
		t.Errorf("DefaultPath not set from project BaseDir")
	}
}

func TestNewGroupTreeFromProjects_AssignsSessions(t *testing.T) {
	projects := []*Project{
		{Name: "MyApp", BaseDir: "/tmp/myapp", Order: 0},
	}
	instances := []*Instance{
		{ID: "1", Title: "work", GroupPath: "myapp", ProjectPath: "/tmp/myapp"},
		{ID: "2", Title: "feat", GroupPath: "myapp", ProjectPath: "/tmp/myapp"},
	}
	tree := NewGroupTreeFromProjects(instances, projects, nil)

	group := tree.Groups["myapp"]
	if group == nil {
		t.Fatal("expected group 'myapp'")
	}
	if len(group.Sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(group.Sessions))
	}
}

func TestNewGroupTreeFromProjects_OrphansIgnored(t *testing.T) {
	projects := []*Project{
		{Name: "MyApp", BaseDir: "/tmp/myapp", Order: 0},
	}
	instances := []*Instance{
		{ID: "1", Title: "orphan", GroupPath: "other-group", ProjectPath: "/tmp/other"},
	}
	tree := NewGroupTreeFromProjects(instances, projects, nil)

	// Only one group from projects, orphan not added
	if len(tree.GroupList) != 1 {
		t.Errorf("expected 1 group, got %d", len(tree.GroupList))
	}
	if len(tree.GroupList[0].Sessions) != 0 {
		t.Errorf("expected 0 sessions in group, got %d", len(tree.GroupList[0].Sessions))
	}
}

func TestNewGroupTreeFromProjects_RestoresExpandedState(t *testing.T) {
	projects := []*Project{
		{Name: "Alpha", BaseDir: "/tmp/alpha", Order: 0},
	}
	storedGroups := []*GroupData{
		{Name: "Alpha", Path: "alpha", Expanded: false},
	}
	tree := NewGroupTreeFromProjects(nil, projects, storedGroups)

	if tree.Groups["alpha"].Expanded {
		t.Error("expected Alpha to be collapsed per stored state")
	}
}

func TestNewGroupTreeFromProjects_OrderFromProjects(t *testing.T) {
	projects := []*Project{
		{Name: "Zebra", BaseDir: "/tmp/z", Order: 0},
		{Name: "Apple", BaseDir: "/tmp/a", Order: 1},
	}
	tree := NewGroupTreeFromProjects(nil, projects, nil)

	if len(tree.GroupList) != 2 {
		t.Fatalf("expected 2 groups")
	}
	// Zebra has Order=0 so comes first despite alphabetical sort
	if tree.GroupList[0].Name != "Zebra" {
		t.Errorf("expected Zebra first (Order=0), got %s", tree.GroupList[0].Name)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/session/... -run TestNewGroupTreeFromProjects -v
```
Expected: compile error (NewGroupTreeFromProjects undefined)

**Step 3: Implement `NewGroupTreeFromProjects`**

Add to `internal/session/groups.go` after `NewGroupTreeWithGroups`:

```go
// projectSlug converts a project name to a group path using the same
// logic as CreateGroup: lowercase, spaces→hyphens, sanitized.
func projectSlug(name string) string {
	sanitized := sanitizeGroupName(name)
	return strings.ToLower(strings.ReplaceAll(sanitized, " ", "-"))
}

// NewGroupTreeFromProjects builds a group tree using projects as the
// authoritative list of top-level groups.
//
//   - projects defines which groups exist and their order (from projects.toml)
//   - instances are assigned to groups by matching inst.GroupPath == projectSlug(project.Name)
//   - storedGroups supplies expanded/collapsed state (from SQLite groups table)
//
// Empty projects always appear in the sidebar. Sessions whose GroupPath does
// not match any project are silently dropped (orphans).
func NewGroupTreeFromProjects(instances []*Instance, projects []*Project, storedGroups []*GroupData) *GroupTree {
	tree := &GroupTree{
		Groups:   make(map[string]*Group),
		Expanded: make(map[string]bool),
	}

	// Build expanded-state lookup from stored groups
	expandedByPath := make(map[string]bool)
	for _, sg := range storedGroups {
		expandedByPath[sg.Path] = sg.Expanded
	}

	// Create one group per project
	for _, p := range projects {
		path := projectSlug(p.Name)
		expanded := true // default expanded
		if e, ok := expandedByPath[path]; ok {
			expanded = e
		}
		group := &Group{
			Name:        p.Name,
			Path:        path,
			Expanded:    expanded,
			Sessions:    []*Instance{},
			Order:       p.Order,
			DefaultPath: p.BaseDir,
		}
		tree.Groups[path] = group
		tree.Expanded[path] = expanded
	}

	// Assign instances to their groups
	for _, inst := range instances {
		if inst.GroupPath == "" {
			continue
		}
		group, exists := tree.Groups[inst.GroupPath]
		if !exists {
			continue // orphan — not in projects.toml
		}
		group.Sessions = append(group.Sessions, inst)
	}

	// Sort sessions within each group by Order
	for _, group := range tree.Groups {
		sort.SliceStable(group.Sessions, func(i, j int) bool {
			return group.Sessions[i].Order < group.Sessions[j].Order
		})
	}

	tree.rebuildGroupList()
	return tree
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/session/... -run TestNewGroupTreeFromProjects -v
```
Expected: all PASS

**Step 5: Commit**

```bash
git add internal/session/groups.go internal/session/groups_test.go
git commit -m "feat(session): add NewGroupTreeFromProjects for projects.toml-driven sidebar"
```

---

### Task 3: Wire `loadSessions` to use `NewGroupTreeFromProjects`

**Files:**
- Modify: `internal/ui/home.go` (loadSessionsMsg struct + loadSessions function)
- Modify: `internal/ui/update_handlers.go` (handleLoadSessions)

**Step 1: Add `projects` field to `loadSessionsMsg`**

In `internal/ui/home.go`, find `loadSessionsMsg` (around line 361) and add the field:

```go
type loadSessionsMsg struct {
	instances    []*session.Instance
	groups       []*session.GroupData
	projects     []*session.Project   // ADD THIS
	err          error
	restoreState *reloadState
	loadMtime    time.Time
}
```

**Step 2: Load projects in `loadSessions`**

In `internal/ui/home.go`, find the `loadSessions` function (around line 1319) and add project loading:

```go
func (h *Home) loadSessions() tea.Msg {
	if h.storage == nil {
		return loadSessionsMsg{instances: []*session.Instance{}, err: fmt.Errorf("storage not initialized")}
	}

	loadMtime, _ := h.storage.GetFileMtime()
	instances, groups, err := h.storage.LoadWithGroups()

	// Load projects for sidebar — errors here are non-fatal (sidebar falls back
	// to DB groups if projects.toml is absent or unreadable)
	projects, _ := session.ListProjects()

	return loadSessionsMsg{
		instances: instances,
		groups:    groups,
		projects:  projects,
		err:       err,
		loadMtime: loadMtime,
	}
}
```

**Step 3: Use `NewGroupTreeFromProjects` in `handleLoadSessions`**

In `internal/ui/update_handlers.go`, find the section around line 95-120 that builds the group tree and replace it:

```go
// Sync group tree with loaded data.
// If projects are available, use them as the authoritative group list;
// otherwise fall back to DB-stored groups (backward compat / empty projects.toml).
if len(msg.projects) > 0 {
	// Re-capture expanded state before rebuilding
	expandedState := make(map[string]bool)
	if h.groupTree != nil {
		for path, group := range h.groupTree.Groups {
			expandedState[path] = group.Expanded
		}
	}
	// Merge in-memory expanded state into stored groups so a just-toggled
	// group doesn't snap back to its stored state during a reload.
	mergedGroups := msg.groups
	for _, sg := range mergedGroups {
		if live, ok := expandedState[sg.Path]; ok {
			sg.Expanded = live
		}
	}
	h.groupTree = session.NewGroupTreeFromProjects(h.instances, msg.projects, mergedGroups)
} else if h.groupTree.GroupCount() == 0 {
	if len(msg.groups) > 0 {
		h.groupTree = session.NewGroupTreeWithGroups(h.instances, msg.groups)
	} else {
		h.groupTree = session.NewGroupTree(h.instances)
	}
} else {
	expandedState := make(map[string]bool)
	for path, group := range h.groupTree.Groups {
		expandedState[path] = group.Expanded
	}
	if len(msg.groups) > 0 {
		h.groupTree = session.NewGroupTreeWithGroups(h.instances, msg.groups)
	} else {
		h.groupTree = session.NewGroupTree(h.instances)
	}
	for path, expanded := range expandedState {
		if group, exists := h.groupTree.Groups[path]; exists {
			group.Expanded = expanded
		}
	}
}
```

**Step 4: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... -v 2>&1 | tail -30
go test ./internal/session/... -v 2>&1 | tail -30
```
Expected: build succeeds, pre-existing failures only

**Step 5: Commit**

```bash
git add internal/ui/home.go internal/ui/update_handlers.go
git commit -m "feat(ui): build sidebar from projects.toml via NewGroupTreeFromProjects"
```

---

### Task 4: Sync TUI project create/rename/delete to `projects.toml`

**Files:**
- Modify: `internal/ui/home.go` (handleGroupDialogKey, ConfirmDeleteGroup handler)

**Context:** `handleGroupDialogKey` is around line 4190. Three modes to fix:

**Step 1: Fix `GroupDialogCreate` — always write to projects.toml**

Find the `GroupDialogCreate` case (around line 4202). Currently it calls `session.AddProject` only if `baseDir != ""`. Change to always call it. The existing code already does this correctly IF a path is entered, but we want to ensure it always persists. Verify the current behavior and ensure it calls `AddProject` unconditionally (with empty baseDir if none was entered).

Current code:
```go
case GroupDialogCreate:
    name := h.groupDialog.GetValue()
    if name != "" {
        group := h.groupTree.CreateGroup(name)
        // Register as a Project so the new-session dialog can pick it up
        if baseDir := h.groupDialog.GetPath(); baseDir != "" {
            _ = session.AddProject(name, baseDir, "")
            h.groupTree.SetDefaultPathForGroup(group.Path, baseDir)
        }
        h.rebuildFlatItems()
        h.saveInstances()
    }
```

Change to always add the project (even if no base dir yet):
```go
case GroupDialogCreate:
    name := h.groupDialog.GetValue()
    if name != "" {
        group := h.groupTree.CreateGroup(name)
        baseDir := h.groupDialog.GetPath()
        _ = session.AddProject(name, baseDir, "")
        if baseDir != "" {
            h.groupTree.SetDefaultPathForGroup(group.Path, baseDir)
        }
        h.rebuildFlatItems()
        h.saveInstances()
    }
```

**Step 2: Fix `GroupDialogRename` — sync rename to projects.toml**

Find the `GroupDialogRename` case (around line 4217):

```go
case GroupDialogRename:
    name := h.groupDialog.GetValue()
    if name != "" {
        oldGroupPath := h.groupDialog.GetGroupPath()
        // Get old name from group tree before renaming
        oldName := ""
        if g, exists := h.groupTree.Groups[oldGroupPath]; exists {
            oldName = g.Name
        }
        h.groupTree.RenameGroup(oldGroupPath, name)
        // Sync rename to projects.toml
        if oldName != "" {
            _ = session.RenameProject(oldName, name)
        }
        h.instancesMu.Lock()
        h.instances = h.groupTree.GetAllInstances()
        h.instancesMu.Unlock()
        h.rebuildFlatItems()
        h.saveInstances()
        // ... rest of existing code (move-to-group logic) unchanged
    }
```

**Step 3: Fix `ConfirmDeleteGroup` — sync delete to projects.toml**

Find the `ConfirmDeleteGroup` case (around line 4059):

```go
case ConfirmDeleteGroup:
    groupPath := h.confirmDialog.GetTargetID()
    h.confirmDialog.Hide()
    // Get group name before deleting (needed to remove from projects.toml)
    groupName := ""
    if g, exists := h.groupTree.Groups[groupPath]; exists {
        groupName = g.Name
    }
    if h.groupTree.DeleteGroup(groupPath) == nil {
        h.setError(fmt.Errorf("cannot delete project: move or delete all sessions first"))
        return h, nil
    }
    // Sync deletion to projects.toml
    if groupName != "" {
        _ = session.RemoveProject(groupName)
    }
    h.instancesMu.Lock()
    h.instances = h.groupTree.GetAllInstances()
    h.instancesMu.Unlock()
    h.rebuildFlatItems()
    h.saveInstances()
```

**Step 4: Build and verify**

```bash
go build ./...
go test ./internal/ui/... -v 2>&1 | grep -E "^(ok|FAIL|---)"
```
Expected: build succeeds

**Step 5: Commit**

```bash
git add internal/ui/home.go
git commit -m "feat(ui): sync project create/rename/delete to projects.toml"
```

---

### Task 5: Remove project-picker step from `NewDialog`

The project picker step in the new-session dialog is now redundant. When 'n' is pressed, the project is already known from cursor context (the `ShowInGroup` call already sets `parentGroupPath` and `defaultPath` from the project). The dialog just needs to show the name/path inputs directly.

**Files:**
- Modify: `internal/ui/newdialog.go`

**Step 1: Remove project-picker state fields**

In `NewDialog` struct, remove:
- `projectList     []string`
- `projectCursor   int`
- `projectSelected bool`
- `projectStep     bool`

**Step 2: Remove `refreshProjectList` and `applySelectedProject` methods**

Delete both functions entirely.

**Step 3: Simplify `ShowInGroup`**

Replace the entire body of `ShowInGroup` after the setup lines. Remove the project-picker logic block entirely. New version:

```go
func (d *NewDialog) ShowInGroup(groupPath, groupName, defaultPath string) {
    if groupPath == "" {
        groupPath = "default"
        groupName = "default"
    }
    d.parentGroupPath = groupPath
    d.parentGroupName = groupName
    d.visible = true
    d.validationErr = ""
    d.nameInput.SetValue("")
    d.suggestionNavigated = false
    d.pathSuggestionCursor = 0
    d.pathCycler.Reset()
    d.pathInput.Blur()
    d.nameInput.Blur()
    d.claudeOptions.Blur()
    d.updateToolOptions()
    d.worktreeEnabled = false
    d.branchInput.SetValue("")
    d.branchAutoSet = false
    if defaultPath != "" {
        d.pathInput.SetValue(defaultPath)
    } else {
        cwd, err := os.Getwd()
        if err == nil {
            d.pathInput.SetValue(cwd)
        }
    }
    if userConfig, err := session.LoadUserConfig(); err == nil && userConfig != nil {
        d.claudeOptions.SetDefaults(userConfig)
    }
    // No project-picker step — project is always determined by cursor context.
    d.focusIndex = 0
    d.nameInput.Focus()
}
```

**Step 4: Remove `IsChoosingProject()`**

Delete the `IsChoosingProject()` method. Search for call sites:

```bash
grep -rn "IsChoosingProject" internal/
```

Remove any call sites in `home.go` that guard key handling with `IsChoosingProject()`.

**Step 5: Remove project-picker rendering from `View()`**

In `newdialog.go` `View()` method, find and delete the block:
```go
// Project picker step
if d.projectStep && !d.projectSelected {
    // ...entire block...
}
```

**Step 6: Remove key handling for project-picker**

In `HandleKey()`, find and delete:
```go
// Handle project picker step (before regular flow)
if d.projectStep && !d.projectSelected {
    // ...entire block...
}
```

**Step 7: Remove `refreshProjectList()` call in `NewNewDialog()`**

In the `NewNewDialog()` constructor, remove the line:
```go
dlg.refreshProjectList()
```

**Step 8: Build and run tests**

```bash
go build ./...
go test ./internal/ui/... -run TestNewDialog -v 2>&1 | tail -40
```
Expected: build succeeds. Pre-existing failures (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`) may still fail — that's OK.

**Step 9: Commit**

```bash
git add internal/ui/newdialog.go internal/ui/home.go
git commit -m "feat(ui): remove project-picker step from new-session dialog"
```

---

### Task 6: Touch DB when CLI changes `projects.toml`

When `hangar project add` or `hangar project remove` runs from the CLI, the TUI (if running) should auto-reload the sidebar. This requires touching the SQLite metadata so the `StorageWatcher` picks up the change.

**Files:**
- Modify: `cmd/hangar/project_cmd.go`

**Step 1: Add a touch helper**

In `project_cmd.go`, add a helper that opens the storage and calls Touch:

```go
// touchStorage bumps the SQLite metadata timestamp so any running TUI
// instance detects the change and reloads (picks up projects.toml updates).
func touchStorage(profile string) {
    storage, err := session.NewStorageWithProfile(profile)
    if err != nil {
        return // Non-fatal: TUI will pick up changes on next tick
    }
    defer storage.Close()
    _ = storage.Touch()
}
```

**Step 2: Call it after `AddProject` and `RemoveProject`**

In `handleProjectAdd`, after the `session.AddProject(...)` success line:
```go
fmt.Printf("Added project %q (base: %s, branch: %s)\n", name, expandedDir, baseBranch)
touchStorage(profile)
```

In `handleProjectRemove`, after the `session.RemoveProject(...)` success line:
```go
fmt.Printf("Removed project %q\n", name)
touchStorage(profile)
```

**Step 3: Check `session.NewStorageWithProfile` signature**

```bash
grep -n "func NewStorageWithProfile\|func.*Storage.*Touch" internal/session/storage.go | head -5
```

Use the correct function name/signature.

**Step 4: Build and test**

```bash
go build ./...
go test ./... -v 2>&1 | grep -E "^(ok|FAIL)"
```

**Step 5: Commit**

```bash
git add cmd/hangar/project_cmd.go
git commit -m "feat(cli): touch DB after project add/remove to trigger TUI reload"
```

---

### Task 7: Handle empty-projects empty state in sidebar

When all projects are empty (no sessions), the sidebar should show the project groups with a helpful message rather than a blank view.

**Files:**
- Modify: `internal/ui/home.go` (renderSessionList or the empty-state section)

**Step 1: Find current empty-state rendering**

```bash
grep -n "No Sessions\|no sessions\|emptyState\|splash\|initialLoading" internal/ui/home.go | head -20
```

**Step 2: Ensure empty groups render**

The group tree now always includes groups for each project (even empty ones). `rebuildFlatItems()` should include group header rows even when their Sessions slice is empty. Verify `Flatten()` in `groups.go` includes empty groups:

```bash
grep -n "func.*Flatten" internal/session/groups.go
```

Read the `Flatten` function to confirm it emits a group header row for empty groups. If it doesn't, add that behavior.

**Step 3: Add test for empty-project rendering in Flatten**

In `groups_test.go`, add:

```go
func TestFlatten_EmptyGroupsIncluded(t *testing.T) {
    projects := []*Project{
        {Name: "EmptyProject", BaseDir: "/tmp/ep", Order: 0},
    }
    tree := NewGroupTreeFromProjects(nil, projects, nil)
    items := tree.Flatten()

    if len(items) == 0 {
        t.Fatal("expected at least one item (the group header) for empty project")
    }
    if items[0].Type != ItemTypeGroup {
        t.Errorf("expected first item to be a group header, got %v", items[0].Type)
    }
    if items[0].Group.Name != "EmptyProject" {
        t.Errorf("expected group name 'EmptyProject', got %s", items[0].Group.Name)
    }
}
```

```bash
go test ./internal/session/... -run TestFlatten_EmptyGroupsIncluded -v
```

**Step 4: Verify build + run all tests**

```bash
go build ./...
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: only pre-existing failures (`TestNewDialog_WorktreeToggle_ViaKeyPress`, `TestNewDialog_TypingResetsSuggestionNavigation`).

**Step 5: Commit**

```bash
git add internal/session/groups.go internal/session/groups_test.go
git commit -m "test(session): verify empty projects appear in sidebar flatten"
```

---

### Task 8: Integration smoke test + full test run

**Step 1: Build the binary**

```bash
go build -o /tmp/hangar-test ./cmd/hangar
```

**Step 2: Verify `project list` still works**

```bash
/tmp/hangar-test project list
```
Expected: shows the projects from `~/.hangar/projects.toml`

**Step 3: Run the full test suite**

```bash
go test -race ./... 2>&1 | grep -E "^(ok|FAIL|---)"
```
Expected: only pre-existing failures

**Step 4: Final commit (if any cleanup needed)**

```bash
git add -p  # review any remaining changes
git commit -m "chore: cleanup after projects-as-source-of-truth implementation"
```
