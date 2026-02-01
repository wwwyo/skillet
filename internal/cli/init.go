package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/skill"
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
			// Default to project if neither specified
			if !initGlobal && !initProject {
				initProject = true
			}

			// Validate --path is only used with --global
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

	agentsDir, err := config.ExpandPath(a.fs, globalPath)
	if err != nil {
		return err
	}

	configPath, err := config.GlobalConfigPath(a.fs)
	if err != nil {
		return err
	}

	if !skipPrompts && !confirmCreation(reader, configPath, agentsDir, enabledTargets, strategy) {
		fmt.Println("Aborted.")
		return nil
	}

	if err := createSkillsDirectories(a, agentsDir); err != nil {
		return err
	}

	cfg, err := saveGlobalConfig(a, configPath, globalPath, enabledTargets, strategy)
	if err != nil {
		return err
	}

	// Ask about migrating existing skills
	if err := runMigrate(a, cfg, migrateRunOptions{
		skipPrompts:    skipPrompts,
		defaultConfirm: false, // Destructive operation, default to no
		scope:          skill.ScopeGlobal,
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
		return config.DefaultGlobalPath
	}

	fmt.Printf("\nGlobal skills path [%s]: ", config.DefaultGlobalPath)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return config.DefaultGlobalPath
	}
	return input
}

func promptTargets(skipPrompts bool) map[string]bool {
	defaultCfg := config.Default()
	enabledTargets := make(map[string]bool)

	if skipPrompts {
		for name := range defaultCfg.Targets {
			enabledTargets[name] = true
		}
		return enabledTargets
	}

	// Build options list
	var options []string
	var defaults []string
	for name := range defaultCfg.Targets {
		options = append(options, name)
		defaults = append(defaults, name) // All selected by default
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

	// Convert selected list to map
	for _, name := range selected {
		enabledTargets[name] = true
	}

	return enabledTargets
}

func promptStrategy(skipPrompts bool) config.Strategy {
	if skipPrompts {
		return config.StrategySymlink
	}

	options := []string{
		string(config.StrategySymlink),
		string(config.StrategyCopy),
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select sync strategy:",
		Options: options,
		Default: string(config.StrategySymlink),
		Help:    "symlink: creates symbolic links (recommended), copy: copies files",
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		os.Exit(1)
	}

	return config.Strategy(selected)
}

func validateTargets(enabledTargets map[string]bool) error {
	for _, enabled := range enabledTargets {
		if enabled {
			return nil
		}
	}
	return fmt.Errorf("at least one target must be selected")
}

func confirmCreation(reader *bufio.Reader, configPath, agentsDir string, enabledTargets map[string]bool, strategy config.Strategy) bool {
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

func createSkillsDirectories(a *app, agentsDir string) error {
	dirs := []string{
		agentsDir,
		a.fs.Join(agentsDir, config.SkillsDir),
		a.fs.Join(agentsDir, config.SkillsDir, config.OptionalDir),
	}

	for _, dir := range dirs {
		if err := a.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func saveGlobalConfig(a *app, configPath, globalPath string, enabledTargets map[string]bool, strategy config.Strategy) (*config.Config, error) {
	if a.fs.Exists(configPath) {
		fmt.Printf("\n• Global configuration already exists at %s\n", configPath)
		// Load and return existing config
		return config.Load(a.fs, configPath)
	}

	cfg := config.Default()
	if globalPath != config.DefaultGlobalPath {
		cfg.GlobalPath = globalPath
	}
	cfg.DefaultStrategy = strategy

	for name, target := range cfg.Targets {
		target.Enabled = enabledTargets[name]
		cfg.Targets[name] = target
	}

	if err := cfg.SaveTo(a.fs, configPath); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("\n✓ Created global configuration at %s\n", configPath)
	fmt.Printf("✓ Initialized global skills at %s\n", strings.Replace(globalPath, "~", "$HOME", 1))
	return cfg, nil
}

func initializeProject(a *app, skipPrompts bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	agentsDir := config.ProjectAgentsDir(cwd, a.fs)

	// Create directory structure
	dirs := []string{
		agentsDir,
		config.ProjectSkillsDir(cwd, a.fs, ""),
		config.ProjectSkillsDir(cwd, a.fs, config.OptionalDir),
	}

	for _, dir := range dirs {
		if err := a.fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	fmt.Printf("Initialized project skillet at %s\n", agentsDir)

	// Ask about migrating existing skills
	cfg, err := config.Load(a.fs, "")
	if err != nil {
		// Config not found, skip migration
		return nil
	}

	if err := runMigrate(a, cfg, migrateRunOptions{
		skipPrompts:    skipPrompts,
		defaultConfirm: false, // Destructive operation, default to no
		scope:          skill.ScopeProject,
		projectRoot:    cwd,
	}); err != nil {
		return err
	}

	return nil
}
