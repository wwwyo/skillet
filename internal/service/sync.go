package service

import "fmt"

// Sync synchronizes skills to targets.
func (s *SkillService) Sync(opts SyncOptions) ([]SyncResult, error) {
	skills, err := s.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if opts.Scope != nil {
		skills = filterByScope(skills, *opts.Scope)
	}

	targets := s.targets.GetAll()
	var results []SyncResult

	for _, t := range targets {
		for _, sk := range skills {
			isInstalled := t.IsInstalledInScope(sk.Name, sk.Scope)
			result := s.syncSkill(t, sk, isInstalled, opts)
			results = append(results, result)
		}
	}

	return results, nil
}

// syncSkill syncs a single skill to a target.
func (s *SkillService) syncSkill(t Target, sk *Skill, isInstalled bool, opts SyncOptions) SyncResult {
	result := SyncResult{
		SkillName: sk.Name,
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

	strategy := s.cfg.DefaultStrategy
	if strategy == "" {
		strategy = StrategySymlink
	}

	installOpts := InstallOptions{
		Strategy: strategy,
		Force:    opts.Force || isInstalled,
	}

	if err := t.Install(sk, installOpts); err != nil {
		result.Action = SyncActionError
		result.Error = err
	}

	return result
}
