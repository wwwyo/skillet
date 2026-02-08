package service

// SkillReader abstracts skill read operations (used by list, sync, status).
type SkillReader interface {
	GetAll() ([]*Skill, error)
	GetByScope(scope Scope) ([]*Skill, error)
	GetResolved() ([]*Skill, error)
}

// SkillFinder abstracts skill search operations (used by remove).
type SkillFinder interface {
	GetByName(name string) (*Skill, error)
	FindInScope(name string, scope Scope) (*Skill, error)
	Exists(name string) bool
}

// SkillRemover abstracts skill deletion operations (used by remove).
type SkillRemover interface {
	Remove(skill *Skill) error
}

// SkillStore combines all skill store operations (for DI wiring).
type SkillStore interface {
	SkillReader
	SkillFinder
	SkillRemover
}

// Target abstracts skill deployment to a target.
type Target interface {
	Name() string
	Install(skill *Skill, opts InstallOptions) error
	Uninstall(skillName string) error
	IsInstalled(skillName string) bool
	IsInstalledInScope(skillName string, scope Scope) bool
	GetSkillsPath(scope Scope) (string, error)
	ListInstalled() ([]string, error)
}

// TargetRegistry manages multiple targets.
type TargetRegistry interface {
	GetAll() []Target
	Get(name string) (Target, bool)
}

// ConfigStore abstracts config file persistence.
type ConfigStore interface {
	Load(path string) (*Config, error)
	Save(cfg *Config, path string) error
	GlobalConfigPath() (string, error)
	FindProjectRoot() (string, error)
	FindProjectRootFrom(startDir string) (string, error)
}
