package skill

import (
	"fmt"
	"regexp"
	"strings"
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

// Priority returns the priority for conflict resolution.
// Higher priority wins. Project > Global.
func (s Scope) Priority() int {
	switch s {
	case ScopeProject:
		return 2
	case ScopeGlobal:
		return 1
	default:
		return 0
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

// Skill represents an AI agent skill (pure data).
type Skill struct {
	Name        string
	Description string
	Path        string // absolute path to the skill directory
}

// NewSkill creates a new Skill. Use for all Skill creation.
// Returns an error if the name is invalid.
func NewSkill(name, description, path string) (*Skill, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	return &Skill{
		Name:        name,
		Description: description,
		Path:        path,
	}, nil
}

// ScopedSkill wraps a Skill with storage context (scope and category).
// Use this when you need to know where a skill is stored.
type ScopedSkill struct {
	*Skill
	Scope    Scope
	Category Category
}

// NewScopedSkill creates a new ScopedSkill.
func NewScopedSkill(skill *Skill, scope Scope, category Category) *ScopedSkill {
	return &ScopedSkill{
		Skill:    skill,
		Scope:    scope,
		Category: category,
	}
}

// Priority returns the priority for conflict resolution based on scope.
func (s *ScopedSkill) Priority() int {
	return s.Scope.Priority()
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
