package orchestrator

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/skill"
)

// Status represents the synchronization status for a target.
type Status struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	InSync    bool
	Error     error
}

// StatusParams contains parameters for creating a new Status.
type StatusParams struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	Error     error
}

// NewStatus creates a new Status from the given parameters.
// InSync is computed automatically: true only when no missing skills, no extra skills, and no error.
func NewStatus(params StatusParams) *Status {
	return &Status{
		Target:    params.Target,
		Installed: params.Installed,
		Missing:   params.Missing,
		Extra:     params.Extra,
		Error:     params.Error,
		InSync:    len(params.Missing) == 0 && len(params.Extra) == 0 && params.Error == nil,
	}
}

// GetStatusOptions contains options for getting status.
type GetStatusOptions struct {
	// Scope limits status to a specific scope (nil for all)
	Scope *skill.Scope
}

// GetStatus returns the synchronization status for all targets.
func (o *Orchestrator) GetStatus(opts ...GetStatusOptions) ([]*Status, error) {
	skills, err := o.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if len(opts) > 0 && opts[0].Scope != nil {
		skills = filterByScope(skills, *opts[0].Scope)
	}

	skillNames := make(map[string]bool)
	for _, s := range skills {
		skillNames[s.Name] = true
	}

	targets := o.registry.GetAll()
	var statuses []*Status

	for _, t := range targets {
		installed, err := t.ListInstalled()
		if err != nil {
			statuses = append(statuses, NewStatus(StatusParams{
				Target: t.Name(),
				Error:  fmt.Errorf("failed to list installed skills: %w", err),
			}))
			continue
		}

		installedSet := make(map[string]bool)
		for _, name := range installed {
			installedSet[name] = true
		}

		var installedList, missingList []string
		for _, s := range skills {
			if t.IsInstalledInScope(s.Name, s.Scope) {
				installedList = append(installedList, s.Name)
			} else {
				missingList = append(missingList, s.Name)
			}
		}

		var extraList []string
		for name := range installedSet {
			if !skillNames[name] {
				extraList = append(extraList, name)
			}
		}

		statuses = append(statuses, NewStatus(StatusParams{
			Target:    t.Name(),
			Installed: installedList,
			Missing:   missingList,
			Extra:     extraList,
		}))
	}

	return statuses, nil
}
