package orchestrator

import (
	"testing"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

func setupMigrateTestEnv() (*fs.MockSystem, *Orchestrator) {
	mock := fs.NewMock()
	mock.HomeDir = "/home/test"

	// Setup global agents directory
	mock.Dirs["/home/test/.agents"] = true
	mock.Dirs["/home/test/.agents/skills"] = true

	// Setup target directories
	mock.Dirs["/home/test/.claude"] = true
	mock.Dirs["/home/test/.claude/skills"] = true
	mock.Dirs["/home/test/.codex"] = true
	mock.Dirs["/home/test/.codex/skills"] = true

	cfg := config.Default()
	store := skill.NewStore(mock, cfg, "")
	registry := target.NewRegistry(mock, "", cfg)
	orch := New(mock, store, registry, cfg, "")

	return mock, orch
}

func TestFindSkillsToMigrate(t *testing.T) {
	t.Run("finds skills in target directories", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add a skill to claude target
		mock.Dirs["/home/test/.claude/skills/my-skill"] = true
		mock.Files["/home/test/.claude/skills/my-skill/SKILL.md"] = []byte("# My Skill")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 1 {
			t.Errorf("FindSkillsToMigrate() claude skills = %d, want 1", len(result["claude"]))
		}
		if result["claude"][0] != "my-skill" {
			t.Errorf("FindSkillsToMigrate() claude skill = %s, want my-skill", result["claude"][0])
		}
	})

	t.Run("skips symlinks", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add a symlink (already managed by skillet)
		mock.Symlinks["/home/test/.claude/skills/linked-skill"] = "/home/test/.agents/skills/linked-skill"

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 0 {
			t.Errorf("FindSkillsToMigrate() should skip symlinks, got %d skills", len(result["claude"]))
		}
	})

	t.Run("finds skills in multiple targets", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add skills to both targets (with SKILL.md)
		mock.Dirs["/home/test/.claude/skills/skill-a"] = true
		mock.Files["/home/test/.claude/skills/skill-a/SKILL.md"] = []byte("---\nname: skill-a\n---")
		mock.Dirs["/home/test/.codex/skills/skill-b"] = true
		mock.Files["/home/test/.codex/skills/skill-b/SKILL.md"] = []byte("---\nname: skill-b\n---")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 1 {
			t.Errorf("FindSkillsToMigrate() claude skills = %d, want 1", len(result["claude"]))
		}
		if len(result["codex"]) != 1 {
			t.Errorf("FindSkillsToMigrate() codex skills = %d, want 1", len(result["codex"]))
		}
	})

	t.Run("returns empty when no skills exist", func(t *testing.T) {
		_, orch := setupMigrateTestEnv()

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		total := 0
		for _, skills := range result {
			total += len(skills)
		}
		if total != 0 {
			t.Errorf("FindSkillsToMigrate() total skills = %d, want 0", total)
		}
	})

	t.Run("finds skills with nested SKILL.md", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add a skill with nested SKILL.md (e.g., skill-a/.system/commands/SKILL.md)
		mock.Dirs["/home/test/.claude/skills/skill-a"] = true
		mock.Dirs["/home/test/.claude/skills/skill-a/.system"] = true
		mock.Dirs["/home/test/.claude/skills/skill-a/.system/commands"] = true
		mock.Files["/home/test/.claude/skills/skill-a/.system/commands/SKILL.md"] = []byte("---\nname: commands\n---")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 1 {
			t.Errorf("FindSkillsToMigrate() claude skills = %d, want 1", len(result["claude"]))
		}
		if result["claude"][0] != "skill-a" {
			t.Errorf("FindSkillsToMigrate() skill name = %s, want skill-a", result["claude"][0])
		}
	})

	t.Run("skips dot-start top-level directories", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		mock.Dirs["/home/test/.claude/skills/.system"] = true
		mock.Dirs["/home/test/.claude/skills/.system/commands"] = true
		mock.Files["/home/test/.claude/skills/.system/commands/SKILL.md"] = []byte("---\nname: commands\n---")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 0 {
			t.Errorf("FindSkillsToMigrate() should skip dot-start directories, got %d", len(result["claude"]))
		}
	})

	t.Run("skips directories without SKILL.md", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add a directory without SKILL.md
		mock.Dirs["/home/test/.claude/skills/not-a-skill"] = true
		mock.Files["/home/test/.claude/skills/not-a-skill/README.md"] = []byte("# Not a skill")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 0 {
			t.Errorf("FindSkillsToMigrate() should skip directories without SKILL.md, got %d", len(result["claude"]))
		}
	})

	t.Run("skips disabled targets", func(t *testing.T) {
		mock := fs.NewMock()
		mock.HomeDir = "/home/test"

		mock.Dirs["/home/test/.agents"] = true
		mock.Dirs["/home/test/.agents/skills"] = true
		mock.Dirs["/home/test/.claude"] = true
		mock.Dirs["/home/test/.claude/skills"] = true
		mock.Dirs["/home/test/.codex"] = true
		mock.Dirs["/home/test/.codex/skills"] = true

		cfg := config.Default()
		// Disable codex target
		codex := cfg.Targets["codex"]
		codex.Enabled = false
		cfg.Targets["codex"] = codex

		store := skill.NewStore(mock, cfg, "")
		registry := target.NewRegistry(mock, "", cfg)
		orch := New(mock, store, registry, cfg, "")

		// Add skills to both targets (with SKILL.md)
		mock.Dirs["/home/test/.claude/skills/skill-a"] = true
		mock.Files["/home/test/.claude/skills/skill-a/SKILL.md"] = []byte("---\nname: skill-a\n---")
		mock.Dirs["/home/test/.codex/skills/skill-b"] = true
		mock.Files["/home/test/.codex/skills/skill-b/SKILL.md"] = []byte("---\nname: skill-b\n---")

		result := orch.FindSkillsToMigrate(MigrateOptions{Scope: skill.ScopeGlobal})

		if len(result["claude"]) != 1 {
			t.Errorf("FindSkillsToMigrate() claude skills = %d, want 1", len(result["claude"]))
		}
		if len(result["codex"]) != 0 {
			t.Errorf("FindSkillsToMigrate() codex skills = %d, want 0 (disabled)", len(result["codex"]))
		}
	})
}

func TestMoveSkillsToAgents(t *testing.T) {
	t.Run("moves skill to agents directory", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Add a skill to claude target
		mock.Dirs["/home/test/.claude/skills/my-skill"] = true
		mock.Files["/home/test/.claude/skills/my-skill/SKILL.md"] = []byte("# My Skill")

		existingSkills := map[string][]string{
			"claude": {"my-skill"},
		}

		opts := MigrateOptions{Scope: skill.ScopeGlobal}
		results := orch.moveSkillsToAgents("/home/test/.agents", existingSkills, opts)

		// Check results
		var movedCount int
		for _, r := range results {
			if r.Action == MigrateActionMoved {
				movedCount++
			}
		}
		if movedCount != 1 {
			t.Errorf("moveSkillsToAgents() moved = %d, want 1", movedCount)
		}

		// Check skill was moved to agents
		if !mock.Exists("/home/test/.agents/skills/my-skill") {
			t.Error("moveSkillsToAgents() skill not moved to agents directory")
		}

		// Check skill was removed from target
		if mock.Exists("/home/test/.claude/skills/my-skill") {
			t.Error("moveSkillsToAgents() skill should be removed from target")
		}
	})

	t.Run("skips if already exists in agents", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Skill already exists in agents
		mock.Dirs["/home/test/.agents/skills/existing-skill"] = true
		mock.Files["/home/test/.agents/skills/existing-skill/SKILL.md"] = []byte("# Existing")

		// Same skill in target
		mock.Dirs["/home/test/.claude/skills/existing-skill"] = true
		mock.Files["/home/test/.claude/skills/existing-skill/SKILL.md"] = []byte("# From Target")

		existingSkills := map[string][]string{
			"claude": {"existing-skill"},
		}

		opts := MigrateOptions{Scope: skill.ScopeGlobal}
		results := orch.moveSkillsToAgents("/home/test/.agents", existingSkills, opts)

		// Check results
		var skippedCount int
		for _, r := range results {
			if r.Action == MigrateActionSkipped {
				skippedCount++
			}
		}
		if skippedCount != 1 {
			t.Errorf("moveSkillsToAgents() skipped = %d, want 1", skippedCount)
		}

		// Check agents skill content unchanged
		content, _ := mock.ReadFile("/home/test/.agents/skills/existing-skill/SKILL.md")
		if string(content) != "# Existing" {
			t.Error("moveSkillsToAgents() should not overwrite existing skill in agents")
		}
	})

	t.Run("handles duplicate skills from multiple targets", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Same skill name in both targets
		mock.Dirs["/home/test/.claude/skills/shared-skill"] = true
		mock.Files["/home/test/.claude/skills/shared-skill/SKILL.md"] = []byte("# Claude Version")

		mock.Dirs["/home/test/.codex/skills/shared-skill"] = true
		mock.Files["/home/test/.codex/skills/shared-skill/SKILL.md"] = []byte("# Codex Version")

		existingSkills := map[string][]string{
			"claude": {"shared-skill"},
			"codex":  {"shared-skill"},
		}

		opts := MigrateOptions{Scope: skill.ScopeGlobal}
		_ = orch.moveSkillsToAgents("/home/test/.agents", existingSkills, opts)

		// Check skill exists in agents (first one wins)
		if !mock.Exists("/home/test/.agents/skills/shared-skill") {
			t.Error("moveSkillsToAgents() skill not moved to agents")
		}

		// Check both targets had their copies removed
		if mock.Exists("/home/test/.claude/skills/shared-skill") {
			t.Error("moveSkillsToAgents() claude skill should be removed")
		}
		if mock.Exists("/home/test/.codex/skills/shared-skill") {
			t.Error("moveSkillsToAgents() codex skill should be removed")
		}
	})

	t.Run("moves multiple different skills", func(t *testing.T) {
		mock, orch := setupMigrateTestEnv()

		// Different skills in different targets
		mock.Dirs["/home/test/.claude/skills/skill-a"] = true
		mock.Dirs["/home/test/.codex/skills/skill-b"] = true

		existingSkills := map[string][]string{
			"claude": {"skill-a"},
			"codex":  {"skill-b"},
		}

		opts := MigrateOptions{Scope: skill.ScopeGlobal}
		_ = orch.moveSkillsToAgents("/home/test/.agents", existingSkills, opts)

		// Check both skills moved to agents
		if !mock.Exists("/home/test/.agents/skills/skill-a") {
			t.Error("moveSkillsToAgents() skill-a not moved to agents")
		}
		if !mock.Exists("/home/test/.agents/skills/skill-b") {
			t.Error("moveSkillsToAgents() skill-b not moved to agents")
		}
	})
}

func TestMigrateActionConstants(t *testing.T) {
	tests := []struct {
		action MigrateAction
		want   string
	}{
		{MigrateActionMoved, "moved"},
		{MigrateActionSkipped, "skipped"},
		{MigrateActionRemoved, "removed"},
		{MigrateActionError, "error"},
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
		result := NewMigrateResult(MigrateResultParams{})

		if result.Found == nil {
			t.Error("NewMigrateResult() Found should not be nil")
		}
	})

	t.Run("HasSkillsToMigrate returns true when found", func(t *testing.T) {
		result := NewMigrateResult(MigrateResultParams{
			Found: map[string][]string{"claude": {"skill-a"}},
		})

		if !result.HasSkillsToMigrate() {
			t.Error("HasSkillsToMigrate() should return true")
		}
	})

	t.Run("HasSkillsToMigrate returns false when empty", func(t *testing.T) {
		result := NewMigrateResult(MigrateResultParams{})

		if result.HasSkillsToMigrate() {
			t.Error("HasSkillsToMigrate() should return false")
		}
	})
}
