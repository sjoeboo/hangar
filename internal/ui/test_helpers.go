package ui

import "github.com/asheshgoplani/agent-deck/internal/session"

// Test helpers for width_test.go
// These are only compiled in test builds

// NewTestHome creates a minimal Home instance for testing
func NewTestHome() *Home {
	return &Home{}
}

// SetFlatItemsForTest sets the flatItems field for testing
func (h *Home) SetFlatItemsForTest(items []session.Item) {
	h.flatItems = items
}

// SetCursorForTest sets the cursor field for testing
func (h *Home) SetCursorForTest(cursor int) {
	h.cursor = cursor
}

// RenderPreviewPaneForTest exposes renderPreviewPane for testing
func (h *Home) RenderPreviewPaneForTest(width, height int) string {
	return h.renderPreviewPane(width, height)
}
