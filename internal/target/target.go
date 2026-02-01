package target

import (
	"github.com/wwwyo/skillet/internal/fs"
)

// Target represents a synchronization target (e.g., claude, codex).
type Target interface {
	// Name returns the target name.
	Name() string
	// GlobalSkillsDir returns the global skills directory path.
	GlobalSkillsDir() string
	// ProjectSkillsDir returns the project skills directory path.
	ProjectSkillsDir() string
}

// BaseTarget is the default implementation of Target.
type BaseTarget struct {
	name        string
	globalPath  string
	projectPath string
	skillsDir   string
	fs          fs.System
	projectRoot string
}

// NewBaseTarget creates a new BaseTarget.
func NewBaseTarget(name, globalPath, projectPath, skillsDir string, fsys fs.System, projectRoot string) Target {
	return &BaseTarget{
		name:        name,
		globalPath:  globalPath,
		projectPath: projectPath,
		skillsDir:   skillsDir,
		fs:          fsys,
		projectRoot: projectRoot,
	}
}

// Name returns the target name.
func (t *BaseTarget) Name() string {
	return t.name
}

// GlobalSkillsDir returns the global skills directory path.
func (t *BaseTarget) GlobalSkillsDir() string {
	expandedPath, err := expandPath(t.fs, t.globalPath)
	if err != nil {
		return t.fs.Join(t.globalPath, t.skillsDir)
	}
	return t.fs.Join(expandedPath, t.skillsDir)
}

// ProjectSkillsDir returns the project skills directory path.
func (t *BaseTarget) ProjectSkillsDir() string {
	if t.projectRoot == "" {
		return ""
	}
	return t.fs.Join(t.projectRoot, t.projectPath, t.skillsDir)
}

// expandPath expands ~ in a path to the home directory.
func expandPath(fsys fs.System, path string) (string, error) {
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
