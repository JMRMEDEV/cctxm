package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jmrmedev/cctxm/internal/filter"
	"github.com/jmrmedev/cctxm/internal/session"
)

type Result struct {
	ExitCode    int
	Duration    time.Duration
	RawLog      string // path to raw log file
	Output      string // raw output content
	Filtered    string // filtered output content
	FilteredLog string // path to filtered log file
}

// Run executes a command in the given working directory, captures output to the
// session directory, and returns the result.
func Run(command string, cwd string, mgr *session.Manager, sessionID string, mode filter.Mode, keywords []string) (*Result, error) {
	start := time.Now()

	shell, flag := shellCommand()
	cmd := exec.Command(shell, flag, command)
	cmd.Dir = cwd

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	output := buf.String()
	rawLogPath := ""
	filteredLogPath := ""

	// Apply filter
	filtered := filter.Apply(command, output, mode, keywords)

	if mgr != nil && sessionID != "" {
		entry := session.CommandEntry{
			Command:   command,
			ExitCode:  exitCode,
			Duration:  duration.Round(time.Millisecond).String(),
			Timestamp: start,
		}
		if logErr := mgr.LogCommand(sessionID, entry); logErr == nil {
			commands, _ := mgr.LoadCommands(sessionID)
			if len(commands) > 0 {
				last := commands[len(commands)-1]
				rawLogPath = filepath.Join(mgr.SessionDir(sessionID), last.RawLog)
				os.WriteFile(rawLogPath, []byte(output), 0644)
				filteredLogPath = filepath.Join(mgr.SessionDir(sessionID), last.Filtered)
				os.WriteFile(filteredLogPath, []byte(filtered.Output), 0644)
			}
		}
	}

	// Print filtered output to stdout
	fmt.Print(filtered.Output)

	return &Result{
		ExitCode:    exitCode,
		Duration:    duration,
		RawLog:      rawLogPath,
		Output:      output,
		Filtered:    filtered.Output,
		FilteredLog: filteredLogPath,
	}, nil
}

func shellCommand() (string, string) {
	if runtime.GOOS == "windows" {
		return "cmd", "/C"
	}
	return "sh", "-c"
}

// ParseArgs splits the args around "--" to separate cctxm flags from the command.
func ParseArgs(args []string) string {
	return strings.Join(args, " ")
}
