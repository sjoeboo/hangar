package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sjoeboo/hangar/internal/git"
	"github.com/sjoeboo/hangar/internal/session"
)

// handleProject dispatches project subcommands
func handleProject(profile string, args []string) {
	if len(args) == 0 {
		handleProjectList(profile, nil)
		return
	}

	switch args[0] {
	case "list", "ls":
		handleProjectList(profile, args[1:])
	case "add", "new":
		handleProjectAdd(profile, args[1:])
	case "remove", "rm", "delete":
		handleProjectRemove(profile, args[1:])
	case "help", "--help", "-h":
		printProjectHelp()
	default:
		fmt.Printf("Unknown project command: %s\n", args[0])
		fmt.Println()
		printProjectHelp()
		os.Exit(1)
	}
}

// printProjectHelp prints usage for project commands
func printProjectHelp() {
	fmt.Println("Usage: hangar project <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                            List all projects")
	fmt.Println("  add <name> <base-dir> [branch]  Add a new project")
	fmt.Println("  remove <name>                   Remove a project")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  hangar project list")
	fmt.Println("  hangar project add myrepo ~/code/myrepo")
	fmt.Println("  hangar project add myrepo ~/code/myrepo main")
	fmt.Println("  hangar project remove myrepo")
}

// handleProjectList lists all projects
func handleProjectList(profile string, args []string) {
	projects, err := session.ListProjects()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading projects: %v\n", err)
		os.Exit(1)
	}

	if len(projects) == 0 {
		fmt.Println("No projects configured.")
		fmt.Println()
		fmt.Println("Add a project with: hangar project add <name> <base-dir>")
		return
	}

	fmt.Printf("%-20s %-40s %s\n", "NAME", "BASE DIR", "BASE BRANCH")
	fmt.Println(strings.Repeat("-", 70))
	for _, p := range projects {
		baseDir := p.BaseDir
		if len(baseDir) > 38 {
			baseDir = "..." + baseDir[len(baseDir)-35:]
		}
		fmt.Printf("%-20s %-40s %s\n", p.Name, baseDir, p.BaseBranch)
	}
	fmt.Printf("\nTotal: %d project(s)\n", len(projects))
}

// handleProjectAdd adds a new project
func handleProjectAdd(profile string, args []string) {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: hangar project add <name> <base-dir> [base-branch]")
		os.Exit(1)
	}

	name := args[0]
	baseDir := args[1]
	baseBranch := ""
	if len(args) >= 3 {
		baseBranch = args[2]
	}

	// Expand ~ in path
	expandedDir := session.ExpandPath(baseDir)

	// Validate that base-dir is a git repo
	if !git.IsGitRepo(expandedDir) {
		fmt.Fprintf(os.Stderr, "Error: %q is not a git repository\n", expandedDir)
		os.Exit(1)
	}

	// Auto-detect base branch if not provided
	if baseBranch == "" {
		baseBranch = session.DetectBaseBranch(expandedDir)
		fmt.Printf("Auto-detected base branch: %s\n", baseBranch)
	}

	if err := session.AddProject(name, expandedDir, baseBranch); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added project %q (base: %s, branch: %s)\n", name, expandedDir, baseBranch)
	touchStorage(profile)
}

// handleProjectRemove removes a project by name
func handleProjectRemove(profile string, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: hangar project remove <name>")
		os.Exit(1)
	}

	name := args[0]

	if err := session.RemoveProject(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed project %q\n", name)
	touchStorage(profile)
}

// touchStorage bumps the SQLite metadata timestamp so a running TUI instance
// reloads after CLI project mutations.
func touchStorage(profile string) {
	storage, err := session.NewStorageWithProfile(profile)
	if err != nil {
		return // best-effort; TUI will reload on next poll
	}
	defer storage.Close()
	_ = storage.GetDB().Touch()
}
