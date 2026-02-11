package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
)

// Store manages config file persistence.
type Store struct {
	fs platformfs.FileSystem
}

// NewStore creates a new Store.
func NewStore(fsys platformfs.FileSystem) *Store {
	return &Store{fs: fsys}
}

// Load loads the configuration from a file.
func (s *Store) Load(path string) (*Config, error) {
	var err error
	if path == "" {
		path, err = s.GlobalConfigPath()
	} else {
		path, err = ExpandPath(s.fs, path)
	}
	if err != nil {
		return nil, err
	}

	if !s.fs.Exists(path) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save saves the configuration to a specific path.
func (s *Store) Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := s.fs.Dir(path)
	if err := s.fs.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := s.fs.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GlobalConfigPath returns the path to the global config file.
func (s *Store) GlobalConfigPath() (string, error) {
	return GlobalConfigPath(s.fs)
}

// FindProjectRoot searches for the project root by looking for .agents directory.
func (s *Store) FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return s.FindProjectRootFrom(cwd)
}

// FindProjectRootFrom searches for the project root starting from the given directory.
func (s *Store) FindProjectRootFrom(startDir string) (string, error) {
	dir := startDir
	for {
		agentsPath := s.fs.Join(dir, AgentsDirName)
		if s.fs.Exists(agentsPath) && s.fs.IsDir(agentsPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("project root not found (no %s directory)", AgentsDirName)
		}
		dir = parent
	}
}
