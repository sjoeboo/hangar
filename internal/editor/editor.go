// Package editor provides logic for opening a worktree directory in a user-configured editor.
// It handles both terminal editors (neovim, vim, emacs, …) and GUI editors (VS Code, Zed, IntelliJ, …),
// routing them to the correct launch mechanism automatically.
package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// terminalEditors is the set of binary basenames that require a real terminal to render.
// All other editors are assumed to be GUI applications launched as detached processes.
var terminalEditors = map[string]bool{
	"nvim":   true,
	"vim":    true,
	"vi":     true,
	"nano":   true,
	"emacs":  true,
	"hx":     true,
	"helix":  true,
	"micro":  true,
	"kak":    true,
	"kakoune": true,
	"ne":     true,
	"joe":    true,
	"mcedit": true,
}

// Preset describes a built-in editor option shown in the editor picker.
type Preset struct {
	Name     string // internal identifier, e.g. "neovim"
	Label    string // display label, e.g. "Neovim"
	Cmd      string // binary to invoke, e.g. "nvim"
	Terminal bool   // true if this editor needs a terminal window
}

// Presets is the ordered list of built-in editors shown in the picker overlay.
var Presets = []Preset{
	{Name: "neovim", Label: "Neovim", Cmd: "nvim", Terminal: true},
	{Name: "zed", Label: "Zed", Cmd: "zed", Terminal: false},
	{Name: "vscode", Label: "VS Code", Cmd: "code", Terminal: false},
	{Name: "intellij", Label: "IntelliJ IDEA", Cmd: "idea", Terminal: false},
	{Name: "custom", Label: "Custom…", Cmd: "", Terminal: false},
}

// IsTerminal reports whether cmd (a binary name or path) is a known terminal editor.
func IsTerminal(cmd string) bool {
	return terminalEditors[strings.ToLower(filepath.Base(cmd))]
}

// GetCmd returns the editor command string to use, preferring $HANGAR_EDITOR over configEditor.
func GetCmd(configEditor string) string {
	if env := os.Getenv("HANGAR_EDITOR"); env != "" {
		return env
	}
	return configEditor
}

// ResolveCommand splits editorStr into a command + args list and substitutes {path}.
// If no {path} placeholder is present anywhere, worktreePath is appended as the final argument.
func ResolveCommand(editorStr, worktreePath string) (cmd string, args []string) {
	tokens := tokenize(editorStr)
	if len(tokens) == 0 {
		return "", nil
	}
	cmd = strings.ReplaceAll(tokens[0], "{path}", worktreePath)
	hasPlaceholder := strings.Contains(tokens[0], "{path}")
	for _, t := range tokens[1:] {
		if strings.Contains(t, "{path}") {
			args = append(args, strings.ReplaceAll(t, "{path}", worktreePath))
			hasPlaceholder = true
		} else {
			args = append(args, t)
		}
	}
	if !hasPlaceholder {
		args = append(args, worktreePath)
	}
	return cmd, args
}

// LaunchInTmux opens a terminal editor in a new tmux window within the given tmux session.
// worktreePath is used as the working directory for the new window.
func LaunchInTmux(cmd string, args []string, tmuxSession, worktreePath string) error {
	// Build: tmux new-window -t <session> -c <dir> -n <editor-name> <cmd> [args...]
	tmuxArgs := []string{"new-window", "-t", tmuxSession, "-c", worktreePath, "-n", filepath.Base(cmd), cmd}
	tmuxArgs = append(tmuxArgs, args...)
	return exec.Command("tmux", tmuxArgs...).Run()
}

// LaunchGUI starts a GUI editor as a detached background process.
// The process is placed in its own process group so it survives Hangar restarts.
func LaunchGUI(cmd string, args []string) error {
	c := exec.Command(cmd, args...)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return c.Start()
}

// Launch opens the editor for worktreePath. For terminal editors it creates a new tmux window;
// for GUI editors it launches a detached background process.
// tmuxSession is only required for terminal editors; it may be empty for GUI editors.
func Launch(editorStr, worktreePath, tmuxSession string) error {
	cmd, args := ResolveCommand(editorStr, worktreePath)
	if cmd == "" {
		return nil
	}
	if IsTerminal(cmd) {
		return LaunchInTmux(cmd, args, tmuxSession, worktreePath)
	}
	return LaunchGUI(cmd, args)
}

// tokenize splits s into whitespace-separated tokens, respecting double-quoted groups.
func tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for _, ch := range s {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ' ' && !inQuote:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}
