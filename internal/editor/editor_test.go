package editor

import (
	"testing"
)

func TestIsTerminal(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		{"nvim", true},
		{"vim", true},
		{"vi", true},
		{"hx", true},
		{"emacs", true},
		{"/usr/bin/nvim", true},
		{"code", false},
		{"zed", false},
		{"idea", false},
		{"my-custom-editor", false},
	}
	for _, c := range cases {
		got := IsTerminal(c.cmd)
		if got != c.want {
			t.Errorf("IsTerminal(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestResolveCommand_AppendFallback(t *testing.T) {
	cmd, args := ResolveCommand("nvim", "/home/user/project")
	if cmd != "nvim" {
		t.Errorf("cmd = %q, want %q", cmd, "nvim")
	}
	if len(args) != 1 || args[0] != "/home/user/project" {
		t.Errorf("args = %v, want [/home/user/project]", args)
	}
}

func TestResolveCommand_PathPlaceholder(t *testing.T) {
	cmd, args := ResolveCommand("my-editor --file {path} --flag", "/some/path")
	if cmd != "my-editor" {
		t.Errorf("cmd = %q, want %q", cmd, "my-editor")
	}
	if len(args) != 3 || args[0] != "--file" || args[1] != "/some/path" || args[2] != "--flag" {
		t.Errorf("args = %v", args)
	}
}

func TestResolveCommand_MultiplePlaceholders(t *testing.T) {
	cmd, args := ResolveCommand("editor --cwd {path} --open {path}", "/repo")
	if cmd != "editor" {
		t.Errorf("cmd = %q", cmd)
	}
	if len(args) != 4 || args[1] != "/repo" || args[3] != "/repo" {
		t.Errorf("args = %v", args)
	}
}

func TestResolveCommand_PathWithSpaces(t *testing.T) {
	// Quoted path in the editor string itself (unusual but possible)
	cmd, args := ResolveCommand("nvim", "/home/user/my project")
	if cmd != "nvim" {
		t.Errorf("cmd = %q", cmd)
	}
	if len(args) != 1 || args[0] != "/home/user/my project" {
		t.Errorf("args = %v", args)
	}
}

func TestResolveCommand_FlagsBeforePath(t *testing.T) {
	cmd, args := ResolveCommand("code --new-window", "/workspace")
	if cmd != "code" {
		t.Errorf("cmd = %q", cmd)
	}
	// path appended as last arg
	if len(args) != 2 || args[0] != "--new-window" || args[1] != "/workspace" {
		t.Errorf("args = %v", args)
	}
}

func TestResolveCommand_EmptyString(t *testing.T) {
	cmd, args := ResolveCommand("", "/path")
	if cmd != "" || args != nil {
		t.Errorf("expected empty cmd and nil args, got cmd=%q args=%v", cmd, args)
	}
}

func TestGetCmd_EnvVarWins(t *testing.T) {
	t.Setenv("HANGAR_EDITOR", "zed")
	got := GetCmd("nvim")
	if got != "zed" {
		t.Errorf("GetCmd with HANGAR_EDITOR=zed and config=nvim: got %q, want %q", got, "zed")
	}
}

func TestGetCmd_FallbackToConfig(t *testing.T) {
	t.Setenv("HANGAR_EDITOR", "")
	got := GetCmd("nvim")
	if got != "nvim" {
		t.Errorf("GetCmd with no env var: got %q, want %q", got, "nvim")
	}
}
