package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerTodoTools() {
	s.addTool(
		mcp.NewTool("hangar_list_todos",
			mcp.WithDescription("List todos, optionally filtered by project path"),
			mcp.WithString("project", mcp.Description("Project path to filter by")),
		),
		s.handleListTodos,
	)

	s.addTool(
		mcp.NewTool("hangar_create_todo",
			mcp.WithDescription("Create a new todo item"),
			mcp.WithString("project", mcp.Required(), mcp.Description("Project path")),
			mcp.WithString("title", mcp.Required(), mcp.Description("Todo title")),
			mcp.WithString("description", mcp.Description("Optional description")),
		),
		s.handleCreateTodo,
	)

	s.addTool(
		mcp.NewTool("hangar_update_todo",
			mcp.WithDescription("Update a todo item's status, title, or description"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Todo ID")),
			mcp.WithString("status", mcp.Description("New status (todo, doing, done)")),
			mcp.WithString("title", mcp.Description("New title")),
			mcp.WithString("description", mcp.Description("New description")),
		),
		s.handleUpdateTodo,
	)

	s.addTool(
		mcp.NewTool("hangar_delete_todo",
			mcp.WithDescription("Delete a todo item"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Todo ID")),
		),
		s.handleDeleteTodo,
	)
}

func (s *Server) handleListTodos(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	todos, err := s.client.ListTodos(project)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list todos: %v", err)), nil
	}
	return jsonResult(todos)
}

func (s *Server) handleCreateTodo(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	title, err := req.RequireString("title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	description := req.GetString("description", "")
	todo, err := s.client.CreateTodo(project, title, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create todo: %v", err)), nil
	}
	return jsonResult(todo)
}

func (s *Server) handleUpdateTodo(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	fields := make(map[string]any)
	if v := req.GetString("status", ""); v != "" {
		fields["status"] = v
	}
	if v := req.GetString("title", ""); v != "" {
		fields["title"] = v
	}
	if v := req.GetString("description", ""); v != "" {
		fields["description"] = v
	}
	if len(fields) == 0 {
		return mcp.NewToolResultError("no fields to update"), nil
	}
	if err := s.client.UpdateTodo(id, fields); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update todo: %v", err)), nil
	}
	return mcp.NewToolResultText("Todo updated"), nil
}

func (s *Server) handleDeleteTodo(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.client.DeleteTodo(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete todo: %v", err)), nil
	}
	return mcp.NewToolResultText("Todo deleted"), nil
}
