package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server is the Hangar MCP server, exposing session and todo tools over stdio.
type Server struct {
	mcpServer *server.MCPServer
	client    *Client
}

// New creates a new MCP server backed by the Hangar REST API at baseURL.
func New(baseURL, version string) *Server {
	s := &Server{
		mcpServer: server.NewMCPServer(
			"hangar",
			version,
			server.WithToolCapabilities(true),
		),
		client: NewClient(baseURL),
	}
	s.registerSessionTools()
	s.registerTodoTools()
	return s
}

// Run starts the stdio MCP server, blocking until stdin is closed.
func (s *Server) Run() error {
	return server.ServeStdio(s.mcpServer)
}

// addTool is a convenience for registering a tool on the underlying MCP server.
func (s *Server) addTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.mcpServer.AddTool(tool, handler)
}
