package service

import (
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "my-skill", false},
		{"valid with underscore", "my_skill", false},
		{"valid with numbers", "skill123", false},
		{"valid alphanumeric", "MySkill2", false},
		{"valid single char", "a", false},
		{"empty name", "", true},
		{"contains forward slash", "my/skill", true},
		{"contains backslash", "my\\skill", true},
		{"contains double dot", "my..skill", true},
		{"path traversal attempt", "../etc/passwd", true},
		{"starts with dot", ".hidden", true},
		{"dotfile", ".gitignore", true},
		{"starts with number", "123skill", false},
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
			skill, err := NewSkill("test", "", "", tt.scope, 0)
			if err != nil {
				t.Fatalf("NewSkill() error = %v", err)
			}
			if got := skill.Priority(); got != tt.want {
				t.Errorf("Skill.Priority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkillPriorityOrder(t *testing.T) {
	projectSkill, _ := NewSkill("test", "", "", ScopeProject, 0)
	globalSkill, _ := NewSkill("test", "", "", ScopeGlobal, 0)

	if projectSkill.Priority() <= globalSkill.Priority() {
		t.Error("Project scope should have higher priority than Global scope")
	}
}
