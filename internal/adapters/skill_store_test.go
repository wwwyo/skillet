package adapters

import (
	"testing"

	"github.com/wwwyo/skillet/internal/service"
)

// testConfig returns a default config for testing
func testConfig() *service.Config {
	return service.DefaultConfig()
}

// setupGlobalSkillsDir creates the global skills directory structure
func setupGlobalSkillsDir(m *MockFileSystem) {
	m.Dirs["/home/test/.agents"] = true
	m.Dirs["/home/test/.agents/skills"] = true
	m.Dirs["/home/test/.agents/skills/optional"] = true
}

// setupProjectSkillsDir creates the project skills directory structure
func setupProjectSkillsDir(m *MockFileSystem, projectRoot string) {
	m.Dirs[projectRoot+"/.agents"] = true
	m.Dirs[projectRoot+"/.agents/skills"] = true
	m.Dirs[projectRoot+"/.agents/skills/optional"] = true
}

// addSkillToMock adds a skill to the mock filesystem
func addSkillToMock(m *MockFileSystem, dir, name, desc string) {
	skillDir := dir + "/" + name
	m.Dirs[skillDir] = true
	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n"
	m.Files[skillDir+"/SKILL.md"] = []byte(content)
}

func TestNewSkillStore(t *testing.T) {
	mock := NewMockFileSystem()
	store := NewSkillStore(mock, testConfig(), "/project")

	if store == nil {
		t.Fatal("NewSkillStore() returned nil")
	}
	if store.fs != mock {
		t.Error("NewSkillStore() fs not set correctly")
	}
	if store.projectRoot != "/project" {
		t.Errorf("NewSkillStore() projectRoot = %v, want %v", store.projectRoot, "/project")
	}
}

func TestSkillStoreGetAll(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "global-default", "Global default skill")
	addSkillToMock(mock, "/home/test/.agents/skills/optional", "global-optional", "Global optional skill")
	addSkillToMock(mock, "/project/.agents/skills", "project-default", "Project default skill")
	addSkillToMock(mock, "/project/.agents/skills/optional", "project-optional", "Project optional skill")

	store := NewSkillStore(mock, testConfig(), "/project")
	skills, err := store.GetAll()

	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(skills) != 4 {
		t.Errorf("GetAll() returned %d skills, want 4", len(skills))
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}

	expectedNames := []string{"global-default", "global-optional", "project-default", "project-optional"}
	for _, name := range expectedNames {
		if !names[name] {
			t.Errorf("GetAll() missing skill %s", name)
		}
	}
}

func TestSkillStoreGetAllWithoutProject(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	addSkillToMock(mock, "/home/test/.agents/skills", "global-skill", "A global skill")

	store := NewSkillStore(mock, testConfig(), "")
	skills, err := store.GetAll()

	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("GetAll() returned %d skills, want 1", len(skills))
	}
}

func TestSkillStoreGetByScope(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "global-skill", "Global skill")
	addSkillToMock(mock, "/project/.agents/skills", "project-skill", "Project skill")

	store := NewSkillStore(mock, testConfig(), "/project")

	t.Run("get global scope", func(t *testing.T) {
		skills, err := store.GetByScope(service.ScopeGlobal)
		if err != nil {
			t.Fatalf("GetByScope(ScopeGlobal) error = %v", err)
		}
		if len(skills) != 1 {
			t.Errorf("GetByScope(ScopeGlobal) returned %d skills, want 1", len(skills))
		}
		if skills[0].Name != "global-skill" {
			t.Errorf("GetByScope(ScopeGlobal) skill name = %v, want %v", skills[0].Name, "global-skill")
		}
	})

	t.Run("get project scope", func(t *testing.T) {
		skills, err := store.GetByScope(service.ScopeProject)
		if err != nil {
			t.Fatalf("GetByScope(ScopeProject) error = %v", err)
		}
		if len(skills) != 1 {
			t.Errorf("GetByScope(ScopeProject) returned %d skills, want 1", len(skills))
		}
		if skills[0].Name != "project-skill" {
			t.Errorf("GetByScope(ScopeProject) skill name = %v, want %v", skills[0].Name, "project-skill")
		}
	})

	t.Run("unknown scope", func(t *testing.T) {
		_, err := store.GetByScope(service.Scope(99))
		if err == nil {
			t.Error("GetByScope(unknown) expected error, got nil")
		}
	})
}

func TestSkillStoreGetByName(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "shared-skill", "Global version")
	addSkillToMock(mock, "/project/.agents/skills", "shared-skill", "Project version")
	addSkillToMock(mock, "/home/test/.agents/skills", "unique-skill", "Unique skill")

	store := NewSkillStore(mock, testConfig(), "/project")

	t.Run("get skill with priority (project wins)", func(t *testing.T) {
		skill, err := store.GetByName("shared-skill")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if skill.Scope != service.ScopeProject {
			t.Errorf("GetByName() returned scope = %v, want project", skill.Scope)
		}
		if skill.Description != "Project version" {
			t.Errorf("GetByName() returned description = %v, want 'Project version'", skill.Description)
		}
	})

	t.Run("get unique skill", func(t *testing.T) {
		skill, err := store.GetByName("unique-skill")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if skill.Name != "unique-skill" {
			t.Errorf("GetByName() returned name = %v, want 'unique-skill'", skill.Name)
		}
	})

	t.Run("skill not found", func(t *testing.T) {
		_, err := store.GetByName("nonexistent")
		if err == nil {
			t.Error("GetByName() expected error for nonexistent skill, got nil")
		}
	})
}

func TestSkillStoreRemove(t *testing.T) {
	t.Run("remove existing skill", func(t *testing.T) {
		mock := NewMockFileSystem()
		setupGlobalSkillsDir(mock)
		addSkillToMock(mock, "/home/test/.agents/skills", "to-remove", "Skill to remove")

		store := NewSkillStore(mock, testConfig(), "")
		s, err := store.FindInScope("to-remove", service.ScopeGlobal)
		if err != nil {
			t.Fatalf("FindInScope() error = %v", err)
		}

		err = store.Remove(s)
		if err != nil {
			t.Fatalf("Remove() error = %v", err)
		}

		if mock.Exists("/home/test/.agents/skills/to-remove") {
			t.Error("Remove() did not delete skill directory")
		}
	})
}

func TestSkillStoreExists(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	addSkillToMock(mock, "/home/test/.agents/skills", "existing", "Existing skill")

	store := NewSkillStore(mock, testConfig(), "")

	if !store.Exists("existing") {
		t.Error("Exists() returned false for existing skill")
	}

	if store.Exists("nonexistent") {
		t.Error("Exists() returned true for nonexistent skill")
	}
}

func TestSkillStoreGetResolved(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "shared-skill", "Global version")
	addSkillToMock(mock, "/project/.agents/skills", "shared-skill", "Project version")
	addSkillToMock(mock, "/home/test/.agents/skills", "global-only", "Global only")
	addSkillToMock(mock, "/project/.agents/skills", "project-only", "Project only")

	store := NewSkillStore(mock, testConfig(), "/project")
	resolved, err := store.GetResolved()

	if err != nil {
		t.Fatalf("GetResolved() error = %v", err)
	}

	if len(resolved) != 3 {
		t.Errorf("GetResolved() returned %d skills, want 3", len(resolved))
	}

	var sharedSkill *service.Skill
	for _, s := range resolved {
		if s.Name == "shared-skill" {
			sharedSkill = s
			break
		}
	}

	if sharedSkill == nil {
		t.Fatal("GetResolved() did not return shared-skill")
	}

	if sharedSkill.Scope != service.ScopeProject {
		t.Errorf("GetResolved() shared-skill scope = %v, want project", sharedSkill.Scope)
	}

	if sharedSkill.Description != "Project version" {
		t.Errorf("GetResolved() shared-skill description = %v, want 'Project version'", sharedSkill.Description)
	}
}

func TestSkillStoreGetResolvedSorted(t *testing.T) {
	mock := NewMockFileSystem()
	setupGlobalSkillsDir(mock)

	addSkillToMock(mock, "/home/test/.agents/skills", "zebra", "Zebra skill")
	addSkillToMock(mock, "/home/test/.agents/skills", "alpha", "Alpha skill")
	addSkillToMock(mock, "/home/test/.agents/skills", "beta", "Beta skill")

	store := NewSkillStore(mock, testConfig(), "")
	resolved, err := store.GetResolved()

	if err != nil {
		t.Fatalf("GetResolved() error = %v", err)
	}

	expectedOrder := []string{"alpha", "beta", "zebra"}
	for i, s := range resolved {
		if s.Name != expectedOrder[i] {
			t.Errorf("GetResolved() skill[%d].Name = %v, want %v", i, s.Name, expectedOrder[i])
		}
	}
}

func TestSkillStoreLoadSkill(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MockFileSystem)
		dir      string
		wantName string
		wantDesc string
		wantErr  bool
	}{
		{
			name: "load valid skill",
			setup: func(m *MockFileSystem) {
				m.Dirs["/skills/my-skill"] = true
				m.Files["/skills/my-skill/SKILL.md"] = []byte("---\nname: my-skill\ndescription: A test skill\n---\n# My Skill\n")
			},
			dir:      "/skills/my-skill",
			wantName: "my-skill",
			wantDesc: "A test skill",
		},
		{
			name: "missing SKILL.md",
			setup: func(m *MockFileSystem) {
				m.Dirs["/skills/no-skill"] = true
			},
			dir:     "/skills/no-skill",
			wantErr: true,
		},
		{
			name: "invalid frontmatter",
			setup: func(m *MockFileSystem) {
				m.Dirs["/skills/invalid"] = true
				m.Files["/skills/invalid/SKILL.md"] = []byte("No frontmatter here\n")
			},
			dir:     "/skills/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockFileSystem()
			tt.setup(mock)
			store := NewSkillStore(mock, testConfig(), "")

			skill, err := store.loadSkill(tt.dir, service.ScopeGlobal, service.CategoryDefault)
			if tt.wantErr {
				if err == nil {
					t.Error("loadSkill() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("loadSkill() unexpected error: %v", err)
			}
			if skill.Name != tt.wantName {
				t.Errorf("loadSkill() Name = %v, want %v", skill.Name, tt.wantName)
			}
			if skill.Description != tt.wantDesc {
				t.Errorf("loadSkill() Description = %v, want %v", skill.Description, tt.wantDesc)
			}
		})
	}
}

func TestSkillStoreLoadAllInDir(t *testing.T) {
	t.Run("load default and optional skills", func(t *testing.T) {
		mock := NewMockFileSystem()
		mock.Dirs["/skills"] = true
		mock.Dirs["/skills/skill-a"] = true
		mock.Files["/skills/skill-a/SKILL.md"] = []byte("---\nname: skill-a\n---\n")
		mock.Dirs["/skills/skill-b"] = true
		mock.Files["/skills/skill-b/SKILL.md"] = []byte("---\nname: skill-b\n---\n")
		mock.Dirs["/skills/optional"] = true
		mock.Dirs["/skills/optional/skill-c"] = true
		mock.Files["/skills/optional/skill-c/SKILL.md"] = []byte("---\nname: skill-c\n---\n")

		store := NewSkillStore(mock, testConfig(), "")
		defaultSkills, optionalSkills, err := store.loadAllInDir("/skills", service.ScopeGlobal)

		if err != nil {
			t.Fatalf("loadAllInDir() error = %v", err)
		}

		if len(defaultSkills) != 2 {
			t.Errorf("loadAllInDir() default skills = %d, want 2", len(defaultSkills))
		}
		if len(optionalSkills) != 1 {
			t.Errorf("loadAllInDir() optional skills = %d, want 1", len(optionalSkills))
		}
	})
}
