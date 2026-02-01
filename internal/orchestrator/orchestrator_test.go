package orchestrator

import (
	"fmt"
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

func TestNew(t *testing.T) {
	mock, store, registry, cfg := setupTestEnv()
	orch := New(mock, store, registry, cfg, "/project")

	if orch == nil {
		t.Fatal("New() returned nil")
	}
	if orch.fs != mock {
		t.Error("New() fs not set correctly")
	}
	if orch.store != store {
		t.Error("New() store not set correctly")
	}
	if orch.registry != registry {
		t.Error("New() registry not set correctly")
	}
	if orch.projectRoot != "/project" {
		t.Errorf("New() projectRoot = %v, want /project", orch.projectRoot)
	}
}

func TestSync(t *testing.T) {
	t.Run("sync skills to targets", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "test-skill", "Test skill")

		orch := New(mock, store, registry, cfg, "")
		results, err := orch.Sync(SyncOptions{})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		if len(results) == 0 {
			t.Error("Sync() returned no results")
		}

		hasInstall := false
		for _, r := range results {
			if r.Action == SyncActionInstall {
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

		orch := New(mock, store, registry, cfg, "")
		results, err := orch.Sync(SyncOptions{DryRun: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		claudeSkillPath := "/home/test/.claude/skills/dry-run-skill"
		if mock.Exists(claudeSkillPath) {
			t.Error("Sync() with DryRun should not install skill")
		}

		hasInstall := false
		for _, r := range results {
			if r.Action == SyncActionInstall && r.SkillName == "dry-run-skill" {
				hasInstall = true
				break
			}
		}
		if !hasInstall {
			t.Error("Sync() DryRun should report install action")
		}
	})

	t.Run("sync with scope filter", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "global-skill", "Global skill")

		orch := New(mock, store, registry, cfg, "")
		globalScope := skill.ScopeGlobal
		results, err := orch.Sync(SyncOptions{Scope: &globalScope})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		for _, r := range results {
			if r.Action == SyncActionInstall && r.SkillName == "global-skill" {
				return
			}
		}
		t.Error("Sync() with global scope should sync global skills")
	})

	t.Run("skip already installed skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "installed-skill", "Already installed")

		mock.Dirs["/home/test/.claude/skills/installed-skill"] = true
		mock.Files["/home/test/.claude/skills/installed-skill/SKILL.md"] = []byte("---\nname: installed-skill\n---\n")
		mock.Dirs["/home/test/.codex/skills/installed-skill"] = true
		mock.Files["/home/test/.codex/skills/installed-skill/SKILL.md"] = []byte("---\nname: installed-skill\n---\n")

		orch := New(mock, store, registry, cfg, "")
		results, err := orch.Sync(SyncOptions{})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		hasSkip := false
		for _, r := range results {
			if r.Action == SyncActionSkip && r.SkillName == "installed-skill" {
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

		mock.Dirs["/home/test/.claude/skills/update-skill"] = true
		mock.Files["/home/test/.claude/skills/update-skill/SKILL.md"] = []byte("---\nname: update-skill\n---\n")
		mock.Dirs["/home/test/.codex/skills/update-skill"] = true
		mock.Files["/home/test/.codex/skills/update-skill/SKILL.md"] = []byte("---\nname: update-skill\n---\n")

		orch := New(mock, store, registry, cfg, "")
		results, err := orch.Sync(SyncOptions{Force: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		hasUpdate := false
		for _, r := range results {
			if r.Action == SyncActionUpdate && r.SkillName == "update-skill" {
				hasUpdate = true
				break
			}
		}
		if !hasUpdate {
			t.Error("Sync() with Force should update already installed skill")
		}
	})
}

func TestGetStatus(t *testing.T) {
	t.Run("status in sync", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "synced-skill", "Synced skill")

		mock.Dirs["/home/test/.claude/skills/synced-skill"] = true
		mock.Dirs["/home/test/.codex/skills/synced-skill"] = true

		orch := New(mock, store, registry, cfg, "")
		statuses, err := orch.GetStatus()

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

		orch := New(mock, store, registry, cfg, "")
		statuses, err := orch.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		for _, status := range statuses {
			if status.InSync {
				t.Errorf("GetStatus() target %s should not be in sync", status.Target)
			}
			if len(status.Missing) == 0 {
				t.Errorf("GetStatus() target %s should have missing skills", status.Target)
			}
		}
	})

	t.Run("status extra skills", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		mock.Dirs["/home/test/.claude/skills/extra-skill"] = true

		orch := New(mock, store, registry, cfg, "")
		statuses, err := orch.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		var claudeStatus *Status
		for _, s := range statuses {
			if s.Target == "claude" {
				claudeStatus = s
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

func TestNewStatus(t *testing.T) {
	t.Run("empty status is in sync", func(t *testing.T) {
		status := NewStatus(StatusParams{Target: "claude"})

		if status.Target != "claude" {
			t.Errorf("NewStatus().Target = %v, want claude", status.Target)
		}
		if !status.InSync {
			t.Error("NewStatus() with no missing/extra should be in sync")
		}
	})

	t.Run("status with missing skills is not in sync", func(t *testing.T) {
		status := NewStatus(StatusParams{
			Target:  "claude",
			Missing: []string{"skill-b"},
		})

		if status.InSync {
			t.Error("Status with missing skills should not be in sync")
		}
	})

	t.Run("status with extra skills is not in sync", func(t *testing.T) {
		status := NewStatus(StatusParams{
			Target: "codex",
			Extra:  []string{"skill-c"},
		})

		if status.InSync {
			t.Error("Status with extra skills should not be in sync")
		}
	})

	t.Run("status with error is not in sync", func(t *testing.T) {
		status := NewStatus(StatusParams{
			Target: "test",
			Error:  fmt.Errorf("test error"),
		})

		if status.InSync {
			t.Error("Status with error should not be in sync")
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("remove existing skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "remove-me", "Skill to remove")

		// Pre-install to targets
		mock.Dirs["/home/test/.claude/skills/remove-me"] = true
		mock.Dirs["/home/test/.codex/skills/remove-me"] = true

		orch := New(mock, store, registry, cfg, "")
		result := orch.Remove(RemoveOptions{Name: "remove-me"})

		if result.Error != nil {
			t.Fatalf("Remove() error = %v", result.Error)
		}

		if !result.StoreRemoved {
			t.Error("Remove() should have removed from store")
		}

		// Check targets were cleaned up
		for _, tr := range result.TargetResults {
			if tr.Error != nil {
				t.Errorf("Remove() target %s error = %v", tr.Target, tr.Error)
			}
		}
	})

	t.Run("remove non-existent skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		orch := New(mock, store, registry, cfg, "")
		result := orch.Remove(RemoveOptions{Name: "does-not-exist"})

		if result.Error == nil {
			t.Error("Remove() should return error for non-existent skill")
		}
	})

	t.Run("remove with invalid name", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		orch := New(mock, store, registry, cfg, "")
		result := orch.Remove(RemoveOptions{Name: "../invalid"})

		if result.Error == nil {
			t.Error("Remove() should return error for invalid name")
		}
	})
}

func TestSyncActionConstants(t *testing.T) {
	tests := []struct {
		action SyncAction
		want   string
	}{
		{SyncActionInstall, "install"},
		{SyncActionUpdate, "update"},
		{SyncActionUninstall, "uninstall"},
		{SyncActionSkip, "skip"},
		{SyncActionError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("SyncAction = %v, want %v", tt.action, tt.want)
			}
		})
	}
}
