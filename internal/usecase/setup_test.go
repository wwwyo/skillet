package usecase_test

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/usecase"
)

func TestSetupServiceSetupGlobalCreatesConfigAndDirs(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"

	svc := usecase.NewSetupService(mock)
	cfg, err := svc.SetupGlobal(usecase.SetupGlobalParams{
		GlobalPath: "~/.agents",
		EnabledTargets: map[string]bool{
			"claude": true,
			"codex":  false,
		},
		Strategy:   config.StrategyCopy,
		ConfigPath: "/home/test/.config/skillet/config.yaml",
	})
	if err != nil {
		t.Fatalf("SetupGlobal() error = %v", err)
	}

	if !mock.Exists("/home/test/.agents") {
		t.Fatal("expected global agents directory to be created")
	}
	if !mock.Exists("/home/test/.agents/skills") {
		t.Fatal("expected global skills directory to be created")
	}
	if !mock.Exists("/home/test/.agents/skills/optional") {
		t.Fatal("expected optional skills directory to be created")
	}
	if !mock.Exists("/home/test/.config/skillet/config.yaml") {
		t.Fatal("expected config file to be created")
	}
	if cfg.DefaultStrategy != config.StrategyCopy {
		t.Fatalf("DefaultStrategy = %v, want %v", cfg.DefaultStrategy, config.StrategyCopy)
	}
	if cfg.Targets["codex"].Enabled {
		t.Fatal("codex target should be disabled")
	}
}

func TestSetupServiceSetupGlobalUpdatesExistingConfig(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	mock.HomeDir = "/home/test"
	mock.Dirs["/home/test/.config/skillet"] = true
	mock.Files["/home/test/.config/skillet/config.yaml"] = []byte(`version: 1
globalPath: /existing/agents
defaultStrategy: symlink
targets:
  claude:
    enabled: true
    globalPath: ~/.claude
  codex:
    enabled: true
    globalPath: ~/.codex
`)

	svc := usecase.NewSetupService(mock)
	cfg, err := svc.SetupGlobal(usecase.SetupGlobalParams{
		GlobalPath: config.DefaultGlobalPath,
		EnabledTargets: map[string]bool{
			"claude": false,
			"codex":  true,
		},
		Strategy:   config.StrategyCopy,
		ConfigPath: "/home/test/.config/skillet/config.yaml",
	})
	if err != nil {
		t.Fatalf("SetupGlobal() error = %v", err)
	}

	if cfg.GlobalPath != "/existing/agents" {
		t.Fatalf("GlobalPath = %q, want %q", cfg.GlobalPath, "/existing/agents")
	}
	if cfg.DefaultStrategy != config.StrategyCopy {
		t.Fatalf("DefaultStrategy = %v, want %v", cfg.DefaultStrategy, config.StrategyCopy)
	}
	if cfg.Targets["claude"].Enabled {
		t.Fatal("claude target should be disabled")
	}
}

func TestSetupServiceSetupProjectCreatesDirs(t *testing.T) {
	mock := platformfs.NewMockFileSystem()
	svc := usecase.NewSetupService(mock)

	if err := svc.SetupProject("/project"); err != nil {
		t.Fatalf("SetupProject() error = %v", err)
	}

	if !mock.Exists("/project/.agents") {
		t.Fatal("expected project agents directory to be created")
	}
	if !mock.Exists("/project/.agents/skills") {
		t.Fatal("expected project skills directory to be created")
	}
	if !mock.Exists("/project/.agents/skills/optional") {
		t.Fatal("expected project optional skills directory to be created")
	}
}
