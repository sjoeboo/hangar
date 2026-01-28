package session

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/asheshgoplani/agent-deck/internal/mcppool"
	"github.com/asheshgoplani/agent-deck/internal/platform"
)

// Global MCP pool instances
var (
	globalPool     *mcppool.Pool
	globalHTTPPool *mcppool.HTTPPool
	globalPoolMu   sync.RWMutex
)

// InitializeGlobalPool creates and starts the global MCP pool
func InitializeGlobalPool(ctx context.Context, config *UserConfig, sessions []*Instance) (*mcppool.Pool, error) {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	log.Printf("[Pool] InitializeGlobalPool called with %d sessions", len(sessions))

	// Return existing pool if already initialized
	if globalPool != nil {
		log.Printf("[Pool] Pool already initialized, returning existing")
		return globalPool, nil
	}

	// Check if pool is enabled
	if !config.MCPPool.Enabled {
		log.Printf("[Pool] Pool disabled in config")
		return nil, nil // Pool disabled, not an error
	}

	// Check platform compatibility for Unix sockets
	// WSL1 and Windows don't reliably support Unix domain sockets
	detectedPlatform := platform.Detect()
	if !platform.SupportsUnixSockets() {
		log.Printf("[Pool] Platform '%s' detected - MCP socket pooling disabled", detectedPlatform)
		log.Printf("[Pool] MCPs will use stdio mode (each session spawns its own MCP processes)")
		if detectedPlatform == platform.PlatformWSL1 {
			log.Printf("[Pool] Tip: WSL2 supports socket pooling. Run 'wsl --set-version <distro> 2' to upgrade")
		}
		return nil, nil // Platform doesn't support sockets, not an error
	}

	log.Printf("[Pool] Platform '%s' detected - socket pooling supported", detectedPlatform)
	log.Printf("[Pool] Pool enabled, creating pool...")

	// Create pool config
	// FallbackStdio is forced to true for safety (Issue #36):
	// - Pool sockets may not be ready immediately after TUI starts
	// - Instant socket check (no blocking) means fallback is essential
	// - Falling back to stdio is safe - MCPs work, just use more memory
	//
	// Note: The config field fallback_to_stdio is effectively ignored and
	// always treated as true. This ensures session creation never fails
	// due to pool initialization timing.
	poolConfig := &mcppool.PoolConfig{
		Enabled:       config.MCPPool.Enabled,
		PoolAll:       config.MCPPool.PoolAll,
		ExcludeMCPs:   config.MCPPool.ExcludeMCPs,
		PoolMCPs:      config.MCPPool.PoolMCPs,
		FallbackStdio: true, // Always true - see Issue #36
	}

	// Create pool
	pool, err := mcppool.NewPool(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	// FIRST: Discover existing sockets from another agent-deck instance
	// This allows multiple TUI instances to share the same pool
	discovered := pool.DiscoverExistingSockets()
	if discovered > 0 {
		log.Printf("[Pool] Reusing %d sockets from another agent-deck instance", discovered)
	}

	// Get all available MCPs from config.toml
	availableMCPs := GetAvailableMCPs()
	log.Printf("[Pool] Available MCPs in config: %d", len(availableMCPs))

	// When pool_all = true, pool ALL available MCPs (not just those in use)
	// This ensures any MCP can be attached via socket immediately
	startedCount := 0
	skippedCount := 0
	for mcpName, def := range availableMCPs {
		shouldPool := pool.ShouldPool(mcpName)
		log.Printf("[Pool] MCP '%s' - should pool: %v", mcpName, shouldPool)

		if !shouldPool {
			continue // Excluded or not in pool_mcps list
		}

		// Skip if already running (discovered from another instance)
		if pool.IsRunning(mcpName) {
			log.Printf("[Pool] %s: already running (discovered from another instance), skipping", mcpName)
			skippedCount++
			continue
		}

		// Start socket proxy for this MCP
		log.Printf("[Pool] Starting socket proxy for %s...", mcpName)
		if err := pool.Start(mcpName, def.Command, def.Args, def.Env); err != nil {
			log.Printf("[Pool] ✗ Failed to start socket proxy for %s: %v", mcpName, err)
		} else {
			log.Printf("[Pool] ✓ Socket proxy started: %s", mcpName)
			startedCount++
		}
	}

	log.Printf("[Pool] Started %d socket proxies, reused %d from other instance", startedCount, skippedCount)

	// Start health monitor for auto-restart of failed proxies
	pool.StartHealthMonitor()

	globalPool = pool

	// Initialize HTTP pool for HTTP/SSE MCPs with auto-start servers
	httpPool := mcppool.NewHTTPPool(ctx)
	httpStarted := 0
	for mcpName, def := range availableMCPs {
		if def.HasAutoStartServer() {
			log.Printf("[HTTP-Pool] Starting HTTP server for %s...", mcpName)
			timeout := time.Duration(def.Server.GetStartupTimeout()) * time.Millisecond
			healthCheck := def.Server.HealthCheck
			if healthCheck == "" {
				healthCheck = def.URL
			}
			if err := httpPool.Start(mcpName, def.URL, healthCheck, def.Server.Command, def.Server.Args, def.Server.Env, timeout); err != nil {
				log.Printf("[HTTP-Pool] ✗ Failed to start HTTP server for %s: %v", mcpName, err)
			} else {
				log.Printf("[HTTP-Pool] ✓ HTTP server started: %s at %s", mcpName, def.URL)
				httpStarted++
			}
		}
	}
	if httpStarted > 0 {
		log.Printf("[HTTP-Pool] Started %d HTTP servers", httpStarted)
		httpPool.StartHealthMonitor()
	}
	globalHTTPPool = httpPool

	return pool, nil
}

// GetGlobalPool returns the global socket pool instance (may be nil if disabled)
func GetGlobalPool() *mcppool.Pool {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()
	return globalPool
}

// GetGlobalHTTPPool returns the global HTTP pool instance (may be nil)
func GetGlobalHTTPPool() *mcppool.HTTPPool {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()
	return globalHTTPPool
}

// GetGlobalPoolRunningCount returns the number of running MCPs in the global pool
func GetGlobalPoolRunningCount() int {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()

	if globalPool != nil {
		return globalPool.GetRunningCount()
	}
	return 0
}

// ShutdownGlobalPool stops the global pools if shouldShutdown is true.
// If shouldShutdown is false, it disconnects from the pools but leaves processes running.
func ShutdownGlobalPool(shouldShutdown bool) error {
	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	// Shutdown socket pool
	if globalPool != nil {
		if shouldShutdown {
			log.Printf("[Pool] Shutting down socket pool (killing MCP processes)")
			err := globalPool.Shutdown()
			globalPool = nil
			if err != nil {
				return err
			}
		} else {
			// Just disconnect - leave MCPs running for next instance
			log.Printf("[Pool] Disconnecting from socket pool (leaving %d MCPs running in background)", globalPool.GetRunningCount())
			globalPool = nil
		}
	}

	// Shutdown HTTP pool
	if globalHTTPPool != nil {
		if shouldShutdown {
			log.Printf("[HTTP-Pool] Shutting down HTTP pool (killing server processes)")
			err := globalHTTPPool.Shutdown()
			globalHTTPPool = nil
			if err != nil {
				return err
			}
		} else {
			log.Printf("[HTTP-Pool] Disconnecting from HTTP pool (leaving %d servers running in background)", globalHTTPPool.GetRunningCount())
			globalHTTPPool = nil
		}
	}

	return nil
}

// StartHTTPServer starts an HTTP MCP server on demand
// This is called when an HTTP MCP with server config is attached to a session
func StartHTTPServer(name string, def *MCPDef) error {
	if !def.HasAutoStartServer() {
		return nil // No server config, nothing to start
	}

	globalPoolMu.Lock()
	defer globalPoolMu.Unlock()

	// Create HTTP pool if it doesn't exist
	if globalHTTPPool == nil {
		globalHTTPPool = mcppool.NewHTTPPool(context.Background())
		globalHTTPPool.StartHealthMonitor()
	}

	// Check if already running
	if globalHTTPPool.IsRunning(name) {
		log.Printf("[HTTP-Pool] %s: already running", name)
		return nil
	}

	// Start the server
	timeout := time.Duration(def.Server.GetStartupTimeout()) * time.Millisecond
	healthCheck := def.Server.HealthCheck
	if healthCheck == "" {
		healthCheck = def.URL
	}

	log.Printf("[HTTP-Pool] Starting HTTP server for %s...", name)
	if err := globalHTTPPool.Start(name, def.URL, healthCheck, def.Server.Command, def.Server.Args, def.Server.Env, timeout); err != nil {
		return err
	}
	log.Printf("[HTTP-Pool] ✓ HTTP server started: %s at %s", name, def.URL)
	return nil
}

// IsHTTPServerRunning checks if an HTTP MCP server is running
func IsHTTPServerRunning(name string) bool {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()

	if globalHTTPPool == nil {
		return false
	}
	return globalHTTPPool.IsRunning(name)
}

// GetHTTPServerStatus returns the status of an HTTP MCP server
func GetHTTPServerStatus(name string) string {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()

	if globalHTTPPool == nil {
		return "not_initialized"
	}

	server := globalHTTPPool.GetServer(name)
	if server == nil {
		return "not_found"
	}

	return server.GetStatus().String()
}
