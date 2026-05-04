package root

import (
	"github.com/spf13/cobra"

	cmdconfig "github.com/jmrmedev/cctxm/cmd/config"
	cmdexec "github.com/jmrmedev/cctxm/cmd/exec"
	cmdinit "github.com/jmrmedev/cctxm/cmd/init"
	cmdread "github.com/jmrmedev/cctxm/cmd/read"
	cmdrules "github.com/jmrmedev/cctxm/cmd/rules"
	cmdsession "github.com/jmrmedev/cctxm/cmd/session"
	cmdtask "github.com/jmrmedev/cctxm/cmd/task"
)

var Cmd = &cobra.Command{
	Use:   "cctxm",
	Short: "Copilot Context Manager — intelligent output filtering for LLM-assisted development",
}

func init() {
	Cmd.PersistentFlags().StringP("session", "s", "", "session ID to use")
	Cmd.PersistentFlags().StringP("project", "p", "", "target project from config")
	Cmd.PersistentFlags().StringP("config", "c", "", "path to config file")

	Cmd.AddCommand(cmdinit.Cmd)
	Cmd.AddCommand(cmdexec.Cmd)
	Cmd.AddCommand(cmdsession.Cmd)
	Cmd.AddCommand(cmdtask.Cmd)
	Cmd.AddCommand(cmdrules.Cmd)
	Cmd.AddCommand(cmdread.Cmd)
	Cmd.AddCommand(cmdconfig.Cmd)
}
