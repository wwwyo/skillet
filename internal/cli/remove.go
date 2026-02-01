package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newRemoveCmd creates the remove command.
func newRemoveCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [skill-name]",
		Short: "Remove a skill from the store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented yet")
		},
	}

	return cmd
}
