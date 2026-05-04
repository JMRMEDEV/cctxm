package task

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	"github.com/jmrmedev/cctxm/internal/session"
	tasklib "github.com/jmrmedev/cctxm/internal/task"
)

var Cmd = &cobra.Command{
	Use:   "task",
	Short: "Manage task context for the active session",
}

var setCmd = &cobra.Command{
	Use:   "set <description>",
	Short: "Set current task and auto-detect filter mode",
	Args:  cobra.ExactArgs(1),
	RunE:  runSet,
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current task, keywords, and filter mode",
	RunE:  runShow,
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear task context",
	RunE:  runClear,
}

var modeCmd = &cobra.Command{
	Use:   "mode <strict|normal|verbose>",
	Short: "Manually override filter mode",
	Args:  cobra.ExactArgs(1),
	RunE:  runMode,
}

func init() {
	Cmd.AddCommand(setCmd, showCmd, clearCmd, modeCmd)
}

func getManager(cmd *cobra.Command) (*session.Manager, string, error) {
	cwd, _ := os.Getwd()
	root := config.FindRoot(cwd)
	if root == "" {
		return nil, "", fmt.Errorf("no .cctxm directory found (run 'cctxm init' first)")
	}
	mgr := session.NewManager(fmt.Sprintf("%s/%s", root, config.CctxmDir))
	sid, _ := mgr.Active()
	if sid == "" {
		return nil, "", fmt.Errorf("no active session (run 'cctxm session start' first)")
	}
	return mgr, sid, nil
}

func runSet(cmd *cobra.Command, args []string) error {
	mgr, sid, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.LoadMeta(sid)
	if err != nil {
		return err
	}

	description := args[0]
	meta.Task = description
	meta.Keywords = tasklib.ExtractKeywords(description)
	meta.FilterMode = tasklib.DetectMode(description)

	if err := mgr.UpdateMeta(sid, meta); err != nil {
		return err
	}

	fmt.Printf("Task: %s\n", meta.Task)
	fmt.Printf("Keywords: %s\n", strings.Join(meta.Keywords, ", "))
	fmt.Printf("Filter mode: %s (auto-detected)\n", meta.FilterMode)
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	mgr, sid, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.LoadMeta(sid)
	if err != nil {
		return err
	}

	if meta.Task == "" {
		fmt.Println("No task set.")
		return nil
	}
	fmt.Printf("Task: %s\n", meta.Task)
	fmt.Printf("Keywords: %s\n", strings.Join(meta.Keywords, ", "))
	fmt.Printf("Filter mode: %s\n", meta.FilterMode)
	return nil
}

func runClear(cmd *cobra.Command, args []string) error {
	mgr, sid, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.LoadMeta(sid)
	if err != nil {
		return err
	}

	meta.Task = ""
	meta.Keywords = nil
	meta.FilterMode = "normal"

	if err := mgr.UpdateMeta(sid, meta); err != nil {
		return err
	}
	fmt.Println("Task cleared. Filter mode reset to normal.")
	return nil
}

func runMode(cmd *cobra.Command, args []string) error {
	mode := strings.ToLower(args[0])
	if mode != "strict" && mode != "normal" && mode != "verbose" {
		return fmt.Errorf("invalid mode '%s' (use strict, normal, or verbose)", mode)
	}

	mgr, sid, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.LoadMeta(sid)
	if err != nil {
		return err
	}

	meta.FilterMode = mode
	if err := mgr.UpdateMeta(sid, meta); err != nil {
		return err
	}
	fmt.Printf("Filter mode set to: %s\n", mode)
	return nil
}
