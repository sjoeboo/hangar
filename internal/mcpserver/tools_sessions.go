package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerSessionTools() {
	s.addTool(
		mcp.NewTool("hangar_list_sessions",
			mcp.WithDescription("List all Hangar sessions with their status, tool, path, and metadata"),
		),
		s.handleListSessions,
	)

	s.addTool(
		mcp.NewTool("hangar_get_session",
			mcp.WithDescription("Get detailed information about a specific session"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
		),
		s.handleGetSession,
	)

	s.addTool(
		mcp.NewTool("hangar_get_output",
			mcp.WithDescription("Get recent terminal output from a session"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
			mcp.WithNumber("lines", mcp.Description("Number of output lines to return (default 50)")),
		),
		s.handleGetOutput,
	)

	s.addTool(
		mcp.NewTool("hangar_send_message",
			mcp.WithDescription("Send a text message to a running session"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
			mcp.WithString("message", mcp.Required(), mcp.Description("Message text to send")),
		),
		s.handleSendMessage,
	)

	s.addTool(
		mcp.NewTool("hangar_start_session",
			mcp.WithDescription("Start a stopped session, optionally with an initial message"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
			mcp.WithString("message", mcp.Description("Optional initial message to send after starting")),
		),
		s.handleStartSession,
	)

	s.addTool(
		mcp.NewTool("hangar_stop_session",
			mcp.WithDescription("Stop a running session"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
		),
		s.handleStopSession,
	)

	s.addTool(
		mcp.NewTool("hangar_restart_session",
			mcp.WithDescription("Restart a session (stop then start)"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Session ID")),
		),
		s.handleRestartSession,
	)

	s.addTool(
		mcp.NewTool("hangar_create_session",
			mcp.WithDescription("Create a new Hangar session"),
			mcp.WithString("title", mcp.Required(), mcp.Description("Session title")),
			mcp.WithString("path", mcp.Required(), mcp.Description("Project path for the session")),
			mcp.WithString("tool", mcp.Description("Tool to use (default: claude)")),
		),
		s.handleCreateSession,
	)
}

func (s *Server) handleListSessions(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessions, err := s.client.ListSessions()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list sessions: %v", err)), nil
	}
	return jsonResult(sessions)
}

func (s *Server) handleGetSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	session, err := s.client.GetSession(id)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get session: %v", err)), nil
	}
	return jsonResult(session)
}

func (s *Server) handleGetOutput(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	lines := req.GetInt("lines", 50)
	output, err := s.client.GetSessionOutput(id, lines)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get output: %v", err)), nil
	}
	return jsonResult(output)
}

func (s *Server) handleSendMessage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	message, err := req.RequireString("message")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.client.SendMessage(id, message); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send message: %v", err)), nil
	}
	return mcp.NewToolResultText("Message sent successfully"), nil
}

func (s *Server) handleStartSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	message := req.GetString("message", "")
	if err := s.client.StartSession(id, message); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start session: %v", err)), nil
	}
	return mcp.NewToolResultText("Session started"), nil
}

func (s *Server) handleStopSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.client.StopSession(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stop session: %v", err)), nil
	}
	return mcp.NewToolResultText("Session stopped"), nil
}

func (s *Server) handleRestartSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.client.RestartSession(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to restart session: %v", err)), nil
	}
	return mcp.NewToolResultText("Session restarted"), nil
}

func (s *Server) handleCreateSession(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := req.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tool := req.GetString("tool", "")
	session, err := s.client.CreateSession(title, path, tool)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create session: %v", err)), nil
	}
	return jsonResult(session)
}

// jsonResult marshals v to JSON and returns it as a text tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
