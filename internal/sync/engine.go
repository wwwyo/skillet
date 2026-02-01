package sync

import (
	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

// Action represents the type of sync action taken.
type Action string

const (
	ActionInstall   Action = "install"
	ActionUpdate    Action = "update"
	ActionUninstall Action = "uninstall"
	ActionSkip      Action = "skip"
	ActionError     Action = "error"
)

// Result represents the result of syncing a single skill to a target.
type Result struct {
	SkillName string
	Target    string
	Action    Action
	Error     error
}

// Status represents the sync status of a target.
type Status struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	InSync    bool
	Error     error
}

// SyncOptions contains options for the sync operation.
type SyncOptions struct {
	TargetName string
	DryRun     bool
	Force      bool
}

// Engine handles synchronization of skills to targets.
type Engine struct {
	fs          fs.System
	store       *skill.Store
	registry    *target.Registry
	cfg         *config.Config
	projectRoot string
}

// NewEngine creates a new sync Engine.
func NewEngine(fsys fs.System, store *skill.Store, registry *target.Registry, cfg *config.Config, projectRoot string) *Engine {
	return &Engine{
		fs:          fsys,
		store:       store,
		registry:    registry,
		cfg:         cfg,
		projectRoot: projectRoot,
	}
}

// Sync synchronizes skills to targets.
func (e *Engine) Sync(opts SyncOptions) ([]Result, error) {
	var results []Result

	targets := e.registry.GetAll()
	if opts.TargetName != "" {
		t, err := e.registry.Get(opts.TargetName)
		if err != nil {
			return nil, err
		}
		targets = []target.Target{t}
	}

	skills, err := e.store.GetResolved()
	if err != nil {
		return nil, err
	}

	for _, t := range targets {
		targetResults := e.syncToTarget(t, skills, opts)
		results = append(results, targetResults...)
	}

	return results, nil
}

// syncToTarget syncs skills to a single target.
func (e *Engine) syncToTarget(t target.Target, skills []*skill.ScopedSkill, opts SyncOptions) []Result {
	var results []Result

	targetDir := t.GlobalSkillsDir()
	if e.projectRoot != "" {
		targetDir = t.ProjectSkillsDir()
	}

	for _, sk := range skills {
		result := e.syncSkill(sk, targetDir, t.Name(), opts)
		results = append(results, result)
	}

	return results
}

// syncSkill syncs a single skill to a target directory.
func (e *Engine) syncSkill(sk *skill.ScopedSkill, targetDir string, targetName string, opts SyncOptions) Result {
	skillTargetPath := e.fs.Join(targetDir, sk.Name)

	// Check if already installed
	exists := e.fs.Exists(skillTargetPath)

	if exists && !opts.Force {
		return Result{
			SkillName: sk.Name,
			Target:    targetName,
			Action:    ActionSkip,
		}
	}

	action := ActionInstall
	if exists {
		action = ActionUpdate
	}

	if !opts.DryRun {
		// Remove existing if force update
		if exists {
			_ = e.fs.RemoveAll(skillTargetPath)
		}

		// Create symlink or copy based on strategy
		if e.cfg.DefaultStrategy == config.StrategySymlink {
			if err := e.fs.MkdirAll(targetDir, 0755); err != nil {
				return Result{
					SkillName: sk.Name,
					Target:    targetName,
					Action:    ActionError,
					Error:     err,
				}
			}
			if err := e.fs.Symlink(sk.Path, skillTargetPath); err != nil {
				return Result{
					SkillName: sk.Name,
					Target:    targetName,
					Action:    ActionError,
					Error:     err,
				}
			}
		} else {
			if err := e.fs.CopyDir(sk.Path, skillTargetPath); err != nil {
				return Result{
					SkillName: sk.Name,
					Target:    targetName,
					Action:    ActionError,
					Error:     err,
				}
			}
		}
	}

	return Result{
		SkillName: sk.Name,
		Target:    targetName,
		Action:    action,
	}
}

// GetStatus returns the sync status for all targets.
func (e *Engine) GetStatus() ([]Status, error) {
	var statuses []Status

	targets := e.registry.GetAll()
	skills, err := e.store.GetResolved()
	if err != nil {
		return nil, err
	}

	skillNames := make(map[string]bool)
	for _, sk := range skills {
		skillNames[sk.Name] = true
	}

	for _, t := range targets {
		status := e.getTargetStatus(t, skillNames)
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// getTargetStatus returns the sync status for a single target.
func (e *Engine) getTargetStatus(t target.Target, skillNames map[string]bool) Status {
	targetDir := t.GlobalSkillsDir()
	if e.projectRoot != "" {
		targetDir = t.ProjectSkillsDir()
	}

	status := Status{
		Target: t.Name(),
	}

	// Get installed skills
	entries, err := e.fs.ReadDir(targetDir)
	if err != nil {
		// Directory doesn't exist yet, all skills are missing
		for name := range skillNames {
			status.Missing = append(status.Missing, name)
		}
		status.InSync = len(skillNames) == 0
		return status
	}

	installed := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || e.fs.IsSymlink(e.fs.Join(targetDir, entry.Name())) {
			installed[entry.Name()] = true
		}
	}

	// Compare with expected skills
	for name := range skillNames {
		if installed[name] {
			status.Installed = append(status.Installed, name)
		} else {
			status.Missing = append(status.Missing, name)
		}
	}

	// Find extra skills (installed but not in store)
	for name := range installed {
		if !skillNames[name] {
			status.Extra = append(status.Extra, name)
		}
	}

	status.InSync = len(status.Missing) == 0 && len(status.Extra) == 0

	return status
}
