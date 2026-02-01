package orchestrator

import (
	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

// Orchestrator coordinates operations between skill stores and targets.
type Orchestrator struct {
	fs          fs.System
	store       *skill.Store
	registry    *target.Registry
	cfg         *config.Config
	projectRoot string
}

// New creates a new Orchestrator.
func New(fsys fs.System, store *skill.Store, registry *target.Registry, cfg *config.Config, projectRoot string) *Orchestrator {
	return &Orchestrator{
		fs:          fsys,
		store:       store,
		registry:    registry,
		cfg:         cfg,
		projectRoot: projectRoot,
	}
}

// filterByScope filters skills by the specified scope.
func filterByScope(skills []*skill.Skill, scope skill.Scope) []*skill.Skill {
	var filtered []*skill.Skill
	for _, s := range skills {
		if s.Scope == scope {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
