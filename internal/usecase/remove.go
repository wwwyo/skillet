package usecase

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

// RemoveOptions contains options for removing a skill.
type RemoveOptions struct {
	// Name is the skill name to remove
	Name string
	// Scope limits removal to a specific scope (nil to auto-detect)
	Scope *skill.Scope
}

// RemoveResult represents the result of a remove operation.
type RemoveResult struct {
	SkillName     string
	Scope         skill.Scope
	StoreRemoved  bool
	TargetResults []RemoveTargetResult
	Error         error
}

// RemoveTargetResult represents the result of removing from a single target.
type RemoveTargetResult struct {
	Target  string
	Removed bool
	Error   error
}

// RemoveService removes skills from store and targets.
type RemoveService struct {
	store   *skill.Store
	targets *TargetRegistry
}

// NewRemoveService creates a new remove service.
func NewRemoveService(fsys platformfs.FileSystem, cfg *config.Config, root string) *RemoveService {
	return &RemoveService{
		store:   skill.NewStore(fsys, cfg, root),
		targets: NewTargetRegistry(fsys, root, cfg),
	}
}

// Remove removes a skill from the store and all targets.
func (s *RemoveService) Remove(opts RemoveOptions) *RemoveResult {
	if err := skill.ValidateName(opts.Name); err != nil {
		return &RemoveResult{SkillName: opts.Name, Error: fmt.Errorf("invalid skill name: %w", err)}
	}

	var sk *skill.Skill
	var err error
	if opts.Scope != nil {
		sk, err = s.store.FindInScope(opts.Name, *opts.Scope)
		if err != nil {
			return &RemoveResult{
				SkillName: opts.Name,
				Scope:     *opts.Scope,
				Error:     fmt.Errorf("skill not found in %s scope: %w", *opts.Scope, err),
			}
		}
	} else {
		sk, err = s.store.GetByName(opts.Name)
		if err != nil {
			return &RemoveResult{SkillName: opts.Name, Error: fmt.Errorf("skill not found: %w", err)}
		}
	}

	// Remove from targets first, before removing from store.
	// This prevents leaving broken symlinks that would be skipped by exists checks.
	targetResults := make([]RemoveTargetResult, 0, len(s.targets.GetAll()))
	for _, t := range s.targets.GetAll() {
		result := RemoveTargetResult{Target: t.Name()}
		if t.IsInstalled(sk.Name) {
			if err := t.Uninstall(sk.Name); err != nil {
				result.Error = err
			} else {
				result.Removed = true
			}
		}
		targetResults = append(targetResults, result)
	}

	if err := s.store.Remove(sk); err != nil {
		return &RemoveResult{
			SkillName: sk.Name,
			Scope:     sk.Scope,
			Error:     fmt.Errorf("failed to remove from store: %w", err),
		}
	}

	return &RemoveResult{
		SkillName:     sk.Name,
		Scope:         sk.Scope,
		StoreRemoved:  true,
		TargetResults: targetResults,
	}
}

// Success returns true if the removal was successful.
func (r *RemoveResult) Success() bool {
	return r.StoreRemoved && r.Error == nil
}
