package config

import (
	"testing"

	"github.com/wwwyo/skillet/internal/fs"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Version != 1 {
		t.Errorf("Default() Version = %v, want 1", cfg.Version)
	}
	if cfg.GlobalPath != DefaultGlobalPath {
		t.Errorf("Default() GlobalPath = %v, want %v", cfg.GlobalPath, DefaultGlobalPath)
	}
	if cfg.DefaultStrategy != StrategySymlink {
		t.Errorf("Default() DefaultStrategy = %v, want symlink", cfg.DefaultStrategy)
	}
	if len(cfg.Targets) != 2 {
		t.Errorf("Default() has %d targets, want 2", len(cfg.Targets))
	}

	// Check Claude target
	claude, ok := cfg.Targets["claude"]
	if !ok {
		t.Error("Default() missing claude target")
	} else {
		if !claude.Enabled {
			t.Error("Default() claude target should be enabled")
		}
		if claude.GlobalPath != "~/.claude" {
			t.Errorf("Default() claude GlobalPath = %v, want ~/.claude", claude.GlobalPath)
		}
	}

	// Check Codex target
	codex, ok := cfg.Targets["codex"]
	if !ok {
		t.Error("Default() missing codex target")
	} else {
		if !codex.Enabled {
			t.Error("Default() codex target should be enabled")
		}
		if codex.GlobalPath != "~/.codex" {
			t.Errorf("Default() codex GlobalPath = %v, want ~/.codex", codex.GlobalPath)
		}
	}
}

func TestGlobalConfigPath(t *testing.T) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/user"

	path, err := GlobalConfigPath(mock)
	if err != nil {
		t.Fatalf("GlobalConfigPath() error = %v", err)
	}

	expected := "/home/user/.config/skillet/config.yaml"
	if path != expected {
		t.Errorf("GlobalConfigPath() = %v, want %v", path, expected)
	}
}

func TestConfigAgentsDir(t *testing.T) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/user"

	t.Run("default path", func(t *testing.T) {
		cfg := Default()
		path, err := cfg.AgentsDir(mock)
		if err != nil {
			t.Fatalf("AgentsDir() error = %v", err)
		}
		expected := "/home/user/.agents"
		if path != expected {
			t.Errorf("AgentsDir() = %v, want %v", path, expected)
		}
	})

	t.Run("custom path", func(t *testing.T) {
		cfg := Default()
		cfg.GlobalPath = "~/dotfiles/.agents"
		path, err := cfg.AgentsDir(mock)
		if err != nil {
			t.Fatalf("AgentsDir() error = %v", err)
		}
		expected := "/home/user/dotfiles/.agents"
		if path != expected {
			t.Errorf("AgentsDir() = %v, want %v", path, expected)
		}
	})
}

func TestConfigSkillsDir(t *testing.T) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/user"

	cfg := Default()

	tests := []struct {
		category string
		want     string
	}{
		{"", "/home/user/.agents/skills"},
		{"optional", "/home/user/.agents/skills/optional"},
	}

	for _, tt := range tests {
		name := tt.category
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			path, err := cfg.SkillsDir(mock, tt.category)
			if err != nil {
				t.Fatalf("SkillsDir() error = %v", err)
			}
			if path != tt.want {
				t.Errorf("SkillsDir() = %v, want %v", path, tt.want)
			}
		})
	}
}

func TestProjectAgentsDir(t *testing.T) {
	mock := fs.NewMock()
	path := ProjectAgentsDir("/project", mock)

	expected := "/project/.agents"
	if path != expected {
		t.Errorf("ProjectAgentsDir() = %v, want %v", path, expected)
	}
}

func TestProjectSkillsDir(t *testing.T) {
	mock := fs.NewMock()

	tests := []struct {
		category string
		want     string
	}{
		{"", "/project/.agents/skills"},
		{"optional", "/project/.agents/skills/optional"},
	}

	for _, tt := range tests {
		name := tt.category
		if name == "" {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			path := ProjectSkillsDir("/project", mock, tt.category)
			if path != tt.want {
				t.Errorf("ProjectSkillsDir() = %v, want %v", path, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("load valid config", func(t *testing.T) {
		mock := fs.NewMock()
		mock.Dirs["/home/test/.agents"] = true
		mock.Files["/home/test/.agents/skillet.yaml"] = []byte(`version: 1
defaultStrategy: symlink
targets:
  claude:
    enabled: true
    globalPath: ~/.claude
`)

		cfg, err := Load(mock, "/home/test/.agents/skillet.yaml")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Version != 1 {
			t.Errorf("Load() Version = %v, want 1", cfg.Version)
		}
		if cfg.DefaultStrategy != StrategySymlink {
			t.Errorf("Load() DefaultStrategy = %v, want symlink", cfg.DefaultStrategy)
		}
	})

	t.Run("load from default path", func(t *testing.T) {
		mock := fs.NewMock()
		mock.HomeDir = "/home/user"
		mock.Dirs["/home/user/.config/skillet"] = true
		mock.Files["/home/user/.config/skillet/config.yaml"] = []byte(`version: 1
defaultStrategy: copy
`)

		cfg, err := Load(mock, "")
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.DefaultStrategy != StrategyCopy {
			t.Errorf("Load() DefaultStrategy = %v, want copy", cfg.DefaultStrategy)
		}
	})

	t.Run("config file not found", func(t *testing.T) {
		mock := fs.NewMock()
		mock.HomeDir = "/home/user"

		_, err := Load(mock, "/nonexistent/config.yaml")
		if err == nil {
			t.Error("Load() expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		mock := fs.NewMock()
		mock.Files["/config.yaml"] = []byte(`version: [invalid yaml`)

		_, err := Load(mock, "/config.yaml")
		if err == nil {
			t.Error("Load() expected error for invalid YAML, got nil")
		}
	})
}

func TestConfigSave(t *testing.T) {
	t.Run("save config", func(t *testing.T) {
		mock := fs.NewMock()
		cfg := Default()
		cfg.fsys = mock
		cfg.path = "/home/test/.agents/skillet.yaml"

		err := cfg.Save()
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if !mock.Exists("/home/test/.agents/skillet.yaml") {
			t.Error("Save() did not create config file")
		}
	})

	t.Run("save without path", func(t *testing.T) {
		mock := fs.NewMock()
		cfg := Default()
		cfg.fsys = mock

		err := cfg.Save()
		if err == nil {
			t.Error("Save() expected error when path not set, got nil")
		}
	})
}

func TestConfigSaveTo(t *testing.T) {
	mock := fs.NewMock()
	cfg := Default()

	err := cfg.SaveTo(mock, "/custom/path/config.yaml")
	if err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	if !mock.Exists("/custom/path/config.yaml") {
		t.Error("SaveTo() did not create config file")
	}
}

func TestFindProjectRootFrom(t *testing.T) {
	t.Run("find project root", func(t *testing.T) {
		mock := fs.NewMock()
		mock.Dirs["/project/.agents"] = true
		mock.Dirs["/project/src"] = true
		mock.Dirs["/project/src/deep"] = true

		root, err := FindProjectRootFrom(mock, "/project/src/deep")
		if err != nil {
			t.Fatalf("FindProjectRootFrom() error = %v", err)
		}

		if root != "/project" {
			t.Errorf("FindProjectRootFrom() = %v, want /project", root)
		}
	})

	t.Run("find project root at current dir", func(t *testing.T) {
		mock := fs.NewMock()
		mock.Dirs["/project/.agents"] = true

		root, err := FindProjectRootFrom(mock, "/project")
		if err != nil {
			t.Fatalf("FindProjectRootFrom() error = %v", err)
		}

		if root != "/project" {
			t.Errorf("FindProjectRootFrom() = %v, want /project", root)
		}
	})

	t.Run("no project root found", func(t *testing.T) {
		mock := fs.NewMock()
		mock.Dirs["/some/directory"] = true

		_, err := FindProjectRootFrom(mock, "/some/directory")
		if err == nil {
			t.Error("FindProjectRootFrom() expected error when no project root, got nil")
		}
	})
}

func TestExpandPath(t *testing.T) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/user"

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{"expand tilde", "~/documents", "/home/user/documents", false},
		{"expand tilde only", "~", "/home/user", false},
		{"no expansion needed", "/absolute/path", "/absolute/path", false},
		{"relative path", "relative/path", "relative/path", false},
		{"empty path", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(mock, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategyConstants(t *testing.T) {
	if StrategySymlink != "symlink" {
		t.Errorf("StrategySymlink = %v, want symlink", StrategySymlink)
	}
	if StrategyCopy != "copy" {
		t.Errorf("StrategyCopy = %v, want copy", StrategyCopy)
	}
}

func TestDirectoryConstants(t *testing.T) {
	if AgentsDir != ".agents" {
		t.Errorf("AgentsDir = %v, want .agents", AgentsDir)
	}
	if SkillsDir != "skills" {
		t.Errorf("SkillsDir = %v, want skills", SkillsDir)
	}
	if OptionalDir != "optional" {
		t.Errorf("OptionalDir = %v, want optional", OptionalDir)
	}
}
