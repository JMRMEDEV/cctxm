package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	cfg "github.com/jmrmedev/cctxm/internal/config"
)

func setupWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".cctxm"), 0755)
	configPath := cfg.ConfigPath(root)
	c := cfg.Default()
	c.Projects = map[string]string{"api": "./services/api"}
	if err := cfg.Save(configPath, c); err != nil {
		t.Fatal(err)
	}
	return root, configPath
}

func buildTestCmd(t *testing.T, configPath string) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "cctxm"}
	root.PersistentFlags().StringP("config", "c", "", "")

	configCmd := &cobra.Command{Use: "config"}
	s := &cobra.Command{Use: "show", RunE: runShow}
	e := &cobra.Command{Use: "edit", RunE: runEdit}
	st := &cobra.Command{Use: "set", Args: cobra.ExactArgs(2), RunE: runSet}
	configCmd.AddCommand(s, e, st)
	root.AddCommand(configCmd)

	root.SetArgs(append([]string{"--config", configPath}, t.Name()))
	return root
}

func TestShowCmd(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "show"})
	if err := root.Execute(); err != nil {
		t.Fatalf("show failed: %v", err)
	}
}

func TestSetCmdProject(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "projects.web", "./frontend"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	c, _ := cfg.Load(configPath)
	if c.Projects["web"] != "./frontend" {
		t.Errorf("expected project 'web' = './frontend', got '%s'", c.Projects["web"])
	}
	if c.Projects["api"] != "./services/api" {
		t.Errorf("expected project 'api' preserved, got '%s'", c.Projects["api"])
	}
}

func TestSetCmdFilterMaxLines(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "filter.max_lines", "500"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	c, _ := cfg.Load(configPath)
	if c.Filter.MaxLines != 500 {
		t.Errorf("expected MaxLines 500, got %d", c.Filter.MaxLines)
	}
}

func TestSetCmdInvalidInt(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "filter.max_lines", "notanumber"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for invalid integer")
	}
}

func TestSetCmdUnknownKey(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "unknown.key", "value"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestSetCmdDefaultProject(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "default_project", "api"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	c, _ := cfg.Load(configPath)
	if c.DefaultProject != "api" {
		t.Errorf("expected DefaultProject 'api', got '%s'", c.DefaultProject)
	}
}

func TestSetCmdRetentionDays(t *testing.T) {
	_, configPath := setupWorkspace(t)
	root := buildTestCmd(t, configPath)
	root.SetArgs([]string{"--config", configPath, "config", "set", "session.retention_days", "14"})
	if err := root.Execute(); err != nil {
		t.Fatalf("set failed: %v", err)
	}

	c, _ := cfg.Load(configPath)
	if c.Session.RetentionDays != 14 {
		t.Errorf("expected RetentionDays 14, got %d", c.Session.RetentionDays)
	}
}
