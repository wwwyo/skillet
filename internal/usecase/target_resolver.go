package usecase

import (
	"fmt"
	"os"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

// InstallOptions contains options for installing a skill.
type InstallOptions struct {
	Strategy config.Strategy
	Force    bool
}

// TargetDef defines default paths for a target.
type TargetDef struct {
	GlobalPath  string
	ProjectPath string
	SkillsDir   string
}

// defaultTargets contains default definitions for all supported targets.
var defaultTargets = map[string]TargetDef{
	"claude": {GlobalPath: "~/.claude", ProjectPath: ".claude", SkillsDir: "skills"},
	"codex":  {GlobalPath: "~/.codex", ProjectPath: ".codex", SkillsDir: "skills"},
}

// Target manages skill deployment to a single target.
type Target struct {
	name        string
	globalPath  string
	projectPath string
	skillsDir   string
	fs          platformfs.FileSystem
	projectRoot string
}

// newTarget creates a new Target.
func newTarget(name, globalPath, projectPath, skillsDir string, fsys platformfs.FileSystem, projectRoot string) *Target {
	return &Target{
		name:        name,
		globalPath:  globalPath,
		projectPath: projectPath,
		skillsDir:   skillsDir,
		fs:          fsys,
		projectRoot: projectRoot,
	}
}

func (t *Target) Name() string {
	return t.name
}

// GetSkillsPath returns the skills directory path for the given scope.
func (t *Target) GetSkillsPath(scope skill.Scope) (string, error) {
	switch scope {
	case skill.ScopeGlobal:
		expanded, err := config.ExpandPath(t.fs, t.globalPath)
		if err != nil {
			return "", err
		}
		return t.fs.Join(expanded, t.skillsDir), nil
	case skill.ScopeProject:
		if t.projectRoot == "" {
			return "", fmt.Errorf("project root not set")
		}
		return t.fs.Join(t.projectRoot, t.projectPath, t.skillsDir), nil
	default:
		return "", fmt.Errorf("unknown scope: %v", scope)
	}
}

// GetInstalledPath returns the path where a skill is installed (checks all scopes).
func (t *Target) GetInstalledPath(skillName string) string {
	if path, err := t.GetSkillsPath(skill.ScopeProject); err == nil {
		fullPath := t.fs.Join(path, skillName)
		if t.fs.Exists(fullPath) {
			return fullPath
		}
	}

	if path, err := t.GetSkillsPath(skill.ScopeGlobal); err == nil {
		fullPath := t.fs.Join(path, skillName)
		if t.fs.Exists(fullPath) {
			return fullPath
		}
	}

	return ""
}

// IsInstalled checks if a skill is installed in any scope.
func (t *Target) IsInstalled(skillName string) bool {
	return t.GetInstalledPath(skillName) != ""
}

// IsInstalledInScope checks if a skill is installed in the specified scope.
func (t *Target) IsInstalledInScope(skillName string, scope skill.Scope) bool {
	path, err := t.GetSkillsPath(scope)
	if err != nil {
		return false
	}
	return t.fs.Exists(t.fs.Join(path, skillName))
}

// Install installs a skill to this target.
func (t *Target) Install(s *skill.Skill, opts InstallOptions) error {
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

	if err := t.fs.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	switch opts.Strategy {
	case config.StrategySymlink:
		if err := t.fs.Symlink(s.Path, destPath); err != nil {
			if err := t.fs.CopyDir(s.Path, destPath); err != nil {
				return fmt.Errorf("failed to install skill: %w", err)
			}
		}
	case config.StrategyCopy:
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
func (t *Target) Uninstall(skillName string) error {
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
func (t *Target) ListInstalled() ([]string, error) {
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
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				skillSet[entry.Name()] = true
			}
		}
		return nil
	}

	if globalPath, err := t.GetSkillsPath(skill.ScopeGlobal); err == nil {
		if err := addFromDir(globalPath); err != nil {
			return nil, err
		}
	}

	if projectPath, err := t.GetSkillsPath(skill.ScopeProject); err == nil {
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

// ListMigratable returns skill names that can be migrated from a specific scope.
func (t *Target) ListMigratable(scope skill.Scope) ([]string, error) {
	targetSkillsDir, err := t.GetSkillsPath(scope)
	if err != nil || targetSkillsDir == "" {
		return nil, err
	}

	if !t.fs.Exists(targetSkillsDir) || !t.fs.IsDir(targetSkillsDir) {
		return nil, nil
	}

	entries, err := t.fs.ReadDir(targetSkillsDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		// Skip symlinks (already managed by skillet).
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		if !entry.IsDir() {
			continue
		}

		skillName := entry.Name()
		if err := skill.ValidateName(skillName); err != nil {
			continue
		}

		skillDir := t.fs.Join(targetSkillsDir, skillName)
		if isValidSkillDir(t.fs, skillDir) {
			names = append(names, skillName)
		}
	}

	return names, nil
}

// TargetRegistry manages multiple targets.
type TargetRegistry struct {
	targets map[string]*Target
}

// NewTargetRegistry creates a new registry with default targets.
func NewTargetRegistry(fsys platformfs.FileSystem, projectRoot string, cfg *config.Config) *TargetRegistry {
	r := &TargetRegistry{targets: make(map[string]*Target)}

	for name, def := range defaultTargets {
		if cfg != nil && !cfg.Targets[name].Enabled {
			continue
		}

		globalPath := def.GlobalPath
		if cfg != nil && cfg.Targets[name].GlobalPath != "" {
			globalPath = cfg.Targets[name].GlobalPath
		}

		r.targets[name] = newTarget(name, globalPath, def.ProjectPath, def.SkillsDir, fsys, projectRoot)
	}

	return r
}

// Get returns a target by name.
func (r *TargetRegistry) Get(name string) (*Target, bool) {
	target, ok := r.targets[name]
	return target, ok
}

// GetAll returns all registered targets.
func (r *TargetRegistry) GetAll() []*Target {
	targets := make([]*Target, 0, len(r.targets))
	for _, t := range r.targets {
		targets = append(targets, t)
	}
	return targets
}

// Names returns all registered target names.
func (r *TargetRegistry) Names() []string {
	names := make([]string, 0, len(r.targets))
	for name := range r.targets {
		names = append(names, name)
	}
	return names
}

const maxValidationDepth = 5

func isValidSkillDir(fsys platformfs.FileSystem, dir string) bool {
	return isValidSkillDirWithDepth(fsys, dir, 0)
}

func isValidSkillDirWithDepth(fsys platformfs.FileSystem, dir string, depth int) bool {
	if depth > maxValidationDepth {
		return false
	}

	if fsys.Exists(fsys.Join(dir, "SKILL.md")) {
		return true
	}

	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && isValidSkillDirWithDepth(fsys, fsys.Join(dir, entry.Name()), depth+1) {
			return true
		}
	}

	return false
}
