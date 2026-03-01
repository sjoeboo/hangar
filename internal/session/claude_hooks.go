package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// hangarHookCommand is the marker command used to identify hangar hooks in settings.json.
const hangarHookCommand = "hangar hook-handler"

// hangarHTTPHookURL is the URL template for the embedded HTTP hook server.
const hangarHTTPHookURL = "http://127.0.0.1:%d/hooks"

// hangarHTTPHookRE matches URLs of the form http://127.0.0.1:PORT/hooks (exact path, no subpaths).
var hangarHTTPHookRE = regexp.MustCompile(`^http://127\.0\.0\.1:\d{1,5}/hooks$`)

// claudeHookEntry represents a single hook entry in Claude Code settings.
type claudeHookEntry struct {
	Type           string            `json:"type"`
	Command        string            `json:"command,omitempty"`
	Async          bool              `json:"async,omitempty"`
	URL            string            `json:"url,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	AllowedEnvVars []string          `json:"allowedEnvVars,omitempty"`
	Timeout        int               `json:"timeout,omitempty"`
}

// claudeHookMatcher represents a matcher block (with optional matcher pattern) in settings.
type claudeHookMatcher struct {
	Matcher string            `json:"matcher,omitempty"`
	Hooks   []claudeHookEntry `json:"hooks"`
}

// hangarHook returns the standard hangar command hook entry.
func hangarHook() claudeHookEntry {
	return claudeHookEntry{
		Type:    "command",
		Command: hangarHookCommand,
		Async:   true,
	}
}

// hangarHTTPHook returns an HTTP hook entry for the given port.
func hangarHTTPHook(port int) claudeHookEntry {
	return claudeHookEntry{
		Type:           "http",
		URL:            fmt.Sprintf(hangarHTTPHookURL, port),
		Headers:        map[string]string{"X-Hangar-Instance-Id": "$HANGAR_INSTANCE_ID"},
		AllowedEnvVars: []string{"HANGAR_INSTANCE_ID"},
		Timeout:        5,
	}
}

// isHangarHook reports whether h is a hangar-managed hook entry (either command or HTTP type).
func isHangarHook(h claudeHookEntry) bool {
	if h.Type == "http" {
		return hangarHTTPHookRE.MatchString(h.URL)
	}
	return strings.Contains(h.Command, hangarHookCommand)
}

// isOrphanedHTTPHook reports whether h is an invalid http hook with no URL,
// written by older buggy versions of hangar.
func isOrphanedHTTPHook(h claudeHookEntry) bool {
	return h.Type == "http" && h.URL == ""
}

// hookEventConfigs defines which Claude Code events we subscribe to and their matcher patterns.
var hookEventConfigs = []struct {
	Event   string
	Matcher string // empty = no matcher
}{
	{Event: "SessionStart"},
	{Event: "UserPromptSubmit"},
	{Event: "Stop"},
	{Event: "PermissionRequest"},
	{Event: "Notification", Matcher: "permission_prompt|elicitation_dialog"},
	{Event: "SessionEnd"},
}

// InjectClaudeHooks injects hangar hook entries into Claude Code's settings.json.
// Uses read-preserve-modify-write pattern to preserve all existing settings and user hooks.
// When port > 0, injects HTTP hooks (upgrading any existing command hooks first).
// When port == 0, injects command hooks.
// Returns true if hooks were newly installed or upgraded, false if already present with correct type.
func InjectClaudeHooks(configDir string, port int) (bool, error) {
	settingsPath := filepath.Join(configDir, "settings.json")

	// Read existing settings (or start fresh)
	var rawSettings map[string]json.RawMessage
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, fmt.Errorf("read settings.json: %w", err)
		}
		rawSettings = make(map[string]json.RawMessage)
	} else {
		if err := json.Unmarshal(data, &rawSettings); err != nil {
			return false, fmt.Errorf("parse settings.json: %w", err)
		}
	}

	// Parse existing hooks section
	var existingHooks map[string]json.RawMessage
	if raw, ok := rawSettings["hooks"]; ok {
		if err := json.Unmarshal(raw, &existingHooks); err != nil {
			// hooks key exists but isn't a valid object; start fresh for hooks
			existingHooks = make(map[string]json.RawMessage)
		}
	} else {
		existingHooks = make(map[string]json.RawMessage)
	}

	if port > 0 {
		// HTTP mode: check if HTTP hooks already installed (idempotent)
		if httpHooksAlreadyInstalled(existingHooks) {
			// Still clean up any orphaned entries that may have been left by older versions.
			if orphanedHTTPHooksPresent(existingHooks) {
				for _, cfg := range hookEventConfigs {
					if raw, ok := existingHooks[cfg.Event]; ok {
						if cleaned, removed := removeOrphanedFromEvent(raw); removed {
							if cleaned == nil {
								delete(existingHooks, cfg.Event)
							} else {
								existingHooks[cfg.Event] = cleaned
							}
						}
					}
				}
				// Write the cleaned-up settings back even though no new hooks are needed.
				hooksRaw, err := json.Marshal(existingHooks)
				if err != nil {
					return false, fmt.Errorf("marshal hooks: %w", err)
				}
				rawSettings["hooks"] = hooksRaw
				finalData, err := json.MarshalIndent(rawSettings, "", "  ")
				if err != nil {
					return false, fmt.Errorf("marshal settings: %w", err)
				}
				if err := os.MkdirAll(configDir, 0755); err != nil {
					return false, fmt.Errorf("create config dir: %w", err)
				}
				tmpPath := settingsPath + ".tmp"
				if err := os.WriteFile(tmpPath, finalData, 0644); err != nil {
					return false, fmt.Errorf("write settings.json.tmp: %w", err)
				}
				if err := os.Rename(tmpPath, settingsPath); err != nil {
					os.Remove(tmpPath)
					return false, fmt.Errorf("rename settings.json: %w", err)
				}
			}
			return false, nil
		}
		// Remove any existing command hooks (upgrade path)
		if commandHooksPresent(existingHooks) {
			for _, cfg := range hookEventConfigs {
				if raw, ok := existingHooks[cfg.Event]; ok {
					cleaned, _ := removeHangarFromEvent(raw)
					if cleaned == nil {
						delete(existingHooks, cfg.Event)
					} else {
						existingHooks[cfg.Event] = cleaned
					}
				}
			}
		}
		// Remove any orphaned http entries from older buggy versions.
		if orphanedHTTPHooksPresent(existingHooks) {
			for _, cfg := range hookEventConfigs {
				if raw, ok := existingHooks[cfg.Event]; ok {
					if cleaned, removed := removeOrphanedFromEvent(raw); removed {
						if cleaned == nil {
							delete(existingHooks, cfg.Event)
						} else {
							existingHooks[cfg.Event] = cleaned
						}
					}
				}
			}
		}
	} else {
		// Command mode: check if command hooks already installed (idempotent)
		if commandHooksAlreadyInstalled(existingHooks) {
			return false, nil
		}
	}

	// Choose the hook entry based on port
	hookEntry := hangarHook()
	if port > 0 {
		hookEntry = hangarHTTPHook(port)
	}

	// Inject our hook entries for each event
	for _, cfg := range hookEventConfigs {
		existingHooks[cfg.Event] = mergeHookEvent(existingHooks[cfg.Event], cfg.Matcher, hookEntry)
	}

	// Marshal hooks back into raw settings
	hooksRaw, err := json.Marshal(existingHooks)
	if err != nil {
		return false, fmt.Errorf("marshal hooks: %w", err)
	}
	rawSettings["hooks"] = hooksRaw

	// Atomic write
	finalData, err := json.MarshalIndent(rawSettings, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal settings: %w", err)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}

	tmpPath := settingsPath + ".tmp"
	if err := os.WriteFile(tmpPath, finalData, 0644); err != nil {
		return false, fmt.Errorf("write settings.json.tmp: %w", err)
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("rename settings.json: %w", err)
	}

	sessionLog.Info("claude_hooks_installed", slog.String("config_dir", configDir))
	return true, nil
}

// RemoveClaudeHooks removes hangar hook entries from Claude Code's settings.json.
// Returns true if hooks were removed, false if none found.
func RemoveClaudeHooks(configDir string) (bool, error) {
	settingsPath := filepath.Join(configDir, "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read settings.json: %w", err)
	}

	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return false, fmt.Errorf("parse settings.json: %w", err)
	}

	hooksRaw, ok := rawSettings["hooks"]
	if !ok {
		return false, nil
	}

	var existingHooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &existingHooks); err != nil {
		return false, nil
	}

	removed := false
	for _, cfg := range hookEventConfigs {
		if raw, ok := existingHooks[cfg.Event]; ok {
			cleaned, didRemove := removeHangarFromEvent(raw)
			if didRemove {
				removed = true
				if cleaned == nil {
					delete(existingHooks, cfg.Event)
				} else {
					existingHooks[cfg.Event] = cleaned
				}
			}
		}
	}

	if !removed {
		return false, nil
	}

	// If hooks map is empty, remove the key entirely
	if len(existingHooks) == 0 {
		delete(rawSettings, "hooks")
	} else {
		hooksData, _ := json.Marshal(existingHooks)
		rawSettings["hooks"] = hooksData
	}

	finalData, err := json.MarshalIndent(rawSettings, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal settings: %w", err)
	}

	tmpPath := settingsPath + ".tmp"
	if err := os.WriteFile(tmpPath, finalData, 0644); err != nil {
		return false, fmt.Errorf("write settings.json.tmp: %w", err)
	}
	if err := os.Rename(tmpPath, settingsPath); err != nil {
		os.Remove(tmpPath)
		return false, fmt.Errorf("rename settings.json: %w", err)
	}

	sessionLog.Info("claude_hooks_removed", slog.String("config_dir", configDir))
	return true, nil
}

// loadHooksMap reads settings.json from configDir and returns the parsed hooks map,
// or nil if the file does not exist, cannot be read, or has no "hooks" key.
func loadHooksMap(configDir string) map[string]json.RawMessage {
	settingsPath := filepath.Join(configDir, "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}
	var rawSettings map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawSettings); err != nil {
		return nil
	}
	hooksRaw, ok := rawSettings["hooks"]
	if !ok {
		return nil
	}
	var hooks map[string]json.RawMessage
	if err := json.Unmarshal(hooksRaw, &hooks); err != nil {
		return nil
	}
	return hooks
}

// CheckClaudeHooksInstalled checks if hangar hooks are present in settings.json.
func CheckClaudeHooksInstalled(configDir string) bool {
	return hooksAlreadyInstalled(loadHooksMap(configDir))
}

// CheckClaudeHTTPHooksInstalled returns true if HTTP hooks (not command hooks) are installed.
func CheckClaudeHTTPHooksInstalled(configDir string) bool {
	return httpHooksAlreadyInstalled(loadHooksMap(configDir))
}

// hooksAlreadyInstalled checks if all required hangar hooks (any type) are present.
func hooksAlreadyInstalled(hooks map[string]json.RawMessage) bool {
	for _, cfg := range hookEventConfigs {
		raw, ok := hooks[cfg.Event]
		if !ok {
			return false
		}
		if !eventHasHangarHook(raw) {
			return false
		}
	}
	return true
}

// commandHooksAlreadyInstalled checks if all required hangar command hooks are present.
// Unlike hooksAlreadyInstalled, this only matches command-type entries.
func commandHooksAlreadyInstalled(hooks map[string]json.RawMessage) bool {
	for _, cfg := range hookEventConfigs {
		raw, ok := hooks[cfg.Event]
		if !ok {
			return false
		}
		if !eventHasCommandHook(raw) {
			return false
		}
	}
	return true
}

// eventHasCommandHook checks if an event has a hangar command hook entry specifically.
func eventHasCommandHook(raw json.RawMessage) bool {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return false
	}
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, hangarHookCommand) {
				return true
			}
		}
	}
	return false
}

// httpHooksAlreadyInstalled checks if all events have HTTP hook entries.
func httpHooksAlreadyInstalled(hooks map[string]json.RawMessage) bool {
	for _, cfg := range hookEventConfigs {
		raw, ok := hooks[cfg.Event]
		if !ok {
			return false
		}
		if !eventHasHTTPHook(raw) {
			return false
		}
	}
	return true
}

// orphanedHTTPHooksPresent reports whether any event has an orphaned http hook (no URL).
func orphanedHTTPHooksPresent(hooks map[string]json.RawMessage) bool {
	for _, cfg := range hookEventConfigs {
		raw, ok := hooks[cfg.Event]
		if !ok {
			continue
		}
		var matchers []claudeHookMatcher
		if err := json.Unmarshal(raw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			for _, h := range m.Hooks {
				if isOrphanedHTTPHook(h) {
					return true
				}
			}
		}
	}
	return false
}

// removeOrphanedFromEvent removes any orphaned http hook entries (no URL) from a single event.
func removeOrphanedFromEvent(raw json.RawMessage) (json.RawMessage, bool) {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return raw, false
	}

	removed := false
	var cleaned []claudeHookMatcher
	for _, m := range matchers {
		var hooks []claudeHookEntry
		for _, h := range m.Hooks {
			if isOrphanedHTTPHook(h) {
				removed = true
				continue
			}
			hooks = append(hooks, h)
		}
		if len(hooks) > 0 {
			m.Hooks = hooks
			cleaned = append(cleaned, m)
		}
	}
	if !removed {
		return raw, false
	}
	if len(cleaned) == 0 {
		return nil, true
	}
	result, _ := json.Marshal(cleaned)
	return result, true
}

// commandHooksPresent reports whether any event has a hangar command hook.
func commandHooksPresent(hooks map[string]json.RawMessage) bool {
	for _, cfg := range hookEventConfigs {
		raw, ok := hooks[cfg.Event]
		if !ok {
			continue
		}
		var matchers []claudeHookMatcher
		if err := json.Unmarshal(raw, &matchers); err != nil {
			continue
		}
		for _, m := range matchers {
			for _, h := range m.Hooks {
				if strings.Contains(h.Command, hangarHookCommand) {
					return true
				}
			}
		}
	}
	return false
}

// eventHasHangarHook checks if a hook event's matcher array contains any hangar hook (command or HTTP).
func eventHasHangarHook(raw json.RawMessage) bool {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return false
	}
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if isHangarHook(h) {
				return true
			}
		}
	}
	return false
}

// eventHasHTTPHook checks if an event has a hangar HTTP hook entry.
func eventHasHTTPHook(raw json.RawMessage) bool {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return false
	}
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if h.Type == "http" && hangarHTTPHookRE.MatchString(h.URL) {
				return true
			}
		}
	}
	return false
}

// mergeHookEvent adds a hook entry to an existing event's matcher array.
// Preserves all existing matchers and hooks.
func mergeHookEvent(existing json.RawMessage, matcher string, hook claudeHookEntry) json.RawMessage {
	var matchers []claudeHookMatcher

	if existing != nil {
		if err := json.Unmarshal(existing, &matchers); err != nil {
			matchers = nil
		}
	}

	// Check if we already have a matcher entry with our hook
	for i, m := range matchers {
		if m.Matcher == matcher {
			// Check if our hook is already in this matcher
			for _, h := range m.Hooks {
				if isHangarHook(h) {
					// Already present
					result, _ := json.Marshal(matchers)
					return result
				}
			}
			// Append our hook to existing matcher
			matchers[i].Hooks = append(matchers[i].Hooks, hook)
			result, _ := json.Marshal(matchers)
			return result
		}
	}

	// No matching matcher found; add a new one
	newMatcher := claudeHookMatcher{
		Matcher: matcher,
		Hooks:   []claudeHookEntry{hook},
	}
	matchers = append(matchers, newMatcher)
	result, _ := json.Marshal(matchers)
	return result
}

var versionRegexp = regexp.MustCompile(`(?:^|[^\d])v?(\d+)\.(\d+)\.(\d+)\b`)

// parseClaudeVersion extracts the semver string from `claude --version` output.
func parseClaudeVersion(output string) (string, error) {
	m := versionRegexp.FindStringSubmatch(strings.TrimSpace(output))
	if m == nil {
		return "", fmt.Errorf("no semver found in %q", output)
	}
	return m[1] + "." + m[2] + "." + m[3], nil
}

// versionAtLeast reports whether version string (e.g. "2.1.63") is >= major.minor.patch.
func versionAtLeast(version string, major, minor, patch int) bool {
	m := versionRegexp.FindStringSubmatch(version)
	if m == nil {
		return false
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	if maj != major {
		return maj > major
	}
	if min != minor {
		return min > minor
	}
	return pat >= patch
}

// claudeSupportsHTTPHooks reports whether the given version supports type:"http" hooks.
// HTTP hooks were introduced in Claude Code 2.1.63.
func claudeSupportsHTTPHooks(version string) bool {
	return versionAtLeast(version, 2, 1, 63)
}

// DetectClaudeVersion runs `claude --version` and returns the parsed semver string.
// Returns empty string and an error if the version cannot be determined.
func DetectClaudeVersion() (string, error) {
	out, err := exec.Command("claude", "--version").Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("claude --version: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("claude --version: %w", err)
	}
	return parseClaudeVersion(string(out))
}

// removeHangarFromEvent removes hangar hook entries from an event's matcher array.
// Returns cleaned JSON and whether any removal happened. Returns nil JSON if the array is empty.
func removeHangarFromEvent(raw json.RawMessage) (json.RawMessage, bool) {
	var matchers []claudeHookMatcher
	if err := json.Unmarshal(raw, &matchers); err != nil {
		return raw, false
	}

	removed := false
	var cleaned []claudeHookMatcher

	for _, m := range matchers {
		var hooks []claudeHookEntry
		for _, h := range m.Hooks {
			if isHangarHook(h) {
				removed = true
				continue
			}
			hooks = append(hooks, h)
		}
		if len(hooks) > 0 {
			m.Hooks = hooks
			cleaned = append(cleaned, m)
		}
		// If len(hooks) == 0, the matcher had only hangar hooks — drop it entirely.
		// (removed is already true from the inner loop above)
	}

	if !removed {
		return raw, false
	}

	if len(cleaned) == 0 {
		return nil, true
	}

	result, _ := json.Marshal(cleaned)
	return result, true
}
