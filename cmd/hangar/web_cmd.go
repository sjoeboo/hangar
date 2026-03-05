package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sjoeboo/hangar/internal/apiserver"
	"github.com/sjoeboo/hangar/internal/pr"
	"github.com/sjoeboo/hangar/internal/session"
)

// handleWeb dispatches the "web" subcommand.
// With no arguments, it prints status (same as "web status").
func handleWeb(profile string, args []string) {
	sub := "status"
	rest := args
	if len(args) > 0 {
		sub = args[0]
		rest = args[1:]
	}
	switch sub {
	case "start":
		fs := flag.NewFlagSet("web start", flag.ExitOnError)
		noOpen := fs.Bool("no-open", false, "Do not open browser after start")
		detach := fs.Bool("detach", false, "Run server in background as a daemon")
		fs.BoolVar(detach, "d", false, "Run server in background as a daemon (shorthand)")
		_ = fs.Parse(normalizeArgs(fs, rest))
		handleWebStart(profile, *noOpen, *detach)
	case "stop":
		handleWebStop()
	case "status":
		handleWebStatus()
	default:
		fmt.Fprintf(os.Stderr, "Unknown web subcommand: %s\n", sub)
		fmt.Fprintln(os.Stderr, "Usage: hangar web [start|stop|status]")
		os.Exit(1)
	}
}

// handleWebStart runs the API + web UI server in the foreground.
// The caller can background it with: hangar web start &
// A PID file at ~/.hangar/web.pid lets "hangar web stop" find the process.
// Pass noOpen=true (via --no-open flag) to suppress auto-opening the browser.
// Pass detach=true (via --detach/-d flag) to re-exec self as a background daemon.
func handleWebStart(profile string, noOpen bool, detach bool) {
	if detach {
		// Re-exec self without --detach, redirect output to log file, then return.
		// The child runs in the foreground but the shell regains control immediately.
		hangarDir, err := session.GetHangarDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve hangar dir: %v\n", err)
			os.Exit(1)
		}
		logsDir := filepath.Join(hangarDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create logs dir: %v\n", err)
			os.Exit(1)
		}
		logPath := filepath.Join(logsDir, "web.log")
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
			os.Exit(1)
		}
		// Build args: all original args minus --detach / -d
		self, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve executable: %v\n", err)
			os.Exit(1)
		}
		childArgs := []string{"web", "start", "--no-open"}
		if profile != "" {
			childArgs = append([]string{"--profile", profile}, childArgs...)
		}
		cmd := exec.Command(self, childArgs...)
		cmd.Stdout = logFile
		cmd.Stderr = logFile
		if err := cmd.Start(); err != nil {
			logFile.Close()
			fmt.Fprintf(os.Stderr, "Failed to start background server: %v\n", err)
			os.Exit(1)
		}
		logFile.Close()
		port, bindAddr := webLoadConfig()
		uiURL := fmt.Sprintf("http://%s:%d/ui/", webDisplayAddr(bindAddr), port)
		fmt.Printf("Hangar web server starting in background\n")
		fmt.Printf("  URL: %s\n", uiURL)
		fmt.Printf("  Log: %s\n", logPath)
		fmt.Printf("  Stop: hangar web stop\n")
		return
	}
	pidFile := webPIDFile()

	// If a PID file exists and the process is alive, don't start a second one.
	if pid := readWebPID(pidFile); pid > 0 && webProcessRunning(pid) {
		fmt.Printf("Web server already running (PID %d).\n", pid)
		port, bindAddr := webLoadConfig()
		uiURL := fmt.Sprintf("http://%s:%d/ui/", webDisplayAddr(bindAddr), port)
		fmt.Printf("  URL:  %s\n", uiURL)
		if !noOpen {
			openBrowser(uiURL)
		}
		return
	}
	// Remove any stale PID file.
	_ = os.Remove(pidFile)

	// Load config for port and bind address.
	port, bindAddr := webLoadConfig()

	// Write PID file so "hangar web stop" can find this process.
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write PID file: %v\n", err)
	}
	defer os.Remove(pidFile)

	// In standalone mode getInstances reads directly from SQLite on each call.
	// This is safe and fast; the TUI's in-memory cache is not available here.
	getInstances := func() []*session.Instance {
		storage, err := session.NewStorageWithProfile(profile)
		if err != nil {
			return nil
		}
		defer storage.Close()
		instances, _ := storage.Load()
		return instances
	}

	// Start the hook file watcher so HTTP hook events update session status.
	watcher, err := session.NewStatusFileWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not start hook watcher: %v\n", err)
		// Continue without hook watcher — polling still works.
		watcher = nil
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a PR manager for standalone mode. It self-initialises gh detection
	// and background-fetches Mine/ReviewRequested lists via Start().
	prManager := pr.New()
	prManager.Start()

	// Poll sessions from SQLite and call UpdateSessionPR for each worktree
	// session so the PR dashboard is populated without the TUI running.
	go func() {
		pollSessionPRs := func() {
			for _, inst := range getInstances() {
				if inst.IsWorktree() && inst.WorktreePath != "" {
					prManager.UpdateSessionPR(inst.ID, inst.WorktreePath)
				}
			}
		}
		pollSessionPRs()
		ticker := time.NewTicker(90 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollSessionPRs()
			}
		}
	}()

	// Build a getPRInfo callback that serves session PR data from the manager.
	getPRInfo := func(sessionID string) *apiserver.PRInfo {
		p, exists := prManager.GetSessionPR(sessionID)
		if !exists || p == nil {
			return nil
		}
		return &apiserver.PRInfo{
			Number:        p.Number,
			Title:         p.Title,
			State:         p.State,
			URL:           p.URL,
			ChecksPassed:  p.ChecksPassed,
			ChecksFailed:  p.ChecksFailed,
			ChecksPending: p.ChecksPending,
			HasChecks:     p.HasChecks,
		}
	}

	cfg := apiserver.APIConfig{Port: port, BindAddress: bindAddr}
	srv := apiserver.New(cfg, watcher, getInstances, getPRInfo, nil, prManager, profile, Version)

	uiURL := fmt.Sprintf("http://%s:%d/ui/", webDisplayAddr(bindAddr), port)
	fmt.Printf("Hangar web server started.\n")
	fmt.Printf("  URL:  %s\n", uiURL)
	fmt.Printf("  API:  http://%s:%d/api/v1/status\n", webDisplayAddr(bindAddr), port)
	fmt.Printf("  PID:  %d\n", os.Getpid())
	fmt.Printf("  Stop: hangar web stop\n")

	if !noOpen {
		// Poll the status endpoint before opening the browser so the tab doesn't
		// hit a "connection refused" while the server is still binding its socket.
		statusURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/status", port)
		go func() {
			client := &http.Client{Timeout: time.Second}
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				resp, err := client.Get(statusURL)
				if err == nil {
					resp.Body.Close()
					openBrowser(uiURL)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	if err := srv.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// handleWebStop sends SIGTERM to a running "hangar web start" process.
func handleWebStop() {
	pidFile := webPIDFile()
	pid := readWebPID(pidFile)
	if pid <= 0 {
		fmt.Println("Web server is not running.")
		return
	}
	proc, err := os.FindProcess(pid)
	if err != nil || !webProcessRunning(pid) {
		fmt.Println("Web server is not running (stale PID file removed).")
		_ = os.Remove(pidFile)
		return
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop server: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Web server stopped (PID %d).\n", pid)
	_ = os.Remove(pidFile)
}

// handleWebStatus checks whether the API server is reachable and prints details.
// This works whether the server was started by the TUI or by "hangar web start".
func handleWebStatus() {
	port, bindAddr := webLoadConfig()
	pidFile := webPIDFile()
	pid := readWebPID(pidFile)

	// Try to reach the status endpoint regardless of PID file.
	url := fmt.Sprintf("http://127.0.0.1:%d/api/v1/status", port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		if pid > 0 && webProcessRunning(pid) {
			fmt.Printf("Web server process is running (PID %d) but not responding on port %d.\n", pid, port)
		} else {
			fmt.Printf("Web server is not running on port %d.\n", port)
			fmt.Printf("  Start:  hangar web start\n")
			fmt.Printf("  Config: ~/.hangar/config.toml  ([api] port, [api] bind_address)\n")
		}
		return
	}
	defer resp.Body.Close()

	// Decode the status response for session counts.
	var status struct {
		Version  string         `json:"version"`
		Uptime   string         `json:"uptime"`
		Sessions int            `json:"sessions"`
		ByStatus map[string]int `json:"by_status"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&status)

	fmt.Printf("Web server is running.\n")
	if pid > 0 {
		fmt.Printf("  PID:      %d (standalone)\n", pid)
	} else {
		fmt.Printf("  PID:      (started by TUI)\n")
	}
	fmt.Printf("  Port:     %d\n", port)
	fmt.Printf("  Bind:     %s\n", bindAddr)
	fmt.Printf("  URL:      http://%s:%d/ui/\n", webDisplayAddr(bindAddr), port)
	fmt.Printf("  Version:  %s\n", status.Version)
	fmt.Printf("  Uptime:   %s\n", status.Uptime)
	fmt.Printf("  Sessions: %d", status.Sessions)
	if len(status.ByStatus) > 0 {
		parts := make([]string, 0, len(status.ByStatus))
		for k, v := range status.ByStatus {
			parts = append(parts, fmt.Sprintf("%s=%d", k, v))
		}
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println()
}

// webPIDFile returns the path to the PID file for a standalone web server.
func webPIDFile() string {
	dir, err := session.GetHangarDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "hangar-web.pid")
	}
	return filepath.Join(dir, "web.pid")
}

// readWebPID reads the PID from the PID file; returns 0 on failure.
func readWebPID(pidFile string) int {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// webProcessRunning returns true if a process with the given PID exists.
func webProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// webLoadConfig reads port and bind address from user config, with defaults.
func webLoadConfig() (port int, bindAddr string) {
	port = 47437
	bindAddr = "0.0.0.0"
	userConfig, err := session.LoadUserConfig()
	if err == nil && userConfig != nil {
		port = userConfig.API.GetPort(&userConfig.Claude)
		bindAddr = userConfig.API.GetBindAddress()
	}
	return
}

// webDisplayAddr converts 0.0.0.0 to localhost for display in URLs.
func webDisplayAddr(bind string) string {
	if bind == "0.0.0.0" || bind == "" {
		return "localhost"
	}
	return bind
}

// openBrowser opens url in the user's default browser. Errors are silently
// ignored — the URL is printed to stdout so the user can open it manually.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start() //nolint:errcheck // best-effort; URL already printed to stdout
}

// webPrintURL prints the configured web UI URL.
func webPrintURL() {
	port, bindAddr := webLoadConfig()
	fmt.Printf("  URL: http://%s:%d/ui/\n", webDisplayAddr(bindAddr), port)
}
