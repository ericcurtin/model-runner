package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindChartDirectory(t *testing.T) {
	// Change to the repository root
	repoRoot := "../../../"
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatalf("Failed to change to repo root: %v", err)
	}

	chartDir, err := findChartDirectory()
	if err != nil {
		t.Fatalf("Failed to find chart directory: %v", err)
	}

	// Verify Chart.yaml exists
	chartYaml := filepath.Join(chartDir, "Chart.yaml")
	if _, err := os.Stat(chartYaml); err != nil {
		t.Errorf("Chart.yaml not found at %s: %v", chartYaml, err)
	}

	// Verify templates directory exists
	templatesDir := filepath.Join(chartDir, "templates")
	if info, err := os.Stat(templatesDir); err != nil || !info.IsDir() {
		t.Errorf("templates directory not found at %s: %v", templatesDir, err)
	}

	// Verify values.yaml exists
	valuesYaml := filepath.Join(chartDir, "values.yaml")
	if _, err := os.Stat(valuesYaml); err != nil {
		t.Errorf("values.yaml not found at %s: %v", valuesYaml, err)
	}
}
