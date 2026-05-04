package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultFileName = "config.yaml"
const CctxmDir = ".cctxm"

type Config struct {
	Projects       map[string]string `yaml:"projects"`
	DefaultProject string            `yaml:"default_project"`
	Filter         FilterConfig      `yaml:"filter"`
	Rules          RulesConfig       `yaml:"rules"`
	Session        SessionConfig     `yaml:"session"`
}

type FilterConfig struct {
	MaxLines           int      `yaml:"max_lines"`
	ContextLines       int      `yaml:"context_lines"`
	ReadThreshold      int      `yaml:"read_threshold"`
	FullReadExtensions []string `yaml:"full_read_extensions"`
}

type RulesConfig struct {
	Inject   []string `yaml:"inject"`
	Suppress []string `yaml:"suppress"`
}

type SessionConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

func Default() *Config {
	return &Config{
		Projects:       map[string]string{},
		DefaultProject: "",
		Filter: FilterConfig{
			MaxLines:      200,
			ContextLines:  10,
			ReadThreshold: 5120,
			FullReadExtensions: []string{
				".md", ".yaml", ".yml", ".toml", ".json", ".env.example",
			},
		},
		Rules: RulesConfig{
			Inject:   []string{".cctxm/rules/*.md"},
			Suppress: []string{".github/copilot/"},
		},
		Session: SessionConfig{
			RetentionDays: 7,
		},
	}
}

// FindRoot walks up from startDir looking for a .cctxm directory.
// Returns the workspace root or empty string if not found.
func FindRoot(startDir string) string {
	dir := startDir
	for {
		if _, err := os.Stat(filepath.Join(dir, CctxmDir)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// ConfigPath returns the full path to config.yaml given a workspace root.
func ConfigPath(root string) string {
	return filepath.Join(root, CctxmDir, DefaultFileName)
}

// Load reads config from the given path, merging with defaults.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes config to the given path.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
