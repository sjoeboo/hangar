package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// handleSession dispatches session subcommands
func handleSession(profile string, args []string) {
	if len(args) == 0 {
		printSessionHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "start":
		handleSessionStart(profile, args[1:])
	case "stop":
		handleSessionStop(profile, args[1:])
	case "restart":
		handleSessionRestart(profile, args[1:])
	case "fork":
		handleSessionFork(profile, args[1:])
	case "attach":
		handleSessionAttach(profile, args[1:])
	case "show":
		handleSessionShow(profile, args[1:])
	case "help", "--help", "-h":
		printSessionHelp()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown session command: %s\n", args[0])
		printSessionHelp()
		os.Exit(1)
	}
}

// printSessionHelp prints help for session commands
func printSessionHelp() {
	fmt.Println("Usage: agent-deck session <command> [options]")
	fmt.Println()
	fmt.Println("Manage individual sessions.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start <id>        Start a session's tmux process")
	fmt.Println("  stop <id>         Stop/kill session process")
	fmt.Println("  restart <id>      Restart session (Claude: reload MCPs)")
	fmt.Println("  fork <id>         Fork Claude session with context")
	fmt.Println("  attach <id>       Attach to session interactively")
	fmt.Println("  show [id]         Show session details (auto-detect current if no id)")
	fmt.Println()
	fmt.Println("Global Options:")
	fmt.Println("  -p, --profile <name>   Use specific profile")
	fmt.Println("  --json                 Output as JSON")
	fmt.Println("  -q, --quiet            Minimal output (exit codes only)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  agent-deck session start my-project")
	fmt.Println("  agent-deck session stop abc123")
	fmt.Println("  agent-deck session restart my-project")
	fmt.Println("  agent-deck session fork my-project -t \"my-project-fork\"")
	fmt.Println("  agent-deck session attach my-project")
	fmt.Println("  agent-deck session show                  # Auto-detect current session")
	fmt.Println("  agent-deck session show my-project --json")
}

// handleSessionStart starts a session's tmux process
func handleSessionStart(profile string, args []string) {
	fs := flag.NewFlagSet("session start", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Minimal output")
	quietShort := fs.Bool("q", false, "Minimal output (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session start <id|title> [options]")
		fmt.Println()
		fmt.Println("Start a session's tmux process.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	quietMode := *quiet || *quietShort
	out := NewCLIOutput(*jsonOutput, quietMode)

	// Load sessions
	storage, instances, _, err := loadSessionData(profile)
	if err != nil {
		out.Error(err.Error(), ErrCodeNotFound)
		os.Exit(1)
	}

	// Resolve session
	inst, errMsg, errCode := ResolveSession(identifier, instances)
	if inst == nil {
		out.Error(errMsg, errCode)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Check if already running
	if inst.Exists() {
		out.Error(fmt.Sprintf("session '%s' is already running", inst.Title), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Start the session
	if err := inst.Start(); err != nil {
		out.Error(fmt.Sprintf("failed to start session: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Save updated state
	if err := saveSessionData(storage, instances); err != nil {
		out.Error(fmt.Sprintf("failed to save session state: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Output success
	jsonData := map[string]interface{}{
		"success": true,
		"id":      inst.ID,
		"title":   inst.Title,
	}
	if tmuxSess := inst.GetTmuxSession(); tmuxSess != nil {
		jsonData["tmux"] = tmuxSess.Name
	}
	out.Success(fmt.Sprintf("Started session: %s", inst.Title), jsonData)
}

// handleSessionStop stops a session process
func handleSessionStop(profile string, args []string) {
	fs := flag.NewFlagSet("session stop", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Minimal output")
	quietShort := fs.Bool("q", false, "Minimal output (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session stop <id|title> [options]")
		fmt.Println()
		fmt.Println("Stop/kill a session's process (tmux session remains).")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	quietMode := *quiet || *quietShort
	out := NewCLIOutput(*jsonOutput, quietMode)

	// Load sessions
	storage, instances, _, err := loadSessionData(profile)
	if err != nil {
		out.Error(err.Error(), ErrCodeNotFound)
		os.Exit(1)
	}

	// Resolve session
	inst, errMsg, errCode := ResolveSession(identifier, instances)
	if inst == nil {
		out.Error(errMsg, errCode)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Check if not running
	if !inst.Exists() {
		out.Error(fmt.Sprintf("session '%s' is not running", inst.Title), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Stop the session by killing the tmux session
	if err := inst.Kill(); err != nil {
		out.Error(fmt.Sprintf("failed to stop session: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Save updated state
	if err := saveSessionData(storage, instances); err != nil {
		out.Error(fmt.Sprintf("failed to save session state: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Output success
	out.Success(fmt.Sprintf("Stopped session: %s", inst.Title), map[string]interface{}{
		"success": true,
		"id":      inst.ID,
		"title":   inst.Title,
	})
}

// handleSessionRestart restarts a session
func handleSessionRestart(profile string, args []string) {
	fs := flag.NewFlagSet("session restart", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Minimal output")
	quietShort := fs.Bool("q", false, "Minimal output (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session restart <id|title> [options]")
		fmt.Println()
		fmt.Println("Restart a session. For Claude sessions, this reloads MCPs.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	quietMode := *quiet || *quietShort
	out := NewCLIOutput(*jsonOutput, quietMode)

	// Load sessions
	storage, instances, _, err := loadSessionData(profile)
	if err != nil {
		out.Error(err.Error(), ErrCodeNotFound)
		os.Exit(1)
	}

	// Resolve session
	inst, errMsg, errCode := ResolveSession(identifier, instances)
	if inst == nil {
		out.Error(errMsg, errCode)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Restart the session
	if err := inst.Restart(); err != nil {
		out.Error(fmt.Sprintf("failed to restart session: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Save updated state
	if err := saveSessionData(storage, instances); err != nil {
		out.Error(fmt.Sprintf("failed to save session state: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Output success
	out.Success(fmt.Sprintf("Restarted session: %s", inst.Title), map[string]interface{}{
		"success": true,
		"id":      inst.ID,
		"title":   inst.Title,
	})
}

// handleSessionFork forks a Claude session
func handleSessionFork(profile string, args []string) {
	fs := flag.NewFlagSet("session fork", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Minimal output")
	quietShort := fs.Bool("q", false, "Minimal output (short)")
	title := fs.String("title", "", "Title for forked session")
	titleShort := fs.String("t", "", "Title for forked session (short)")
	group := fs.String("group", "", "Group for forked session")
	groupShort := fs.String("g", "", "Group for forked session (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session fork <id|title> [options]")
		fmt.Println()
		fmt.Println("Fork a Claude session with conversation context.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  agent-deck session fork my-project")
		fmt.Println("  agent-deck session fork my-project -t \"my-fork\"")
		fmt.Println("  agent-deck session fork my-project -t \"my-fork\" -g \"experiments\"")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	quietMode := *quiet || *quietShort
	out := NewCLIOutput(*jsonOutput, quietMode)

	// Merge short and long flags
	forkTitle := mergeFlags(*title, *titleShort)
	forkGroup := mergeFlags(*group, *groupShort)

	// Load sessions
	storage, instances, groupsData, err := loadSessionData(profile)
	if err != nil {
		out.Error(err.Error(), ErrCodeNotFound)
		os.Exit(1)
	}

	// Resolve session
	inst, errMsg, errCode := ResolveSession(identifier, instances)
	if inst == nil {
		out.Error(errMsg, errCode)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Verify it's a Claude session
	if inst.Tool != "claude" {
		out.Error(fmt.Sprintf("session '%s' is not a Claude session (tool: %s)", inst.Title, inst.Tool), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Verify it can be forked
	if !inst.CanFork() {
		out.Error(fmt.Sprintf("session '%s' cannot be forked: no active Claude session ID", inst.Title), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Default title if not provided
	if forkTitle == "" {
		forkTitle = inst.Title + "-fork"
	}

	// Default group to parent's group
	if forkGroup == "" {
		forkGroup = inst.GroupPath
	}

	// Create the forked instance
	forkedInst, _, err := inst.CreateForkedInstance(forkTitle, forkGroup)
	if err != nil {
		out.Error(fmt.Sprintf("failed to create fork: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Start the forked session
	if err := forkedInst.Start(); err != nil {
		out.Error(fmt.Sprintf("failed to start forked session: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Add to instances
	instances = append(instances, forkedInst)

	// Rebuild group tree and ensure group exists
	groupTree := session.NewGroupTreeWithGroups(instances, groupsData)
	if forkedInst.GroupPath != "" {
		groupTree.CreateGroup(forkedInst.GroupPath)
	}

	// Save
	if err := storage.SaveWithGroups(instances, groupTree); err != nil {
		out.Error(fmt.Sprintf("failed to save: %v", err), ErrCodeInvalidOperation)
		os.Exit(1)
	}

	// Output success
	out.Success(fmt.Sprintf("Forked session: %s -> %s (%s)", inst.Title, forkedInst.Title, TruncateID(forkedInst.ID)), map[string]interface{}{
		"success":   true,
		"parent_id": inst.ID,
		"new_id":    forkedInst.ID,
		"new_title": forkedInst.Title,
	})
}

// handleSessionAttach attaches to a session interactively
func handleSessionAttach(profile string, args []string) {
	fs := flag.NewFlagSet("session attach", flag.ExitOnError)

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session attach <id|title>")
		fmt.Println()
		fmt.Println("Attach to a session interactively.")
		fmt.Println("Press Ctrl+Q to detach.")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)

	// Load sessions
	_, instances, _, err := loadSessionData(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Resolve session (allow current session detection)
	inst, errMsg, errCode := ResolveSessionOrCurrent(identifier, instances)
	if inst == nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Check if session exists
	if !inst.Exists() {
		fmt.Fprintf(os.Stderr, "Error: session '%s' is not running\n", inst.Title)
		os.Exit(1)
	}

	// Attach to the session
	tmuxSession := inst.GetTmuxSession()
	if tmuxSession == nil {
		fmt.Fprintf(os.Stderr, "Error: no tmux session for '%s'\n", inst.Title)
		os.Exit(1)
	}

	// Create context for attach
	ctx := context.Background()

	if err := tmuxSession.Attach(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to attach: %v\n", err)
		os.Exit(1)
	}
}

// handleSessionShow shows session details
func handleSessionShow(profile string, args []string) {
	fs := flag.NewFlagSet("session show", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	quiet := fs.Bool("quiet", false, "Minimal output")
	quietShort := fs.Bool("q", false, "Minimal output (short)")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck session show [id|title] [options]")
		fmt.Println()
		fmt.Println("Show session details. If no ID is provided, auto-detects current session.")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	identifier := fs.Arg(0)
	quietMode := *quiet || *quietShort
	out := NewCLIOutput(*jsonOutput, quietMode)

	// Load sessions
	_, instances, _, err := loadSessionData(profile)
	if err != nil {
		out.Error(err.Error(), ErrCodeNotFound)
		os.Exit(1)
	}

	// Resolve session (allow current session detection)
	inst, errMsg, errCode := ResolveSessionOrCurrent(identifier, instances)
	if inst == nil {
		out.Error(errMsg, errCode)
		if errCode == ErrCodeNotFound {
			os.Exit(2)
		}
		os.Exit(1)
	}

	// Update status
	_ = inst.UpdateStatus()

	// Get MCP info if Claude session
	var mcpInfo *session.MCPInfo
	if inst.Tool == "claude" {
		mcpInfo = inst.GetMCPInfo()
	}

	// Prepare JSON output
	jsonData := map[string]interface{}{
		"id":         inst.ID,
		"title":      inst.Title,
		"status":     StatusString(inst.Status),
		"path":       inst.ProjectPath,
		"group":      inst.GroupPath,
		"tool":       inst.Tool,
		"created_at": inst.CreatedAt.Format(time.RFC3339),
	}

	if inst.Command != "" {
		jsonData["command"] = inst.Command
	}

	if inst.Tool == "claude" {
		jsonData["claude_session_id"] = inst.ClaudeSessionID
		jsonData["can_fork"] = inst.CanFork()
		jsonData["can_restart"] = inst.CanRestart()

		if mcpInfo != nil && mcpInfo.HasAny() {
			jsonData["mcps"] = map[string]interface{}{
				"local":   mcpInfo.Local,
				"global":  mcpInfo.Global,
				"project": mcpInfo.Project,
			}
		}
	}

	if inst.Exists() {
		tmuxSession := inst.GetTmuxSession()
		if tmuxSession != nil {
			jsonData["tmux_session"] = tmuxSession.Name
		}
	}

	// Build human-readable output
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Session: %s\n", inst.Title))
	sb.WriteString(fmt.Sprintf("ID:      %s\n", inst.ID))
	sb.WriteString(fmt.Sprintf("Status:  %s %s\n", StatusSymbol(inst.Status), StatusString(inst.Status)))
	sb.WriteString(fmt.Sprintf("Path:    %s\n", FormatPath(inst.ProjectPath)))

	if inst.GroupPath != "" {
		sb.WriteString(fmt.Sprintf("Group:   %s\n", inst.GroupPath))
	}

	sb.WriteString(fmt.Sprintf("Tool:    %s\n", inst.Tool))

	if inst.Command != "" {
		sb.WriteString(fmt.Sprintf("Command: %s\n", inst.Command))
	}

	if inst.Tool == "claude" {
		if inst.ClaudeSessionID != "" {
			truncatedID := inst.ClaudeSessionID
			if len(truncatedID) > 36 {
				truncatedID = truncatedID[:36] + "..."
			}
			canForkStr := "no"
			if inst.CanFork() {
				canForkStr = "yes"
			}
			sb.WriteString(fmt.Sprintf("Claude:  session_id=%s (can fork: %s)\n", truncatedID, canForkStr))
		} else {
			sb.WriteString("Claude:  no session ID detected\n")
		}

		if mcpInfo != nil && mcpInfo.HasAny() {
			var mcpParts []string
			for _, name := range mcpInfo.Local {
				mcpParts = append(mcpParts, name+" (local)")
			}
			for _, name := range mcpInfo.Global {
				mcpParts = append(mcpParts, name+" (global)")
			}
			for _, name := range mcpInfo.Project {
				mcpParts = append(mcpParts, name+" (project)")
			}
			sb.WriteString(fmt.Sprintf("MCPs:    %s\n", strings.Join(mcpParts, ", ")))
		}
	}

	sb.WriteString(fmt.Sprintf("Created: %s\n", inst.CreatedAt.Format("2006-01-02 15:04:05")))

	if !inst.LastAccessedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("Accessed: %s\n", inst.LastAccessedAt.Format("2006-01-02 15:04:05")))
	}

	if inst.Exists() {
		tmuxSession := inst.GetTmuxSession()
		if tmuxSession != nil {
			sb.WriteString(fmt.Sprintf("Tmux:    %s\n", tmuxSession.Name))
		}
	}

	out.Print(sb.String(), jsonData)
}

// loadSessionData loads storage and session data for a profile
// The Storage.LoadWithGroups() method already handles tmux reconnection internally
func loadSessionData(profile string) (*session.Storage, []*session.Instance, []*session.GroupData, error) {
	storage, err := session.NewStorageWithProfile(profile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	instances, groupsData, err := storage.LoadWithGroups()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load sessions: %w", err)
	}

	// LoadWithGroups already reconnects tmux sessions and updates status

	return storage, instances, groupsData, nil
}

// saveSessionData saves session data with groups
func saveSessionData(storage *session.Storage, instances []*session.Instance) error {
	// Rebuild group tree from instances
	groupTree := session.NewGroupTree(instances)
	return storage.SaveWithGroups(instances, groupTree)
}
