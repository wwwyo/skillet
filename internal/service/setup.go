package service

import "fmt"

// SetupService handles initialization operations.
type SetupService struct {
	fs          FileSystem
	configStore ConfigStore
}

// NewSetupService creates a new SetupService.
func NewSetupService(fs FileSystem, cs ConfigStore) *SetupService {
	return &SetupService{
		fs:          fs,
		configStore: cs,
	}
}

// SetupGlobal performs global initialization.
func (s *SetupService) SetupGlobal(params SetupGlobalParams) (*Config, error) {
	agentsDir, err := ExpandPath(s.fs, params.GlobalPath)
	if err != nil {
		return nil, err
	}

	// Create directory structure
	dirs := []string{
		agentsDir,
		s.fs.Join(agentsDir, SkillsDirName),
		s.fs.Join(agentsDir, SkillsDirName, OptionalDirName),
	}
	for _, dir := range dirs {
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Check if config already exists
	if s.fs.Exists(params.ConfigPath) {
		cfg, err := s.configStore.Load(params.ConfigPath)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// Create new config
	cfg := DefaultConfig()
	if params.GlobalPath != DefaultGlobalPath {
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
	agentsDir := ProjectAgentsDir(projectRoot, s.fs)

	dirs := []string{
		agentsDir,
		ProjectSkillsDir(projectRoot, s.fs, ""),
		ProjectSkillsDir(projectRoot, s.fs, OptionalDirName),
	}

	for _, dir := range dirs {
		if err := s.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
