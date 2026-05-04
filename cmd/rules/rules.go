package rules

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	ruleslib "github.com/jmrmedev/cctxm/internal/rules"
)

var Cmd = &cobra.Command{
	Use:   "rules",
	Short: "Manage Copilot instruction injection",
}

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Generate .github/copilot/ from cctxm rules",
	RunE:  runInject,
}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore original .github/copilot/ from backup",
	RunE:  runRestore,
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Preview injected content",
	RunE:  runShow,
}

func init() {
	Cmd.AddCommand(injectCmd, restoreCmd, showCmd)
}

func resolveRoot(cmd *cobra.Command) (string, string, *config.Config, error) {
	cwd, _ := os.Getwd()
	root := config.FindRoot(cwd)
	if root == "" {
		return "", "", nil, fmt.Errorf("no .cctxm directory found (run 'cctxm init' first)")
	}
	cctxmDir := fmt.Sprintf("%s/%s", root, config.CctxmDir)
	cfg, err := config.Load(config.ConfigPath(root))
	if err != nil {
		cfg = config.Default()
	}
	return root, cctxmDir, cfg, nil
}

func runInject(cmd *cobra.Command, args []string) error {
	root, cctxmDir, cfg, err := resolveRoot(cmd)
	if err != nil {
		return err
	}
	if err := ruleslib.Inject(root, cctxmDir, cfg.Rules.Inject); err != nil {
		return err
	}
	fmt.Println("Injected cctxm instructions into .github/copilot/instructions.md")
	if len(cfg.Rules.Inject) > 0 {
		fmt.Printf("User rules included from: %v\n", cfg.Rules.Inject)
	}
	return nil
}

func runRestore(cmd *cobra.Command, args []string) error {
	root, cctxmDir, _, err := resolveRoot(cmd)
	if err != nil {
		return err
	}
	if err := ruleslib.Restore(root, cctxmDir); err != nil {
		return err
	}
	fmt.Println("Restored original .github/copilot/ from backup.")
	return nil
}

func runShow(cmd *cobra.Command, args []string) error {
	root, _, cfg, err := resolveRoot(cmd)
	if err != nil {
		return err
	}
	content := ruleslib.Show(root, cfg.Rules.Inject)
	fmt.Print(content)
	return nil
}
