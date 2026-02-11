package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

var (
	// version is set via ldflags during build: -ldflags "-X github.com/wwwyo/skillet/internal/cli.version=v1.0.0"
	version = "v0.0.0"
	cfgFile string
)

func init() {
	if !semver.IsValid(version) {
		panic(fmt.Sprintf("invalid version set via ldflags: %q (must be valid semver)", version))
	}
}

// app represents the CLI application with its dependencies.
type app struct {
	fs          platformfs.FileSystem
	config      *config.Config
	configStore *config.Store
}

// newApp creates a new app instance.
func newApp() *app {
	fsys := platformfs.NewFileSystem()
	return &app{
		fs:          fsys,
		configStore: config.NewStore(fsys),
	}
}

// findProjectRoot returns project root path when available.
func (a *app) findProjectRoot() (root string, rootErr error) {
	root, rootErr = a.configStore.FindProjectRoot()
	if rootErr != nil {
		return "", rootErr
	}
	return root, nil
}

// newSkillStore creates a skill.Store and returns the project root.
// The caller can decide how to handle a missing project root.
func (a *app) newSkillStore() (*skill.Store, string, error) {
	root, err := a.findProjectRoot()
	return skill.NewStore(a.fs, a.config, root), root, err
}

// newRootCmd creates the root command for skillet.
func newRootCmd(a *app) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "skillet",
		Short:   "AI Agent Skills Manager",
		Long:    `Skillet manages AI agent skills as a Single Source of Truth (SSOT) for distribution and synthesis.`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := a.configStore.Load(cfgFile)
			if err != nil {
				if cmd.Name() != "init" && cmd.Name() != "migrate" {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cfg = config.DefaultConfig()
			}
			a.config = cfg
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "~/.config/skillet/config.yaml", "config file path")

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
