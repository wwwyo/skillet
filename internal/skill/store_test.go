package skill

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
)

// testConfig returns a default config for testing
func testConfig() *config.Config {
	return config.Default()
}

// setupGlobalSkillsDir creates the global skills directory structure
func setupGlobalSkillsDir(m *fs.MockSystem) {
	m.Dirs["/home/test/.agents"] = true
	m.Dirs["/home/test/.agents/skills"] = true
	m.Dirs["/home/test/.agents/skills/optional"] = true
}

// setupProjectSkillsDir creates the project skills directory structure
func setupProjectSkillsDir(m *fs.MockSystem, projectRoot string) {
	m.Dirs[projectRoot+"/.agents"] = true
	m.Dirs[projectRoot+"/.agents/skills"] = true
	m.Dirs[projectRoot+"/.agents/skills/optional"] = true
}

// addSkillToMock adds a skill to the mock filesystem
func addSkillToMock(m *fs.MockSystem, dir, name, desc string) {
	skillDir := dir + "/" + name
	m.Dirs[skillDir] = true
	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n"
	m.Files[skillDir+"/SKILL.md"] = []byte(content)
}

func TestNewStore(t *testing.T) {
	mock := fs.NewMock()
	store := NewStore(mock, testConfig(), "/project")

	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	if store.fs != mock {
		t.Error("NewStore() fs not set correctly")
	}
	if store.projectRoot != "/project" {
		t.Errorf("NewStore() projectRoot = %v, want %v", store.projectRoot, "/project")
	}
}

func TestStoreGetAll(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	// Add global skills
	addSkillToMock(mock, "/home/test/.agents/skills", "global-default", "Global default skill")
	addSkillToMock(mock, "/home/test/.agents/skills/optional", "global-optional", "Global optional skill")

	// Add project skills
	addSkillToMock(mock, "/project/.agents/skills", "project-default", "Project default skill")
	addSkillToMock(mock, "/project/.agents/skills/optional", "project-optional", "Project optional skill")

	store := NewStore(mock, testConfig(), "/project")
	skills, err := store.GetAll()

	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(skills) != 4 {
		t.Errorf("GetAll() returned %d skills, want 4", len(skills))
	}

	// Check all skill names are present
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

func TestStoreGetAllWithoutProject(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	addSkillToMock(mock, "/home/test/.agents/skills", "global-skill", "A global skill")

	store := NewStore(mock, testConfig(), "")
	skills, err := store.GetAll()

	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("GetAll() returned %d skills, want 1", len(skills))
	}
}

func TestStoreGetByScope(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "global-skill", "Global skill")
	addSkillToMock(mock, "/project/.agents/skills", "project-skill", "Project skill")

	store := NewStore(mock, testConfig(), "/project")

	t.Run("get global scope", func(t *testing.T) {
		skills, err := store.GetByScope(ScopeGlobal)
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
		skills, err := store.GetByScope(ScopeProject)
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
		_, err := store.GetByScope(Scope(99))
		if err == nil {
			t.Error("GetByScope(unknown) expected error, got nil")
		}
	})
}

func TestStoreGetByName(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	addSkillToMock(mock, "/home/test/.agents/skills", "shared-skill", "Global version")
	addSkillToMock(mock, "/project/.agents/skills", "shared-skill", "Project version")
	addSkillToMock(mock, "/home/test/.agents/skills", "unique-skill", "Unique skill")

	store := NewStore(mock, testConfig(), "/project")

	t.Run("get skill with priority (project wins)", func(t *testing.T) {
		skill, err := store.GetByName("shared-skill")
		if err != nil {
			t.Fatalf("GetByName() error = %v", err)
		}
		if skill.Scope != ScopeProject {
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

func TestStoreRemove(t *testing.T) {
	t.Run("remove existing skill", func(t *testing.T) {
		mock := fs.NewMock()
		setupGlobalSkillsDir(mock)
		addSkillToMock(mock, "/home/test/.agents/skills", "to-remove", "Skill to remove")

		store := NewStore(mock, testConfig(), "")
		s, err := store.FindInScope("to-remove", ScopeGlobal)
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

func TestStoreExists(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	addSkillToMock(mock, "/home/test/.agents/skills", "existing", "Existing skill")

	store := NewStore(mock, testConfig(), "")

	if !store.Exists("existing") {
		t.Error("Exists() returned false for existing skill")
	}

	if store.Exists("nonexistent") {
		t.Error("Exists() returned true for nonexistent skill")
	}
}

func TestStoreGetResolved(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)
	setupProjectSkillsDir(mock, "/project")

	// Add skills with same name in different scopes
	addSkillToMock(mock, "/home/test/.agents/skills", "shared-skill", "Global version")
	addSkillToMock(mock, "/project/.agents/skills", "shared-skill", "Project version")

	// Add unique skills
	addSkillToMock(mock, "/home/test/.agents/skills", "global-only", "Global only")
	addSkillToMock(mock, "/project/.agents/skills", "project-only", "Project only")

	store := NewStore(mock, testConfig(), "/project")
	resolved, err := store.GetResolved()

	if err != nil {
		t.Fatalf("GetResolved() error = %v", err)
	}

	// Should have 3 resolved skills (shared-skill resolved to project, plus 2 unique)
	if len(resolved) != 3 {
		t.Errorf("GetResolved() returned %d skills, want 3", len(resolved))
	}

	// Find shared-skill and verify it's the project version
	var sharedSkill *Skill
	for _, s := range resolved {
		if s.Name == "shared-skill" {
			sharedSkill = s
			break
		}
	}

	if sharedSkill == nil {
		t.Fatal("GetResolved() did not return shared-skill")
	}

	if sharedSkill.Scope != ScopeProject {
		t.Errorf("GetResolved() shared-skill scope = %v, want project", sharedSkill.Scope)
	}

	if sharedSkill.Description != "Project version" {
		t.Errorf("GetResolved() shared-skill description = %v, want 'Project version'", sharedSkill.Description)
	}
}

func TestStoreGetResolvedSorted(t *testing.T) {
	mock := fs.NewMock()
	setupGlobalSkillsDir(mock)

	// Add skills in non-alphabetical order
	addSkillToMock(mock, "/home/test/.agents/skills", "zebra", "Zebra skill")
	addSkillToMock(mock, "/home/test/.agents/skills", "alpha", "Alpha skill")
	addSkillToMock(mock, "/home/test/.agents/skills", "beta", "Beta skill")

	store := NewStore(mock, testConfig(), "")
	resolved, err := store.GetResolved()

	if err != nil {
		t.Fatalf("GetResolved() error = %v", err)
	}

	// Verify skills are sorted by name
	expectedOrder := []string{"alpha", "beta", "zebra"}
	for i, s := range resolved {
		if s.Name != expectedOrder[i] {
			t.Errorf("GetResolved() skill[%d].Name = %v, want %v", i, s.Name, expectedOrder[i])
		}
	}
}
