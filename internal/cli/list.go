package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/skill"
)

// newListCmd creates the list command.
func newListCmd(a *app) *cobra.Command {
	scopeFlags := NewScopeFlags(skill.ScopeProject)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		Long: `List all available skills.

Use --global or --project to filter by scope.
If neither is specified, shows all skills.`,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			store, _, rootErr := a.newSkillStore()

			if scopeFlags.Project && rootErr != nil {
				return fmt.Errorf("not in a project directory")
			}

			var skills []*skill.Skill
			var err error

			if !scopeFlags.IsSet() {
				skills, err = store.GetAll()
			} else {
				scope, scopeErr := scopeFlags.GetScope()
				if scopeErr != nil {
					return scopeErr
				}
				skills, err = store.GetByScope(scope)
			}

			if err != nil {
				return fmt.Errorf("failed to list skills: %w", err)
			}

			if len(skills) == 0 {
				fmt.Println("No skills found")
				return nil
			}

			if err := printSkillsByScope(skills); err != nil {
				return err
			}
			return nil
		},
	}

	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// printSkillsByScope displays skills in a table format grouped by scope.
func printSkillsByScope(skills []*skill.Skill) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if _, err := fmt.Fprintf(w, "NAME\tSCOPE\tCATEGORY\tDESCRIPTION\n"); err != nil {
		return fmt.Errorf("failed to write table header: %w", err)
	}
	if _, err := fmt.Fprintf(w, "----\t-----\t--------\t-----------\n"); err != nil {
		return fmt.Errorf("failed to write table separator: %w", err)
	}

	for _, s := range skills {
		category := "default"
		if s.Category == skill.CategoryOptional {
			category = "optional"
		}
		desc := truncate(s.Description, 60)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Scope, category, desc); err != nil {
			return fmt.Errorf("failed to write skill row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}

	return nil
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
