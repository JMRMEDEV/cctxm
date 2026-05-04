package root

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandExists(t *testing.T) {
	if Cmd == nil {
		t.Fatal("root command is nil")
	}
	if Cmd.Use != "cctxm" {
		t.Errorf("expected Use 'cctxm', got '%s'", Cmd.Use)
	}
}

func TestSubcommandsRegistered(t *testing.T) {
	expected := []string{"init", "exec", "session", "task", "rules", "read", "config"}
	cmds := Cmd.Commands()

	registered := make(map[string]bool)
	for _, c := range cmds {
		registered[c.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("subcommand '%s' not registered", name)
		}
	}
}

func TestGlobalFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"session", "s"},
		{"project", "p"},
		{"config", "c"},
	}

	for _, f := range flags {
		flag := Cmd.PersistentFlags().Lookup(f.name)
		if flag == nil {
			t.Errorf("global flag '--%s' not found", f.name)
			continue
		}
		if flag.Shorthand != f.shorthand {
			t.Errorf("flag '--%s' shorthand: expected '%s', got '%s'", f.name, f.shorthand, flag.Shorthand)
		}
	}
}

func TestSessionSubcommands(t *testing.T) {
	expected := []string{"start", "list", "show", "restore", "clean", "logs"}
	var sessionCmd *cobra.Command
	for _, c := range Cmd.Commands() {
		if c.Name() == "session" {
			sessionCmd = c
			break
		}
	}
	if sessionCmd == nil {
		t.Fatal("session command not found")
		return
	}

	registered := make(map[string]bool)
	for _, c := range sessionCmd.Commands() {
		registered[c.Name()] = true
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("session subcommand '%s' not registered", name)
		}
	}
}

func TestTaskSubcommands(t *testing.T) {
	expected := []string{"set", "show", "clear", "mode"}
	var taskCmd *cobra.Command
	for _, c := range Cmd.Commands() {
		if c.Name() == "task" {
			taskCmd = c
			break
		}
	}
	if taskCmd == nil {
		t.Fatal("task command not found")
		return
	}

	registered := make(map[string]bool)
	for _, c := range taskCmd.Commands() {
		registered[c.Name()] = true
	}
	for _, name := range expected {
		if !registered[name] {
			t.Errorf("task subcommand '%s' not registered", name)
		}
	}
}
