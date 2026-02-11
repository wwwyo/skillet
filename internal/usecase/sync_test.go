package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/usecase"
)

func setupSyncEnv() (*platformfs.MockFileSystem, *usecase.SyncService) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.agents/skills/optional"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := config.DefaultConfig()
	return mock, usecase.NewSyncService(mock, cfg, "")
}

func addGlobalSkill(m *platformfs.MockFileSystem, name string) {
	skillDir := "/home/test/.agents/skills/" + name
	m.Dirs[skillDir] = true
	m.Files[skillDir+"/SKILL.md"] = []byte("---\nname: " + name + "\n---\n")
}

func TestSyncInstall(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.agents/skills/optional"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	addGlobalSkill(mock, "test-skill")

	cfg := config.DefaultConfig()
	svc := usecase.NewSyncService(mock, cfg, "")

	results, err := svc.Sync(usecase.SyncOptions{})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	foundInstall := false
	for _, r := range results {
		if r.SkillName == "test-skill" && r.Action == usecase.SyncActionInstall {
			foundInstall = true
			break
		}
	}
	if !foundInstall {
		t.Fatal("Sync() did not return install action for test-skill")
	}
}

func TestSyncDryRun(t *testing.T) {
	mock, svc := setupSyncEnv()
	addGlobalSkill(mock, "dry-run-skill")

	results, err := svc.Sync(usecase.SyncOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if mock.Exists("/home/test/.claude/skills/dry-run-skill") {
		t.Fatal("Sync() with DryRun should not install skill")
	}

	foundInstall := false
	for _, r := range results {
		if r.SkillName == "dry-run-skill" && r.Action == usecase.SyncActionInstall {
			foundInstall = true
		}
	}
	if !foundInstall {
		t.Fatal("Sync() DryRun did not report install action")
	}
}

func TestSyncScopeFilter(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"
	cfg := config.DefaultConfig()

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/project/.agents"] = true
	mock.Dirs["/project/.agents/skills"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	addGlobalSkill(mock, "global-skill")
	projectSkillDir := "/project/.agents/skills/project-skill"
	mock.Dirs[projectSkillDir] = true
	mock.Files[projectSkillDir+"/SKILL.md"] = []byte("---\nname: project-skill\n---\n")

	svc := usecase.NewSyncService(mock, cfg, "/project")

	scope := skill.ScopeGlobal
	results, err := svc.Sync(usecase.SyncOptions{Scope: &scope})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	for _, r := range results {
		if r.SkillName == "project-skill" {
			t.Fatal("Sync() with global scope should not include project skill")
		}
	}
}
