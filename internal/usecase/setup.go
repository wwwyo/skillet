package usecase

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
)

// SetupGlobalParams contains parameters for global setup.
type SetupGlobalParams struct {
	GlobalPath     string
	EnabledTargets map[string]bool
	Strategy       config.Strategy
	ConfigPath     string
}

// SetupService handles initialization operations.
type SetupService struct {
	fs          platformfs.FileSystem
	configStore *config.Store
}

// NewSetupService creates a new setup service.
func NewSetupService(fsys platformfs.FileSystem) *SetupService {
	return &SetupService{
		fs:          fsys,
		configStore: config.NewStore(fsys),
	}
}

// SetupGlobal performs global initialization.
func (s *SetupService) SetupGlobal(params SetupGlobalParams) (*config.Config, error) {
	agentsDir, err := config.ExpandPath(s.fs, params.GlobalPath)
	if err != nil {
		return nil, err
	}

	// Create directory structure.
	dirs := []string{
		agentsDir,
		s.fs.Join(agentsDir, config.SkillsDirName),
		s.fs.Join(agentsDir, config.SkillsDirName, config.OptionalDirName),
	}
	for _, dir := range dirs {
		if err := s.fs.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Update existing config with new params.
	if s.fs.Exists(params.ConfigPath) {
		cfg, err := s.configStore.Load(params.ConfigPath)
		if err != nil {
			return nil, err
		}
		if params.GlobalPath != config.DefaultGlobalPath {
			cfg.GlobalPath = params.GlobalPath
		}
		cfg.DefaultStrategy = params.Strategy
		for name, target := range cfg.Targets {
			target.Enabled = params.EnabledTargets[name]
			cfg.Targets[name] = target
		}
		if err := s.configStore.Save(cfg, params.ConfigPath); err != nil {
			return nil, fmt.Errorf("failed to update config file: %w", err)
		}
		return cfg, nil
	}

	// Create new config.
	cfg := config.DefaultConfig()
	if params.GlobalPath != config.DefaultGlobalPath {
		cfg.GlobalPath = params.GlobalPath
	}
	cfg.DefaultStrategy = params.Strategy

	for name, target := range cfg.Targets {
		target.Enabled = params.EnabledTargets[name]
		cfg.Targets[name] = target
	}

	if err := s.configStore.Save(cfg, params.ConfigPath); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}

	return cfg, nil
}

// SetupProject performs project initialization.
func (s *SetupService) SetupProject(projectRoot string) error {
	agentsDir := config.ProjectAgentsDir(projectRoot, s.fs)

	dirs := []string{
		agentsDir,
		config.ProjectSkillsDir(projectRoot, s.fs, ""),
		config.ProjectSkillsDir(projectRoot, s.fs, config.OptionalDirName),
	}

	for _, dir := range dirs {
		if err := s.fs.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
