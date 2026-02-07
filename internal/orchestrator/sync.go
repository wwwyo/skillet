package orchestrator

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

// SyncAction represents the type of sync action taken.
type SyncAction string

const (
	SyncActionInstall   SyncAction = "install"
	SyncActionUpdate    SyncAction = "update"
	SyncActionUninstall SyncAction = "uninstall"
	SyncActionSkip      SyncAction = "skip"
	SyncActionError     SyncAction = "error"
)

// SyncResult represents the result of a sync operation for a single skill.
type SyncResult struct {
	SkillName string
	Target    string
	Action    SyncAction
	Error     error
}

// SyncOptions contains options for synchronization.
type SyncOptions struct {
	// DryRun only shows what would be done without making changes
	DryRun bool
	// Force overwrites existing installations
	Force bool
	// Scope limits sync to a specific scope (nil for all)
	Scope *skill.Scope
}

// Sync synchronizes skills to targets.
func (o *Orchestrator) Sync(opts SyncOptions) ([]SyncResult, error) {
	skills, err := o.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if opts.Scope != nil {
		skills = filterByScope(skills, *opts.Scope)
	}

	targets := o.registry.GetAll()
	var results []SyncResult

	for _, t := range targets {
		for _, s := range skills {
			isInstalled := t.IsInstalledInScope(s.Name, s.Scope)
			result := o.syncSkill(t, s, isInstalled, opts)
			results = append(results, result)
		}
	}

	return results, nil
}

// syncSkill syncs a single skill to a target.
func (o *Orchestrator) syncSkill(t target.Target, s *skill.Skill, isInstalled bool, opts SyncOptions) SyncResult {
	result := SyncResult{
		SkillName: s.Name,
		Target:    t.Name(),
	}

	if isInstalled && !opts.Force {
		result.Action = SyncActionSkip
		return result
	}

	if isInstalled {
		result.Action = SyncActionUpdate
	} else {
		result.Action = SyncActionInstall
	}

	if opts.DryRun {
		return result
	}

	strategy := o.cfg.DefaultStrategy
	if strategy == "" {
		strategy = config.StrategySymlink
	}

	installOpts := target.InstallOptions{
		Strategy: strategy,
		Force:    opts.Force || isInstalled,
	}

	if err := t.Install(s, installOpts); err != nil {
		result.Action = SyncActionError
		result.Error = err
	}

	return result
}
