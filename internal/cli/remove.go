package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/usecase"
)

// newRemoveCmd creates the remove command.
func newRemoveCmd(a *app) *cobra.Command {
	scopeFlags := NewScopeFlags(skill.ScopeProject)

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a skill from the store and targets",
		Long: `Remove a skill from the skill store and all targets.

By default, attempts to find the skill in any scope (project scope takes priority).
Use --global or --project to specify a particular scope.

This removes the skill from both the skillet store and all configured targets
(e.g., ~/.claude/skills).`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, rootErr := a.findProjectRoot()
			if rootErr != nil {
				root = ""
			}
			if scopeFlags.Project && rootErr != nil {
				return fmt.Errorf("not in a project directory")
			}
			svc := usecase.NewRemoveService(a.fs, a.config, root)

			opts := usecase.RemoveOptions{Name: args[0]}
			if scopeFlags.IsSet() {
				scope, err := scopeFlags.GetScope()
				if err != nil {
					return err
				}
				opts.Scope = &scope
			}

			result := svc.Remove(opts)
			if result.Error != nil {
				return result.Error
			}

			printRemoveResult(result)

			return nil
		},
	}

	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// printRemoveResult prints the result of a remove operation.
func printRemoveResult(result *usecase.RemoveResult) {
	fmt.Printf("Removed skill '%s' from %s scope\n", result.SkillName, result.Scope)

	for _, tr := range result.TargetResults {
		if tr.Removed {
			fmt.Printf("  Removed from target '%s'\n", tr.Target)
		} else if tr.Error != nil {
			fmt.Printf("  Warning: failed to remove from %s: %v\n", tr.Target, tr.Error)
		}
	}
}
