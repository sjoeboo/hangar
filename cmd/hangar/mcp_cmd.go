package main

import (
	"fmt"
	"os"

	"github.com/sjoeboo/hangar/internal/mcpserver"
	"github.com/sjoeboo/hangar/internal/session"
)

// handleMCPServer runs the "hangar mcp-server" subcommand.
// This is a stdio MCP server — Claude Code spawns it as a child process
// and communicates via JSON-RPC on stdin/stdout.
func handleMCPServer(args []string) {
	port, _ := webLoadConfig()
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	_, _ = session.GetHangarDir() // ensure config dir initialized

	srv := mcpserver.New(baseURL, Version)
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "mcp-server error: %v\n", err)
		os.Exit(1)
	}
}
