package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/service"
)

// newListCmd creates the list command.
func newListCmd(a *app) *cobra.Command {
	scopeFlags := NewScopeFlags(service.ScopeProject)

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

			var skills []*service.Skill
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

			printSkillsByScope(skills)
			return nil
		},
	}

	AddScopeFlags(cmd, &scopeFlags)

	return cmd
}

// printSkillsByScope groups and displays skills by scope.
func printSkillsByScope(skills []*service.Skill) {
	grouped := make(map[service.Scope][]*service.Skill)
	for _, s := range skills {
		grouped[s.Scope] = append(grouped[s.Scope], s)
	}

	for _, scope := range []service.Scope{service.ScopeGlobal, service.ScopeProject} {
		if scopeSkills := grouped[scope]; len(scopeSkills) > 0 {
			printSkillSection(scope, scopeSkills)
		}
	}
	fmt.Println()
}

// printSkillSection prints a single scope section.
func printSkillSection(scope service.Scope, skills []*service.Skill) {
	fmt.Printf("\n%s skills:\n", capitalizeFirst(scope.String()))
	fmt.Println(strings.Repeat("-", 40))

	for _, s := range skills {
		printSkill(s)
	}
}

// printSkill prints a single skill entry.
func printSkill(s *service.Skill) {
	categoryMark := ""
	if s.Category == service.CategoryOptional {
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
