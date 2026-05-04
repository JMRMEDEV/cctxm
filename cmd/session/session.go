package session

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	sess "github.com/jmrmedev/cctxm/internal/session"
	tasklib "github.com/jmrmedev/cctxm/internal/task"
)

var Cmd = &cobra.Command{
	Use:   "session",
	Short: "Manage cctxm sessions",
}

var startCmd = &cobra.Command{
	Use:   "start <description>",
	Short: "Create and activate a new session",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sessions",
	RunE:  runList,
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active session details",
	RunE:  runShow,
}

var restoreCmd = &cobra.Command{
	Use:   "restore <session-id>",
	Short: "Restore and activate a session",
	Args:  cobra.ExactArgs(1),
	RunE:  runRestore,
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove old sessions",
	RunE:  runClean,
}

var logsCmd = &cobra.Command{
	Use:   "logs [command-number]",
	Short: "List or view session log files",
	RunE:  runLogs,
}

func init() {
	cleanCmd.Flags().Bool("all", false, "remove all sessions")
	Cmd.AddCommand(startCmd, listCmd, showCmd, restoreCmd, cleanCmd, logsCmd)
}

func getManager(cmd *cobra.Command) (*sess.Manager, error) {
	cfgFlag, _ := cmd.Flags().GetString("config")
	var root string
	if cfgFlag != "" {
		// Config flag points to config file; derive .cctxm dir from it
		root = cfgFlag
		// Walk up to find .cctxm parent
		cwd, _ := os.Getwd()
		root = config.FindRoot(cwd)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = config.FindRoot(cwd)
	}
	if root == "" {
		return nil, fmt.Errorf("no .cctxm directory found (run 'cctxm init' first)")
	}
	return sess.NewManager(fmt.Sprintf("%s/%s", root, config.CctxmDir)), nil
}

func runStart(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.Start(args[0])
	if err != nil {
		return err
	}

	// Auto-detect keywords and filter mode from description
	meta.Keywords = tasklib.ExtractKeywords(args[0])
	meta.FilterMode = tasklib.DetectMode(args[0])
	mgr.UpdateMeta(meta.ID, meta)

	fmt.Printf("Session created: %s\n", meta.ID)
	fmt.Printf("Task: %s\n", meta.Task)
	fmt.Printf("Filter mode: %s\n", meta.FilterMode)
	if len(meta.Keywords) > 0 {
		fmt.Printf("Keywords: %v\n", meta.Keywords)
	}
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	sessions, err := mgr.List()
	if err != nil {
		return err
	}
	active, _ := mgr.Active()

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}
	for _, s := range sessions {
		marker := "  "
		if s.ID == active {
			marker = "* "
		}
		fmt.Printf("%s%s  %s  (%d commands)  %s\n",
			marker, s.ID, s.Task, s.CommandCount,
			s.CreatedAt.Format("2006-01-02 15:04"))
	}
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	active, err := mgr.Active()
	if err != nil || active == "" {
		fmt.Println("No active session.")
		return nil
	}
	meta, err := mgr.LoadMeta(active)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	fmt.Printf("Session: %s\n", meta.ID)
	fmt.Printf("Task: %s\n", meta.Task)
	fmt.Printf("Filter mode: %s\n", meta.FilterMode)
	fmt.Printf("Commands run: %d\n", meta.CommandCount)
	fmt.Printf("Created: %s\n", meta.CreatedAt.Format("2006-01-02 15:04:05"))
	if len(meta.Keywords) > 0 {
		fmt.Printf("Keywords: %v\n", meta.Keywords)
	}
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	meta, err := mgr.Restore(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("Session restored: %s\n", meta.ID)
	fmt.Printf("Task: %s\n", meta.Task)
	fmt.Printf("Filter mode: %s\n", meta.FilterMode)
	fmt.Printf("Commands run: %d\n", meta.CommandCount)
	return nil
}

func runClean(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	all, _ := cmd.Flags().GetBool("all")
	days := 7
	if all {
		days = 0
	}
	removed, err := mgr.Clean(days)
	if err != nil {
		return err
	}
	fmt.Printf("Removed %d session(s).\n", removed)
	return nil
}

func runLogs(cmd *cobra.Command, args []string) error {
	mgr, err := getManager(cmd)
	if err != nil {
		return err
	}
	active, _ := mgr.Active()
	if active == "" {
		fmt.Println("No active session.")
		return nil
	}
	commands, err := mgr.LoadCommands(active)
	if err != nil {
		fmt.Println("No commands logged yet.")
		return nil
	}

	if len(args) > 0 {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid command number: %s", args[0])
		}
		for _, c := range commands {
			if c.Number == num {
				path := fmt.Sprintf("%s/%s", mgr.SessionDir(active), c.RawLog)
				data, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("log file not found: %s", path)
				}
				fmt.Print(string(data))
				return nil
			}
		}
		return fmt.Errorf("command #%d not found", num)
	}

	for _, c := range commands {
		fmt.Printf("#%d  %s  (exit %d, %s)  %s\n",
			c.Number, c.Command, c.ExitCode, c.Duration,
			c.Timestamp.Format("15:04:05"))
	}
	return nil
}
