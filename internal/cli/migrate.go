package cli

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/sync"
	"github.com/wwwyo/skillet/internal/target"
)

var migrateYes bool

func newMigrateCmd(a *app) *cobra.Command {
	scopeFlags := NewScopeFlags(skill.ScopeProject)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate existing skills from targets to agents directory",
		Long: `Migrate existing skills from AI client directories to the central agents directory.

This command finds skills in target directories (e.g., .claude/skills/) that are not
symlinks, moves them to the agents directory, and creates links back to the targets.

Use --global or --project to specify which scope to migrate:
  --global  - Migrate from global targets (e.g., ~/.claude/skills/) to ~/.agents/
  --project - Migrate from project targets (e.g., .claude/skills/) to .agents/ (default)

Use this after setting up skillet to consolidate existing skills.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := scopeFlags.GetScope()
			if err != nil {
				return err
			}

			cfg, err := config.Load(a.fs, "")
			if err != nil {
				return fmt.Errorf("failed to load config: %w (run 'skillet init -g' first)", err)
			}

			projectRoot := ""
			if scope == skill.ScopeProject {
				projectRoot, err = config.FindProjectRoot(a.fs)
				if err != nil {
					return fmt.Errorf("failed to find project root: %w", err)
				}
			}

			return runMigrate(a, cfg, MigrateOptions{
				SkipPrompts:    migrateYes,
				DefaultConfirm: true, // Explicit command, default to yes
				Scope:          scope,
				ProjectRoot:    projectRoot,
			})
		},
	}

	cmd.Flags().BoolVarP(&migrateYes, "yes", "y", false, "Skip confirmation prompts")
	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// MigrateOptions contains options for migration.
type MigrateOptions struct {
	SkipPrompts    bool
	DefaultConfirm bool // Default value for confirmation prompt
	Scope          skill.Scope
	ProjectRoot    string
}

// runMigrate executes the migration logic. Exported for use by init command.
func runMigrate(a *app, cfg *config.Config, opts MigrateOptions) error {
	var agentsDir string
	var err error

	if opts.Scope == skill.ScopeProject {
		agentsDir = config.ProjectAgentsDir(opts.ProjectRoot, a.fs)
	} else {
		agentsDir, err = cfg.AgentsDir(a.fs)
		if err != nil {
			return err
		}
	}

	// Find existing skills in enabled targets
	existingSkills := findExistingSkills(a, cfg, opts.Scope, opts.ProjectRoot)
	if len(existingSkills) == 0 {
		fmt.Println("No skills to migrate.")
		return nil
	}

	// Show what was found
	fmt.Println("\nFound existing skills:")
	for targetName, skills := range existingSkills {
		for _, skillName := range skills {
			fmt.Printf("  %s: %s\n", targetName, skillName)
		}
	}

	// Ask if user wants to migrate
	if !opts.SkipPrompts {
		var migrate bool
		prompt := &survey.Confirm{
			Message: "Migrate existing skills to agents directory?",
			Default: opts.DefaultConfirm,
		}
		if err := survey.AskOne(prompt, &migrate); err != nil {
			return nil
		}
		if !migrate {
			return nil
		}
	}

	// Move skills to agents directory
	if err := moveSkillsToAgents(a, cfg, agentsDir, existingSkills, opts.Scope, opts.ProjectRoot); err != nil {
		return err
	}

	// Use sync engine to create links back to targets
	store := skill.NewStore(a.fs, cfg, opts.ProjectRoot)
	registry := target.NewRegistry(a.fs, opts.ProjectRoot, cfg)
	engine := sync.NewEngine(a.fs, store, registry, cfg, opts.ProjectRoot)

	results, err := engine.Sync(sync.SyncOptions{Force: true})
	if err != nil {
		return fmt.Errorf("failed to sync skills: %w", err)
	}

	// Report results
	fmt.Println("\nSynced to targets:")
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("  ⚠ %s → %s: %v\n", r.SkillName, r.Target, r.Error)
		} else if r.Action == sync.ActionInstall || r.Action == sync.ActionUpdate {
			fmt.Printf("  ✓ %s → %s\n", r.SkillName, r.Target)
		}
	}

	return nil
}

func findExistingSkills(a *app, cfg *config.Config, scope skill.Scope, projectRoot string) map[string][]string {
	result := make(map[string][]string)

	for name, targetCfg := range cfg.Targets {
		if !targetCfg.Enabled {
			continue
		}

		var targetSkillsDir string
		var err error
		if scope == skill.ScopeProject {
			// Project scope: .claude/skills/, .codex/skills/ etc.
			// Derive project path from target name (e.g., "claude" -> ".claude")
			targetSkillsDir = a.fs.Join(projectRoot, "."+name, "skills")
		} else {
			// Global scope: ~/.claude/skills/, ~/.codex/skills/ etc.
			targetSkillsDir, err = config.ExpandPath(a.fs, targetCfg.GlobalPath+"/skills")
			if err != nil {
				continue
			}
		}

		if !a.fs.Exists(targetSkillsDir) || !a.fs.IsDir(targetSkillsDir) {
			continue
		}

		entries, err := a.fs.ReadDir(targetSkillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// Skip symlinks (already managed by skillet)
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			if entry.IsDir() {
				skillName := entry.Name()
				if err := skill.ValidateName(skillName); err != nil {
					continue
				}
				skillDir := a.fs.Join(targetSkillsDir, skillName)
				// Only include valid skills (directories containing SKILL.md somewhere)
				if hasSkillFile(a.fs, skillDir) {
					result[name] = append(result[name], skillName)
				}
			}
		}
	}

	return result
}

// hasSkillFile checks if a directory contains SKILL.md anywhere in its tree.
func hasSkillFile(fsys fs.System, dir string) bool {
	// Check current directory
	if fsys.Exists(fsys.Join(dir, "SKILL.md")) {
		return true
	}

	// Check subdirectories recursively
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if hasSkillFile(fsys, fsys.Join(dir, entry.Name())) {
				return true
			}
		}
	}

	return false
}

func moveSkillsToAgents(a *app, cfg *config.Config, agentsDir string, existingSkills map[string][]string, scope skill.Scope, projectRoot string) error {
	skillsDir := a.fs.Join(agentsDir, config.SkillsDir)
	moved := make(map[string]bool)

	for targetName, skills := range existingSkills {
		targetCfg, ok := cfg.Targets[targetName]
		if !ok {
			continue
		}

		var targetSkillsDir string
		if scope == skill.ScopeProject {
			// Project scope: .claude/skills/, .codex/skills/ etc.
			targetSkillsDir = a.fs.Join(projectRoot, "."+targetName, "skills")
		} else {
			// Global scope: ~/.claude/skills/, ~/.codex/skills/ etc.
			targetPath, err := config.ExpandPath(a.fs, targetCfg.GlobalPath)
			if err != nil {
				return err
			}
			targetSkillsDir = a.fs.Join(targetPath, "skills")
		}

		for _, skillName := range skills {
			// Skip if already moved from another target
			if moved[skillName] {
				// Just remove the duplicate from target
				srcPath := a.fs.Join(targetSkillsDir, skillName)
				if err := a.fs.RemoveAll(srcPath); err != nil {
					fmt.Printf("  ⚠ Failed to remove duplicate %s from %s: %v\n", skillName, targetName, err)
				}
				continue
			}

			srcPath := a.fs.Join(targetSkillsDir, skillName)
			dstPath := a.fs.Join(skillsDir, skillName)

			// Check if destination already exists
			if a.fs.Exists(dstPath) {
				fmt.Printf("  • Skipping %s (already exists in agents)\n", skillName)
				// Remove from target since it's already in agents
				if err := a.fs.RemoveAll(srcPath); err != nil {
					fmt.Printf("  ⚠ Failed to remove %s from %s: %v\n", skillName, targetName, err)
				}
				continue
			}

			// Move skill to agents directory
			if err := a.fs.Rename(srcPath, dstPath); err != nil {
				fmt.Printf("  ⚠ Failed to move %s: %v\n", skillName, err)
				continue
			}

			moved[skillName] = true
			fmt.Printf("  ✓ Moved %s to agents\n", skillName)
		}
	}

	return nil
}
