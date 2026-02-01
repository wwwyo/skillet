package target

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

// InstallOptions contains options for installing a skill.
type InstallOptions struct {
	// Strategy specifies how to install (symlink or copy)
	Strategy config.Strategy
	// Force overwrites existing installations
	Force bool
}

// Target represents an AI agent target for skill synchronization.
type Target interface {
	// Name returns the target name (e.g., "claude", "codex")
	Name() string
	// Install installs a skill to this target
	Install(skill *skill.Skill, opts InstallOptions) error
	// Uninstall removes a skill from this target
	Uninstall(skillName string) error
	// IsInstalled checks if a skill is installed
	IsInstalled(skillName string) bool
	// IsInstalledInScope checks if a skill is installed in a specific scope
	IsInstalledInScope(skillName string, scope skill.Scope) bool
	// GetInstalledPath returns the path where a skill is installed
	GetInstalledPath(skillName string) string
	// GetSkillsPath returns the skills directory path for the given scope
	GetSkillsPath(scope skill.Scope) (string, error)
	// ListInstalled returns all installed skills
	ListInstalled() ([]string, error)
}

// BaseTarget provides common functionality for targets.
type BaseTarget struct {
	name        string
	globalPath  string
	projectPath string
	skillsDir   string
	fs          fs.System
	projectRoot string
}

// NewBaseTarget creates a new BaseTarget.
func NewBaseTarget(name, globalPath, projectPath, skillsDir string, fsys fs.System, projectRoot string) *BaseTarget {
	return &BaseTarget{
		name:        name,
		globalPath:  globalPath,
		projectPath: projectPath,
		skillsDir:   skillsDir,
		fs:          fsys,
		projectRoot: projectRoot,
	}
}

func (t *BaseTarget) Name() string {
	return t.name
}

// GetSkillsPath returns the skills directory path for the given scope.
func (t *BaseTarget) GetSkillsPath(scope skill.Scope) (string, error) {
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
// Returns empty string if not found.
func (t *BaseTarget) GetInstalledPath(skillName string) string {
	// Check project first (higher priority)
	if path, err := t.GetSkillsPath(skill.ScopeProject); err == nil {
		fullPath := t.fs.Join(path, skillName)
		if t.fs.Exists(fullPath) {
			return fullPath
		}
	}

	// Then check global
	if path, err := t.GetSkillsPath(skill.ScopeGlobal); err == nil {
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
func (t *BaseTarget) IsInstalledInScope(skillName string, scope skill.Scope) bool {
	path, err := t.GetSkillsPath(scope)
	if err != nil {
		return false
	}
	return t.fs.Exists(t.fs.Join(path, skillName))
}

// Install installs a skill to this target.
func (t *BaseTarget) Install(s *skill.Skill, opts InstallOptions) error {
	destDir, err := t.GetSkillsPath(s.Scope)
	if err != nil {
		return err
	}

	destPath := t.fs.Join(destDir, s.Name)

	// Check if already installed
	if t.fs.Exists(destPath) {
		if !opts.Force {
			return fmt.Errorf("skill already installed: %s", s.Name)
		}
		// Remove existing
		if err := t.fs.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing skill: %w", err)
		}
	}

	// Ensure destination directory exists
	if err := t.fs.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	// Install based on strategy
	switch opts.Strategy {
	case config.StrategySymlink:
		// Try symlink first
		if err := t.fs.Symlink(s.Path, destPath); err != nil {
			// Fallback to copy if symlink fails (e.g., cross-filesystem)
			if err := t.fs.CopyDir(s.Path, destPath); err != nil {
				return fmt.Errorf("failed to install skill: %w", err)
			}
		}
	case config.StrategyCopy:
		if err := t.fs.CopyDir(s.Path, destPath); err != nil {
			return fmt.Errorf("failed to copy skill: %w", err)
		}
	default:
		// Default to symlink with copy fallback
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

	// Helper to add skills from a directory
	addFromDir := func(dir string) error {
		if dir == "" || !t.fs.Exists(dir) {
			return nil
		}
		entries, err := t.fs.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read skills directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
				skillSet[entry.Name()] = true
			}
		}
		return nil
	}

	// List from global path
	if globalPath, err := t.GetSkillsPath(skill.ScopeGlobal); err == nil {
		if err := addFromDir(globalPath); err != nil {
			return nil, err
		}
	}

	// List from project path
	if projectPath, err := t.GetSkillsPath(skill.ScopeProject); err == nil {
		if err := addFromDir(projectPath); err != nil {
			return nil, err
		}
	}

	// Convert to slice
	skills := make([]string, 0, len(skillSet))
	for name := range skillSet {
		skills = append(skills, name)
	}

	return skills, nil
}
