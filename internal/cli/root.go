package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
)

var (
	// version is set via ldflags during build: -ldflags "-X github.com/wwwyo/skillet/internal/cli.version=v1.0.0"
	// Default value for development builds
	version = "v0.0.0"
	cfgFile string
)

func init() {
	if !semver.IsValid(version) {
		// Panic if invalid version was set via ldflags (build-time error)
		panic(fmt.Sprintf("invalid version set via ldflags: %q (must be valid semver)", version))
	}
}

// app represents the CLI application with its dependencies.
type app struct {
	fs     fs.System
	config *config.Config
}

// newApp creates a new app instance.
func newApp() *app {
	return &app{
		fs: fs.New(),
	}
}

// newRootCmd creates the root command for skillet.
func newRootCmd(a *app) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "skillet",
		Short:   "AI Agent Skills Manager",
		Long:    `Skillet manages AI agent skills as a Single Source of Truth (SSOT) for distribution and synthesis.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.Load(a.fs, cfgFile)
			if err != nil {
				// Config might not exist yet, which is fine for init and migrate commands
				if cmd.Name() != "init" && cmd.Name() != "migrate" {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cfg = config.Default()
			}
			a.config = cfg
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "~/.config/skillet/config.yaml", "config file path")

	// Add subcommands
	rootCmd.AddCommand(newInitCmd(a))
	rootCmd.AddCommand(newRemoveCmd(a))
	rootCmd.AddCommand(newListCmd(a))
	rootCmd.AddCommand(newSyncCmd(a))
	rootCmd.AddCommand(newStatusCmd(a))
	rootCmd.AddCommand(newMigrateCmd(a))

	return rootCmd
}

// Execute runs the CLI application.
func Execute() {
	a := newApp()
	rootCmd := newRootCmd(a)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
