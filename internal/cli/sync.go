package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/sync"
	"github.com/wwwyo/skillet/internal/target"
)

// newSyncCmd creates the sync command.
func newSyncCmd(a *app) *cobra.Command {
	var dryRun bool
	var force bool
	var targetName string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize skills to targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := config.FindProjectRoot(a.fs)
			if err != nil {
				projectRoot = ""
			}

			store := skill.NewStore(a.fs, a.config, projectRoot)
			registry := target.NewRegistry(a.fs, projectRoot, a.config)
			engine := sync.NewEngine(a.fs, store, registry, a.config, projectRoot)

			results, err := engine.Sync(sync.SyncOptions{
				TargetName: targetName,
				DryRun:     dryRun,
				Force:      force,
			})
			if err != nil {
				return err
			}

			for _, r := range results {
				if r.Error != nil {
					fmt.Printf("⚠ %s → %s: %v\n", r.SkillName, r.Target, r.Error)
				} else {
					fmt.Printf("✓ %s → %s (%s)\n", r.SkillName, r.Target, r.Action)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().BoolVar(&force, "force", false, "Force update even if already installed")
	cmd.Flags().StringVar(&targetName, "target", "", "Sync to specific target only")

	return cmd
}
