package router

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jmrmedev/cctxm/internal/config"
)

func TestResolveExplicitProject(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "services", "api")
	os.MkdirAll(apiDir, 0755)

	cfg := config.Default()
	cfg.Projects = map[string]string{"api": "./services/api"}

	dir, err := Resolve(cfg, root, "api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != apiDir {
		t.Errorf("expected '%s', got '%s'", apiDir, dir)
	}
}

func TestResolveDefaultProject(t *testing.T) {
	root := t.TempDir()
	uiDir := filepath.Join(root, "frontend")
	os.MkdirAll(uiDir, 0755)

	cfg := config.Default()
	cfg.Projects = map[string]string{"ui": "./frontend"}
	cfg.DefaultProject = "ui"

	dir, err := Resolve(cfg, root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dir != uiDir {
		t.Errorf("expected '%s', got '%s'", uiDir, dir)
	}
}

func TestResolveFallbackToCwd(t *testing.T) {
	cfg := config.Default()

	dir, err := Resolve(cfg, "/some/root", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cwd, _ := os.Getwd()
	if dir != cwd {
		t.Errorf("expected cwd '%s', got '%s'", cwd, dir)
	}
}

func TestResolveUnknownProject(t *testing.T) {
	cfg := config.Default()
	cfg.Projects = map[string]string{"api": "./services/api"}

	_, err := Resolve(cfg, "/root", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown project")
	}
	if !strings.Contains(err.Error(), "unknown project") {
		t.Errorf("expected 'unknown project' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "api") {
		t.Errorf("expected available projects listed in error, got: %v", err)
	}
}

func TestResolveDirNotExist(t *testing.T) {
	cfg := config.Default()
	cfg.Projects = map[string]string{"api": "./services/api"}

	_, err := Resolve(cfg, "/nonexistent/root", "api")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' in error, got: %v", err)
	}
}

func TestResolveAbsolutePath(t *testing.T) {
	dir := t.TempDir()

	cfg := config.Default()
	cfg.Projects = map[string]string{"ext": dir}

	resolved, err := Resolve(cfg, "/other/root", "ext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != dir {
		t.Errorf("expected '%s', got '%s'", dir, resolved)
	}
}

func TestResolveNotADirectory(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "afile")
	os.WriteFile(filePath, []byte("hi"), 0644)

	cfg := config.Default()
	cfg.Projects = map[string]string{"bad": "./afile"}

	_, err := Resolve(cfg, root, "bad")
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' in error, got: %v", err)
	}
}
