package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".cctxm")
	os.MkdirAll(dir, 0755)
	return NewManager(dir)
}

func TestStartCreatesSession(t *testing.T) {
	mgr := newTestManager(t)

	meta, err := mgr.Start("fix auth bug")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !strings.HasPrefix(meta.ID, "s_") {
		t.Errorf("expected ID starting with 's_', got '%s'", meta.ID)
	}
	if meta.Task != "fix auth bug" {
		t.Errorf("expected task 'fix auth bug', got '%s'", meta.Task)
	}
	if meta.FilterMode != "normal" {
		t.Errorf("expected filter mode 'normal', got '%s'", meta.FilterMode)
	}

	// Verify directory exists
	dir := mgr.SessionDir(meta.ID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("session directory not created")
	}

	// Verify it's set as active
	active, _ := mgr.Active()
	if active != meta.ID {
		t.Errorf("expected active session '%s', got '%s'", meta.ID, active)
	}
}

func TestListSessions(t *testing.T) {
	mgr := newTestManager(t)

	mgr.Start("first task")
	time.Sleep(10 * time.Millisecond)
	mgr.Start("second task")

	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Newest first
	if sessions[0].Task != "second task" {
		t.Errorf("expected newest first, got '%s'", sessions[0].Task)
	}
}

func TestListEmptyDir(t *testing.T) {
	mgr := newTestManager(t)

	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestRestoreSession(t *testing.T) {
	mgr := newTestManager(t)

	meta1, _ := mgr.Start("first")
	mgr.Start("second")

	restored, err := mgr.Restore(meta1.ID)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	if restored.Task != "first" {
		t.Errorf("expected task 'first', got '%s'", restored.Task)
	}

	active, _ := mgr.Active()
	if active != meta1.ID {
		t.Errorf("expected active '%s', got '%s'", meta1.ID, active)
	}
}

func TestRestoreNonExistent(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.Restore("s_nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
}

func TestCleanOldSessions(t *testing.T) {
	mgr := newTestManager(t)

	meta1, _ := mgr.Start("old task")
	// Backdate the first session
	m, _ := mgr.LoadMeta(meta1.ID)
	m.CreatedAt = time.Now().AddDate(0, 0, -10)
	mgr.UpdateMeta(meta1.ID, m)

	meta2, _ := mgr.Start("recent task") // This is now active

	removed, err := mgr.Clean(7)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	sessions, _ := mgr.List()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 remaining, got %d", len(sessions))
	}
	if sessions[0].ID != meta2.ID {
		t.Errorf("expected recent session to remain")
	}
}

func TestCleanPreservesActive(t *testing.T) {
	mgr := newTestManager(t)

	meta, _ := mgr.Start("active task")
	m, _ := mgr.LoadMeta(meta.ID)
	m.CreatedAt = time.Now().AddDate(0, 0, -30)
	mgr.UpdateMeta(meta.ID, m)

	removed, _ := mgr.Clean(7)
	if removed != 0 {
		t.Errorf("expected 0 removed (active session protected), got %d", removed)
	}
}

func TestCleanAll(t *testing.T) {
	mgr := newTestManager(t)

	mgr.Start("task 1")
	time.Sleep(10 * time.Millisecond)
	mgr.Start("task 2")

	// Clean all (days <= 0), but active is still protected
	removed, _ := mgr.Clean(0)
	if removed != 1 {
		t.Errorf("expected 1 removed (active protected), got %d", removed)
	}
}

func TestLogCommand(t *testing.T) {
	mgr := newTestManager(t)

	meta, _ := mgr.Start("test task")

	entry := CommandEntry{
		Command:   "docker logs my-app",
		ExitCode:  0,
		Duration:  "1.2s",
		Timestamp: time.Now(),
	}
	if err := mgr.LogCommand(meta.ID, entry); err != nil {
		t.Fatalf("LogCommand failed: %v", err)
	}

	commands, err := mgr.LoadCommands(meta.ID)
	if err != nil {
		t.Fatalf("LoadCommands failed: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}
	if commands[0].Number != 1 {
		t.Errorf("expected number 1, got %d", commands[0].Number)
	}
	if commands[0].RawLog != "001-docker-logs.raw.log" {
		t.Errorf("unexpected raw log name: %s", commands[0].RawLog)
	}

	// Log a second command
	entry2 := CommandEntry{
		Command:   "npm test",
		ExitCode:  1,
		Duration:  "5.3s",
		Timestamp: time.Now(),
	}
	mgr.LogCommand(meta.ID, entry2)

	commands, _ = mgr.LoadCommands(meta.ID)
	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}
	if commands[1].Number != 2 {
		t.Errorf("expected number 2, got %d", commands[1].Number)
	}

	// Verify meta command count updated
	updated, _ := mgr.LoadMeta(meta.ID)
	if updated.CommandCount != 2 {
		t.Errorf("expected command count 2, got %d", updated.CommandCount)
	}
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"docker logs my-app", "docker-logs"},
		{"npm test", "npm-test"},
		{"", "cmd"},
		{"NPM RUN BUILD", "npm-run"},
	}
	for _, tt := range tests {
		got := sanitizeLabel(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeLabel(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
