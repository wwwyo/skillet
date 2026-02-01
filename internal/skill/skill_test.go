package skill

import (
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{"valid simple name", "my-skill", false},
		{"valid with underscore", "my_skill", false},
		{"valid with numbers", "skill123", false},
		{"valid alphanumeric", "MySkill2", false},
		{"valid single char", "a", false},

		// Invalid names - empty
		{"empty name", "", true},

		// Invalid names - path traversal
		{"contains forward slash", "my/skill", true},
		{"contains backslash", "my\\skill", true},
		{"contains double dot", "my..skill", true},
		{"path traversal attempt", "../etc/passwd", true},

		// Invalid names - hidden files
		{"starts with dot", ".hidden", true},
		{"dotfile", ".gitignore", true},

		// Valid - numbers at start are allowed by the regex
		{"starts with number", "123skill", false},

		// Invalid names - pattern mismatch
		{"starts with hyphen", "-skill", true},
		{"starts with underscore", "_skill", true},
		{"contains space", "my skill", true},
		{"contains special char", "skill@name", true},
		{"contains exclamation", "skill!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseScope(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Scope
		wantErr bool
	}{
		{"global scope", "global", ScopeGlobal, false},
		{"project scope", "project", ScopeProject, false},
		{"unknown scope", "unknown", ScopeGlobal, true},
		{"empty scope", "", ScopeGlobal, true},
		{"uppercase", "Global", ScopeGlobal, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseScope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseScope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseScope(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestScopeString(t *testing.T) {
	tests := []struct {
		scope Scope
		want  string
	}{
		{ScopeGlobal, "global"},
		{ScopeProject, "project"},
		{Scope(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.scope.String(); got != tt.want {
				t.Errorf("Scope.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCategoryString(t *testing.T) {
	tests := []struct {
		category Category
		want     string
	}{
		{CategoryDefault, "default"},
		{CategoryOptional, "optional"},
		{Category(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.category.String(); got != tt.want {
				t.Errorf("Category.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSkill(t *testing.T) {
	skill := NewSkill("test-skill")

	if skill.Name != "test-skill" {
		t.Errorf("NewSkill() Name = %v, want %v", skill.Name, "test-skill")
	}
	if skill.Description != "" {
		t.Errorf("NewSkill() Description = %v, want empty", skill.Description)
	}
}

func TestSkillBuilderPattern(t *testing.T) {
	skill := NewSkill("test-skill").
		WithDescription("A test skill").
		WithPath("/path/to/skill").
		WithScope(ScopeProject).
		WithCategory(CategoryDefault)

	if skill.Name != "test-skill" {
		t.Errorf("skill.Name = %v, want %v", skill.Name, "test-skill")
	}
	if skill.Description != "A test skill" {
		t.Errorf("skill.Description = %v, want %v", skill.Description, "A test skill")
	}
	if skill.Path != "/path/to/skill" {
		t.Errorf("skill.Path = %v, want %v", skill.Path, "/path/to/skill")
	}
	if skill.Scope != ScopeProject {
		t.Errorf("skill.Scope = %v, want %v", skill.Scope, ScopeProject)
	}
	if skill.Category != CategoryDefault {
		t.Errorf("skill.Category = %v, want %v", skill.Category, CategoryDefault)
	}
}

func TestSkillPriority(t *testing.T) {
	tests := []struct {
		name  string
		scope Scope
		want  int
	}{
		{"project priority", ScopeProject, 2},
		{"global priority", ScopeGlobal, 1},
		{"unknown priority", Scope(99), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := NewSkill("test").WithScope(tt.scope)
			if got := skill.Priority(); got != tt.want {
				t.Errorf("Skill.Priority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkillPriorityOrder(t *testing.T) {
	projectSkill := NewSkill("test").WithScope(ScopeProject)
	globalSkill := NewSkill("test").WithScope(ScopeGlobal)

	if projectSkill.Priority() <= globalSkill.Priority() {
		t.Error("Project scope should have higher priority than Global scope")
	}
}
