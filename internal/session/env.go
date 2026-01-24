package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// buildEnvSourceCommand builds shell commands to source .env files before the main command.
// Returns empty string if no env files are configured.
// Order of sourcing (later overrides earlier):
//  1. Global [shell].env_files (in order)
//  2. [shell].init_script (for direnv, nvm, etc.)
//  3. Tool-specific env_file ([claude].env_file, [gemini].env_file, [tools.X].env_file)
func (i *Instance) buildEnvSourceCommand() string {
	var sources []string
	config, _ := LoadUserConfig()
	if config == nil {
		return ""
	}

	ignoreMissing := config.Shell.GetIgnoreMissingEnvFiles()

	// 1. Global env_files from [shell] section
	for _, envFile := range config.Shell.EnvFiles {
		resolved := resolveEnvFilePath(envFile, i.ProjectPath)
		sources = append(sources, buildSourceCmd(resolved, ignoreMissing))
	}

	// 2. Shell init script (direnv, nvm, pyenv, etc.)
	if config.Shell.InitScript != "" {
		script := config.Shell.InitScript
		if isFilePath(script) {
			resolved := expandHomePath(script)
			sources = append(sources, buildSourceCmd(resolved, ignoreMissing))
		} else {
			// Inline command (e.g., 'eval "$(direnv hook bash)"')
			sources = append(sources, script)
		}
	}

	// 3. Tool-specific env_file
	toolEnvFile := i.getToolEnvFile()
	if toolEnvFile != "" {
		resolved := resolveEnvFilePath(toolEnvFile, i.ProjectPath)
		sources = append(sources, buildSourceCmd(resolved, ignoreMissing))
	}

	if len(sources) == 0 {
		return ""
	}

	// Join all sources with && and add trailing && for the main command
	return strings.Join(sources, " && ") + " && "
}

// buildSourceCmd creates a shell command to source a file.
// If ignoreMissing is true, wraps in a file existence check.
func buildSourceCmd(path string, ignoreMissing bool) string {
	if ignoreMissing {
		// Use [ -f file ] && source file pattern for safe sourcing
		return fmt.Sprintf(`[ -f "%s" ] && source "%s"`, path, path)
	}
	return fmt.Sprintf(`source "%s"`, path)
}

// resolveEnvFilePath resolves an env file path.
// - Expands ~ to home directory
// - Absolute paths are returned as-is
// - Relative paths are resolved relative to workDir
func resolveEnvFilePath(path, workDir string) string {
	expanded := expandHomePath(path)
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded)
	}
	return filepath.Clean(filepath.Join(workDir, expanded))
}

// expandHomePath expands ~ prefix to the user's home directory.
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return unchanged if home dir unavailable
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// isFilePath checks if a string looks like a file path (vs inline command).
func isFilePath(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "~/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~")
}

// getToolEnvFile returns the env_file setting for the current tool.
func (i *Instance) getToolEnvFile() string {
	config, _ := LoadUserConfig()
	if config == nil {
		return ""
	}

	switch i.Tool {
	case "claude":
		return config.Claude.EnvFile
	case "gemini":
		return config.Gemini.EnvFile
	default:
		// Check custom tools
		if def := GetToolDef(i.Tool); def != nil {
			return def.EnvFile
		}
	}
	return ""
}
