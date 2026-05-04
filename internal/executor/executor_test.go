package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jmrmedev/cctxm/internal/filter"
	"github.com/jmrmedev/cctxm/internal/session"
)

func newTestSession(t *testing.T) (*session.Manager, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), ".cctxm")
	os.MkdirAll(dir, 0755)
	mgr := session.NewManager(dir)
	meta, err := mgr.Start("test")
	if err != nil {
		t.Fatal(err)
	}
	return mgr, meta.ID
}

func TestRunSimpleCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	result, err := Run("echo hello", os.TempDir(), nil, "", filter.ModeVerbose, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output to contain 'hello', got '%s'", result.Output)
	}
}

func TestRunWithNonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	result, err := Run("exit 42", os.TempDir(), nil, "", filter.ModeNormal, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

func TestRunCapturesStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	result, err := Run("echo errout >&2", os.TempDir(), nil, "", filter.ModeVerbose, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(result.Output, "errout") {
		t.Errorf("expected stderr in output, got '%s'", result.Output)
	}
}

func TestRunWithSession(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	mgr, sid := newTestSession(t)

	result, err := Run("echo session-test", os.TempDir(), mgr, sid, filter.ModeVerbose, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	commands, err := mgr.LoadCommands(sid)
	if err != nil {
		t.Fatalf("LoadCommands failed: %v", err)
	}
	if len(commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(commands))
	}

	if result.RawLog == "" {
		t.Fatal("expected raw log path")
	}
	data, err := os.ReadFile(result.RawLog)
	if err != nil {
		t.Fatalf("failed to read raw log: %v", err)
	}
	if !strings.Contains(string(data), "session-test") {
		t.Errorf("raw log missing output")
	}

	// Verify filtered log was also written
	if result.FilteredLog == "" {
		t.Fatal("expected filtered log path")
	}
}

func TestRunInSpecificDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dir := t.TempDir()
	result, err := Run("pwd", dir, nil, "", filter.ModeVerbose, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(result.Output, dir) {
		t.Errorf("expected output to contain '%s', got '%s'", dir, result.Output)
	}
}

func TestRunDuration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	result, err := Run("sleep 0.1", os.TempDir(), nil, "", filter.ModeNormal, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Duration.Milliseconds() < 50 {
		t.Errorf("expected duration >= 50ms, got %v", result.Duration)
	}
}

func TestParseArgs(t *testing.T) {
	got := ParseArgs([]string{"docker", "logs", "my-app"})
	if got != "docker logs my-app" {
		t.Errorf("expected 'docker logs my-app', got '%s'", got)
	}
}
