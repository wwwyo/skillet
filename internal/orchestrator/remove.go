package orchestrator

import (
	"fmt"

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
	TargetResults []TargetRemoveResult
	Error         error
}

// TargetRemoveResult represents the result of removing from a single target.
type TargetRemoveResult struct {
	Target  string
	Removed bool
	Error   error
}

// RemoveResultParams contains parameters for creating a RemoveResult.
type RemoveResultParams struct {
	SkillName     string
	Scope         skill.Scope
	StoreRemoved  bool
	TargetResults []TargetRemoveResult
	Error         error
}

// NewRemoveResult creates a new RemoveResult from the given parameters.
func NewRemoveResult(params RemoveResultParams) *RemoveResult {
	return &RemoveResult{
		SkillName:     params.SkillName,
		Scope:         params.Scope,
		StoreRemoved:  params.StoreRemoved,
		TargetResults: params.TargetResults,
		Error:         params.Error,
	}
}

// Success returns true if the removal was successful.
func (r *RemoveResult) Success() bool {
	return r.StoreRemoved && r.Error == nil
}

// Remove removes a skill from the store and all targets.
func (o *Orchestrator) Remove(opts RemoveOptions) *RemoveResult {
	if err := skill.ValidateName(opts.Name); err != nil {
		return NewRemoveResult(RemoveResultParams{
			SkillName: opts.Name,
			Error:     fmt.Errorf("invalid skill name: %w", err),
		})
	}

	var s *skill.Skill
	var err error
	if opts.Scope != nil {
		s, err = o.store.FindInScope(opts.Name, *opts.Scope)
		if err != nil {
			return NewRemoveResult(RemoveResultParams{
				SkillName: opts.Name,
				Scope:     *opts.Scope,
				Error:     fmt.Errorf("skill not found in %s scope: %w", *opts.Scope, err),
			})
		}
	} else {
		s, err = o.store.GetByName(opts.Name)
		if err != nil {
			return NewRemoveResult(RemoveResultParams{
				SkillName: opts.Name,
				Error:     fmt.Errorf("skill not found: %w", err),
			})
		}
	}

	if err := o.store.Remove(s); err != nil {
		return NewRemoveResult(RemoveResultParams{
			SkillName: s.Name,
			Scope:     s.Scope,
			Error:     fmt.Errorf("failed to remove from store: %w", err),
		})
	}

	var targetResults []TargetRemoveResult
	for _, t := range o.registry.GetAll() {
		result := TargetRemoveResult{Target: t.Name()}

		if t.IsInstalled(s.Name) {
			if err := t.Uninstall(s.Name); err != nil {
				result.Error = err
			} else {
				result.Removed = true
			}
		}

		targetResults = append(targetResults, result)
	}

	return NewRemoveResult(RemoveResultParams{
		SkillName:     s.Name,
		Scope:         s.Scope,
		StoreRemoved:  true,
		TargetResults: targetResults,
	})
}
