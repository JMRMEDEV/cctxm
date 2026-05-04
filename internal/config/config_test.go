package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Filter.MaxLines != 200 {
		t.Errorf("expected MaxLines 200, got %d", cfg.Filter.MaxLines)
	}
	if cfg.Filter.ContextLines != 10 {
		t.Errorf("expected ContextLines 10, got %d", cfg.Filter.ContextLines)
	}
	if cfg.Filter.ReadThreshold != 5120 {
		t.Errorf("expected ReadThreshold 5120, got %d", cfg.Filter.ReadThreshold)
	}
	if cfg.Session.RetentionDays != 7 {
		t.Errorf("expected RetentionDays 7, got %d", cfg.Session.RetentionDays)
	}
	if len(cfg.Filter.FullReadExtensions) == 0 {
		t.Error("expected non-empty FullReadExtensions")
	}
	if len(cfg.Rules.Suppress) == 0 {
		t.Error("expected non-empty Suppress")
	}
}

func TestLoadNonExistent(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	// Should return defaults
	if cfg.Filter.MaxLines != 200 {
		t.Errorf("expected default MaxLines 200, got %d", cfg.Filter.MaxLines)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".cctxm", "config.yaml")

	cfg := Default()
	cfg.Projects = map[string]string{
		"api":    "./services/api",
		"web-ui": "./frontend",
	}
	cfg.DefaultProject = "api"
	cfg.Filter.MaxLines = 500

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.DefaultProject != "api" {
		t.Errorf("expected DefaultProject 'api', got '%s'", loaded.DefaultProject)
	}
	if loaded.Filter.MaxLines != 500 {
		t.Errorf("expected MaxLines 500, got %d", loaded.Filter.MaxLines)
	}
	if loaded.Projects["api"] != "./services/api" {
		t.Errorf("expected project 'api' = './services/api', got '%s'", loaded.Projects["api"])
	}
	if loaded.Projects["web-ui"] != "./frontend" {
		t.Errorf("expected project 'web-ui' = './frontend', got '%s'", loaded.Projects["web-ui"])
	}
	// Defaults should still be present for unset fields
	if loaded.Filter.ContextLines != 10 {
		t.Errorf("expected default ContextLines 10, got %d", loaded.Filter.ContextLines)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("{{invalid yaml"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestFindRoot(t *testing.T) {
	// Create a nested structure with .cctxm at root
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".cctxm"), 0755)
	nested := filepath.Join(root, "services", "api", "src")
	os.MkdirAll(nested, 0755)

	found := FindRoot(nested)
	if found != root {
		t.Errorf("expected root '%s', got '%s'", root, found)
	}
}

func TestFindRootNotFound(t *testing.T) {
	dir := t.TempDir()
	found := FindRoot(dir)
	if found != "" {
		t.Errorf("expected empty string, got '%s'", found)
	}
}

func TestConfigPath(t *testing.T) {
	p := ConfigPath("/workspace")
	expected := filepath.Join("/workspace", ".cctxm", "config.yaml")
	if p != expected {
		t.Errorf("expected '%s', got '%s'", expected, p)
	}
}
