package adapters

import (
	"cmp"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wwwyo/skillet/internal/service"
)

// SkillStore implements service.SkillStore.
type SkillStore struct {
	fs          service.FileSystem
	cfg         *service.Config
	projectRoot string
}

// Compile-time interface check.
var _ service.SkillStore = (*SkillStore)(nil)

// NewSkillStore creates a new SkillStore.
func NewSkillStore(fs service.FileSystem, cfg *service.Config, projectRoot string) *SkillStore {
	return &SkillStore{
		fs:          fs,
		cfg:         cfg,
		projectRoot: projectRoot,
	}
}

// GetAll returns all skills from all scopes.
func (s *SkillStore) GetAll() ([]*service.Skill, error) {
	var allSkills []*service.Skill

	globalSkills, err := s.getGlobalSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to load global skills: %w", err)
	}
	allSkills = append(allSkills, globalSkills...)

	projectSkills, err := s.getProjectSkills()
	if err != nil {
		return nil, fmt.Errorf("failed to load project skills: %w", err)
	}
	allSkills = append(allSkills, projectSkills...)

	return allSkills, nil
}

// GetByScope returns skills from a specific scope.
func (s *SkillStore) GetByScope(scope service.Scope) ([]*service.Skill, error) {
	switch scope {
	case service.ScopeGlobal:
		return s.getGlobalSkills()
	case service.ScopeProject:
		return s.getProjectSkills()
	default:
		return nil, fmt.Errorf("unknown scope: %v", scope)
	}
}

// GetByName returns a skill by name, respecting priority.
func (s *SkillStore) GetByName(name string) (*service.Skill, error) {
	if err := service.ValidateName(name); err != nil {
		return nil, fmt.Errorf("invalid skill name %q: %w", name, err)
	}
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	var best *service.Skill
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
func (s *SkillStore) Remove(skill *service.Skill) error {
	if err := s.fs.RemoveAll(skill.Path); err != nil {
		return fmt.Errorf("failed to remove skill: %w", err)
	}
	return nil
}

// Exists checks if a skill exists by name in any scope.
func (s *SkillStore) Exists(name string) bool {
	_, err := s.GetByName(name)
	return err == nil
}

// GetResolved returns all skills after resolving conflicts.
func (s *SkillStore) GetResolved() ([]*service.Skill, error) {
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	best := make(map[string]*service.Skill)
	for _, skill := range allSkills {
		if cur, ok := best[skill.Name]; !ok || skill.Priority() > cur.Priority() {
			best[skill.Name] = skill
		}
	}

	return slices.SortedFunc(maps.Values(best), func(a, b *service.Skill) int {
		return cmp.Compare(a.Name, b.Name)
	}), nil
}

// FindInScope finds a skill by name in a specific scope.
func (s *SkillStore) FindInScope(name string, scope service.Scope) (*service.Skill, error) {
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

// getGlobalSkills loads skills from global directories.
func (s *SkillStore) getGlobalSkills() ([]*service.Skill, error) {
	skillsDir, err := s.cfg.SkillsDir(s.fs, "")
	if err != nil {
		return nil, err
	}

	defaultSkills, optionalSkills, err := s.loadAllInDir(skillsDir, service.ScopeGlobal)
	if err != nil {
		return nil, err
	}

	return append(defaultSkills, optionalSkills...), nil
}

// getProjectSkills loads skills from project directories.
func (s *SkillStore) getProjectSkills() ([]*service.Skill, error) {
	if s.projectRoot == "" {
		return nil, nil
	}

	skillsDir := service.ProjectSkillsDir(s.projectRoot, s.fs, "")
	defaultSkills, optionalSkills, err := s.loadAllInDir(skillsDir, service.ScopeProject)
	if err != nil {
		return nil, err
	}

	return append(defaultSkills, optionalSkills...), nil
}

// --- Loader functionality (integrated) ---

const (
	maxSearchDepth = 5
	optionalDir    = "optional"
)

// skillMetadata represents the YAML frontmatter in SKILL.md.
type skillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// loadSkill loads a skill from a directory.
func (s *SkillStore) loadSkill(dir string, scope service.Scope, category service.Category) (*service.Skill, error) {
	skillFile := s.findSkillFile(dir)
	if skillFile == "" {
		return nil, fmt.Errorf("SKILL.md not found in %s", dir)
	}

	content, err := s.fs.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	meta, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}

	return service.NewSkill(s.fs.Base(dir), strings.TrimSpace(meta.Description), dir, scope, category)
}

// findSkillFile finds SKILL.md in a directory or its subdirectories.
func (s *SkillStore) findSkillFile(dir string) string {
	return s.findSkillFileWithDepth(dir, 0)
}

func (s *SkillStore) findSkillFileWithDepth(dir string, depth int) string {
	if depth > maxSearchDepth {
		return ""
	}

	skillFile := s.fs.Join(dir, "SKILL.md")
	if s.fs.Exists(skillFile) {
		return skillFile
	}

	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if found := s.findSkillFileWithDepth(s.fs.Join(dir, entry.Name()), depth+1); found != "" {
				return found
			}
		}
	}

	return ""
}

// parseFrontmatter extracts and parses YAML frontmatter from content.
func parseFrontmatter(content string) (*skillMetadata, error) {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	matches := re.FindStringSubmatch(content)

	if len(matches) < 2 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	var meta skillMetadata
	if err := yaml.Unmarshal([]byte(matches[1]), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &meta, nil
}

// listSkillsInDir returns all skill names in a directory.
func (s *SkillStore) listSkillsInDir(dir string) ([]string, error) {
	if !s.fs.Exists(dir) {
		return nil, nil
	}

	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&service.ModeSymlink != 0 {
			skillDir := s.fs.Join(dir, entry.Name())
			if service.IsValidSkillDir(s.fs, skillDir) {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

// loadAllInDir loads skills from a directory.
func (s *SkillStore) loadAllInDir(dir string, scope service.Scope) (defaultSkills, optionalSkills []*service.Skill, err error) {
	names, err := s.listSkillsInDir(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, name := range names {
		if name == optionalDir {
			continue
		}
		skill, err := s.loadSkill(s.fs.Join(dir, name), scope, service.CategoryDefault)
		if err != nil {
			continue
		}
		defaultSkills = append(defaultSkills, skill)
	}

	optDir := s.fs.Join(dir, optionalDir)
	optNames, err := s.listSkillsInDir(optDir)
	if err != nil {
		return defaultSkills, nil, nil
	}

	for _, name := range optNames {
		skill, err := s.loadSkill(s.fs.Join(optDir, name), scope, service.CategoryOptional)
		if err != nil {
			continue
		}
		optionalSkills = append(optionalSkills, skill)
	}

	return defaultSkills, optionalSkills, nil
}
