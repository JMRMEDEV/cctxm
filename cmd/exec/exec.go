package exec

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	"github.com/jmrmedev/cctxm/internal/executor"
	"github.com/jmrmedev/cctxm/internal/filter"
	"github.com/jmrmedev/cctxm/internal/router"
	"github.com/jmrmedev/cctxm/internal/session"
)

var Cmd = &cobra.Command{
	Use:   "exec [flags] -- <command>",
	Short: "Execute a command with output filtering",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runExec,
}

func init() {
	Cmd.Flags().Bool("strict", false, "force strict filter mode")
	Cmd.Flags().Bool("verbose", false, "force verbose filter mode")
}

func runExec(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Flags().GetString("config")
	var cfg *config.Config
	var workspaceRoot string

	cwd, _ := os.Getwd()
	workspaceRoot = config.FindRoot(cwd)

	if cfgPath != "" {
		c, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		cfg = c
	} else if workspaceRoot != "" {
		c, err := config.Load(config.ConfigPath(workspaceRoot))
		if err != nil {
			return err
		}
		cfg = c
	} else {
		cfg = config.Default()
		workspaceRoot = cwd
	}

	// Resolve working directory
	project, _ := cmd.Flags().GetString("project")
	execDir, err := router.Resolve(cfg, workspaceRoot, project)
	if err != nil {
		return err
	}

	// Resolve session + filter mode/keywords
	var mgr *session.Manager
	var sessionID string
	mode := filter.ModeNormal
	var keywords []string

	if workspaceRoot != "" {
		mgr = session.NewManager(fmt.Sprintf("%s/%s", workspaceRoot, config.CctxmDir))
		sidFlag, _ := cmd.Flags().GetString("session")
		if sidFlag != "" {
			sessionID = sidFlag
		} else {
			sessionID, _ = mgr.Active()
		}

		// Load task context for filter mode and keywords
		if sessionID != "" {
			if meta, err := mgr.LoadMeta(sessionID); err == nil {
				mode = filter.ParseMode(meta.FilterMode)
				keywords = meta.Keywords
			}
		}
	}

	// CLI flag overrides
	if strict, _ := cmd.Flags().GetBool("strict"); strict {
		mode = filter.ModeStrict
	}
	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		mode = filter.ModeVerbose
	}

	command := executor.ParseArgs(args)
	result, err := executor.Run(command, execDir, mgr, sessionID, mode, keywords)
	if err != nil {
		return err
	}

	if result.RawLog != "" {
		fmt.Fprintf(os.Stderr, "\n[cctxm] Full output: %s\n", result.RawLog)
	}

	os.Exit(result.ExitCode)
	return nil
}
