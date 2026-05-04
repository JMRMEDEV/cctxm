package router

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmrmedev/cctxm/internal/config"
)

// Resolve returns the absolute working directory for a given project name.
// If project is empty, it falls back to defaultProject from config, then cwd.
func Resolve(cfg *config.Config, workspaceRoot, project string) (string, error) {
	if project == "" {
		project = cfg.DefaultProject
	}
	if project == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		return cwd, nil
	}

	dir, ok := cfg.Projects[project]
	if !ok {
		available := make([]string, 0, len(cfg.Projects))
		for k := range cfg.Projects {
			available = append(available, k)
		}
		return "", fmt.Errorf("unknown project '%s' (available: %v)", project, available)
	}

	if filepath.IsAbs(dir) {
		return dir, nil
	}
	abs := filepath.Join(workspaceRoot, dir)
	abs = filepath.Clean(abs)

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("project '%s' directory does not exist: %s", project, abs)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project '%s' path is not a directory: %s", project, abs)
	}

	return abs, nil
}
