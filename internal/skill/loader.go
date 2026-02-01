package skill

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wwwyo/skillet/internal/fs"
)

// maxSearchDepth is the maximum depth to search for SKILL.md files.
const maxSearchDepth = 5

// Loader loads skills from the file system.
type Loader struct {
	fs fs.System
}

// NewLoader creates a new Loader.
func NewLoader(fsys fs.System) *Loader {
	return &Loader{fs: fsys}
}

// skillMetadata represents the YAML frontmatter in SKILL.md.
type skillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Load loads a skill from a directory.
// The directory must contain a SKILL.md file (directly or in subdirectories) with YAML frontmatter.
// The skill name is always the directory name (for consistent symlink creation).
func (l *Loader) Load(dir string, scope Scope, category Category) (*Skill, error) {
	skillFile := l.findSkillFile(dir)

	if skillFile == "" {
		return nil, fmt.Errorf("SKILL.md not found in %s", dir)
	}

	content, err := l.fs.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	meta, err := l.parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}

	return NewSkill(l.fs.Base(dir), strings.TrimSpace(meta.Description), dir, scope, category)
}

// findSkillFile finds SKILL.md in a directory or its subdirectories.
// Returns the path to the first SKILL.md found, or empty string if not found.
func (l *Loader) findSkillFile(dir string) string {
	return l.findSkillFileWithDepth(dir, 0)
}

// findSkillFileWithDepth finds SKILL.md with depth tracking to prevent infinite recursion.
func (l *Loader) findSkillFileWithDepth(dir string, depth int) string {
	if depth > maxSearchDepth {
		return ""
	}

	// Check current directory first
	skillFile := l.fs.Join(dir, "SKILL.md")
	if l.fs.Exists(skillFile) {
		return skillFile
	}

	// Check subdirectories recursively
	entries, err := l.fs.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if found := l.findSkillFileWithDepth(l.fs.Join(dir, entry.Name()), depth+1); found != "" {
				return found
			}
		}
	}

	return ""
}

// LoadFromPath loads a skill from a path.
// If the path is a directory, it loads from that directory.
// If the path is a file, it assumes it's a SKILL.md and loads from its parent directory.
func (l *Loader) LoadFromPath(path string, scope Scope, category Category) (*Skill, error) {
	if l.fs.IsDir(path) {
		return l.Load(path, scope, category)
	}

	// Assume it's a SKILL.md file
	return l.Load(l.fs.Dir(path), scope, category)
}

// parseFrontmatter extracts and parses YAML frontmatter from content.
// Frontmatter is delimited by --- at the start and end.
func (l *Loader) parseFrontmatter(content string) (*skillMetadata, error) {
	// Match YAML frontmatter between --- delimiters
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

// ListSkillsInDir returns all skill names in a directory.
// It expects the directory to contain subdirectories, each being a skill.
func (l *Loader) ListSkillsInDir(dir string) ([]string, error) {
	if !l.fs.Exists(dir) {
		return nil, nil
	}

	entries, err := l.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var skills []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 {
			skillDir := l.fs.Join(dir, entry.Name())
			if IsValidSkillDir(l.fs, skillDir) {
				skills = append(skills, entry.Name())
			}
		}
	}

	return skills, nil
}

const optionalDir = "optional"

// LoadAllInDir loads skills from a directory.
// Returns default skills (directly under dir) and optional skills (under dir/optional/).
func (l *Loader) LoadAllInDir(dir string, scope Scope) (defaultSkills, optionalSkills []*Skill, err error) {
	// Load default skills (excluding optional subdirectory)
	names, err := l.ListSkillsInDir(dir)
	if err != nil {
		return nil, nil, err
	}

	for _, name := range names {
		if name == optionalDir {
			continue
		}
		skill, err := l.Load(l.fs.Join(dir, name), scope, CategoryDefault)
		if err != nil {
			continue
		}
		defaultSkills = append(defaultSkills, skill)
	}

	// Load optional skills
	optDir := l.fs.Join(dir, optionalDir)
	optNames, err := l.ListSkillsInDir(optDir)
	if err != nil {
		return defaultSkills, nil, nil // optional dir may not exist
	}

	for _, name := range optNames {
		skill, err := l.Load(l.fs.Join(optDir, name), scope, CategoryOptional)
		if err != nil {
			continue
		}
		optionalSkills = append(optionalSkills, skill)
	}

	return defaultSkills, optionalSkills, nil
}
