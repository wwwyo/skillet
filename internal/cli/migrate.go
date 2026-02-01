package cli

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/orchestrator"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

func newMigrateCmd(a *app) *cobra.Command {
	var skipPrompts bool
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

			return runMigrate(a, cfg, migrateRunOptions{
				skipPrompts:    skipPrompts,
				defaultConfirm: true,
				scope:          scope,
				projectRoot:    projectRoot,
			})
		},
	}

	cmd.Flags().BoolVarP(&skipPrompts, "yes", "y", false, "Skip confirmation prompts")
	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// migrateRunOptions contains CLI-specific options for migration.
type migrateRunOptions struct {
	skipPrompts    bool
	defaultConfirm bool
	scope          skill.Scope
	projectRoot    string
}

// runMigrate executes the migration logic. Exported for use by init command.
func runMigrate(a *app, cfg *config.Config, opts migrateRunOptions) error {
	store := skill.NewStore(a.fs, cfg, opts.projectRoot)
	registry := target.NewRegistry(a.fs, opts.projectRoot, cfg)
	orch := orchestrator.New(a.fs, store, registry, cfg, opts.projectRoot)

	migrateOpts := orchestrator.MigrateOptions{
		Scope:       opts.scope,
		ProjectRoot: opts.projectRoot,
	}

	// Find existing skills
	existingSkills := orch.FindSkillsToMigrate(migrateOpts)
	if len(existingSkills) == 0 {
		fmt.Println("No skills to migrate.")
		return nil
	}

	// Show what was found
	printFoundSkills(existingSkills)

	// Ask for confirmation
	if !opts.skipPrompts {
		confirmed, err := promptMigrateConfirmation(opts.defaultConfirm)
		if err != nil || !confirmed {
			return nil
		}
	}

	// Execute migration
	result, err := orch.Migrate(migrateOpts, existingSkills)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print results
	printMoveResults(result.MoveResults)
	printMigrateSyncResults(result.SyncResults)

	return nil
}

// printFoundSkills prints the skills found for migration.
func printFoundSkills(found map[string][]string) {
	fmt.Println("\nFound existing skills:")
	for targetName, skills := range found {
		for _, skillName := range skills {
			fmt.Printf("  %s: %s\n", targetName, skillName)
		}
	}
}

// promptMigrateConfirmation asks the user to confirm migration.
func promptMigrateConfirmation(defaultValue bool) (bool, error) {
	var confirmed bool
	prompt := &survey.Confirm{
		Message: "Migrate existing skills to agents directory?",
		Default: defaultValue,
	}
	if err := survey.AskOne(prompt, &confirmed); err != nil {
		return false, err
	}
	return confirmed, nil
}

// printMoveResults prints the results of moving skills.
func printMoveResults(results []orchestrator.MoveResult) {
	if len(results) == 0 {
		return
	}

	for _, r := range results {
		switch r.Action {
		case orchestrator.MigrateActionMoved:
			fmt.Printf("  ✓ Moved %s to agents\n", r.SkillName)
		case orchestrator.MigrateActionSkipped:
			fmt.Printf("  • Skipping %s (%s)\n", r.SkillName, r.Message)
		case orchestrator.MigrateActionRemoved:
			// Silent for duplicates
		case orchestrator.MigrateActionError:
			fmt.Printf("  ⚠ Failed to process %s: %v\n", r.SkillName, r.Error)
		}
	}
}

// printMigrateSyncResults prints the sync results after migration.
func printMigrateSyncResults(results []orchestrator.SyncResult) {
	fmt.Println("\nSynced to targets:")
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("  ⚠ %s → %s: %v\n", r.SkillName, r.Target, r.Error)
		} else if r.Action == orchestrator.SyncActionInstall || r.Action == orchestrator.SyncActionUpdate {
			fmt.Printf("  ✓ %s → %s\n", r.SkillName, r.Target)
		}
	}
}
