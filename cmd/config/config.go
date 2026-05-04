package config

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cfg "github.com/jmrmedev/cctxm/internal/config"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "View and edit configuration",
}

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Print current config",
	RunE:  runShow,
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config in $EDITOR",
	RunE:  runEdit,
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE:  runSet,
}

func init() {
	Cmd.AddCommand(showCmd, editCmd, setCmd)
}

func resolveConfigPath(cmd *cobra.Command) (string, error) {
	cfgFlag, _ := cmd.Flags().GetString("config")
	if cfgFlag != "" {
		return cfgFlag, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := cfg.FindRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("no .cctxm directory found (run 'cctxm init' first)")
	}
	return cfg.ConfigPath(root), nil
}

func runShow(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	c, err := cfg.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func runEdit(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runSet(cmd *cobra.Command, args []string) error {
	path, err := resolveConfigPath(cmd)
	if err != nil {
		return err
	}
	c, err := cfg.Load(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	key, value := args[0], args[1]

	switch {
	case strings.HasPrefix(key, "projects."):
		name := strings.TrimPrefix(key, "projects.")
		if c.Projects == nil {
			c.Projects = map[string]string{}
		}
		c.Projects[name] = value
	case key == "default_project":
		c.DefaultProject = value
	case key == "filter.max_lines":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		c.Filter.MaxLines = v
	case key == "filter.context_lines":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		c.Filter.ContextLines = v
	case key == "filter.read_threshold":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		c.Filter.ReadThreshold = v
	case key == "session.retention_days":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid integer: %s", value)
		}
		c.Session.RetentionDays = v
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	if err := cfg.Save(path, c); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}
