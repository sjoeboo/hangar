package ui

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"ghe.spotify.net/mnicholson/hangar/internal/git"
	"ghe.spotify.net/mnicholson/hangar/internal/session"
	"ghe.spotify.net/mnicholson/hangar/internal/tmux"
)

// handleLoadSessions processes loadSessionsMsg, updating in-memory state from
// the newly loaded instances and groups, restoring cursor/viewport state, and
// triggering initial preview/diff fetches.
func (h *Home) handleLoadSessions(msg loadSessionsMsg) tea.Cmd {
	// Clear loading indicators and store file mtime for external change detection
	h.reloadMu.Lock()
	h.isReloading = false
	if !msg.loadMtime.IsZero() {
		h.lastLoadMtime = msg.loadMtime
	}
	h.reloadMu.Unlock()
	h.initialLoading = false // First load complete, hide splash

	// Show hooks installation prompt (after splash screen is gone)
	if h.pendingHooksPrompt && !h.setupWizard.IsVisible() {
		h.confirmDialog.ShowInstallHooks()
		h.confirmDialog.SetSize(h.width, h.height)
	}

	if msg.err != nil {
		h.setError(msg.err)
	} else {
		// Fix stale state: re-capture current cursor AND expanded groups.
		// Between storageChangedMsg (which saved restoreState) and now,
		// the user may have navigated or toggled groups.
		if msg.restoreState != nil {
			// Re-capture cursor position from OLD flatItems
			if h.cursor >= 0 && h.cursor < len(h.flatItems) {
				currentItem := h.flatItems[h.cursor]
				switch currentItem.Type {
				case session.ItemTypeSession:
					if currentItem.Session != nil {
						msg.restoreState.cursorSessionID = currentItem.Session.ID
						msg.restoreState.cursorGroupPath = ""
					}
				case session.ItemTypeGroup:
					msg.restoreState.cursorGroupPath = currentItem.Path
					msg.restoreState.cursorSessionID = ""
				}
			}
			msg.restoreState.viewOffset = h.viewOffset

			// Re-capture expanded groups (user may have toggled between
			// storageChangedMsg and now)
			if h.groupTree != nil {
				msg.restoreState.expandedGroups = make(map[string]bool)
				for _, group := range h.groupTree.GroupList {
					if group.Expanded {
						msg.restoreState.expandedGroups[group.Path] = true
					}
				}
			}
		}

		h.instancesMu.Lock()
		oldCount := len(h.instances)
		h.instances = msg.instances
		newCount := len(msg.instances)
		uiLog.Debug("reload_load_sessions", slog.Int("old_count", oldCount), slog.Int("new_count", newCount), slog.String("profile", h.profile))
		// Rebuild instanceByID map for O(1) lookup
		h.instanceByID = make(map[string]*session.Instance, len(h.instances))
		for _, inst := range h.instances {
			h.instanceByID[inst.ID] = inst
		}
		// Deduplicate Claude session IDs on load to fix any existing duplicates
		// This ensures no two sessions share the same Claude session ID
		session.UpdateClaudeSessionsWithDedup(h.instances)
		// Collect OpenCode detection commands for restored sessions without IDs
		// Using tea.Cmd pattern ensures save is triggered after detection completes
		var detectionCmds []tea.Cmd
		for _, inst := range h.instances {
			if inst.Tool == "opencode" && inst.OpenCodeSessionID == "" {
				detectionCmds = append(detectionCmds, h.detectOpenCodeSessionCmd(inst))
			}
		}
		h.instancesMu.Unlock()
		// Invalidate status counts cache
		h.cachedStatusCounts.valid.Store(false)
		// Build group tree. When projects.toml is populated, it is the authoritative
		// source for which groups exist (empty projects always show). Fall back to
		// DB-stored groups when no projects are configured.
		if len(msg.projects) > 0 {
			// Merge live expanded state into storedGroups so a just-toggled group
			// doesn't snap back to its stored state during an auto-reload.
			expandedState := make(map[string]bool)
			if h.groupTree != nil {
				for path, group := range h.groupTree.Groups {
					expandedState[path] = group.Expanded
				}
			}
			for _, sg := range msg.groups {
				if live, ok := expandedState[sg.Path]; ok {
					sg.Expanded = live
				}
			}
			h.groupTree = session.NewGroupTreeFromProjects(h.instances, msg.projects, msg.groups)
		} else if h.groupTree == nil || h.groupTree.GroupCount() == 0 {
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
		h.search.SetItems(h.instances)

		// Re-apply pending title changes that were lost during reload.
		// This happens when a rename's save was skipped (isReloading=true)
		// and the reload replaced instances with stale disk data.
		if len(h.pendingTitleChanges) > 0 {
			applied := false
			for id, title := range h.pendingTitleChanges {
				if inst := h.getInstanceByID(id); inst != nil && inst.Title != title {
					inst.Title = title
					inst.SyncTmuxDisplayName()
					applied = true
					uiLog.Info("pending_rename_reapplied",
						slog.String("session_id", id),
						slog.String("title", title))
				}
			}
			// Clear pending changes and persist if any were re-applied
			h.pendingTitleChanges = make(map[string]string)
			if applied {
				h.forceSaveInstances()
			}
		}

		// Restore state if provided (from auto-reload)
		if msg.restoreState != nil {
			h.restoreState(*msg.restoreState)
			h.syncViewport()
		} else {
			h.rebuildFlatItems()
			// Restore cursor from persisted UI state (initial load only)
			if h.pendingCursorRestore != nil {
				restored := false
				if h.pendingCursorRestore.CursorSessionID != "" {
					for i, item := range h.flatItems {
						if item.Type == session.ItemTypeSession &&
							item.Session != nil &&
							item.Session.ID == h.pendingCursorRestore.CursorSessionID {
							h.cursor = i
							restored = true
							break
						}
					}
				}
				if !restored && h.pendingCursorRestore.CursorGroupPath != "" {
					for i, item := range h.flatItems {
						if item.Type == session.ItemTypeGroup && item.Path == h.pendingCursorRestore.CursorGroupPath {
							h.cursor = i
							break
						}
					}
				}
				h.pendingCursorRestore = nil
				h.syncViewport()
			}
			// Save after dedup to persist any ID changes (initial load only)
			h.saveInstances()
		}
		// Trigger immediate preview fetch for initial selection
		h.updateDiffStat()
		if selected := h.getSelectedSession(); selected != nil {
			h.previewFetchingMu.Lock()
			h.previewFetchingID = selected.ID
			h.previewFetchingMu.Unlock()
			// Batch preview fetch with any OpenCode detection commands and diff fetch
			allCmds := append(detectionCmds, h.fetchPreview(selected))
			if dir := h.effectiveDir(selected); dir != "" && git.IsGitRepo(dir) {
				allCmds = append(allCmds, fetchDiffCmd(dir, selected.ID))
			}
			return tea.Batch(allCmds...)
		}
		// No selection, but still run detection commands if any
		if len(detectionCmds) > 0 {
			return tea.Batch(detectionCmds...)
		}
	}
	return nil
}

// handleSessionCreated processes sessionCreatedMsg, adding the new session to
// in-memory state, updating the group tree, saving, and triggering preview fetch.
func (h *Home) handleSessionCreated(msg sessionCreatedMsg) tea.Cmd {
	// Handle reload scenario: session was already started in tmux, we MUST save it to JSON
	// even during reload, otherwise the session becomes orphaned (exists in tmux but not in storage)
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading && msg.err == nil && msg.instance != nil {
		// CRITICAL: Save the new session to JSON immediately to prevent orphaning
		// Skip in-memory state update (reload will handle that), but persist to disk
		uiLog.Debug("reload_save_session_created", slog.String("id", msg.instance.ID), slog.String("title", msg.instance.Title))
		h.instancesMu.Lock()
		h.instances = append(h.instances, msg.instance)
		h.instancesMu.Unlock()
		// Force save to persist the session even during reload
		h.forceSaveInstances()
		// Trigger another reload to pick up the new session in the UI
		if h.storageWatcher != nil {
			h.storageWatcher.TriggerReload()
		}
		h.pendingTodoID = ""
		h.pendingTodoPrompt = ""
		return nil
	}
	if msg.err != nil {
		h.setError(msg.err)
	} else {
		h.instancesMu.Lock()
		h.instances = append(h.instances, msg.instance)
		h.instanceByID[msg.instance.ID] = msg.instance
		// Run dedup to ensure the new session doesn't have a duplicate ID
		session.UpdateClaudeSessionsWithDedup(h.instances)
		h.instancesMu.Unlock()
		// Invalidate status counts cache
		h.cachedStatusCounts.valid.Store(false)

		// Track as launching for animation
		h.launchingSessions[msg.instance.ID] = time.Now()

		// Expand the group so the session is visible
		if msg.instance.GroupPath != "" {
			h.groupTree.ExpandGroupWithParents(msg.instance.GroupPath)
		}

		// Add to existing group tree instead of rebuilding
		h.groupTree.AddSession(msg.instance)
		h.rebuildFlatItems()
		h.search.SetItems(h.instances)

		// Auto-select the new session
		for i, item := range h.flatItems {
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == msg.instance.ID {
				h.cursor = i
				h.syncViewport()
				break
			}
		}

		// Save both instances AND groups (critical fix: was losing groups!)
		// Use forceSave to bypass mtime check - new session creation MUST persist
		h.forceSaveInstances()

		// Link pending todo to the newly created session
		if h.pendingTodoID != "" {
			if err := h.storage.UpdateTodoStatus(h.pendingTodoID, session.TodoStatusInProgress, msg.instance.ID); err != nil {
				uiLog.Warn("link_todo_err", slog.String("todo", h.pendingTodoID), slog.String("err", err.Error()))
			}
			h.pendingTodoID = ""
		}

		// Start fetching preview for the new session
		cmds := []tea.Cmd{h.fetchPreview(msg.instance)}
		if h.pendingTodoPrompt != "" {
			capturedInst := msg.instance
			capturedPrompt := h.pendingTodoPrompt
			h.pendingTodoPrompt = ""
			cmds = append(cmds, func() tea.Msg {
				time.Sleep(4 * time.Second)
				if err := capturedInst.SendText(capturedPrompt); err != nil {
					uiLog.Warn("todo_prompt_send_failed",
						slog.String("id", capturedInst.ID),
						slog.String("err", err.Error()))
				}
				return todoPromptSentMsg{}
			})
		}
		return tea.Batch(cmds...)
	}
	return nil
}

// handleSessionForked processes sessionForkedMsg, adding the forked session to
// in-memory state and updating the group tree.
func (h *Home) handleSessionForked(msg sessionForkedMsg) tea.Cmd {
	// Clean up forking state for source session
	if msg.sourceID != "" {
		delete(h.forkingSessions, msg.sourceID)
	}

	// Handle reload scenario: forked session was already started in tmux, we MUST save it
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading && msg.err == nil && msg.instance != nil {
		// CRITICAL: Save the forked session to JSON immediately to prevent orphaning
		uiLog.Debug("reload_save_session_forked", slog.String("id", msg.instance.ID), slog.String("title", msg.instance.Title))
		h.instancesMu.Lock()
		h.instances = append(h.instances, msg.instance)
		h.instancesMu.Unlock()
		h.forceSaveInstances()
		if h.storageWatcher != nil {
			h.storageWatcher.TriggerReload()
		}
		return nil
	}

	if msg.err != nil {
		h.setError(msg.err)
	} else {
		h.instancesMu.Lock()
		h.instances = append(h.instances, msg.instance)
		h.instanceByID[msg.instance.ID] = msg.instance
		// Run dedup to ensure the forked session doesn't have a duplicate ID
		// This is critical: fork detection may have picked up wrong session
		session.UpdateClaudeSessionsWithDedup(h.instances)
		h.instancesMu.Unlock()
		// Invalidate status counts cache
		h.cachedStatusCounts.valid.Store(false)

		// Track as launching for animation
		h.launchingSessions[msg.instance.ID] = time.Now()

		// Expand the group so the session is visible
		if msg.instance.GroupPath != "" {
			h.groupTree.ExpandGroupWithParents(msg.instance.GroupPath)
		}

		// Add to existing group tree instead of rebuilding
		h.groupTree.AddSession(msg.instance)
		h.rebuildFlatItems()
		h.search.SetItems(h.instances)

		// Auto-select the forked session
		for i, item := range h.flatItems {
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == msg.instance.ID {
				h.cursor = i
				h.syncViewport()
				break
			}
		}

		// Save both instances AND groups
		// Use forceSave to bypass mtime check - forked session MUST persist
		h.forceSaveInstances()

		// Start fetching preview for the forked session
		return h.fetchPreview(msg.instance)
	}
	return nil
}

// handleSessionDeleted processes sessionDeletedMsg, removing the session from
// in-memory state, group tree, and storage, then showing an undo hint.
func (h *Home) handleSessionDeleted(msg sessionDeletedMsg) tea.Cmd {
	// CRITICAL FIX: Skip processing during reload to prevent state corruption
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading {
		uiLog.Debug("reload_skip_session_deleted")
		return nil
	}

	// Report kill error if any (session may still be running in tmux)
	if msg.killErr != nil {
		h.setError(fmt.Errorf("warning: tmux session may still be running: %w", msg.killErr))
	}

	// Find and remove from list
	var deletedInstance *session.Instance
	h.instancesMu.Lock()
	for i, s := range h.instances {
		if s.ID == msg.deletedID {
			deletedInstance = s
			h.instances = append(h.instances[:i], h.instances[i+1:]...)
			break
		}
	}
	delete(h.instanceByID, msg.deletedID)
	h.instancesMu.Unlock()

	// Push to undo stack before removing from group tree
	if deletedInstance != nil {
		h.pushUndoStack(deletedInstance)
	}

	// Invalidate status counts cache
	h.cachedStatusCounts.valid.Store(false)
	// Invalidate preview cache for deleted session
	h.invalidatePreviewCache(msg.deletedID)
	h.logActivityMu.Lock()
	delete(h.lastLogActivity, msg.deletedID)
	h.logActivityMu.Unlock()
	// Remove from group tree (preserves empty groups)
	if deletedInstance != nil {
		h.groupTree.RemoveSession(deletedInstance)
	}
	h.rebuildFlatItems()
	// Update search items
	h.search.SetItems(h.instances)
	// Explicitly delete from database to prevent resurrection on reload
	if err := h.storage.DeleteInstance(msg.deletedID); err != nil {
		uiLog.Warn("delete_instance_db_err", slog.String("id", msg.deletedID), slog.String("err", err.Error()))
	}
	// Save both instances AND groups (critical fix: was losing groups!)
	// Use forceSave to bypass mtime check - delete MUST persist
	h.forceSaveInstances()

	// Orphan any todo linked to the deleted session
	if err := h.storage.OrphanTodosForSession(msg.deletedID); err != nil {
		uiLog.Warn("orphan_todo_err", slog.String("session", msg.deletedID), slog.String("err", err.Error()))
	}

	// Show undo hint (using setError as a transient message)
	if deletedInstance != nil {
		h.setError(fmt.Errorf("deleted '%s'. Ctrl+Z to undo", deletedInstance.Title))
	}
	return nil
}

// handleBulkDeleted processes bulkDeletedMsg, removing multiple sessions from
// in-memory state, group tree, and storage in a single batch operation.
func (h *Home) handleBulkDeleted(msg bulkDeletedMsg) tea.Cmd {
	// CRITICAL FIX: Skip processing during reload to prevent state corruption
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading {
		uiLog.Debug("reload_skip_bulk_deleted")
		return nil
	}

	for _, id := range msg.deletedIDs {
		var deletedInstance *session.Instance
		h.instancesMu.Lock()
		for i, inst := range h.instances {
			if inst.ID == id {
				deletedInstance = inst
				h.instances = append(h.instances[:i], h.instances[i+1:]...)
				break
			}
		}
		delete(h.instanceByID, id)
		h.instancesMu.Unlock()

		// Push to undo stack before removing from group tree
		if deletedInstance != nil {
			h.pushUndoStack(deletedInstance)
			// Remove from group tree (preserves empty groups)
			h.groupTree.RemoveSession(deletedInstance)
		}
		// Invalidate caches for deleted session
		h.cachedStatusCounts.valid.Store(false)
		h.invalidatePreviewCache(id)
		h.logActivityMu.Lock()
		delete(h.lastLogActivity, id)
		h.logActivityMu.Unlock()
		// Explicitly delete from database to prevent resurrection on reload
		if err := h.storage.DeleteInstance(id); err != nil {
			uiLog.Warn("bulk_delete_instance_db_err", slog.String("id", id), slog.String("err", err.Error()))
		}
		// Orphan any todos linked to the deleted session
		if err := h.storage.OrphanTodosForSession(id); err != nil {
			uiLog.Warn("bulk_orphan_todo_err", slog.String("session", id), slog.String("err", err.Error()))
		}
	}
	h.rebuildFlatItems()
	// Update search items
	h.search.SetItems(h.instances)
	h.forceSaveInstances()
	// Exit bulk select mode and clear selections
	h.bulkSelectMode = false
	h.selectedSessionIDs = make(map[string]bool)
	if len(msg.killErrs) > 0 {
		h.setError(fmt.Errorf("deleted %d sessions (warning: some tmux sessions may still be running)", len(msg.deletedIDs)))
	} else {
		h.setError(fmt.Errorf("deleted %d sessions. Ctrl+Z to undo (one at a time)", len(msg.deletedIDs)))
	}
	return nil
}

// handleSessionRestored processes sessionRestoredMsg, re-adding a previously
// deleted session back into in-memory state and the group tree.
func (h *Home) handleSessionRestored(msg sessionRestoredMsg) tea.Cmd {
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading {
		uiLog.Debug("reload_skip_session_restored")
		return nil
	}
	if msg.err != nil {
		h.setError(fmt.Errorf("failed to restore session: %w", msg.err))
		return nil
	}

	// Re-add to instances (mirrors sessionCreatedMsg pattern)
	h.instancesMu.Lock()
	h.instances = append(h.instances, msg.instance)
	h.instanceByID[msg.instance.ID] = msg.instance
	session.UpdateClaudeSessionsWithDedup(h.instances)
	h.instancesMu.Unlock()
	h.cachedStatusCounts.valid.Store(false)

	// Track as launching for animation
	h.launchingSessions[msg.instance.ID] = time.Now()

	// Expand the group so the restored session is visible
	if msg.instance.GroupPath != "" {
		h.groupTree.ExpandGroupWithParents(msg.instance.GroupPath)
	}

	// Add to group tree and rebuild
	h.groupTree.AddSession(msg.instance)
	h.rebuildFlatItems()
	h.search.SetItems(h.instances)

	// Move cursor to restored session
	for i, item := range h.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == msg.instance.ID {
			h.cursor = i
			h.syncViewport()
			break
		}
	}

	// Use forceSave to bypass mtime check - restore MUST persist
	h.forceSaveInstances()
	h.setError(fmt.Errorf("restored '%s'", msg.instance.Title))
	return h.fetchPreview(msg.instance)
}

// handleOpenCodeDetectionComplete processes openCodeDetectionCompleteMsg,
// updating the detected session ID on the instance and persisting to storage.
func (h *Home) handleOpenCodeDetectionComplete(msg openCodeDetectionCompleteMsg) tea.Cmd {
	// OpenCode session detection completed
	// CRITICAL: Find the CURRENT instance by ID and update it
	// The original pointer may have been replaced by storage watcher reload
	if msg.sessionID != "" {
		uiLog.Debug("opencode_detection_complete", slog.String("instance_id", msg.instanceID), slog.String("session_id", msg.sessionID))
		// Update the CURRENT instance (not the original pointer which may be stale)
		if inst := h.getInstanceByID(msg.instanceID); inst != nil {
			inst.OpenCodeSessionID = msg.sessionID
			inst.OpenCodeDetectedAt = time.Now()
			uiLog.Debug("opencode_instance_updated", slog.String("instance_id", msg.instanceID), slog.String("session_id", msg.sessionID))
		} else {
			uiLog.Warn("opencode_instance_not_found", slog.String("instance_id", msg.instanceID))
		}
	} else {
		uiLog.Debug("opencode_detection_no_session", slog.String("instance_id", msg.instanceID))
		// Mark detection as completed even when no session found
		// This allows UI to show "No session found" instead of "Detecting..."
		if inst := h.getInstanceByID(msg.instanceID); inst != nil {
			inst.OpenCodeDetectedAt = time.Now()
			uiLog.Debug("opencode_marked_complete", slog.String("instance_id", msg.instanceID))
		}
	}
	// CRITICAL: Force save to persist the detected session ID to storage
	// This uses forceSaveInstances() to bypass isReloading check, preventing
	// the race condition where detection completes during a storage watcher reload
	h.forceSaveInstances()
	return nil
}

// handleMaintenanceComplete processes maintenanceCompleteMsg, displaying a
// summary of maintenance actions taken and scheduling auto-clear.
func (h *Home) handleMaintenanceComplete(msg maintenanceCompleteMsg) tea.Cmd {
	r := msg.result
	// Build a summary string
	var parts []string
	if r.PrunedLogs > 0 {
		parts = append(parts, fmt.Sprintf("%d logs pruned", r.PrunedLogs))
	}
	if r.PrunedBackups > 0 {
		parts = append(parts, fmt.Sprintf("%d backups cleaned", r.PrunedBackups))
	}
	if r.ArchivedSessions > 0 {
		parts = append(parts, fmt.Sprintf("%d sessions archived", r.ArchivedSessions))
	}
	if len(parts) > 0 {
		h.maintenanceMsg = "Maintenance: " + strings.Join(parts, ", ") + fmt.Sprintf(" (%s)", r.Duration.Round(time.Millisecond))
		h.maintenanceMsgTime = time.Now()
		// Auto-clear after 30 seconds
		return tea.Tick(30*time.Second, func(_ time.Time) tea.Msg {
			return clearMaintenanceMsg{}
		})
	}
	return nil
}

// handleStorageChanged processes storageChangedMsg, preserving UI state and
// triggering a reload from disk, then resuming the storage change listener.
func (h *Home) handleStorageChanged(msg storageChangedMsg) tea.Cmd {
	uiLog.Debug("reload_storage_changed", slog.String("profile", h.profile), slog.Int("instances", len(h.instances)))

	// Show reload indicator and increment version to invalidate in-flight background saves
	h.reloadMu.Lock()
	h.isReloading = true
	h.reloadVersion++
	h.reloadMu.Unlock()

	// Preserve UI state before reload
	state := h.preserveState()

	// Reload from disk
	cmd := func() tea.Msg {
		// Capture file mtime BEFORE loading to detect external changes later
		loadMtime, _ := h.storage.GetFileMtime()
		instances, groups, err := h.storage.LoadWithGroups()
		projects, _ := session.ListProjects()
		uiLog.Debug("reload_load_with_groups", slog.Int("instances", len(instances)), slog.Any("error", err))
		return loadSessionsMsg{
			instances:    instances,
			groups:       groups,
			projects:     projects,
			err:          err,
			restoreState: &state, // Pass state to restore after load
			loadMtime:    loadMtime,
		}
	}

	// Continue listening for next change
	return tea.Batch(cmd, listenForReloads(h.storageWatcher))
}

// handleStatusUpdate processes statusUpdateMsg, clearing the attaching flag,
// triggering a status refresh, and syncing cursor to any notification-switched session.
func (h *Home) handleStatusUpdate(msg statusUpdateMsg) tea.Cmd {
	// Clear attach flag - we've returned from the attached session
	h.isAttaching.Store(false) // Atomic store for thread safety

	// Trigger status update on attach return to reflect current state
	// Acknowledgment was already done on attach (if session was waiting),
	// so this just refreshes the display with current busy indicator state.
	h.triggerStatusUpdate()

	// Cursor sync: if user switched sessions via notification bar during attach,
	// move cursor to the session they were last viewing
	h.lastNotifSwitchMu.Lock()
	switchedID := h.lastNotifSwitchID
	h.lastNotifSwitchID = ""
	h.lastNotifSwitchMu.Unlock()

	if switchedID != "" {
		found := false
		for i, item := range h.flatItems {
			if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == switchedID {
				h.cursor = i
				h.syncViewport()
				found = true
				break
			}
		}
		// If session is in a collapsed group, expand it first
		if !found {
			h.instancesMu.RLock()
			inst, ok := h.instanceByID[switchedID]
			h.instancesMu.RUnlock()
			if ok && inst.GroupPath != "" && h.groupTree != nil {
				h.groupTree.ExpandGroupWithParents(inst.GroupPath)
				h.rebuildFlatItems()
				for i, item := range h.flatItems {
					if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == switchedID {
						h.cursor = i
						h.syncViewport()
						break
					}
				}
			}
		}
	}

	// Skip save during reload to avoid overwriting external changes (CLI)
	h.reloadMu.Lock()
	reloading := h.isReloading
	h.reloadMu.Unlock()
	if reloading {
		return nil
	}

	// PERFORMANCE FIX: Skip save on attach return for 10 seconds
	// Saving can also be blocking (JSON serialization + file write).
	// Combine with periodic save instead of saving on every attach/detach.
	// We'll let the next tickMsg handle background save if needed.

	return nil
}

// handlePreviewDebounce processes previewDebounceMsg, triggering preview fetch,
// worktree dirty check, PR status fetch, and remote URL fetch for the selected session.
func (h *Home) handlePreviewDebounce(msg previewDebounceMsg) tea.Cmd {
	// PERFORMANCE: Debounce period elapsed - check if this fetch is still relevant
	// If user continued navigating, pendingPreviewID will have changed
	h.previewDebounceMu.Lock()
	isPending := h.pendingPreviewID == msg.sessionID
	if isPending {
		h.pendingPreviewID = "" // Clear pending state
	}
	h.previewDebounceMu.Unlock()

	if !isPending {
		return nil // Superseded by newer navigation
	}

	// Find session and trigger actual fetch
	h.instancesMu.RLock()
	inst := h.instanceByID[msg.sessionID]
	h.instancesMu.RUnlock()

	if inst != nil {
		var cmds []tea.Cmd

		// Preview fetch
		h.previewFetchingMu.Lock()
		needsPreviewFetch := h.previewFetchingID != inst.ID
		if needsPreviewFetch {
			h.previewFetchingID = inst.ID
		}
		h.previewFetchingMu.Unlock()
		if needsPreviewFetch {
			cmds = append(cmds, h.fetchPreview(inst))
		}

		// Worktree dirty status check (lazy, 10s TTL)
		if inst.IsWorktree() && inst.WorktreePath != "" {
			cachedAtDirty, hasCachedDirty := h.cache.GetWorktreeDirtyCachedAt(inst.ID)
			needsCheck := !hasCachedDirty || time.Since(cachedAtDirty) > worktreeDirtyCacheTTL
			if needsCheck {
				h.cache.TouchWorktreeDirty(inst.ID) // Prevent duplicate fetches
			}
			if needsCheck {
				sid := inst.ID
				wtPath := inst.WorktreePath
				cmds = append(cmds, func() tea.Msg {
					dirty, err := git.HasUncommittedChanges(wtPath)
					return worktreeDirtyCheckMsg{sessionID: sid, isDirty: dirty, err: err}
				})
			}
		}

		// PR status check (lazy, 60s TTL, requires gh CLI)
		if h.ghPath != "" && inst.IsWorktree() && inst.WorktreePath != "" {
			_, cachedAtPR, hasCachedPR := h.cache.HasPREntry(inst.ID)
			needsFetch := !hasCachedPR || time.Since(cachedAtPR) > prCacheTTL
			if needsFetch {
				h.cache.TouchPR(inst.ID) // Prevent duplicate fetches
			}
			if needsFetch {
				sid := inst.ID
				wtPath := inst.WorktreePath
				ghPath := h.ghPath
				cmds = append(cmds, func() tea.Msg {
					return fetchPRInfo(sid, wtPath, ghPath)
				})
			}
		}

		// Remote URL fetch (lazy, 5m TTL)
		if inst.IsWorktree() && inst.WorktreePath != "" {
			cachedAtRemote, hasCachedRemote := h.cache.GetWorktreeRemoteCachedAt(inst.ID)
			needsRemote := !hasCachedRemote || time.Since(cachedAtRemote) > worktreeRemoteCacheTTL
			if needsRemote {
				h.cache.TouchWorktreeRemote(inst.ID) // Prevent duplicate fetches
			}
			if needsRemote {
				sid := inst.ID
				wtPath := inst.WorktreePath
				remoteLabels := h.remoteLabels
				cmds = append(cmds, func() tea.Msg {
					out, err := exec.Command("git", "-C", wtPath, "config", "--get", "remote.origin.url").Output()
					url := strings.TrimSpace(string(out))
					return worktreeRemoteCheckMsg{sessionID: sid, remoteURL: normalizeRemoteURL(url, remoteLabels), err: err}
				})
			}
		}

		if len(cmds) > 0 {
			return tea.Batch(cmds...)
		}
	}
	return nil
}

// handlePRFetched processes prFetchedMsg, updating the PR cache and advancing
// any linked todo status based on the PR state.
func (h *Home) handlePRFetched(msg prFetchedMsg) tea.Cmd {
	// Update PR cache (nil pr means no PR found — still record so we don't re-fetch immediately)
	h.cache.SetPR(msg.sessionID, msg.pr)

	// If WorktreeFinishDialog is open for this session, push updated PR data
	if h.worktreeFinishDialog.IsVisible() && h.worktreeFinishDialog.GetSessionID() == msg.sessionID {
		h.worktreeFinishDialog.SetPR(msg.pr, true)
	}

	// Auto-advance todo status based on PR state
	if msg.pr != nil {
		if todo, err := h.storage.FindTodoBySessionID(msg.sessionID); err == nil && todo != nil {
			var newStatus session.TodoStatus
			switch msg.pr.State {
			case "OPEN", "DRAFT":
				if todo.Status != session.TodoStatusInReview && todo.Status != session.TodoStatusDone {
					newStatus = session.TodoStatusInReview
				}
			case "MERGED":
				if todo.Status != session.TodoStatusDone {
					newStatus = session.TodoStatusDone
				}
			// "CLOSED" (PR closed without merging): intentionally no transition — todo stays in its current state
			}
			if newStatus != "" {
				if err := h.storage.UpdateTodoStatus(todo.ID, newStatus, msg.sessionID); err != nil {
					uiLog.Warn("pr_todo_status_err", slog.String("todo", todo.ID), slog.String("err", err.Error()))
				}
			}
		}
	}
	return nil
}

// handleReviewSessionCreated processes reviewSessionCreatedMsg, adding the review
// session to in-memory state, navigating to it, and optionally sending an initial prompt.
func (h *Home) handleReviewSessionCreated(msg reviewSessionCreatedMsg) tea.Cmd {
	if msg.err != nil {
		h.setError(msg.err)
		return nil
	}
	inst := msg.instance
	h.instancesMu.Lock()
	h.instances = append(h.instances, inst)
	h.instanceByID[inst.ID] = inst
	h.instancesMu.Unlock()
	// Invalidate status counts cache
	h.cachedStatusCounts.valid.Store(false)

	// Track as launching for animation
	h.launchingSessions[inst.ID] = time.Now()

	// Expand the group so the session is visible
	if inst.GroupPath != "" {
		h.groupTree.ExpandGroupWithParents(inst.GroupPath)
	}

	h.groupTree.AddSession(inst)
	h.rebuildFlatItems()
	h.search.SetItems(h.instances)

	// Auto-select the new session
	for i, item := range h.flatItems {
		if item.Type == session.ItemTypeSession && item.Session != nil && item.Session.ID == inst.ID {
			h.cursor = i
			h.syncViewport()
			break
		}
	}
	h.forceSaveInstances()
	if msg.initialPrompt == "" {
		return nil
	}
	capturedInst := inst
	capturedPrompt := msg.initialPrompt
	return func() tea.Msg {
		time.Sleep(4 * time.Second)
		if err := capturedInst.SendText(capturedPrompt); err != nil {
			uiLog.Warn("review_prompt_send_failed",
				slog.String("id", capturedInst.ID),
				slog.String("err", err.Error()))
		}
		return reviewPromptSentMsg{}
	}
}

// handleWorktreeFinishResult processes worktreeFinishResultMsg, removing the
// finished worktree session from all state and deleting linked todos.
func (h *Home) handleWorktreeFinishResult(msg worktreeFinishResultMsg) tea.Cmd {
	if msg.err != nil {
		// Show error in dialog (user can go back or cancel)
		if h.worktreeFinishDialog.IsVisible() {
			h.worktreeFinishDialog.SetError(msg.err.Error())
		} else {
			h.setError(msg.err)
		}
		return nil
	}

	// Success: remove session from instances and clean up
	h.worktreeFinishDialog.Hide()

	h.instancesMu.Lock()
	for i, s := range h.instances {
		if s.ID == msg.sessionID {
			h.instances = append(h.instances[:i], h.instances[i+1:]...)
			break
		}
	}
	inst := h.instanceByID[msg.sessionID]
	delete(h.instanceByID, msg.sessionID)
	h.instancesMu.Unlock()

	// Invalidate caches
	h.cachedStatusCounts.valid.Store(false)
	// Invalidate all UICache entries for the deleted session
	h.cache.InvalidateSession(msg.sessionID)
	h.logActivityMu.Lock()
	delete(h.lastLogActivity, msg.sessionID)
	h.logActivityMu.Unlock()

	// Remove from group tree and rebuild
	if inst != nil {
		h.groupTree.RemoveSession(inst)
	}
	h.rebuildFlatItems()
	h.search.SetItems(h.instances)

	// Delete from database and save
	if err := h.storage.DeleteInstance(msg.sessionID); err != nil {
		uiLog.Warn("worktree_finish_delete_err", slog.String("id", msg.sessionID), slog.String("err", err.Error()))
	}
	h.forceSaveInstances()

	// Delete the todo linked to this session (worktree finish = work complete)
	if err := h.storage.DeleteTodosForSession(msg.sessionID); err != nil {
		uiLog.Warn("delete_todo_for_session_err", slog.String("session", msg.sessionID), slog.String("err", err.Error()))
	}

	// Show success message
	h.setError(fmt.Errorf("Finished worktree '%s'", msg.sessionTitle))
	return nil
}

// handleTick processes tickMsg, performing all periodic maintenance tasks:
// error dismissal, animation cleanup, preview cache refresh, and PR status polling.
func (h *Home) handleTick(msg tickMsg) tea.Cmd {
	// Auto-dismiss errors after 5 seconds
	if h.err != nil && !h.errTime.IsZero() && time.Since(h.errTime) > 5*time.Second {
		h.clearError()
	}

	// PERFORMANCE: Detect when navigation has settled (300ms since last up/down)
	// This allows background updates to resume after rapid navigation stops
	const navigationSettleTime = 300 * time.Millisecond
	if h.isNavigating && time.Since(h.lastNavigationTime) > navigationSettleTime {
		h.isNavigating = false
	}

	// PERFORMANCE: Skip background updates during rapid navigation
	// This prevents subprocess spawning while user is scrolling through sessions
	if !h.isNavigating {
		// PERFORMANCE: Adaptive status updates - only when user is active
		// If user hasn't interacted for 2+ seconds, skip status updates.
		// This prevents background polling during idle periods.
		const userActivityWindow = 2 * time.Second
		if !h.lastUserInputTime.IsZero() && time.Since(h.lastUserInputTime) < userActivityWindow {
			// User is active - trigger status updates
			// NOTE: RefreshExistingSessions() moved to background worker (processStatusUpdate)
			// to avoid blocking the main goroutine with subprocess calls
			h.triggerStatusUpdate()
		}
		// User idle - no updates needed (cache refresh happens in background worker)
	}

	// Update animation frame for launching spinner (8 frames, cycles every tick)
	h.animationFrame = (h.animationFrame + 1) % 8

	// Periodic UI state save (every 5 ticks = ~10 seconds)
	h.uiStateSaveTicks++
	if h.uiStateSaveTicks >= 5 {
		h.uiStateSaveTicks = 0
		h.saveUIState()
	}

	// Fast log size check every 10 seconds (catches runaway logs before they cause issues)
	// This is much faster than full maintenance - just checks file sizes
	if time.Since(h.lastLogCheck) >= logCheckInterval {
		h.lastLogCheck = time.Now()
		go func() {
			logSettings := session.GetLogSettings()
			// Fast check - only truncate, no orphan cleanup
			_, _ = tmux.TruncateLargeLogFiles(logSettings.MaxSizeMB, logSettings.MaxLines)
		}()
	}

	// Prune stale caches and limiters every 20 seconds
	if time.Since(h.lastCachePrune) >= 20*time.Second {
		h.lastCachePrune = time.Now()
		h.pruneAnalyticsCache()

		// Prune dead pipes and connect new sessions
		if pm := tmux.GetPipeManager(); pm != nil {
			h.instancesMu.RLock()
			for _, inst := range h.instances {
				if ts := inst.GetTmuxSession(); ts != nil && ts.Exists() {
					if !pm.IsConnected(ts.Name) {
						go func(name string) {
							_ = pm.Connect(name)
						}(ts.Name)
					}
				}
			}
			h.instancesMu.RUnlock()
		}
	}

	// Full log maintenance (orphan cleanup, etc) every 5 minutes
	if time.Since(h.lastLogMaintenance) >= logMaintenanceInterval {
		h.lastLogMaintenance = time.Now()
		go func() {
			logSettings := session.GetLogSettings()
			tmux.RunLogMaintenance(logSettings.MaxSizeMB, logSettings.MaxLines, logSettings.RemoveOrphans)
		}()
	}

	// Clean up expired animation entries (launching, resuming, MCP loading, forking)
	// For Claude: remove after 20s timeout (animation shows for ~6-15s)
	// For others: remove after 5s timeout
	const claudeTimeout = 20 * time.Second
	const defaultTimeout = 5 * time.Second

	// Use consolidated cleanup helper for all animation maps
	// Note: cleanupExpiredAnimations accesses instanceByID which is thread-safe on main goroutine
	h.cleanupExpiredAnimations(h.launchingSessions, claudeTimeout, defaultTimeout)
	h.cleanupExpiredAnimations(h.resumingSessions, claudeTimeout, defaultTimeout)
	h.cleanupExpiredAnimations(h.forkingSessions, claudeTimeout, defaultTimeout)

	// Notification bar sync handled by background worker (syncNotificationsBackground)
	// which runs even when TUI is paused during tea.Exec

	// Fetch preview for currently selected session (if stale/missing and not fetching)
	// Cache expires after 2 seconds to show live terminal updates without excessive fetching
	var previewCmd tea.Cmd
	h.instancesMu.RLock()
	selected := h.getSelectedSession()
	h.instancesMu.RUnlock()
	if selected != nil {
		_, cachedTime, hasCached := h.cache.GetPreview(selected.ID)
		cacheExpired := !hasCached || time.Since(cachedTime) > previewCacheTTL
		// Only fetch if cache is stale/missing AND not currently fetching this session
		h.previewFetchingMu.Lock()
		if cacheExpired && h.previewFetchingID != selected.ID {
			h.previewFetchingID = selected.ID
			previewCmd = h.fetchPreview(selected)
		}
		h.previewFetchingMu.Unlock()
	}
	// PR fetch for currently selected worktree session (if missing or TTL expired)
	// Handles startup case where no navigation ever fires previewDebounceMsg
	var prCmd tea.Cmd
	if selected != nil && h.ghPath != "" && selected.IsWorktree() && selected.WorktreePath != "" {
		_, cachedAtPR, hasCachedPR := h.cache.HasPREntry(selected.ID)
		needsFetch := !hasCachedPR || time.Since(cachedAtPR) > prCacheTTL
		if needsFetch {
			h.cache.TouchPR(selected.ID) // Prevent duplicate fetches
		}
		if needsFetch {
			sid := selected.ID
			wtPath := selected.WorktreePath
			ghPath := h.ghPath
			prCmd = func() tea.Msg {
				return fetchPRInfo(sid, wtPath, ghPath)
			}
		}
	}
	return tea.Batch(h.tick(), previewCmd, prCmd)
}
