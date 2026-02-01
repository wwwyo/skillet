package sync

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

// setupTestEnv creates a test environment with mock filesystem
func setupTestEnv() (*fs.MockSystem, *skill.Store, *target.Registry, *config.Config) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/test"

	// Setup global skills directory
	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.agents/skills/optional"] = true

	// Setup target directories
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := config.Default()
	store := skill.NewStore(mock, cfg, "")
	registry := target.NewRegistry(mock, "", cfg)

	return mock, store, registry, cfg
}

// addSkillToStore adds a skill to the mock filesystem
// category: "default" for skills directly under skills/, "optional" for skills/optional/
func addSkillToStore(m *fs.MockSystem, category, name, desc string) {
	var skillDir string
	if category == "default" {
		skillDir = "/home/test/.agents/skills/" + name
	} else {
		skillDir = "/home/test/.agents/skills/" + category + "/" + name
	}
	m.Dirs[skillDir] = true
	content := "---\nname: " + name + "\ndescription: " + desc + "\n---\n"
	m.Files[skillDir+"/SKILL.md"] = []byte(content)
}

func TestNewEngine(t *testing.T) {
	mock, store, registry, cfg := setupTestEnv()
	engine := NewEngine(mock, store, registry, cfg, "/project")

	if engine == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if engine.fs != mock {
		t.Error("NewEngine() fs not set correctly")
	}
	if engine.store != store {
		t.Error("NewEngine() store not set correctly")
	}
	if engine.registry != registry {
		t.Error("NewEngine() registry not set correctly")
	}
	if engine.projectRoot != "/project" {
		t.Errorf("NewEngine() projectRoot = %v, want /project", engine.projectRoot)
	}
}

func TestEngineSync(t *testing.T) {
	t.Run("sync skills to targets", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "test-skill", "Test skill")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Should have results for each target
		if len(results) == 0 {
			t.Error("Sync() returned no results")
		}

		// Check that install action was taken
		hasInstall := false
		for _, r := range results {
			if r.Action == ActionInstall {
				hasInstall = true
				break
			}
		}
		if !hasInstall {
			t.Error("Sync() should have install action")
		}
	})

	t.Run("sync dry run", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "dry-run-skill", "Dry run test")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{DryRun: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Verify skill was not actually installed
		claudeSkillPath := "/home/test/.claude/skills/dry-run-skill"
		if mock.Exists(claudeSkillPath) {
			t.Error("Sync() with DryRun should not install skill")
		}

		// But should still report install action
		hasInstall := false
		for _, r := range results {
			if r.Action == ActionInstall && r.SkillName == "dry-run-skill" {
				hasInstall = true
				break
			}
		}
		if !hasInstall {
			t.Error("Sync() DryRun should report install action")
		}
	})

	t.Run("sync to specific target", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "target-skill", "Target test")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{TargetName: "claude"})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Should only have results for claude target
		for _, r := range results {
			if r.Target != "claude" && r.Action != ActionError {
				t.Errorf("Sync() with TargetName=claude returned result for target %s", r.Target)
			}
		}
	})

	t.Run("sync unknown target", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		engine := NewEngine(mock, store, registry, cfg, "")
		_, err := engine.Sync(SyncOptions{TargetName: "unknown"})

		if err == nil {
			t.Error("Sync() expected error for unknown target, got nil")
		}
	})

	t.Run("skip already installed skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "installed-skill", "Already installed")

		// Pre-install skill
		mock.Dirs["/home/test/.claude/skills/installed-skill"] = true
		mock.Files["/home/test/.claude/skills/installed-skill/SKILL.md"] = []byte("---\nname: installed-skill\n---\n")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{TargetName: "claude"})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Should skip already installed skill
		hasSkip := false
		for _, r := range results {
			if r.Action == ActionSkip && r.SkillName == "installed-skill" {
				hasSkip = true
				break
			}
		}
		if !hasSkip {
			t.Error("Sync() should skip already installed skill")
		}
	})

	t.Run("force update installed skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "update-skill", "Force update test")

		// Pre-install skill
		mock.Dirs["/home/test/.claude/skills/update-skill"] = true
		mock.Files["/home/test/.claude/skills/update-skill/SKILL.md"] = []byte("---\nname: update-skill\n---\n")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{TargetName: "claude", Force: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Should update installed skill
		hasUpdate := false
		for _, r := range results {
			if r.Action == ActionUpdate && r.SkillName == "update-skill" {
				hasUpdate = true
				break
			}
		}
		if !hasUpdate {
			t.Error("Sync() with Force should update already installed skill")
		}
	})

	t.Run("uninstall extra skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		// Install a skill that's not in the store
		mock.Dirs["/home/test/.claude/skills/extra-skill"] = true
		mock.Files["/home/test/.claude/skills/extra-skill/SKILL.md"] = []byte("---\nname: extra-skill\n---\n")

		engine := NewEngine(mock, store, registry, cfg, "")
		results, err := engine.Sync(SyncOptions{TargetName: "claude"})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		// Should uninstall extra skill
		hasUninstall := false
		for _, r := range results {
			if r.Action == ActionUninstall && r.SkillName == "extra-skill" {
				hasUninstall = true
				break
			}
		}
		if !hasUninstall {
			t.Error("Sync() should uninstall extra skill not in store")
		}
	})
}

func TestEngineGetStatus(t *testing.T) {
	t.Run("status in sync", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "synced-skill", "Synced skill")

		// Install skill to match store
		mock.Dirs["/home/test/.claude/skills/synced-skill"] = true
		mock.Dirs["/home/test/.codex/skills/synced-skill"] = true

		engine := NewEngine(mock, store, registry, cfg, "")
		statuses, err := engine.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		for _, status := range statuses {
			if !status.InSync {
				t.Errorf("GetStatus() target %s should be in sync", status.Target)
			}
		}
	})

	t.Run("status missing skills", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "missing-skill", "Missing skill")

		engine := NewEngine(mock, store, registry, cfg, "")
		statuses, err := engine.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		for _, status := range statuses {
			if status.InSync {
				t.Errorf("GetStatus() target %s should not be in sync (missing skill)", status.Target)
			}
			if len(status.Missing) == 0 {
				t.Errorf("GetStatus() target %s should have missing skills", status.Target)
			}
		}
	})

	t.Run("status extra skills", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		// Install a skill that's not in the store
		mock.Dirs["/home/test/.claude/skills/extra-skill"] = true

		engine := NewEngine(mock, store, registry, cfg, "")
		statuses, err := engine.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		var claudeStatus *Status
		for i := range statuses {
			if statuses[i].Target == "claude" {
				claudeStatus = &statuses[i]
				break
			}
		}

		if claudeStatus == nil {
			t.Fatal("GetStatus() did not return claude target")
		}

		if claudeStatus.InSync {
			t.Error("GetStatus() claude should not be in sync (extra skill)")
		}
		if len(claudeStatus.Extra) == 0 {
			t.Error("GetStatus() claude should have extra skills")
		}
	})
}

func TestActionConstants(t *testing.T) {
	tests := []struct {
		action Action
		want   string
	}{
		{ActionInstall, "install"},
		{ActionUpdate, "update"},
		{ActionUninstall, "uninstall"},
		{ActionSkip, "skip"},
		{ActionError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("Action = %v, want %v", tt.action, tt.want)
			}
		})
	}
}

func TestResultStruct(t *testing.T) {
	result := Result{
		SkillName: "test-skill",
		Target:    "claude",
		Action:    ActionInstall,
		Error:     nil,
	}

	if result.SkillName != "test-skill" {
		t.Errorf("Result.SkillName = %v, want test-skill", result.SkillName)
	}
	if result.Target != "claude" {
		t.Errorf("Result.Target = %v, want claude", result.Target)
	}
	if result.Action != ActionInstall {
		t.Errorf("Result.Action = %v, want install", result.Action)
	}
}

func TestStatusStruct(t *testing.T) {
	status := Status{
		Target:    "claude",
		Installed: []string{"skill-a"},
		Missing:   []string{"skill-b"},
		Extra:     []string{"skill-c"},
		InSync:    false,
		Error:     nil,
	}

	if status.Target != "claude" {
		t.Errorf("Status.Target = %v, want claude", status.Target)
	}
	if len(status.Installed) != 1 {
		t.Errorf("Status.Installed = %v, want 1 item", status.Installed)
	}
	if len(status.Missing) != 1 {
		t.Errorf("Status.Missing = %v, want 1 item", status.Missing)
	}
	if len(status.Extra) != 1 {
		t.Errorf("Status.Extra = %v, want 1 item", status.Extra)
	}
}
