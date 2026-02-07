package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/service"
)

var initGlobal bool
var initProject bool
var initPath string
var initYes bool

// newInitCmd creates the init command.
func newInitCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize skillet configuration",
		Long: `Initialize skillet for global or project use.

Use --global to initialize global skills (default: ~/.agents/)
  Config is stored at ~/.config/skillet/config.yaml
  Use --path to specify a custom location (e.g., for dotfiles)
Use --project to initialize project-level configuration at ./.agents/

If neither flag is specified, project initialization is assumed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !initGlobal && !initProject {
				initProject = true
			}

			if initPath != "" && !initGlobal {
				return fmt.Errorf("--path can only be used with --global")
			}

			if initGlobal {
				if err := initializeGlobal(a, initPath, initYes); err != nil {
					return err
				}
			}

			if initProject {
				if err := initializeProject(a, initYes); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&initGlobal, "global", "g", false, "Initialize global configuration")
	cmd.Flags().BoolVarP(&initProject, "project", "p", false, "Initialize project configuration")
	cmd.Flags().StringVar(&initPath, "path", "", "Custom path for initialization (only with --global)")
	cmd.Flags().BoolVarP(&initYes, "yes", "y", false, "Skip confirmation prompts")

	return cmd
}

func initializeGlobal(a *app, customPath string, skipPrompts bool) error {
	reader := bufio.NewReader(os.Stdin)

	globalPath := promptGlobalPath(reader, customPath, skipPrompts)
	enabledTargets := promptTargets(skipPrompts)
	if err := validateTargets(enabledTargets); err != nil {
		return err
	}
	strategy := promptStrategy(skipPrompts)

	agentsDir, err := service.ExpandPath(a.fs, globalPath)
	if err != nil {
		return err
	}

	configPath, err := service.GlobalConfigPath(a.fs)
	if err != nil {
		return err
	}

	if !skipPrompts && !confirmCreation(reader, configPath, agentsDir, enabledTargets, strategy) {
		fmt.Println("Aborted.")
		return nil
	}

	setupSvc := service.NewSetupService(a.fs, a.configStore)
	cfg, err := setupSvc.SetupGlobal(service.SetupGlobalParams{
		GlobalPath:     globalPath,
		EnabledTargets: enabledTargets,
		Strategy:       strategy,
		ConfigPath:     configPath,
	})
	if err != nil {
		return err
	}

	if a.fs.Exists(configPath) {
		fmt.Printf("\n✓ Global configuration at %s\n", configPath)
	} else {
		fmt.Printf("\n✓ Created global configuration at %s\n", configPath)
	}
	fmt.Printf("✓ Initialized global skills at %s\n", strings.Replace(globalPath, "~", "$HOME", 1))

	if err := runMigrate(a, cfg, migrateRunOptions{
		skipPrompts:    skipPrompts,
		defaultConfirm: false,
		scope:          service.ScopeGlobal,
		projectRoot:    "",
	}); err != nil {
		return err
	}

	return nil
}

func promptGlobalPath(reader *bufio.Reader, customPath string, skipPrompts bool) string {
	if customPath != "" {
		return customPath
	}
	if skipPrompts {
		return service.DefaultGlobalPath
	}

	fmt.Printf("\nGlobal skills path [%s]: ", service.DefaultGlobalPath)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return service.DefaultGlobalPath
	}
	return input
}

func promptTargets(skipPrompts bool) map[string]bool {
	defaultCfg := service.DefaultConfig()
	enabledTargets := make(map[string]bool)

	if skipPrompts {
		for name := range defaultCfg.Targets {
			enabledTargets[name] = true
		}
		return enabledTargets
	}

	var options []string
	var defaults []string
	for name := range defaultCfg.Targets {
		options = append(options, name)
		defaults = append(defaults, name)
	}

	var selected []string
	prompt := &survey.MultiSelect{
		Message: "Select targets (Space: toggle, Enter: confirm):",
		Options: options,
		Default: defaults,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		os.Exit(1)
	}

	for _, name := range selected {
		enabledTargets[name] = true
	}

	return enabledTargets
}

func promptStrategy(skipPrompts bool) service.Strategy {
	if skipPrompts {
		return service.StrategySymlink
	}

	options := []string{
		string(service.StrategySymlink),
		string(service.StrategyCopy),
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select sync strategy:",
		Options: options,
		Default: string(service.StrategySymlink),
		Help:    "symlink: creates symbolic links (recommended), copy: copies files",
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		os.Exit(1)
	}

	return service.Strategy(selected)
}

func validateTargets(enabledTargets map[string]bool) error {
	for _, enabled := range enabledTargets {
		if enabled {
			return nil
		}
	}
	return fmt.Errorf("at least one target must be selected")
}

func confirmCreation(reader *bufio.Reader, configPath, agentsDir string, enabledTargets map[string]bool, strategy service.Strategy) bool {
	fmt.Println()
	fmt.Println("This will create:")
	fmt.Printf("  Config: %s\n", configPath)
	fmt.Printf("  Skills: %s/skills/\n", agentsDir)
	fmt.Print("  Targets: ")

	var targetNames []string
	for name := range enabledTargets {
		if enabledTargets[name] {
			targetNames = append(targetNames, name)
		}
	}
	fmt.Println(strings.Join(targetNames, ", "))
	fmt.Printf("  Strategy: %s\n", strategy)
	fmt.Println()
	fmt.Print("Continue? [Y/n]: ")

	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))
	return confirm == "" || confirm == "y" || confirm == "yes"
}

func initializeProject(a *app, skipPrompts bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	setupSvc := service.NewSetupService(a.fs, a.configStore)
	if err := setupSvc.SetupProject(cwd); err != nil {
		return err
	}

	fmt.Printf("Initialized project skillet at %s\n", service.ProjectAgentsDir(cwd, a.fs))

	cfg, err := a.configStore.Load("")
	if err != nil {
		return nil
	}

	if err := runMigrate(a, cfg, migrateRunOptions{
		skipPrompts:    skipPrompts,
		defaultConfirm: false,
		scope:          service.ScopeProject,
		projectRoot:    cwd,
	}); err != nil {
		return err
	}

	return nil
}
