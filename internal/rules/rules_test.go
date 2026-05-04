package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	cctxmDir := filepath.Join(root, ".cctxm")
	os.MkdirAll(filepath.Join(cctxmDir, "overridden"), 0755)
	os.MkdirAll(filepath.Join(cctxmDir, "rules"), 0755)
	os.MkdirAll(filepath.Join(root, ".git", "info"), 0755)
	return root, cctxmDir
}

func TestInjectCreatesInstructions(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	err := Inject(root, cctxmDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	instrPath := filepath.Join(root, ".github", "copilot", "instructions.md")
	data, err := os.ReadFile(instrPath)
	if err != nil {
		t.Fatal("instructions.md not created")
	}
	content := string(data)

	if !strings.Contains(content, "cctxm exec") {
		t.Error("should contain cctxm exec instruction")
	}
	if !strings.Contains(content, "cctxm read") {
		t.Error("should contain cctxm read instruction")
	}
	if !strings.Contains(content, "session start") {
		t.Error("should contain session start instruction")
	}
}

func TestInjectBackupsOriginal(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	// Create original copilot file
	origDir := filepath.Join(root, ".github", "copilot")
	os.MkdirAll(origDir, 0755)
	os.WriteFile(filepath.Join(origDir, "original.md"), []byte("original content"), 0644)

	err := Inject(root, cctxmDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Check backup exists
	backupPath := filepath.Join(cctxmDir, "overridden", "copilot", "original.md")
	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal("backup not created")
	}
	if string(data) != "original content" {
		t.Error("backup content mismatch")
	}
}

func TestInjectAppendsUserRules(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	// Create user rule
	os.WriteFile(filepath.Join(cctxmDir, "rules", "coding.md"), []byte("# Always use TypeScript"), 0644)

	err := Inject(root, cctxmDir, []string{".cctxm/rules/*.md"})
	if err != nil {
		t.Fatal(err)
	}

	instrPath := filepath.Join(root, ".github", "copilot", "instructions.md")
	data, _ := os.ReadFile(instrPath)
	if !strings.Contains(string(data), "Always use TypeScript") {
		t.Error("should contain user rule content")
	}
	if !strings.Contains(string(data), "coding.md") {
		t.Error("should contain source attribution")
	}
}

func TestInjectAddsGitExclude(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	Inject(root, cctxmDir, nil)

	excludePath := filepath.Join(root, ".git", "info", "exclude")
	data, err := os.ReadFile(excludePath)
	if err != nil {
		t.Fatal("exclude file not found")
	}
	if !strings.Contains(string(data), ".github/copilot/") {
		t.Error("should add copilot dir to git exclude")
	}
}

func TestInjectIdempotent(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	origDir := filepath.Join(root, ".github", "copilot")
	os.MkdirAll(origDir, 0755)
	os.WriteFile(filepath.Join(origDir, "original.md"), []byte("original"), 0644)

	// Inject twice
	Inject(root, cctxmDir, nil)
	Inject(root, cctxmDir, nil)

	// Backup should still have original content
	data, _ := os.ReadFile(filepath.Join(cctxmDir, "overridden", "copilot", "original.md"))
	if string(data) != "original" {
		t.Error("second inject should not overwrite backup")
	}
}

func TestRestore(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	// Create original and inject
	origDir := filepath.Join(root, ".github", "copilot")
	os.MkdirAll(origDir, 0755)
	os.WriteFile(filepath.Join(origDir, "original.md"), []byte("original content"), 0644)

	Inject(root, cctxmDir, nil)

	// Verify injected
	instrPath := filepath.Join(root, ".github", "copilot", "instructions.md")
	if _, err := os.Stat(instrPath); os.IsNotExist(err) {
		t.Fatal("inject didn't work")
	}

	// Restore
	err := Restore(root, cctxmDir)
	if err != nil {
		t.Fatal(err)
	}

	// Original should be back
	data, err := os.ReadFile(filepath.Join(origDir, "original.md"))
	if err != nil {
		t.Fatal("original not restored")
	}
	if string(data) != "original content" {
		t.Error("restored content mismatch")
	}

	// Injected file should be gone (replaced by restore)
	if _, err := os.Stat(instrPath); err == nil {
		t.Error("injected instructions.md should be removed after restore")
	}
}

func TestRestoreNoBackup(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	// Create injected dir without backup
	origDir := filepath.Join(root, ".github", "copilot")
	os.MkdirAll(origDir, 0755)
	os.WriteFile(filepath.Join(origDir, "instructions.md"), []byte("injected"), 0644)

	err := Restore(root, cctxmDir)
	if err != nil {
		t.Fatal(err)
	}

	// Dir should be removed
	if _, err := os.Stat(origDir); err == nil {
		t.Error("copilot dir should be removed when no backup")
	}
}

func TestShow(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)
	os.WriteFile(filepath.Join(cctxmDir, "rules", "my-rule.md"), []byte("# My Rule"), 0644)

	content := Show(root, []string{".cctxm/rules/*.md"})

	if !strings.Contains(content, "cctxm exec") {
		t.Error("should contain core instructions")
	}
	if !strings.Contains(content, "My Rule") {
		t.Error("should contain user rule")
	}
}
