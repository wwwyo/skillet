package cli

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/usecase"
)

// newSyncCmd creates the sync command.
func newSyncCmd(a *app) *cobra.Command {
	var (
		dryRun bool
		force  bool
	)
	scopeFlags := NewScopeFlags(skill.ScopeProject)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize skills to targets",
		Long: `Synchronize skills from the skill store to AI agent targets.

By default, syncs all skills to all enabled targets.
Use --global or --project to sync only skills from a specific scope.
Use --dry-run to see what would be done without making changes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, rootErr := a.findProjectRoot()
			if rootErr != nil {
				root = ""
			}
			if scopeFlags.Project && rootErr != nil {
				return fmt.Errorf("not in a project directory")
			}
			svc := usecase.NewSyncService(a.fs, a.config, root)

			opts := usecase.SyncOptions{
				DryRun: dryRun,
				Force:  force,
			}

			if scopeFlags.IsSet() {
				scope, err := scopeFlags.GetScope()
				if err != nil {
					return err
				}
				opts.Scope = &scope
			}

			results, err := svc.Sync(opts)
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			if dryRun {
				fmt.Println("Dry run - no changes made:")
			}

			// Group results by target.
			byTarget := make(map[string][]usecase.SyncResult)
			for _, r := range results {
				byTarget[r.Target] = append(byTarget[r.Target], r)
			}

			targetNames := make([]string, 0, len(byTarget))
			for name := range byTarget {
				targetNames = append(targetNames, name)
			}
			slices.Sort(targetNames)

			for _, tName := range targetNames {
				targetResults := byTarget[tName]
				fmt.Printf("\nTarget: %s\n", tName)

				var installs, updates, uninstalls, skips, errors int

				for _, r := range targetResults {
					switch r.Action {
					case usecase.SyncActionInstall:
						fmt.Printf("  + %s (install)\n", r.SkillName)
						installs++
					case usecase.SyncActionUpdate:
						fmt.Printf("  ~ %s (update)\n", r.SkillName)
						updates++
					case usecase.SyncActionUninstall:
						fmt.Printf("  - %s (uninstall)\n", r.SkillName)
						uninstalls++
					case usecase.SyncActionSkip:
						skips++
					case usecase.SyncActionError:
						fmt.Printf("  ! %s (error: %v)\n", r.SkillName, r.Error)
						errors++
					}
				}

				summary := []string{}
				if installs > 0 {
					summary = append(summary, fmt.Sprintf("%d installed", installs))
				}
				if updates > 0 {
					summary = append(summary, fmt.Sprintf("%d updated", updates))
				}
				if uninstalls > 0 {
					summary = append(summary, fmt.Sprintf("%d uninstalled", uninstalls))
				}
				if skips > 0 {
					summary = append(summary, fmt.Sprintf("%d skipped", skips))
				}
				if errors > 0 {
					summary = append(summary, fmt.Sprintf("%d errors", errors))
				}

				if len(summary) > 0 {
					fmt.Printf("  Summary: ")
					for i, s := range summary {
						if i > 0 {
							fmt.Print(", ")
						}
						fmt.Print(s)
					}
					fmt.Println()
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already installed")
	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}
