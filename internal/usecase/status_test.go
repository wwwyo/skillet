package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/usecase"
)

func setupStatusEnv() (*platformfs.MockFileSystem, *usecase.StatusService) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := config.DefaultConfig()
	return mock, usecase.NewStatusService(mock, cfg, "")
}

func TestGetStatusMissing(t *testing.T) {
	mock, svc := setupStatusEnv()
	mock.Dirs["/home/test/.agents/skills/missing-skill"] = true
	mock.Files["/home/test/.agents/skills/missing-skill/SKILL.md"] = []byte("---\nname: missing-skill\n---\n")

	statuses, err := svc.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	for _, s := range statuses {
		if s.InSync {
			t.Fatalf("target %s should be out of sync", s.Target)
		}
	}
}

func TestGetStatusExtra(t *testing.T) {
	mock, svc := setupStatusEnv()
	mock.Dirs["/home/test/.claude/skills/extra-skill"] = true

	statuses, err := svc.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Target == "claude" {
			if len(s.Extra) == 0 {
				t.Fatal("claude should report extra skill")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("claude target not found")
	}
}
