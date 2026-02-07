package service

import "fmt"

const (
	// ConfigDir is the directory name for skillet configuration.
	ConfigDir = ".config/skillet"
	// ConfigFileName is the name of the config file.
	ConfigFileName = "config.yaml"
	// AgentsDirName is the directory name for agents configuration.
	AgentsDirName = ".agents"
	// DefaultGlobalPath is the default path for global skills.
	DefaultGlobalPath = "~/.agents"
	// SkillsDirName is the directory name for skills.
	SkillsDirName = "skills"
	// OptionalDirName is the directory name for optional (selectable) skills.
	OptionalDirName = "optional"
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
}

// Default returns the default configuration.
func DefaultConfig() *Config {
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
func (c *Config) AgentsDir(fsys FileSystem) (string, error) {
	path := c.GlobalPath
	if path == "" {
		path = DefaultGlobalPath
	}
	return ExpandPath(fsys, path)
}

// SkillsDir returns the expanded global skills directory path.
func (c *Config) SkillsDir(fsys FileSystem, category string) (string, error) {
	agentsDir, err := c.AgentsDir(fsys)
	if err != nil {
		return "", err
	}
	return fsys.Join(agentsDir, SkillsDirName, category), nil
}

// GetAgentsDir returns the agents directory for the given scope.
// If projectRoot is non-empty, returns the project agents directory.
// Otherwise, returns the global agents directory.
func (c *Config) GetAgentsDir(fsys FileSystem, projectRoot string) (string, error) {
	if projectRoot != "" {
		return ProjectAgentsDir(projectRoot, fsys), nil
	}
	return c.AgentsDir(fsys)
}

// ProjectAgentsDir returns the path to the project agents directory.
func ProjectAgentsDir(projectRoot string, fsys FileSystem) string {
	return fsys.Join(projectRoot, AgentsDirName)
}

// ProjectSkillsDir returns the path to the project skills directory.
func ProjectSkillsDir(projectRoot string, fsys FileSystem, category string) string {
	return fsys.Join(projectRoot, AgentsDirName, SkillsDirName, category)
}

// ExpandPath expands ~ in a path to the home directory.
func ExpandPath(fsys FileSystem, path string) (string, error) {
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

// GlobalConfigPath returns the path to the global config file (~/.config/skillet/config.yaml).
func GlobalConfigPath(fsys FileSystem) (string, error) {
	home, err := fsys.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return fsys.Join(home, ConfigDir, ConfigFileName), nil
}
