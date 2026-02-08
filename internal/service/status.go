package service

import "fmt"

// GetStatus returns the synchronization status for all targets.
func (s *SkillService) GetStatus(opts ...GetStatusOptions) ([]*Status, error) {
	skills, err := s.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if len(opts) > 0 && opts[0].Scope != nil {
		skills = filterByScope(skills, *opts[0].Scope)
	}

	skillNames := make(map[string]bool)
	for _, sk := range skills {
		skillNames[sk.Name] = true
	}

	targets := s.targets.GetAll()
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
		for _, sk := range skills {
			if t.IsInstalledInScope(sk.Name, sk.Scope) {
				installedList = append(installedList, sk.Name)
			} else {
				missingList = append(missingList, sk.Name)
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
