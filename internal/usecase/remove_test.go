package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/usecase"
)

func TestRemoveExistingSkill(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.agents/skills/remove-me"] = true
	mock.Files["/home/test/.agents/skills/remove-me/SKILL.md"] = []byte("---\nname: remove-me\n---\n")

	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.claude/skills/remove-me"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true
	mock.Dirs["/home/test/.codex/skills/remove-me"] = true

	cfg := config.DefaultConfig()
	svc := usecase.NewRemoveService(mock, cfg, "")

	result := svc.Remove(usecase.RemoveOptions{Name: "remove-me"})
	if result.Error != nil {
		t.Fatalf("Remove() error = %v", result.Error)
	}
	if !result.StoreRemoved {
		t.Fatal("Remove() should remove from store")
	}
	if mock.Exists("/home/test/.agents/skills/remove-me") {
		t.Fatal("skill should be removed from store")
	}
}
