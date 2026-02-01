package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/skill"
	"github.com/wwwyo/skillet/internal/sync"
	"github.com/wwwyo/skillet/internal/target"
)

// newStatusCmd creates the status command.
func newStatusCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show sync status",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := config.FindProjectRoot(a.fs)
			if err != nil {
				projectRoot = ""
			}

			store := skill.NewStore(a.fs, a.config, projectRoot)
			registry := target.NewRegistry(a.fs, projectRoot, a.config)
			engine := sync.NewEngine(a.fs, store, registry, a.config, projectRoot)

			statuses, err := engine.GetStatus()
			if err != nil {
				return err
			}

			for _, s := range statuses {
				fmt.Printf("\n%s:\n", s.Target)
				if s.InSync {
					fmt.Println("  ✓ In sync")
				} else {
					if len(s.Missing) > 0 {
						fmt.Println("  Missing:")
						for _, name := range s.Missing {
							fmt.Printf("    - %s\n", name)
						}
					}
					if len(s.Extra) > 0 {
						fmt.Println("  Extra:")
						for _, name := range s.Extra {
							fmt.Printf("    - %s\n", name)
						}
					}
				}
			}

			return nil
		},
	}

	return cmd
}
