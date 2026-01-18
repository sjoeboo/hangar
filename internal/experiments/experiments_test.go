package experiments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Setenv("AGENTDECK_PROFILE", "_test")
	os.Exit(m.Run())
}

func TestListExperiments(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some experiment folders
	folders := []string{
		"2025-01-15-redis-cache",
		"2025-01-16-api-test",
		"2025-01-18-database-migration",
		"not-dated-folder",
	}
	for _, f := range folders {
		os.MkdirAll(filepath.Join(tmpDir, f), 0755)
	}

	experiments, err := ListExperiments(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(experiments) != 4 {
		t.Errorf("expected 4 experiments, got %d", len(experiments))
	}
}

func TestFuzzyFind(t *testing.T) {
	experiments := []Experiment{
		{Name: "redis-cache", Path: "/tmp/2025-01-15-redis-cache"},
		{Name: "redis-server", Path: "/tmp/2025-01-16-redis-server"},
		{Name: "api-test", Path: "/tmp/2025-01-17-api-test"},
	}

	matches := FuzzyFind(experiments, "redis")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches for 'redis', got %d", len(matches))
	}

	matches = FuzzyFind(experiments, "rds")
	if len(matches) < 1 {
		t.Error("expected at least 1 fuzzy match for 'rds'")
	}
}

func TestCreateExperiment(t *testing.T) {
	tmpDir := t.TempDir()
	today := time.Now().Format("2006-01-02")

	exp, err := CreateExperiment(tmpDir, "my-project", true)
	if err != nil {
		t.Fatal(err)
	}

	expectedName := today + "-my-project"
	if !strings.HasSuffix(exp.Path, expectedName) {
		t.Errorf("expected path ending with %q, got %q", expectedName, exp.Path)
	}

	// Verify directory was created
	if _, err := os.Stat(exp.Path); os.IsNotExist(err) {
		t.Error("experiment directory was not created")
	}
}

func TestCreateExperiment_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	today := time.Now().Format("2006-01-02")

	// Create first experiment
	exp1, _ := CreateExperiment(tmpDir, "my-project", true)

	// Create second with same name - should add suffix
	exp2, err := CreateExperiment(tmpDir, "my-project", true)
	if err != nil {
		t.Fatal(err)
	}

	if exp1.Path == exp2.Path {
		t.Error("expected different paths for duplicate names")
	}

	expectedSuffix := today + "-my-project-2"
	if !strings.HasSuffix(exp2.Path, expectedSuffix) {
		t.Errorf("expected path ending with %q, got %q", expectedSuffix, exp2.Path)
	}
}
