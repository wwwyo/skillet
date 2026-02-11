package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/usecase"
)

func setupMigrateEnv() (*platformfs.MockFileSystem, *usecase.MigrateService) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := config.DefaultConfig()
	syncSvc := usecase.NewSyncService(mock, cfg, "")

	return mock, usecase.NewMigrateService(mock, cfg, "", syncSvc)
}

func TestFindSkillsToMigrateSkipsSymlink(t *testing.T) {
	mock, svc := setupMigrateEnv()
	mock.Symlinks["/home/test/.claude/skills/linked"] = "/home/test/.agents/skills/linked"

	found := svc.FindSkillsToMigrate(usecase.MigrateOptions{Scope: skill.ScopeGlobal})
	if len(found["claude"]) != 0 {
		t.Fatalf("expected no migratable skills for symlink, got %d", len(found["claude"]))
	}
}

func TestFindSkillsToMigrateFindsDirectorySkill(t *testing.T) {
	mock, svc := setupMigrateEnv()
	mock.Dirs["/home/test/.claude/skills/my-skill"] = true
	mock.Files["/home/test/.claude/skills/my-skill/SKILL.md"] = []byte("# my skill")

	found := svc.FindSkillsToMigrate(usecase.MigrateOptions{Scope: skill.ScopeGlobal})
	if len(found["claude"]) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(found["claude"]))
	}
}
