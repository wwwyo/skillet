package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/wwwyo/skillet/internal/fs"
)

const (
	// ConfigDir is the directory name for skillet configuration.
	ConfigDir = ".config/skillet"
	// ConfigFileName is the name of the config file.
	ConfigFileName = "config.yaml"
	// AgentsDir is the directory name for agents configuration.
	AgentsDir = ".agents"
	// DefaultGlobalPath is the default path for global skills.
	DefaultGlobalPath = "~/.agents"
	// SkillsDir is the directory name for skills.
	SkillsDir = "skills"
	// OptionalDir is the directory name for optional (selectable) skills.
	OptionalDir = "optional"
)

// Strategy represents the synchronization strategy.
type Strategy string

const (
	// StrategySymlink uses symbolic links for synchronization.
	StrategySymlink Strategy = "symlink"
	// StrategyCopy uses file copies for synchronization.
	StrategyCopy Strategy = "copy"
)

// TargetConfig represents configuration for a specific target.
type TargetConfig struct {
	Enabled    bool   `yaml:"enabled"`
	GlobalPath string `yaml:"globalPath,omitempty"`
}

// Config represents the global configuration.
type Config struct {
	Version         int                     `yaml:"version"`
	GlobalPath      string                  `yaml:"globalPath,omitempty"`
	DefaultStrategy Strategy                `yaml:"defaultStrategy"`
	Targets         map[string]TargetConfig `yaml:"targets"`

	// Internal fields (not serialized)
	path string
	fsys fs.System
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Version:         1,
		GlobalPath:      DefaultGlobalPath,
		DefaultStrategy: StrategySymlink,
		Targets: map[string]TargetConfig{
			"claude": {
				Enabled:    true,
				GlobalPath: "~/.claude",
			},
			"codex": {
				Enabled:    true,
				GlobalPath: "~/.codex",
			},
		},
	}
}

// AgentsDir returns the expanded global agents directory path.
func (c *Config) AgentsDir(fsys fs.System) (string, error) {
	path := c.GlobalPath
	if path == "" {
		path = DefaultGlobalPath
	}
	return ExpandPath(fsys, path)
}

// SkillsDir returns the expanded global skills directory path.
func (c *Config) SkillsDir(fsys fs.System, category string) (string, error) {
	agentsDir, err := c.AgentsDir(fsys)
	if err != nil {
		return "", err
	}
	return fsys.Join(agentsDir, SkillsDir, category), nil
}

// GlobalConfigPath returns the path to the global config file (~/.config/skillet/config.yaml).
func GlobalConfigPath(fsys fs.System) (string, error) {
	home, err := fsys.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return fsys.Join(home, ConfigDir, ConfigFileName), nil
}

// ProjectAgentsDir returns the path to the project agents directory.
func ProjectAgentsDir(projectRoot string, fsys fs.System) string {
	return fsys.Join(projectRoot, AgentsDir)
}

// GetAgentsDir returns the agents directory for the given scope.
// If projectRoot is non-empty, returns the project agents directory.
// Otherwise, returns the global agents directory.
func (c *Config) GetAgentsDir(fsys fs.System, projectRoot string) (string, error) {
	if projectRoot != "" {
		return ProjectAgentsDir(projectRoot, fsys), nil
	}
	return c.AgentsDir(fsys)
}

// ProjectSkillsDir returns the path to the project skills directory.
func ProjectSkillsDir(projectRoot string, fsys fs.System, category string) string {
	return fsys.Join(projectRoot, AgentsDir, SkillsDir, category)
}

// Load loads the configuration from a file.
func Load(fsys fs.System, path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = GlobalConfigPath(fsys)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		path, err = ExpandPath(fsys, path)
		if err != nil {
			return nil, err
		}
	}

	if !fsys.Exists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.path = path
	cfg.fsys = fsys

	return &cfg, nil
}

// Save saves the configuration to a file.
func (c *Config) Save() error {
	if c.path == "" {
		return fmt.Errorf("config path not set")
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := c.fsys.Dir(c.path)
	if err := c.fsys.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := c.fsys.WriteFile(c.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SaveTo saves the configuration to a specific path.
func (c *Config) SaveTo(fsys fs.System, path string) error {
	c.fsys = fsys
	c.path = path
	return c.Save()
}

// FindProjectRoot searches for the project root by looking for .agents directory.
// Uses the current working directory as the starting point.
func FindProjectRoot(fsys fs.System) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return FindProjectRootFrom(fsys, cwd)
}

// FindProjectRootFrom searches for the project root starting from the given directory.
// This is useful for testing and when the starting directory is known.
func FindProjectRootFrom(fsys fs.System, startDir string) (string, error) {
	dir := startDir
	for {
		agentsPath := fsys.Join(dir, AgentsDir)
		if fsys.Exists(agentsPath) && fsys.IsDir(agentsPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return empty to indicate not found
			return "", fmt.Errorf("project root not found (no %s directory)", AgentsDir)
		}
		dir = parent
	}
}

// ExpandPath expands ~ in a path to the home directory.
func ExpandPath(fsys fs.System, path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] == '~' {
		home, err := fsys.UserHomeDir()
		if err != nil {
			return "", err
		}
		return home + path[1:], nil
	}

	return path, nil
}
