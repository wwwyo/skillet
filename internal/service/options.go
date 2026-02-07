package service

// SyncAction represents the type of sync action taken.
type SyncAction string

const (
	SyncActionInstall   SyncAction = "install"
	SyncActionUpdate    SyncAction = "update"
	SyncActionUninstall SyncAction = "uninstall"
	SyncActionSkip      SyncAction = "skip"
	SyncActionError     SyncAction = "error"
)

// SyncResult represents the result of a sync operation for a single skill.
type SyncResult struct {
	SkillName string
	Target    string
	Action    SyncAction
	Error     error
}

// SyncOptions contains options for synchronization.
type SyncOptions struct {
	// DryRun only shows what would be done without making changes
	DryRun bool
	// Force overwrites existing installations
	Force bool
	// Scope limits sync to a specific scope (nil for all)
	Scope *Scope
}

// Status represents the synchronization status for a target.
type Status struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	InSync    bool
	Error     error
}

// StatusParams contains parameters for creating a new Status.
type StatusParams struct {
	Target    string
	Installed []string
	Missing   []string
	Extra     []string
	Error     error
}

// NewStatus creates a new Status from the given parameters.
// InSync is computed automatically: true only when no missing skills, no extra skills, and no error.
func NewStatus(params StatusParams) *Status {
	return &Status{
		Target:    params.Target,
		Installed: params.Installed,
		Missing:   params.Missing,
		Extra:     params.Extra,
		Error:     params.Error,
		InSync:    len(params.Missing) == 0 && len(params.Extra) == 0 && params.Error == nil,
	}
}

// GetStatusOptions contains options for getting status.
type GetStatusOptions struct {
	// Scope limits status to a specific scope (nil for all)
	Scope *Scope
}

// MigrateAction represents the type of action taken during migration.
type MigrateAction string

const (
	MigrateActionMoved   MigrateAction = "moved"
	MigrateActionSkipped MigrateAction = "skipped"
	MigrateActionRemoved MigrateAction = "removed"
	MigrateActionError   MigrateAction = "error"
)

// MigrateOptions contains options for migration.
type MigrateOptions struct {
	Scope       Scope
	ProjectRoot string
}

// MigrateResult represents the result of a migration operation.
type MigrateResult struct {
	Found       map[string][]string // target -> skill names
	MoveResults []MoveResult
	SyncResults []SyncResult
}

// MoveResult represents the result of moving a single skill.
type MoveResult struct {
	SkillName  string
	FromTarget string
	Action     MigrateAction
	Message    string
	Error      error
}

// MigrateResultParams contains parameters for creating a MigrateResult.
type MigrateResultParams struct {
	Found       map[string][]string
	MoveResults []MoveResult
	SyncResults []SyncResult
}

// NewMigrateResult creates a new MigrateResult from the given parameters.
func NewMigrateResult(params MigrateResultParams) *MigrateResult {
	found := params.Found
	if found == nil {
		found = make(map[string][]string)
	}
	return &MigrateResult{
		Found:       found,
		MoveResults: params.MoveResults,
		SyncResults: params.SyncResults,
	}
}

// HasSkillsToMigrate returns true if there are skills to migrate.
func (r *MigrateResult) HasSkillsToMigrate() bool {
	return len(r.Found) > 0
}

// RemoveOptions contains options for removing a skill.
type RemoveOptions struct {
	// Name is the skill name to remove
	Name string
	// Scope limits removal to a specific scope (nil to auto-detect)
	Scope *Scope
}

// RemoveResult represents the result of a remove operation.
type RemoveResult struct {
	SkillName     string
	Scope         Scope
	StoreRemoved  bool
	TargetResults []TargetRemoveResult
	Error         error
}

// TargetRemoveResult represents the result of removing from a single target.
type TargetRemoveResult struct {
	Target  string
	Removed bool
	Error   error
}

// RemoveResultParams contains parameters for creating a RemoveResult.
type RemoveResultParams struct {
	SkillName     string
	Scope         Scope
	StoreRemoved  bool
	TargetResults []TargetRemoveResult
	Error         error
}

// NewRemoveResult creates a new RemoveResult from the given parameters.
func NewRemoveResult(params RemoveResultParams) *RemoveResult {
	return &RemoveResult{
		SkillName:     params.SkillName,
		Scope:         params.Scope,
		StoreRemoved:  params.StoreRemoved,
		TargetResults: params.TargetResults,
		Error:         params.Error,
	}
}

// Success returns true if the removal was successful.
func (r *RemoveResult) Success() bool {
	return r.StoreRemoved && r.Error == nil
}

// InstallOptions contains options for installing a skill.
type InstallOptions struct {
	// Strategy specifies how to install (symlink or copy)
	Strategy Strategy
	// Force overwrites existing installations
	Force bool
}

// SetupGlobalParams contains parameters for global setup.
type SetupGlobalParams struct {
	GlobalPath     string
	EnabledTargets map[string]bool
	Strategy       Strategy
	ConfigPath     string
}
