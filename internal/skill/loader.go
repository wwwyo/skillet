package skill

import (
	"fmt"
	"strings"

	"github.com/wwwyo/skillet/internal/fs"
	"gopkg.in/yaml.v3"
)

const skillFileName = "SKILL.md"

// Loader loads skills from the filesystem.
type Loader struct {
	fs fs.System
}

// NewLoader creates a new Loader.
func NewLoader(fsys fs.System) *Loader {
	return &Loader{fs: fsys}
}

// skillFrontmatter represents the YAML frontmatter in SKILL.md.
type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Load loads a skill from a directory.
func (l *Loader) Load(dir string) (*Skill, error) {
	skillFile := l.fs.Join(dir, skillFileName)
	if !l.fs.Exists(skillFile) {
		return nil, fmt.Errorf("SKILL.md not found in %s", dir)
	}

	content, err := l.fs.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	fm, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	name := fm.Name
	if name == "" {
		name = l.fs.Base(dir)
	}

	return NewSkill(name, strings.TrimSpace(fm.Description), dir)
}

// LoadFromPath loads a skill from a path (directory or SKILL.md file).
func (l *Loader) LoadFromPath(path string) (*Skill, error) {
	if l.fs.Base(path) == skillFileName {
		path = l.fs.Dir(path)
	}
	return l.Load(path)
}

// IsValidSkillDir checks if a directory is a valid skill directory.
func (l *Loader) IsValidSkillDir(dir string) bool {
	skillFile := l.fs.Join(dir, skillFileName)
	return l.fs.Exists(skillFile)
}

// ListSkillsInDir returns a list of skill names in a directory.
func (l *Loader) ListSkillsInDir(dir string) ([]string, error) {
	if !l.fs.Exists(dir) {
		return nil, nil
	}

	entries, err := l.fs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			// Check if it's a symlink to a directory
			entryPath := l.fs.Join(dir, entry.Name())
			if !l.fs.IsSymlink(entryPath) {
				continue
			}
		}
		entryPath := l.fs.Join(dir, entry.Name())
		if l.IsValidSkillDir(entryPath) {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}

// LoadAllInDir loads all skills from a directory.
func (l *Loader) LoadAllInDir(dir string) ([]*Skill, error) {
	return l.LoadAllInDirExcluding(dir, "")
}

// LoadAllInDirExcluding loads all skills from a directory, excluding a subdirectory.
func (l *Loader) LoadAllInDirExcluding(dir string, exclude string) ([]*Skill, error) {
	names, err := l.ListSkillsInDir(dir)
	if err != nil {
		return nil, err
	}

	var skills []*Skill
	for _, name := range names {
		if exclude != "" && name == exclude {
			continue
		}
		skillDir := l.fs.Join(dir, name)
		skill, err := l.Load(skillDir)
		if err != nil {
			// Skip invalid skills
			continue
		}
		skills = append(skills, skill)
	}

	return skills, nil
}

// parseFrontmatter extracts YAML frontmatter from content.
func parseFrontmatter(content []byte) (*skillFrontmatter, error) {
	str := string(content)
	if !strings.HasPrefix(str, "---") {
		return nil, fmt.Errorf("no frontmatter found")
	}

	// Find end of frontmatter
	endIdx := strings.Index(str[3:], "---")
	if endIdx == -1 {
		return nil, fmt.Errorf("no frontmatter found")
	}

	fmContent := str[3 : 3+endIdx]
	var fm skillFrontmatter
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &fm, nil
}
