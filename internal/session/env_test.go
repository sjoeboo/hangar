package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"absolute path", "/var/log/test.log", "/var/log/test.log"},
		{"relative path", ".env", ".env"},
		{"tilde prefix", "~/.secrets", filepath.Join(home, ".secrets")},
		{"just tilde", "~", home},
		{"tilde in middle", "/path/~/.env", "/path/~/.env"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandHomePath(tt.input)
			if result != tt.expected {
				t.Errorf("expandHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolveEnvFilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	workDir := "/projects/myapp"

	tests := []struct {
		name     string
		path     string
		workDir  string
		expected string
	}{
		{"absolute path", "/etc/env", workDir, "/etc/env"},
		{"home path", "~/.secrets", workDir, filepath.Join(home, ".secrets")},
		{"relative path", ".env", workDir, "/projects/myapp/.env"},
		{"relative subdir", "config/.env", workDir, "/projects/myapp/config/.env"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEnvFilePath(tt.path, tt.workDir)
			if result != tt.expected {
				t.Errorf("resolveEnvFilePath(%q, %q) = %q, want %q", tt.path, tt.workDir, result, tt.expected)
			}
		})
	}
}

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"/etc/env", true},
		{"~/env", true},
		{"./env", true},
		{"../env", true},
		{"~", true},
		{"eval $(direnv hook bash)", false},
		{"source ~/.bashrc", false},
		{".env", false}, // Treated as inline command, not file path
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("isFilePath(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildSourceCmd(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		ignoreMissing bool
		wantContains  []string
	}{
		{
			name:          "ignore missing",
			path:          "/path/.env",
			ignoreMissing: true,
			wantContains:  []string{`[ -f "/path/.env" ]`, `source "/path/.env"`},
		},
		{
			name:          "strict mode",
			path:          "/path/.env",
			ignoreMissing: false,
			wantContains:  []string{`source "/path/.env"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSourceCmd(tt.path, tt.ignoreMissing)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("buildSourceCmd(%q, %v) = %q, want to contain %q", tt.path, tt.ignoreMissing, result, want)
				}
			}
		})
	}
}

func TestShellSettings_GetIgnoreMissingEnvFiles(t *testing.T) {
	trueBool := true
	falseBool := false

	tests := []struct {
		name     string
		settings ShellSettings
		expected bool
	}{
		{"nil pointer defaults to true", ShellSettings{}, true},
		{"explicit true", ShellSettings{IgnoreMissingEnvFiles: &trueBool}, true},
		{"explicit false", ShellSettings{IgnoreMissingEnvFiles: &falseBool}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.settings.GetIgnoreMissingEnvFiles()
			if result != tt.expected {
				t.Errorf("GetIgnoreMissingEnvFiles() = %v, want %v", result, tt.expected)
			}
		})
	}
}
