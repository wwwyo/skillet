package cli

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/orchestrator"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/target"
)

const statusSeparator = "----------------------------------------"

// newStatusCmd creates the status command.
func newStatusCmd(a *app) *cobra.Command {
	scopeFlags := NewScopeFlags(skill.ScopeProject)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show synchronization status",
		Long: `Show the synchronization status between the skill store and targets.

Displays which skills are installed, missing, or extra for each target.
By default, shows status for all scopes. Use --global or --project to filter.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find project root
			projectRoot, err := config.FindProjectRoot(a.fs)
			if err != nil {
				projectRoot = ""
			}

			store := skill.NewStore(a.fs, a.config, projectRoot)
			registry := target.NewRegistry(a.fs, projectRoot, a.config)
			orch := orchestrator.New(a.fs, store, registry, a.config, projectRoot)

			// Build options
			var opts orchestrator.GetStatusOptions
			if scopeFlags.IsSet() {
				scope, err := scopeFlags.GetScope()
				if err != nil {
					return err
				}
				opts.Scope = &scope
			}

			statuses, err := orch.GetStatus(opts)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			// Sort for consistent output
			slices.SortFunc(statuses, func(a, b *orchestrator.Status) int {
				return cmp.Compare(a.Target, b.Target)
			})

			// Print each status
			for _, status := range statuses {
				printTargetStatus(status)
			}

			// Print summary
			printStatusSummary(statuses)

			return nil
		},
	}

	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// printTargetStatus prints the status for a single target.
func printTargetStatus(status *orchestrator.Status) {
	fmt.Printf("\nTarget: %s\n", status.Target)
	fmt.Println(statusSeparator)

	if status.Error != nil {
		fmt.Printf("  Status: Error - %v\n", status.Error)
		return
	}

	if status.InSync {
		fmt.Println("  Status: In sync")
	} else {
		fmt.Println("  Status: Out of sync")
	}

	printSkillList("Installed", status.Installed, "+")
	printSkillList("Missing", status.Missing, "-")
	printSkillList("Extra", status.Extra, "?")
}

// printSkillList prints a list of skills with a header and prefix.
func printSkillList(header string, skills []string, prefix string) {
	if len(skills) == 0 {
		return
	}
	fmt.Printf("  %s (%d):\n", header, len(skills))
	for _, name := range skills {
		fmt.Printf("    %s %s\n", prefix, name)
	}
}

// printStatusSummary prints a summary of all statuses.
func printStatusSummary(statuses []*orchestrator.Status) {
	if len(statuses) == 0 {
		fmt.Println("\nNo targets found.")
		return
	}

	var inSyncCount, outOfSyncCount, errorCount int
	for _, s := range statuses {
		switch {
		case s.Error != nil:
			errorCount++
		case s.InSync:
			inSyncCount++
		default:
			outOfSyncCount++
		}
	}

	fmt.Printf("\nSummary: %d target(s), %d in sync, %d out of sync",
		len(statuses), inSyncCount, outOfSyncCount)
	if errorCount > 0 {
		fmt.Printf(", %d error(s)", errorCount)
	}
	fmt.Println()
}
