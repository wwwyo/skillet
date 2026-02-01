package skill

import (
	"testing"

	"github.com/wwwyo/skillet/internal/fs"
)

func TestLoaderLoad(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*fs.MockSystem)
		dir         string
		wantName    string
		wantDesc    string
		wantErr     bool
		errContains string
	}{
		{
			name: "load valid skill",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/my-skill"] = true
				m.Files["/skills/my-skill/SKILL.md"] = []byte(`---
name: my-skill
description: A test skill
---
# My Skill
This is my skill content.
`)
			},
			dir:      "/skills/my-skill",
			wantName: "my-skill",
			wantDesc: "A test skill",
			wantErr:  false,
		},
		{
			name: "load skill without name uses directory name",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/fallback-skill"] = true
				m.Files["/skills/fallback-skill/SKILL.md"] = []byte(`---
description: A skill without name
---
# Content
`)
			},
			dir:      "/skills/fallback-skill",
			wantName: "fallback-skill",
			wantDesc: "A skill without name",
			wantErr:  false,
		},
		{
			name: "missing SKILL.md",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/no-skill"] = true
			},
			dir:         "/skills/no-skill",
			wantErr:     true,
			errContains: "SKILL.md not found",
		},
		{
			name: "invalid frontmatter",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/invalid"] = true
				m.Files["/skills/invalid/SKILL.md"] = []byte(`No frontmatter here
Just regular content.
`)
			},
			dir:         "/skills/invalid",
			wantErr:     true,
			errContains: "no frontmatter found",
		},
		{
			name: "malformed yaml",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/malformed"] = true
				m.Files["/skills/malformed/SKILL.md"] = []byte(`---
name: [invalid yaml
  broken: true
---
`)
			},
			dir:         "/skills/malformed",
			wantErr:     true,
			errContains: "failed to parse YAML",
		},
		{
			name: "trim description whitespace",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/whitespace"] = true
				m.Files["/skills/whitespace/SKILL.md"] = []byte(`---
name: whitespace
description: "  spaced description  "
---
`)
			},
			dir:      "/skills/whitespace",
			wantName: "whitespace",
			wantDesc: "spaced description",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := fs.NewMock()
			tt.setup(mock)
			loader := NewLoader(mock)

			skill, err := loader.Load(tt.dir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Load() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error: %v", err)
				return
			}

			if skill.Name != tt.wantName {
				t.Errorf("Load() Name = %v, want %v", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("Load() Description = %v, want %v", skill.Description, tt.wantDesc)
			}
			if skill.Path != tt.dir {
				t.Errorf("Load() Path = %v, want %v", skill.Path, tt.dir)
			}
		})
	}
}

func TestLoaderLoadFromPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*fs.MockSystem)
		path     string
		wantName string
		wantErr  bool
	}{
		{
			name: "load from directory",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/test-skill"] = true
				m.Files["/skills/test-skill/SKILL.md"] = []byte(`---
name: test-skill
---
`)
			},
			path:     "/skills/test-skill",
			wantName: "test-skill",
			wantErr:  false,
		},
		{
			name: "load from SKILL.md file",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/file-skill"] = true
				m.Files["/skills/file-skill/SKILL.md"] = []byte(`---
name: file-skill
---
`)
			},
			path:     "/skills/file-skill/SKILL.md",
			wantName: "file-skill",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := fs.NewMock()
			tt.setup(mock)
			loader := NewLoader(mock)

			skill, err := loader.LoadFromPath(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadFromPath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadFromPath() unexpected error: %v", err)
				return
			}

			if skill.Name != tt.wantName {
				t.Errorf("LoadFromPath() Name = %v, want %v", skill.Name, tt.wantName)
			}
		})
	}
}

func TestLoaderIsValidSkillDir(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*fs.MockSystem)
		dir   string
		want  bool
	}{
		{
			name: "valid skill directory",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/valid"] = true
				m.Files["/skills/valid/SKILL.md"] = []byte(`---
name: valid
---
`)
			},
			dir:  "/skills/valid",
			want: true,
		},
		{
			name: "directory without SKILL.md",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills/invalid"] = true
				m.Files["/skills/invalid/README.md"] = []byte("# Readme")
			},
			dir:  "/skills/invalid",
			want: false,
		},
		{
			name: "non-existent directory",
			setup: func(m *fs.MockSystem) {
			},
			dir:  "/skills/nonexistent",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := fs.NewMock()
			tt.setup(mock)
			loader := NewLoader(mock)

			got := loader.IsValidSkillDir(tt.dir)
			if got != tt.want {
				t.Errorf("IsValidSkillDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoaderListSkillsInDir(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*fs.MockSystem)
		dir     string
		want    []string
		wantErr bool
	}{
		{
			name: "list multiple skills",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
				m.Dirs["/skills/skill-a"] = true
				m.Files["/skills/skill-a/SKILL.md"] = []byte(`---
name: skill-a
---
`)
				m.Dirs["/skills/skill-b"] = true
				m.Files["/skills/skill-b/SKILL.md"] = []byte(`---
name: skill-b
---
`)
			},
			dir:     "/skills",
			want:    []string{"skill-a", "skill-b"},
			wantErr: false,
		},
		{
			name: "skip directories without SKILL.md",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
				m.Dirs["/skills/valid"] = true
				m.Files["/skills/valid/SKILL.md"] = []byte(`---
name: valid
---
`)
				m.Dirs["/skills/invalid"] = true
				m.Files["/skills/invalid/README.md"] = []byte("# Readme")
			},
			dir:     "/skills",
			want:    []string{"valid"},
			wantErr: false,
		},
		{
			name: "non-existent directory returns nil",
			setup: func(m *fs.MockSystem) {
			},
			dir:     "/nonexistent",
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty directory",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
			},
			dir:     "/skills",
			want:    nil,
			wantErr: false,
		},
		{
			name: "include symlinked skills",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
				m.Dirs["/source/linked-skill"] = true
				m.Files["/source/linked-skill/SKILL.md"] = []byte(`---
name: linked-skill
---
`)
				m.Symlinks["/skills/linked-skill"] = "/source/linked-skill"
				// MockSystem needs the symlink path to also have the SKILL.md accessible
				// Since MockSystem.Exists doesn't resolve symlinks for sub-paths,
				// we also register the file at the symlink path
				m.Files["/skills/linked-skill/SKILL.md"] = []byte(`---
name: linked-skill
---
`)
			},
			dir:     "/skills",
			want:    []string{"linked-skill"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := fs.NewMock()
			tt.setup(mock)
			loader := NewLoader(mock)

			got, err := loader.ListSkillsInDir(tt.dir)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListSkillsInDir() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ListSkillsInDir() unexpected error: %v", err)
				return
			}

			if !stringSliceEqual(got, tt.want) {
				t.Errorf("ListSkillsInDir() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoaderLoadAllInDir(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*fs.MockSystem)
		dir       string
		scope     Scope
		category  Category
		wantNames []string
		wantErr   bool
	}{
		{
			name: "load all skills with scope and category",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
				m.Dirs["/skills/skill-a"] = true
				m.Files["/skills/skill-a/SKILL.md"] = []byte(`---
name: skill-a
---
`)
				m.Dirs["/skills/skill-b"] = true
				m.Files["/skills/skill-b/SKILL.md"] = []byte(`---
name: skill-b
---
`)
			},
			dir:       "/skills",
			scope:     ScopeGlobal,
			category:  CategoryDefault,
			wantNames: []string{"skill-a", "skill-b"},
			wantErr:   false,
		},
		{
			name: "skip invalid skills",
			setup: func(m *fs.MockSystem) {
				m.Dirs["/skills"] = true
				m.Dirs["/skills/valid"] = true
				m.Files["/skills/valid/SKILL.md"] = []byte(`---
name: valid
---
`)
				m.Dirs["/skills/invalid"] = true
				m.Files["/skills/invalid/SKILL.md"] = []byte(`No frontmatter`)
			},
			dir:       "/skills",
			scope:     ScopeProject,
			category:  CategoryOptional,
			wantNames: []string{"valid"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := fs.NewMock()
			tt.setup(mock)
			loader := NewLoader(mock)

			skills, err := loader.LoadAllInDir(tt.dir, tt.scope, tt.category)

			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadAllInDir() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LoadAllInDir() unexpected error: %v", err)
				return
			}

			var gotNames []string
			for _, s := range skills {
				gotNames = append(gotNames, s.Name)
				// Verify scope and category are set
				if s.Scope != tt.scope {
					t.Errorf("LoadAllInDir() skill.Scope = %v, want %v", s.Scope, tt.scope)
				}
				if s.Category != tt.category {
					t.Errorf("LoadAllInDir() skill.Category = %v, want %v", s.Category, tt.category)
				}
			}

			if !stringSliceEqual(gotNames, tt.wantNames) {
				t.Errorf("LoadAllInDir() names = %v, want %v", gotNames, tt.wantNames)
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]int)
	for _, v := range a {
		aMap[v]++
	}
	for _, v := range b {
		aMap[v]--
		if aMap[v] < 0 {
			return false
		}
	}
	return true
}
