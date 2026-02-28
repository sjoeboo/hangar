package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// Project represents a git repository that hangar manages sessions for.
type Project struct {
	Name       string `toml:"name"`
	BaseDir    string `toml:"base_dir"`
	BaseBranch string `toml:"base_branch"`
	Order      int    `toml:"order,omitempty"`
}

// projectsFile is the on-disk format for ~/.hangar/projects.toml
type projectsFile struct {
	Project []Project `toml:"project"`
}

// GetProjectsFilePath returns ~/.hangar/projects.toml
func GetProjectsFilePath() (string, error) {
	dir, err := GetHangarDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.toml"), nil
}

// LoadProjects reads ~/.hangar/projects.toml
// Returns empty list if file doesn't exist.
func LoadProjects() ([]*Project, error) {
	path, err := GetProjectsFilePath()
	if err != nil {
		return nil, err
	}

	// Return empty list if file doesn't exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []*Project{}, nil
	}

	var pf projectsFile
	if _, err := toml.DecodeFile(path, &pf); err != nil {
		return nil, fmt.Errorf("failed to parse projects.toml: %w", err)
	}

	projects := make([]*Project, len(pf.Project))
	for i := range pf.Project {
		p := pf.Project[i]
		projects[i] = &p
	}

	return projects, nil
}

// SaveProjects writes the projects list atomically.
func SaveProjects(projects []*Project) error {
	path, err := GetProjectsFilePath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Build the file struct
	pf := projectsFile{}
	for _, p := range projects {
		pf.Project = append(pf.Project, *p)
	}

	// Write to temp file then rename (atomic write)
	tmpPath := path + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	enc := toml.NewEncoder(f)
	if err := enc.Encode(pf); err != nil {
		f.Close()
		return fmt.Errorf("failed to encode projects: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to save projects.toml: %w", err)
	}

	return nil
}

// AddProject adds a new project. Returns error if name already exists.
func AddProject(name, baseDir, baseBranch string) error {
	projects, err := LoadProjects()
	if err != nil {
		return err
	}

	// Check for duplicate name
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			return fmt.Errorf("project %q already exists", name)
		}
	}

	// Determine order
	order := len(projects)

	project := &Project{
		Name:       name,
		BaseDir:    baseDir,
		BaseBranch: baseBranch,
		Order:      order,
	}

	projects = append(projects, project)
	return SaveProjects(projects)
}

// RemoveProject removes a project by name.
func RemoveProject(name string) error {
	projects, err := LoadProjects()
	if err != nil {
		return err
	}

	var filtered []*Project
	found := false
	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}

	if !found {
		return fmt.Errorf("project %q not found", name)
	}

	return SaveProjects(filtered)
}

// RenameProject renames a project from oldName to newName, preserving all
// other fields (BaseDir, BaseBranch, Order). Matching is case-insensitive.
// Returns an error if the project is not found.
func RenameProject(oldName, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("project name must not be empty")
	}

	projects, err := LoadProjects()
	if err != nil {
		return err
	}

	// Guard against colliding with a different existing project (allow case-only renames)
	for _, p := range projects {
		if strings.EqualFold(p.Name, newName) && !strings.EqualFold(p.Name, oldName) {
			return fmt.Errorf("project %q already exists", newName)
		}
	}

	found := false
	for _, p := range projects {
		if strings.EqualFold(p.Name, oldName) {
			p.Name = newName
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("project %q not found", oldName)
	}

	return SaveProjects(projects)
}

// ListProjects returns projects sorted by Order then Name.
func ListProjects() ([]*Project, error) {
	projects, err := LoadProjects()
	if err != nil {
		return nil, err
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Order != projects[j].Order {
			return projects[i].Order < projects[j].Order
		}
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// GetProject returns a project by name.
func GetProject(name string) (*Project, error) {
	projects, err := LoadProjects()
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}

	return nil, fmt.Errorf("project %q not found", name)
}

// DetectBaseBranch auto-detects the default branch for a git repo.
// Tries: git symbolic-ref refs/remotes/origin/HEAD, then falls back to "main" or "master".
func DetectBaseBranch(repoPath string) string {
	// Try symbolic-ref first
	cmd := exec.Command("git", "-C", repoPath, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	output, err := cmd.Output()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		// Strip "origin/" prefix
		if strings.HasPrefix(ref, "origin/") {
			return strings.TrimPrefix(ref, "origin/")
		}
		if ref != "" {
			return ref
		}
	}

	// Fall back to checking if main branch exists
	checkBranch := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/main")
	if checkBranch.Run() == nil {
		return "main"
	}

	// Fall back to master
	checkMaster := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "--quiet", "refs/heads/master")
	if checkMaster.Run() == nil {
		return "master"
	}

	// Default fallback
	return "main"
}
