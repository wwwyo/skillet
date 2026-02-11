package usecase

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

// StatusResult represents the synchronization status for a target.
type StatusResult struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	InSync    bool
	Error     error
}

// StatusOptions contains options for getting status.
type StatusOptions struct {
	// Scope limits status to a specific scope (nil for all)
	Scope *skill.Scope
}

// StatusService returns synchronization status across targets.
type StatusService struct {
	store   *skill.Store
	targets *TargetRegistry
}

// NewStatusService creates a new status service.
func NewStatusService(fsys platformfs.FileSystem, cfg *config.Config, root string) *StatusService {
	return &StatusService{
		store:   skill.NewStore(fsys, cfg, root),
		targets: NewTargetRegistry(fsys, root, cfg),
	}
}

// GetStatus returns the synchronization status for all targets.
func (s *StatusService) GetStatus(opts ...StatusOptions) ([]*StatusResult, error) {
	skills, err := s.store.GetResolved()
	if err != nil {
		return nil, fmt.Errorf("failed to get skills: %w", err)
	}

	if len(opts) > 0 && opts[0].Scope != nil {
		skills = filterSkillsByScope(skills, *opts[0].Scope)
	}

	skillNames := make(map[string]bool, len(skills))
	for _, sk := range skills {
		skillNames[sk.Name] = true
	}

	targets := s.targets.GetAll()
	statuses := make([]*StatusResult, 0, len(targets))

	for _, t := range targets {
		installed, err := t.ListInstalled()
		if err != nil {
			statuses = append(statuses, &StatusResult{
				Target: t.Name(),
				Error:  fmt.Errorf("failed to list installed skills: %w", err),
			})
			continue
		}

		installedSet := make(map[string]bool, len(installed))
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

		statuses = append(statuses, &StatusResult{
			Target:    t.Name(),
			Installed: installedList,
			Missing:   missingList,
			Extra:     extraList,
			InSync:    len(missingList) == 0 && len(extraList) == 0,
		})
	}

	return statuses, nil
}
