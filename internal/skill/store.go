package skill

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strings"

	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"gopkg.in/yaml.v3"
)

// SkillsPathResolver resolves scope-specific skill root directories.
type SkillsPathResolver interface {
	GlobalSkillsDir(fsys platformfs.FileSystem) (string, error)
	ProjectSkillsDir(fsys platformfs.FileSystem, projectRoot string) string
}

// Store manages skill persistence and retrieval.
type Store struct {
	fs          platformfs.FileSystem
	paths       SkillsPathResolver
	projectRoot string
}

// NewStore creates a new Store.
func NewStore(fsys platformfs.FileSystem, paths SkillsPathResolver, projectRoot string) *Store {
	return &Store{
		fs:          fsys,
		paths:       paths,
		projectRoot: projectRoot,
	}
}

// GetAll returns all skills from all scopes.
func (s *Store) GetAll() ([]*Skill, error) {
	var allSkills []*Skill

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
func (s *Store) GetByName(name string) (*Skill, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("invalid skill name %q: %w", name, err)
	}
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	var best *Skill
	for _, sk := range allSkills {
		if sk.Name == name && (best == nil || sk.Priority() > best.Priority()) {
			best = sk
		}
	}

	if best == nil {
		return nil, fmt.Errorf("skill not found: %s", name)
	}

	return best, nil
}

// Remove removes a skill from the store.
func (s *Store) Remove(sk *Skill) error {
	if err := s.fs.RemoveAll(sk.Path); err != nil {
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
func (s *Store) GetResolved() ([]*Skill, error) {
	allSkills, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	best := make(map[string]*Skill)
	for _, sk := range allSkills {
		if cur, ok := best[sk.Name]; !ok || sk.Priority() > cur.Priority() {
			best[sk.Name] = sk
		}
	}

	return slices.SortedFunc(maps.Values(best), func(a, b *Skill) int {
		return cmp.Compare(a.Name, b.Name)
	}), nil
}

// FindInScope finds a skill by name in a specific scope.
func (s *Store) FindInScope(name string, scope Scope) (*Skill, error) {
	skills, err := s.GetByScope(scope)
	if err != nil {
		return nil, err
	}

	for _, sk := range skills {
		if sk.Name == name {
			return sk, nil
		}
	}

	return nil, fmt.Errorf("skill %s not found in %s scope", name, scope)
}

// getGlobalSkills loads skills from global directories.
func (s *Store) getGlobalSkills() ([]*Skill, error) {
	skillsDir, err := s.paths.GlobalSkillsDir(s.fs)
	if err != nil {
		return nil, err
	}

	defaultSkills, optionalSkills, err := s.loadAllInDir(skillsDir, ScopeGlobal)
	if err != nil {
		return nil, err
	}

	return append(defaultSkills, optionalSkills...), nil
}

// getProjectSkills loads skills from project directories.
func (s *Store) getProjectSkills() ([]*Skill, error) {
	if s.projectRoot == "" {
		return nil, nil
	}

	skillsDir := s.paths.ProjectSkillsDir(s.fs, s.projectRoot)
	defaultSkills, optionalSkills, err := s.loadAllInDir(skillsDir, ScopeProject)
	if err != nil {
		return nil, err
	}

	return append(defaultSkills, optionalSkills...), nil
}

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
func (s *Store) loadSkill(dir string, scope Scope, category Category) (*Skill, error) {
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

	return NewSkill(s.fs.Base(dir), strings.TrimSpace(meta.Description), dir, scope, category)
}

// findSkillFile finds SKILL.md in a directory or its subdirectories.
func (s *Store) findSkillFile(dir string) string {
	return s.findSkillFileWithDepth(dir, 0)
}

func (s *Store) findSkillFileWithDepth(dir string, depth int) string {
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

var frontmatterRegex = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)

// parseFrontmatter extracts and parses YAML frontmatter from content.
func parseFrontmatter(content string) (*skillMetadata, error) {
	matches := frontmatterRegex.FindStringSubmatch(content)

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
func (s *Store) listSkillsInDir(dir string) ([]string, error) {
	if !s.fs.Exists(dir) {
		return nil, nil
	}

	entries, err := s.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			skillDir := s.fs.Join(dir, entry.Name())
			if isValidSkillDir(s.fs, skillDir) {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

// loadAllInDir loads skills from a directory.
func (s *Store) loadAllInDir(dir string, scope Scope) (defaultSkills, optionalSkills []*Skill, err error) {
	names, err := s.listSkillsInDir(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, name := range names {
		if name == optionalDir {
			continue
		}
		sk, loadErr := s.loadSkill(s.fs.Join(dir, name), scope, CategoryDefault)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load skill %q: %v\n", name, loadErr)
			continue
		}
		defaultSkills = append(defaultSkills, sk)
	}

	optDir := s.fs.Join(dir, optionalDir)
	optNames, err := s.listSkillsInDir(optDir)
	if err != nil {
		return defaultSkills, nil, nil
	}

	for _, name := range optNames {
		sk, loadErr := s.loadSkill(s.fs.Join(optDir, name), scope, CategoryOptional)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load optional skill %q: %v\n", name, loadErr)
			continue
		}
		optionalSkills = append(optionalSkills, sk)
	}

	return defaultSkills, optionalSkills, nil
}
