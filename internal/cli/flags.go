package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/skill"
)

// ScopeFlags holds the scope-related flags for commands.
type ScopeFlags struct {
	Global       bool
	Project      bool
	DefaultScope skill.Scope
}

// NewScopeFlags creates a new ScopeFlags with the given default scope.
func NewScopeFlags(defaultScope skill.Scope) ScopeFlags {
	return ScopeFlags{DefaultScope: defaultScope}
}

// AddScopeFlags adds --global and --project flags to a command.
func AddScopeFlags(cmd *cobra.Command, flags *ScopeFlags) {
	cmd.Flags().BoolVarP(&flags.Global, "global", "g", false, "Use global scope")
	cmd.Flags().BoolVarP(&flags.Project, "project", "p", false, "Use project scope")
}

// GetScope returns the scope based on the flags.
// Returns an error if both flags are set.
// If neither is set, returns the DefaultScope.
func (f *ScopeFlags) GetScope() (skill.Scope, error) {
	if f.Global && f.Project {
		return 0, fmt.Errorf("cannot specify both --global and --project")
	}

	if f.Global {
		return skill.ScopeGlobal, nil
	}
	if f.Project {
		return skill.ScopeProject, nil
	}

	return f.DefaultScope, nil
}

// IsSet returns true if either flag is explicitly set.
func (f *ScopeFlags) IsSet() bool {
	return f.Global || f.Project
}
