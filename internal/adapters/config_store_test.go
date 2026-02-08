package adapters

import (
	"testing"

	"github.com/wwwyo/skillet/internal/service"
)

func TestConfigStoreLoad(t *testing.T) {
	t.Run("load valid config", func(t *testing.T) {
		mock := NewMockFileSystem()
		mock.Dirs["/home/test/.agents"] = true
		mock.Files["/home/test/.agents/skillet.yaml"] = []byte(`version: 1
defaultStrategy: symlink
targets:
  claude:
    enabled: true
    globalPath: ~/.claude
`)

		cs := NewConfigStore(mock)
		cfg, err := cs.Load("/home/test/.agents/skillet.yaml")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Version != 1 {
			t.Errorf("Load() Version = %v, want 1", cfg.Version)
		}
		if cfg.DefaultStrategy != service.StrategySymlink {
			t.Errorf("Load() DefaultStrategy = %v, want symlink", cfg.DefaultStrategy)
		}
	})

	t.Run("load from default path", func(t *testing.T) {
		mock := NewMockFileSystem()
		mock.HomeDir = "/home/user"
		mock.Dirs["/home/user/.config/skillet"] = true
		mock.Files["/home/user/.config/skillet/config.yaml"] = []byte(`version: 1
defaultStrategy: copy
`)

		cs := NewConfigStore(mock)
		cfg, err := cs.Load("")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.DefaultStrategy != service.StrategyCopy {
			t.Errorf("Load() DefaultStrategy = %v, want copy", cfg.DefaultStrategy)
		}
	})

	t.Run("config file not found", func(t *testing.T) {
		mock := NewMockFileSystem()
		cs := NewConfigStore(mock)
		_, err := cs.Load("/nonexistent/config.yaml")
		if err == nil {
			t.Error("Load() expected error for nonexistent file, got nil")
		}
	})
}

func TestConfigStoreSave(t *testing.T) {
	mock := NewMockFileSystem()
	cs := NewConfigStore(mock)
	cfg := service.DefaultConfig()

	err := cs.Save(cfg, "/home/test/.agents/skillet.yaml")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !mock.Exists("/home/test/.agents/skillet.yaml") {
		t.Error("Save() did not create config file")
	}
}

func TestConfigStoreFindProjectRootFrom(t *testing.T) {
	t.Run("find project root", func(t *testing.T) {
		mock := NewMockFileSystem()
		mock.Dirs["/project/.agents"] = true
		mock.Dirs["/project/src"] = true
		mock.Dirs["/project/src/deep"] = true

		cs := NewConfigStore(mock)
		root, err := cs.FindProjectRootFrom("/project/src/deep")
		if err != nil {
			t.Fatalf("FindProjectRootFrom() error = %v", err)
		}

		if root != "/project" {
			t.Errorf("FindProjectRootFrom() = %v, want /project", root)
		}
	})

	t.Run("no project root found", func(t *testing.T) {
		mock := NewMockFileSystem()
		mock.Dirs["/some/directory"] = true

		cs := NewConfigStore(mock)
		_, err := cs.FindProjectRootFrom("/some/directory")
		if err == nil {
			t.Error("FindProjectRootFrom() expected error when no project root, got nil")
		}
	})
}
