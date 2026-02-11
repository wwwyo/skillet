package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/usecase"
)

func TestNewTargetRegistryRespectsEnabledTargets(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	cfg := config.DefaultConfig()

	claude := cfg.Targets["claude"]
	claude.Enabled = false
	cfg.Targets["claude"] = claude

	registry := usecase.NewTargetRegistry(mock, "", cfg)

	if _, ok := registry.Get("claude"); ok {
		t.Fatal("claude should not be registered when disabled")
	}
	if _, ok := registry.Get("codex"); !ok {
		t.Fatal("codex should be registered")
	}
}

func TestTargetGetSkillsPathUsesConfigOverride(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	cfg := config.DefaultConfig()

	claude := cfg.Targets["claude"]
	claude.GlobalPath = "/opt/claude-custom"
	cfg.Targets["claude"] = claude

	registry := usecase.NewTargetRegistry(mock, "", cfg)
	target, ok := registry.Get("claude")
	if !ok {
		t.Fatal("claude target not found")
	}

	path, err := target.GetSkillsPath(skill.ScopeGlobal)
	if err != nil {
		t.Fatalf("GetSkillsPath() error = %v", err)
	}
	if path != "/opt/claude-custom/skills" {
		t.Fatalf("GetSkillsPath() = %q, want %q", path, "/opt/claude-custom/skills")
	}
}

func TestTargetGetSkillsPathProjectRequiresRoot(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	cfg := config.DefaultConfig()

	registry := usecase.NewTargetRegistry(mock, "", cfg)
	target, ok := registry.Get("claude")
	if !ok {
		t.Fatal("claude target not found")
	}

	_, err := target.GetSkillsPath(skill.ScopeProject)
	if err == nil {
		t.Fatal("expected error when project root is not set")
	}
}

func TestTargetInstallCopyAndUninstall(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"
	mock.Dirs["/home/test/.agents/skills/test-skill"] = true
	mock.Files["/home/test/.agents/skills/test-skill/SKILL.md"] = []byte("---\nname: test-skill\n---\n")

	cfg := config.DefaultConfig()
	registry := usecase.NewTargetRegistry(mock, "", cfg)
	target, ok := registry.Get("claude")
	if !ok {
		t.Fatal("claude target not found")
	}

	sk, err := skill.NewSkill(
		"test-skill",
		"desc",
		"/home/test/.agents/skills/test-skill",
		skill.ScopeGlobal,
		skill.CategoryDefault,
	)
	if err != nil {
		t.Fatalf("NewSkill() error = %v", err)
	}

	err = target.Install(sk, usecase.InstallOptions{Strategy: config.StrategyCopy})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if !mock.Exists("/home/test/.claude/skills/test-skill/SKILL.md") {
		t.Fatal("expected skill to be installed in target path")
	}

	if err := target.Uninstall("test-skill"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if mock.Exists("/home/test/.claude/skills/test-skill") {
		t.Fatal("expected skill to be removed from target path")
	}
}
