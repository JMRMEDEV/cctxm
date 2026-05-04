package init

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jmrmedev/cctxm/internal/config"
	"github.com/jmrmedev/cctxm/internal/rules"
)

var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize cctxm in the current workspace",
	RunE:  runInit,
}

func init() {
	Cmd.Flags().Bool("skip-rules", false, "don't touch .github/copilot/")
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cctxmDir := filepath.Join(cwd, config.CctxmDir)

	// Create directory structure
	dirs := []string{
		cctxmDir,
		filepath.Join(cctxmDir, "sessions"),
		filepath.Join(cctxmDir, "rules"),
		filepath.Join(cctxmDir, "overridden"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", d, err)
		}
	}

	// Write default config if it doesn't exist
	cfgPath := config.ConfigPath(cwd)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg := config.Default()
		if err := config.Save(cfgPath, cfg); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		fmt.Printf("Created %s\n", cfgPath)
	}

	// Write example rule file
	exampleRule := filepath.Join(cctxmDir, "rules", "example.md")
	if _, err := os.Stat(exampleRule); os.IsNotExist(err) {
		content := `# Example CCTXM Rule

Add your coding standards, project conventions, or workflow rules here.
Files in this directory are injected into Copilot's instructions.
`
		os.WriteFile(exampleRule, []byte(content), 0644)
	}

	fmt.Printf("Initialized cctxm in %s\n", cwd)
	fmt.Println("Next steps:")
	fmt.Println("  cctxm config set projects.<name> <path>  — map your projects")
	fmt.Println("  cctxm session start \"<task>\"              — start a session")
	fmt.Println("  cctxm rules inject                        — inject Copilot instructions")

	// Inject rules unless --skip-rules
	skipRules, _ := cmd.Flags().GetBool("skip-rules")
	if !skipRules {
		cfg, _ := config.Load(cfgPath)
		if err := rules.Inject(cwd, cctxmDir, cfg.Rules.Inject); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to inject rules: %v\n", err)
		} else {
			fmt.Println("Injected cctxm instructions into .github/copilot/")
		}
	}

	return nil
}
