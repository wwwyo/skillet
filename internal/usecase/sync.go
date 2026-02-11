package usecase

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
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

// SyncService synchronizes skills to targets.
type SyncService struct {
	store   *skill.Store
	targets *TargetRegistry
	cfg     *config.Config
}

// NewSyncService creates a new sync service.
func NewSyncService(fsys platformfs.FileSystem, cfg *config.Config, root string) *SyncService {
	return &SyncService{
		store:   skill.NewStore(fsys, cfg, root),
		targets: NewTargetRegistry(fsys, root, cfg),
		cfg:     cfg,
	}
}

// Sync synchronizes skills to targets.
func (s *SyncService) Sync(opts SyncOptions) ([]SyncResult, error) {
	skills, err := s.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if opts.Scope != nil {
		skills = filterSkillsByScope(skills, *opts.Scope)
	}

	targets := s.targets.GetAll()
	results := make([]SyncResult, 0, len(targets)*len(skills))

	for _, t := range targets {
		for _, sk := range skills {
			isInstalled := t.IsInstalledInScope(sk.Name, sk.Scope)
			result := s.syncSkill(t, sk, isInstalled, opts)
			results = append(results, result)
		}
	}

	return results, nil
}

func (s *SyncService) syncSkill(t *Target, sk *skill.Skill, isInstalled bool, opts SyncOptions) SyncResult {
	result := SyncResult{SkillName: sk.Name, Target: t.Name()}

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

	strategy := s.cfg.DefaultStrategy
	if strategy == "" {
		strategy = config.StrategySymlink
	}

	installOpts := InstallOptions{Strategy: strategy, Force: opts.Force || isInstalled}
	if err := t.Install(sk, installOpts); err != nil {
		result.Action = SyncActionError
		result.Error = err
	}

	return result
}

func filterSkillsByScope(skills []*skill.Skill, scope skill.Scope) []*skill.Skill {
	filtered := make([]*skill.Skill, 0, len(skills))
	for _, s := range skills {
		if s.Scope == scope {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
