package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
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
			// Find project root
			projectRoot, err := config.FindProjectRoot(a.fs)
			if err != nil {
				// If no project root, only show global
				if scopeFlags.Project {
					return fmt.Errorf("not in a project directory")
				}
				projectRoot = ""
			}

			store := skill.NewStore(a.fs, a.config, projectRoot)

			var skills []*skill.ScopedSkill

			if !scopeFlags.IsSet() {
				// Show all
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

			printSkillsByScope(skills)
			return nil
		},
	}

	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// printSkillsByScope groups and displays skills by scope.
func printSkillsByScope(skills []*skill.ScopedSkill) {
	grouped := make(map[skill.Scope][]*skill.ScopedSkill)
	for _, s := range skills {
		grouped[s.Scope] = append(grouped[s.Scope], s)
	}

	for _, scope := range []skill.Scope{skill.ScopeGlobal, skill.ScopeProject} {
		if scopeSkills := grouped[scope]; len(scopeSkills) > 0 {
			printSkillSection(scope, scopeSkills)
		}
	}
	fmt.Println()
}

// printSkillSection prints a single scope section.
func printSkillSection(scope skill.Scope, skills []*skill.ScopedSkill) {
	fmt.Printf("\n%s skills:\n", capitalizeFirst(scope.String()))
	fmt.Println(strings.Repeat("-", 40))

	for _, s := range skills {
		printSkill(s)
	}
}

// printSkill prints a single skill entry.
func printSkill(s *skill.ScopedSkill) {
	categoryMark := ""
	if s.Category == skill.CategoryOptional {
		categoryMark = " [optional]"
	}
	fmt.Printf("  %s%s\n", s.Name, categoryMark)

	if s.Description != "" {
		fmt.Printf("    %s\n", s.Description)
	}
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
