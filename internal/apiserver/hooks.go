package apiserver

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/sjoeboo/hangar/internal/session"
)

// hookPayload is the JSON body Claude Code sends for HTTP hook events.
type hookPayload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Matcher       string `json:"matcher,omitempty"`
}

// handleHook serves POST /hooks — the backward-compatible Claude Code webhook receiver.
func (s *APIServer) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	instanceID := r.Header.Get("X-Hangar-Instance-Id")
	if instanceID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16)) // 64KB max
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload hookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := session.MapEventToStatus(payload.HookEventName)

	// Notification events: only "waiting" for permission_prompt/elicitation_dialog
	if payload.HookEventName == "Notification" {
		if payload.Matcher == "permission_prompt" || payload.Matcher == "elicitation_dialog" {
			status = "waiting"
		}
	}

	if status != "" {
		s.watcher.Notify(instanceID, status, payload.SessionID, payload.HookEventName)
		select {
		case s.hub.broadcast <- WsMessage{
			Type: "hook_changed",
			Data: WsHookChangedData{
				InstanceID:    instanceID,
				HookEventName: payload.HookEventName,
				Status:        status,
			},
		}:
		default:
			// hub not running or full — skip broadcast
		}
	}

	w.WriteHeader(http.StatusOK)
}
