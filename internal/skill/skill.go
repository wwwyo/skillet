package skill

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wwwyo/skillet/internal/fs"
)

// Scope represents the scope level of a skill.
type Scope int

const (
	// ScopeGlobal represents skills stored in ~/.agents/skills/
	ScopeGlobal Scope = iota
	// ScopeProject represents skills stored in <project>/.agents/skills/
	ScopeProject
)

func (s Scope) String() string {
	switch s {
	case ScopeGlobal:
		return "global"
	case ScopeProject:
		return "project"
	default:
		return "unknown"
	}
}

// Category represents the category of a skill within a scope.
type Category int

const (
	// CategoryDefault represents skills that are always active (placed directly under skills/).
	CategoryDefault Category = iota
	// CategoryOptional represents skills that are optionally available (placed under skills/optional/).
	CategoryOptional
)

func (c Category) String() string {
	switch c {
	case CategoryDefault:
		return "default"
	case CategoryOptional:
		return "optional"
	default:
		return "unknown"
	}
}

// Skill represents an AI agent skill.
type Skill struct {
	Name        string
	Description string
	Path        string   // absolute path to the skill directory
	Scope       Scope    // where this skill is stored (global, project)
	Category    Category // whether the skill is always active or available on demand
}

// NewSkill creates a new Skill. Use for all Skill creation.
// Returns an error if the name is invalid.
func NewSkill(name, description, path string, scope Scope, category Category) (*Skill, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	return &Skill{
		Name:        name,
		Description: description,
		Path:        path,
		Scope:       scope,
		Category:    category,
	}, nil
}

// Priority returns the priority of this skill for conflict resolution.
// Higher priority wins. Project > Global.
func (s *Skill) Priority() int {
	switch s.Scope {
	case ScopeProject:
		return 2
	case ScopeGlobal:
		return 1
	default:
		return 0
	}
}

// validNamePattern matches valid skill names (alphanumeric, hyphen, underscore).
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ValidateName checks if a skill name is valid and safe.
// Returns an error if the name contains path traversal characters or is invalid.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("skill name cannot contain path separators: %s", name)
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("skill name cannot contain '..': %s", name)
	}

	// Check for hidden files
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("skill name cannot start with '.': %s", name)
	}

	// Validate against pattern
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("skill name must start with alphanumeric and contain only alphanumeric, hyphen, or underscore: %s", name)
	}

	return nil
}

// maxValidationDepth is the maximum depth to search for SKILL.md files.
const maxValidationDepth = 5

// IsValidSkillDir checks if a directory is a valid skill directory.
// A valid skill directory contains SKILL.md either directly or in a subdirectory.
func IsValidSkillDir(fsys fs.System, dir string) bool {
	return isValidSkillDirWithDepth(fsys, dir, 0)
}

func isValidSkillDirWithDepth(fsys fs.System, dir string, depth int) bool {
	if depth > maxValidationDepth {
		return false
	}

	// Check current directory
	if fsys.Exists(fsys.Join(dir, "SKILL.md")) {
		return true
	}

	// Check subdirectories recursively
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if isValidSkillDirWithDepth(fsys, fsys.Join(dir, entry.Name()), depth+1) {
				return true
			}
		}
	}

	return false
}
