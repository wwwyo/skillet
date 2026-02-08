package adapters

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/service"
)

// TargetDef defines default paths for a target.
type TargetDef struct {
	GlobalPath  string
	ProjectPath string
	SkillsDir   string
}

// DefaultTargets contains default definitions for all supported targets.
var DefaultTargets = map[string]TargetDef{
	"claude": {GlobalPath: "~/.claude", ProjectPath: ".claude", SkillsDir: "skills"},
	"codex":  {GlobalPath: "~/.codex", ProjectPath: ".codex", SkillsDir: "skills"},
}

// BaseTarget implements service.Target.
type BaseTarget struct {
	name        string
	globalPath  string
	projectPath string
	skillsDir   string
	fs          service.FileSystem
	projectRoot string
}

// Compile-time interface check.
var _ service.Target = (*BaseTarget)(nil)

// NewBaseTarget creates a new BaseTarget.
func NewBaseTarget(name, globalPath, projectPath, skillsDir string, fs service.FileSystem, projectRoot string) *BaseTarget {
	return &BaseTarget{
		name:        name,
		globalPath:  globalPath,
		projectPath: projectPath,
		skillsDir:   skillsDir,
		fs:          fs,
		projectRoot: projectRoot,
	}
}

func (t *BaseTarget) Name() string {
	return t.name
}

// GetSkillsPath returns the skills directory path for the given scope.
func (t *BaseTarget) GetSkillsPath(scope service.Scope) (string, error) {
	switch scope {
	case service.ScopeGlobal:
		expanded, err := service.ExpandPath(t.fs, t.globalPath)
		if err != nil {
			return "", err
		}
		return t.fs.Join(expanded, t.skillsDir), nil
	case service.ScopeProject:
		if t.projectRoot == "" {
			return "", fmt.Errorf("project root not set")
		}
		return t.fs.Join(t.projectRoot, t.projectPath, t.skillsDir), nil
	default:
		return "", fmt.Errorf("unknown scope: %v", scope)
	}
}

// GetInstalledPath returns the path where a skill is installed (checks all scopes).
func (t *BaseTarget) GetInstalledPath(skillName string) string {
	if path, err := t.GetSkillsPath(service.ScopeProject); err == nil {
		fullPath := t.fs.Join(path, skillName)
		if t.fs.Exists(fullPath) {
			return fullPath
		}
	}

	if path, err := t.GetSkillsPath(service.ScopeGlobal); err == nil {
		fullPath := t.fs.Join(path, skillName)
		if t.fs.Exists(fullPath) {
			return fullPath
		}
	}

	return ""
}

// IsInstalled checks if a skill is installed in any scope.
func (t *BaseTarget) IsInstalled(skillName string) bool {
	return t.GetInstalledPath(skillName) != ""
}

// IsInstalledInScope checks if a skill is installed in the specified scope.
func (t *BaseTarget) IsInstalledInScope(skillName string, scope service.Scope) bool {
	path, err := t.GetSkillsPath(scope)
	if err != nil {
		return false
	}
	return t.fs.Exists(t.fs.Join(path, skillName))
}

// Install installs a skill to this target.
func (t *BaseTarget) Install(s *service.Skill, opts service.InstallOptions) error {
	destDir, err := t.GetSkillsPath(s.Scope)
	if err != nil {
		return err
	}

	destPath := t.fs.Join(destDir, s.Name)

	if t.fs.Exists(destPath) {
		if !opts.Force {
			return fmt.Errorf("skill already installed: %s", s.Name)
		}
		if err := t.fs.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	if err := t.fs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	switch opts.Strategy {
	case service.StrategySymlink:
		if err := t.fs.Symlink(s.Path, destPath); err != nil {
			if err := t.fs.CopyDir(s.Path, destPath); err != nil {
				return fmt.Errorf("failed to install skill: %w", err)
			}
		}
	case service.StrategyCopy:
		if err := t.fs.CopyDir(s.Path, destPath); err != nil {
			return fmt.Errorf("failed to copy skill: %w", err)
		}
	default:
		if err := t.fs.Symlink(s.Path, destPath); err != nil {
			if err := t.fs.CopyDir(s.Path, destPath); err != nil {
				return fmt.Errorf("failed to install skill: %w", err)
			}
		}
	}

	return nil
}

// Uninstall removes a skill from this target.
func (t *BaseTarget) Uninstall(skillName string) error {
	path := t.GetInstalledPath(skillName)
	if path == "" {
		return fmt.Errorf("skill not installed: %s", skillName)
	}

	if err := t.fs.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to uninstall skill: %w", err)
	}

	return nil
}

// ListInstalled returns all installed skills from all scopes.
func (t *BaseTarget) ListInstalled() ([]string, error) {
	skillSet := make(map[string]bool)

	addFromDir := func(dir string) error {
		if dir == "" || !t.fs.Exists(dir) {
			return nil
		}
		entries, err := t.fs.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read skills directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&service.ModeSymlink != 0 {
				skillSet[entry.Name()] = true
			}
		}
		return nil
	}

	if globalPath, err := t.GetSkillsPath(service.ScopeGlobal); err == nil {
		if err := addFromDir(globalPath); err != nil {
			return nil, err
		}
	}

	if projectPath, err := t.GetSkillsPath(service.ScopeProject); err == nil {
		if err := addFromDir(projectPath); err != nil {
			return nil, err
		}
	}

	skills := make([]string, 0, len(skillSet))
	for name := range skillSet {
		skills = append(skills, name)
	}

	return skills, nil
}

// Registry implements service.TargetRegistry.
type Registry struct {
	targets map[string]service.Target
}

// Compile-time interface check.
var _ service.TargetRegistry = (*Registry)(nil)

// NewRegistry creates a new Registry with default targets.
func NewRegistry(fs service.FileSystem, projectRoot string, cfg *service.Config) *Registry {
	r := &Registry{targets: make(map[string]service.Target)}

	for name, def := range DefaultTargets {
		if cfg != nil && !cfg.Targets[name].Enabled {
			continue
		}

		globalPath := def.GlobalPath
		if cfg != nil && cfg.Targets[name].GlobalPath != "" {
			globalPath = cfg.Targets[name].GlobalPath
		}

		r.targets[name] = NewBaseTarget(name, globalPath, def.ProjectPath, def.SkillsDir, fs, projectRoot)
	}

	return r
}

// Get returns a target by name.
func (r *Registry) Get(name string) (service.Target, bool) {
	target, ok := r.targets[name]
	return target, ok
}

// GetAll returns all registered targets.
func (r *Registry) GetAll() []service.Target {
	targets := make([]service.Target, 0, len(r.targets))
	for _, t := range r.targets {
		targets = append(targets, t)
	}
	return targets
}

// Names returns all registered target names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.targets))
	for name := range r.targets {
		names = append(names, name)
	}
	return names
}
