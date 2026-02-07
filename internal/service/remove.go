package service

import "fmt"

// Remove removes a skill from the store and all targets.
func (s *SkillService) Remove(opts RemoveOptions) *RemoveResult {
	if err := ValidateName(opts.Name); err != nil {
		return NewRemoveResult(RemoveResultParams{
			SkillName: opts.Name,
			Error:     fmt.Errorf("invalid skill name: %w", err),
		})
	}

	var sk *Skill
	var err error
	if opts.Scope != nil {
		sk, err = s.store.FindInScope(opts.Name, *opts.Scope)
		if err != nil {
			return NewRemoveResult(RemoveResultParams{
				SkillName: opts.Name,
				Scope:     *opts.Scope,
				Error:     fmt.Errorf("skill not found in %s scope: %w", *opts.Scope, err),
			})
		}
	} else {
		sk, err = s.store.GetByName(opts.Name)
		if err != nil {
			return NewRemoveResult(RemoveResultParams{
				SkillName: opts.Name,
				Error:     fmt.Errorf("skill not found: %w", err),
			})
		}
	}

	if err := s.store.Remove(sk); err != nil {
		return NewRemoveResult(RemoveResultParams{
			SkillName: sk.Name,
			Scope:     sk.Scope,
			Error:     fmt.Errorf("failed to remove from store: %w", err),
		})
	}

	var targetResults []TargetRemoveResult
	for _, t := range s.targets.GetAll() {
		result := TargetRemoveResult{Target: t.Name()}

		if t.IsInstalled(sk.Name) {
			if err := t.Uninstall(sk.Name); err != nil {
				result.Error = err
			} else {
				result.Removed = true
			}
		}

		targetResults = append(targetResults, result)
	}

	return NewRemoveResult(RemoveResultParams{
		SkillName:     sk.Name,
		Scope:         sk.Scope,
		StoreRemoved:  true,
		TargetResults: targetResults,
	})
}
