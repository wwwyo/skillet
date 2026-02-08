package service_test

import (
	"fmt"
	"testing"

	"github.com/wwwyo/skillet/internal/adapters"
	"github.com/wwwyo/skillet/internal/service"
)

// setupTestEnv creates a test environment with mock filesystem
func setupTestEnv() (*adapters.MockFileSystem, *adapters.SkillStore, *adapters.Registry, *service.Config) {
	mock := adapters.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.agents/skills/optional"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := service.DefaultConfig()
	store := adapters.NewSkillStore(mock, cfg, "")
	registry := adapters.NewRegistry(mock, "", cfg)

	return mock, store, registry, cfg
}

// addSkillToStore adds a skill to the mock filesystem
func addSkillToStore(m *adapters.MockFileSystem, category, name, desc string) {
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

func TestNewSkillService(t *testing.T) {
	mock, store, registry, cfg := setupTestEnv()
	svc := service.NewSkillService(mock, store, registry, cfg, "/project")

	if svc == nil {
		t.Fatal("NewSkillService() returned nil")
	}
}

func TestSync(t *testing.T) {
	t.Run("sync skills to targets", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()
		addSkillToStore(mock, "default", "test-skill", "Test skill")

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		results, err := svc.Sync(service.SyncOptions{})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		if len(results) == 0 {
			t.Error("Sync() returned no results")
		}

		hasInstall := false
		for _, r := range results {
			if r.Action == service.SyncActionInstall {
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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		results, err := svc.Sync(service.SyncOptions{DryRun: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		claudeSkillPath := "/home/test/.claude/skills/dry-run-skill"
		if mock.Exists(claudeSkillPath) {
			t.Error("Sync() with DryRun should not install skill")
		}

		hasInstall := false
		for _, r := range results {
			if r.Action == service.SyncActionInstall && r.SkillName == "dry-run-skill" {
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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		globalScope := service.ScopeGlobal
		results, err := svc.Sync(service.SyncOptions{Scope: &globalScope})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		for _, r := range results {
			if r.Action == service.SyncActionInstall && r.SkillName == "global-skill" {
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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		results, err := svc.Sync(service.SyncOptions{})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		hasSkip := false
		for _, r := range results {
			if r.Action == service.SyncActionSkip && r.SkillName == "installed-skill" {
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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		results, err := svc.Sync(service.SyncOptions{Force: true})

		if err != nil {
			t.Fatalf("Sync() error = %v", err)
		}

		hasUpdate := false
		for _, r := range results {
			if r.Action == service.SyncActionUpdate && r.SkillName == "update-skill" {
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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		statuses, err := svc.GetStatus()

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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		statuses, err := svc.GetStatus()

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

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		statuses, err := svc.GetStatus()

		if err != nil {
			t.Fatalf("GetStatus() error = %v", err)
		}

		var claudeStatus *service.Status
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
		status := service.NewStatus(service.StatusParams{Target: "claude"})
		if !status.InSync {
			t.Error("NewStatus() with no missing/extra should be in sync")
		}
	})

	t.Run("status with missing skills is not in sync", func(t *testing.T) {
		status := service.NewStatus(service.StatusParams{
			Target:  "claude",
			Missing: []string{"skill-b"},
		})
		if status.InSync {
			t.Error("Status with missing skills should not be in sync")
		}
	})

	t.Run("status with error is not in sync", func(t *testing.T) {
		status := service.NewStatus(service.StatusParams{
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

		mock.Dirs["/home/test/.claude/skills/remove-me"] = true
		mock.Dirs["/home/test/.codex/skills/remove-me"] = true

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		result := svc.Remove(service.RemoveOptions{Name: "remove-me"})

		if result.Error != nil {
			t.Fatalf("Remove() error = %v", result.Error)
		}

		if !result.StoreRemoved {
			t.Error("Remove() should have removed from store")
		}

		// Verify targets were also cleaned up
		for _, tr := range result.TargetResults {
			if !tr.Removed {
				t.Errorf("Remove() should have removed from target %s", tr.Target)
			}
		}

		if mock.Exists("/home/test/.claude/skills/remove-me") {
			t.Error("Remove() should have deleted skill from claude target")
		}
		if mock.Exists("/home/test/.codex/skills/remove-me") {
			t.Error("Remove() should have deleted skill from codex target")
		}
	})

	t.Run("remove non-existent skill", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		result := svc.Remove(service.RemoveOptions{Name: "does-not-exist"})

		if result.Error == nil {
			t.Error("Remove() should return error for non-existent skill")
		}
	})

	t.Run("remove with invalid name", func(t *testing.T) {
		mock, store, registry, cfg := setupTestEnv()

		svc := service.NewSkillService(mock, store, registry, cfg, "")
		result := svc.Remove(service.RemoveOptions{Name: "../invalid"})

		if result.Error == nil {
			t.Error("Remove() should return error for invalid name")
		}
	})
}

func TestFindSkillsToMigrate(t *testing.T) {
	setupMigrateEnv := func() (*adapters.MockFileSystem, *service.SkillService) {
		mock := adapters.NewMockFileSystem()
		mock.HomeDir = "/home/test"
		mock.Dirs["/home/test/.agents"] = true
		mock.Dirs["/home/test/.agents/skills"] = true
		mock.Dirs["/home/test/.claude"] = true
		mock.Dirs["/home/test/.claude/skills"] = true
		mock.Dirs["/home/test/.codex"] = true
		mock.Dirs["/home/test/.codex/skills"] = true

		cfg := service.DefaultConfig()
		store := adapters.NewSkillStore(mock, cfg, "")
		registry := adapters.NewRegistry(mock, "", cfg)
		svc := service.NewSkillService(mock, store, registry, cfg, "")

		return mock, svc
	}

	t.Run("finds skills in target directories", func(t *testing.T) {
		mock, svc := setupMigrateEnv()
		mock.Dirs["/home/test/.claude/skills/my-skill"] = true
		mock.Files["/home/test/.claude/skills/my-skill/SKILL.md"] = []byte("# My Skill")

		result := svc.FindSkillsToMigrate(service.MigrateOptions{Scope: service.ScopeGlobal})

		if len(result["claude"]) != 1 {
			t.Errorf("FindSkillsToMigrate() claude skills = %d, want 1", len(result["claude"]))
		}
	})

	t.Run("skips symlinks", func(t *testing.T) {
		mock, svc := setupMigrateEnv()
		mock.Symlinks["/home/test/.claude/skills/linked-skill"] = "/home/test/.agents/skills/linked-skill"

		result := svc.FindSkillsToMigrate(service.MigrateOptions{Scope: service.ScopeGlobal})

		if len(result["claude"]) != 0 {
			t.Errorf("FindSkillsToMigrate() should skip symlinks, got %d skills", len(result["claude"]))
		}
	})

	t.Run("returns empty when no skills exist", func(t *testing.T) {
		_, svc := setupMigrateEnv()
		result := svc.FindSkillsToMigrate(service.MigrateOptions{Scope: service.ScopeGlobal})

		total := 0
		for _, skills := range result {
			total += len(skills)
		}
		if total != 0 {
			t.Errorf("FindSkillsToMigrate() total skills = %d, want 0", total)
		}
	})
}

func TestSyncActionConstants(t *testing.T) {
	tests := []struct {
		action service.SyncAction
		want   string
	}{
		{service.SyncActionInstall, "install"},
		{service.SyncActionUpdate, "update"},
		{service.SyncActionUninstall, "uninstall"},
		{service.SyncActionSkip, "skip"},
		{service.SyncActionError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("SyncAction = %v, want %v", tt.action, tt.want)
			}
		})
	}
}

func TestMigrateActionConstants(t *testing.T) {
	tests := []struct {
		action service.MigrateAction
		want   string
	}{
		{service.MigrateActionMoved, "moved"},
		{service.MigrateActionSkipped, "skipped"},
		{service.MigrateActionRemoved, "removed"},
		{service.MigrateActionError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.action) != tt.want {
				t.Errorf("MigrateAction = %v, want %v", tt.action, tt.want)
			}
		})
	}
}

func TestNewMigrateResult(t *testing.T) {
	t.Run("creates result with nil found", func(t *testing.T) {
		result := service.NewMigrateResult(service.MigrateResultParams{})
		if result.Found == nil {
			t.Error("NewMigrateResult() Found should not be nil")
		}
	})

	t.Run("HasSkillsToMigrate returns true when found", func(t *testing.T) {
		result := service.NewMigrateResult(service.MigrateResultParams{
			Found: map[string][]string{"claude": {"skill-a"}},
		})
		if !result.HasSkillsToMigrate() {
			t.Error("HasSkillsToMigrate() should return true")
		}
	})

	t.Run("HasSkillsToMigrate returns false when empty", func(t *testing.T) {
		result := service.NewMigrateResult(service.MigrateResultParams{})
		if result.HasSkillsToMigrate() {
			t.Error("HasSkillsToMigrate() should return false")
		}
	})
}
