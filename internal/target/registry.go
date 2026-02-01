package target

import (
	"fmt"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
)

// Registry manages available targets.
type Registry struct {
	targets map[string]Target
}

// NewRegistry creates a new Registry with default targets.
func NewRegistry(fsys fs.System, projectRoot string, cfg *config.Config) *Registry {
	r := &Registry{targets: make(map[string]Target)}

	for name, def := range DefaultTargets {
		// Skip if disabled in config
		if cfg != nil && !cfg.Targets[name].Enabled {
			continue
		}

		// Use config's globalPath if set, otherwise use default
		globalPath := def.GlobalPath
		if cfg != nil && cfg.Targets[name].GlobalPath != "" {
			globalPath = cfg.Targets[name].GlobalPath
		}

		r.targets[name] = NewBaseTarget(name, globalPath, def.ProjectPath, def.SkillsDir, fsys, projectRoot)
	}

	return r
}

// Get returns a target by name.
func (r *Registry) Get(name string) (Target, error) {
	target, ok := r.targets[name]
	if !ok {
		return nil, fmt.Errorf("unknown target: %s", name)
	}
	return target, nil
}

// GetAll returns all registered targets.
func (r *Registry) GetAll() []Target {
	targets := make([]Target, 0, len(r.targets))
	for _, t := range r.targets {
		targets = append(targets, t)
	}
	return targets
}

// Names returns all registered target names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.targets))
	for name := range r.targets {
		names = append(names, name)
	}
	return names
}

// Register adds a custom target to the registry.
func (r *Registry) Register(target Target) {
	r.targets[target.Name()] = target
}
