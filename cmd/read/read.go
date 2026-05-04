package read

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	"github.com/jmrmedev/cctxm/internal/reader"
	"github.com/jmrmedev/cctxm/internal/session"
)

var Cmd = &cobra.Command{
	Use:   "read <file>",
	Short: "Smart file reading with size-based filtering",
	Args:  cobra.ExactArgs(1),
	RunE:  runRead,
}

func init() {
	Cmd.Flags().Bool("full", false, "force full file read")
	Cmd.Flags().String("search", "", "search for specific terms")
}

func runRead(cmd *cobra.Command, args []string) error {
	fullFlag, _ := cmd.Flags().GetBool("full")
	searchFlag, _ := cmd.Flags().GetString("search")

	// Load threshold from config
	threshold := reader.DefaultThreshold
	cwd, _ := os.Getwd()
	root := config.FindRoot(cwd)
	var keywords []string

	if root != "" {
		if cfg, err := config.Load(config.ConfigPath(root)); err == nil {
			if cfg.Filter.ReadThreshold > 0 {
				threshold = cfg.Filter.ReadThreshold
			}
		}
		mgr := session.NewManager(fmt.Sprintf("%s/%s", root, config.CctxmDir))
		if sid, _ := mgr.Active(); sid != "" {
			if meta, err := mgr.LoadMeta(sid); err == nil {
				keywords = meta.Keywords
			}
		}
	}

	out, err := reader.Read(args[0], keywords, threshold, fullFlag, searchFlag)
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}
