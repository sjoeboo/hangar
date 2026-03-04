package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sjoeboo/hangar/internal/session"
)

// handleTower implements the "hangar tower" command.
// By default it ensures the API server and Tower session are running, then
// exits — Tower runs in the background and is accessible via the TUI.
//
// Flags:
//   --attach / -a          Attach the terminal to the Tower tmux session.
//   --happy                Run Tower via "happy {command}" (remote access wrapper).
//   --wrapper <cmd>        Use a custom wrapper; {command} is replaced with the
//                          full claude invocation (e.g. "mytool {command}").
func handleTower(profile string, args []string) {
	attach := false
	wrapper := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--attach", "-a":
			attach = true
		case "--happy":
			wrapper = "happy {command}"
		case "--wrapper":
			if i+1 < len(args) {
				i++
				wrapper = args[i]
			}
		}
	}

	// Step 1: Ensure the API server is reachable (Tower's MCP server needs it).
	port, _ := webLoadConfig()
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/status", port)

	if !apiReachable(apiURL) {
		fmt.Println("Starting Hangar web server...")
		startWebInBackground()
		if !waitForAPI(apiURL, 5*time.Second) {
			fmt.Fprintln(os.Stderr, "Error: API server failed to start within 5 seconds.")
			fmt.Fprintln(os.Stderr, "Try running 'hangar web start' manually.")
			os.Exit(1)
		}
	}

	// Step 2: Load sessions and look for an existing Tower session.
	storage, instances, groupsData, err := loadSessionData(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var towerInst *session.Instance
	for _, inst := range instances {
		if inst.SessionType == "tower" {
			towerInst = inst
			break
		}
	}

	// Step 3: Create if not found, or update wrapper if it changed.
	needRestart := false
	if towerInst == nil {
		towerInst, err = createTowerSession(storage, instances, groupsData, wrapper)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Tower session: %v\n", err)
			os.Exit(1)
		}
		// Reload so the instance has a connected tmux session object.
		_, instances, _, err = loadSessionData(profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading sessions: %v\n", err)
			os.Exit(1)
		}
		for _, inst := range instances {
			if inst.SessionType == "tower" {
				towerInst = inst
				break
			}
		}
	} else if wrapper != "" && towerInst.Wrapper != wrapper {
		// Wrapper changed — update and restart so it takes effect.
		fmt.Printf("Updating Tower wrapper to: %s\n", wrapper)
		towerInst.Wrapper = wrapper
		if err := saveSessionData(storage, instances); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save wrapper change: %v\n", err)
		}
		if towerInst.Exists() {
			_ = towerInst.Kill()
			needRestart = true
		}
	}

	// Ensure Command is set (guards against sessions created before this fix).
	if towerInst.Command == "" {
		towerInst.Command = "claude"
	}

	// Step 4: Start if not running (or restart after wrapper change).
	_ = needRestart
	if !towerInst.Exists() {
		fmt.Println("Starting Tower session...")
		if err := towerInst.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting Tower session: %v\n", err)
			os.Exit(1)
		}
		towerInst.PostStartSync(3 * time.Second)
		if err := saveSessionData(storage, instances); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save session state: %v\n", err)
		}
	}

	// Step 5: Attach (only when --attach / -a is passed).
	if !attach {
		fmt.Printf("Tower is running (ID: %s). Use 'hangar tower --attach' or the TUI to connect.\n", towerInst.ID[:8])
		return
	}

	tmuxSess := towerInst.GetTmuxSession()
	if tmuxSess == nil {
		fmt.Fprintln(os.Stderr, "Error: Tower session has no tmux session.")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := tmuxSess.Attach(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error attaching to Tower: %v\n", err)
		os.Exit(1)
	}
}

// createTowerSession scaffolds the Tower directory, writes .mcp.json and
// CLAUDE.md, creates the session instance, and persists it.
func createTowerSession(storage *session.Storage, instances []*session.Instance, groupsData []*session.GroupData, wrapper string) (*session.Instance, error) {
	hangarDir, err := session.GetHangarDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get hangar dir: %w", err)
	}
	towerDir := filepath.Join(hangarDir, "tower")
	if err := os.MkdirAll(towerDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create tower dir: %w", err)
	}

	// Write .mcp.json so Claude Code discovers the Hangar MCP server.
	mcpJSON := `{
  "mcpServers": {
    "hangar": {
      "type": "stdio",
      "command": "hangar",
      "args": ["mcp-server"]
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(towerDir, ".mcp.json"), []byte(mcpJSON), 0600); err != nil {
		return nil, fmt.Errorf("failed to write .mcp.json: %w", err)
	}

	// Write .claude/settings.local.json — pre-approve the hangar MCP server and
	// hangar CLI commands so Tower never prompts for permission on its own tools.
	claudeDir := filepath.Join(towerDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create .claude dir: %w", err)
	}
	settingsJSON := `{
  "permissions": {
    "allow": [
      "Bash(hangar:*)"
    ]
  },
  "enableAllProjectMcpServers": true,
  "enabledMcpjsonServers": [
    "hangar"
  ]
}
`
	settingsPath := filepath.Join(claudeDir, "settings.local.json")
	// Only write if it doesn't exist — preserve any user customisations.
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		if err := os.WriteFile(settingsPath, []byte(settingsJSON), 0600); err != nil {
			return nil, fmt.Errorf("failed to write settings.local.json: %w", err)
		}
	}

	// Write CLAUDE.md — the system prompt that tells Claude it's Tower.
	claudeMD := `# Tower — Hangar Control Agent

You are Tower, a control agent for all AI coding sessions managed by Hangar.
You have no project of your own — your purpose is to oversee and assist
all other sessions.

## Your MCP Tools

You have access to Hangar tools for:
- Listing and inspecting all sessions (status, output, PR info)
- Sending messages/prompts to any running session
- Starting, stopping, and restarting sessions
- Creating new sessions
- Managing todos across projects

## How to Respond

- **"How are the agents?"** → call ` + "`hangar_list_sessions`" + `, present as a table:
  Title | Status | Tool | Project/Branch
- **"What is X working on?"** → call ` + "`hangar_get_session`" + ` + ` + "`hangar_get_output`" + `
- **"Send X a message"** → call ` + "`hangar_send_message`" + `; confirm first if ambiguous
- **"Create a session for Y"** → ask for path if not provided, then ` + "`hangar_create_session`" + `
- Keep responses concise; use tables for session lists
- Flag any sessions in "error" status prominently

## Self-Awareness

Your own session ID is available as $HANGAR_INSTANCE_ID. You are session_type="tower".
Do not send messages to yourself.
`
	if err := os.WriteFile(filepath.Join(towerDir, "CLAUDE.md"), []byte(claudeMD), 0644); err != nil {
		return nil, fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	// Create the Instance.
	inst := session.NewInstance("Tower", towerDir)
	inst.Tool = "claude"
	inst.Command = "claude" // Required: buildClaudeCommand only fires when baseCommand == "claude"
	inst.Wrapper = wrapper  // e.g. "happy {command}" for remote access; empty = direct
	inst.SessionType = "tower"
	inst.GroupPath = "" // Tower lives at the root level, pinned to top by sort.

	instances = append(instances, inst)
	groupTree := session.NewGroupTreeWithGroups(instances, groupsData)
	if err := storage.SaveWithGroups(instances, groupTree); err != nil {
		return nil, fmt.Errorf("failed to save tower session: %w", err)
	}

	fmt.Printf("Created Tower session (ID: %s)\n", inst.ID[:8])
	return inst, nil
}

// apiReachable checks whether the Hangar API responds within 1 second.
func apiReachable(url string) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// startWebInBackground spawns "hangar web start" as a detached background process.
func startWebInBackground() {
	cmd := exec.Command("hangar", "web", "start")
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Start detached so it outlives us if we exec into tmux.
	_ = cmd.Start()
	// Don't wait — we'll poll for readiness.
}

// waitForAPI polls the API endpoint until it responds or the timeout expires.
func waitForAPI(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if apiReachable(url) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
