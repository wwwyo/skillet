package skill

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
)

// Store manages skills across all scopes.
type Store struct {
	fs          fs.System
	cfg         *config.Config
	loader      *Loader
	projectRoot string
}

// NewStore creates a new Store.
func NewStore(fsys fs.System, cfg *config.Config, projectRoot string) *Store {
	return &Store{
		fs:          fsys,
		cfg:         cfg,
		loader:      NewLoader(fsys),
		projectRoot: projectRoot,
	}
}

// GetAll returns all skills from all scopes.
func (s *Store) GetAll() ([]*Skill, error) {
	var allSkills []*Skill

	// Load global skills
	globalSkills, err := s.getGlobalSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to load global skills: %w", err)
	}
	allSkills = append(allSkills, globalSkills...)

	// Load project skills
	projectSkills, err := s.getProjectSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to load project skills: %w", err)
	}
	allSkills = append(allSkills, projectSkills...)

	return allSkills, nil
}

// GetByScope returns skills from a specific scope.
func (s *Store) GetByScope(scope Scope) ([]*Skill, error) {
	switch scope {
	case ScopeGlobal:
		return s.getGlobalSkills()
	case ScopeProject:
		return s.getProjectSkills()
	default:
		return nil, fmt.Errorf("unknown scope: %v", scope)
	}
}

// GetByName returns a skill by name, respecting priority.
// Project scope has highest priority, followed by global.
func (s *Store) GetByName(name string) (*Skill, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("invalid skill name %q: %w", name, err)
	}
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	var best *Skill
	for _, skill := range allSkills {
		if skill.Name == name && (best == nil || skill.Priority() > best.Priority()) {
			best = skill
		}
	}

	if best == nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return best, nil
}

// Remove removes a skill from the store.
func (s *Store) Remove(skill *Skill) error {
	if err := s.fs.RemoveAll(skill.Path); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}
	return nil
}

// Exists checks if a skill exists by name in any scope.
func (s *Store) Exists(name string) bool {
	_, err := s.GetByName(name)
	return err == nil
}

// GetResolved returns all skills after resolving conflicts.
// Higher priority scopes override lower priority ones.
func (s *Store) GetResolved() ([]*Skill, error) {
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	// Keep highest priority for each name
	best := make(map[string]*Skill)
	for _, skill := range allSkills {
		if cur, ok := best[skill.Name]; !ok || skill.Priority() > cur.Priority() {
			best[skill.Name] = skill
		}
	}

	return slices.SortedFunc(maps.Values(best), func(a, b *Skill) int {
		return cmp.Compare(a.Name, b.Name)
	}), nil
}

// getGlobalSkills loads skills from global directories.
func (s *Store) getGlobalSkills() ([]*Skill, error) {
	var skills []*Skill

	// Default skills (directly under skills/, excluding optional/)
	skillsDir, err := s.cfg.SkillsDir(s.fs, "")
	if err != nil {
		return nil, err
	}
	defaultSkills, err := s.loader.LoadAllInDirExcluding(skillsDir, ScopeGlobal, CategoryDefault, config.OptionalDir)
	if err != nil {
		return nil, err
	}
	skills = append(skills, defaultSkills...)

	// Optional skills
	optionalDir, err := s.cfg.SkillsDir(s.fs, config.OptionalDir)
	if err != nil {
		return nil, err
	}
	optionalSkills, err := s.loader.LoadAllInDir(optionalDir, ScopeGlobal, CategoryOptional)
	if err != nil {
		return nil, err
	}
	skills = append(skills, optionalSkills...)

	return skills, nil
}

// getProjectSkills loads skills from project directories.
func (s *Store) getProjectSkills() ([]*Skill, error) {
	if s.projectRoot == "" {
		return nil, nil
	}

	var skills []*Skill

	// Default skills (directly under skills/, excluding optional/)
	skillsDir := config.ProjectSkillsDir(s.projectRoot, s.fs, "")
	defaultSkills, err := s.loader.LoadAllInDirExcluding(skillsDir, ScopeProject, CategoryDefault, config.OptionalDir)
	if err != nil {
		return nil, err
	}
	skills = append(skills, defaultSkills...)

	// Optional skills
	optionalDir := config.ProjectSkillsDir(s.projectRoot, s.fs, config.OptionalDir)
	optionalSkills, err := s.loader.LoadAllInDir(optionalDir, ScopeProject, CategoryOptional)
	if err != nil {
		return nil, err
	}
	skills = append(skills, optionalSkills...)

	return skills, nil
}

// FindInScope finds a skill by name in a specific scope.
func (s *Store) FindInScope(name string, scope Scope) (*Skill, error) {
	skills, err := s.GetByScope(scope)
	if err != nil {
		return nil, err
	}

	for _, skill := range skills {
		if skill.Name == name {
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill %s not found in %s scope", name, scope)
}
