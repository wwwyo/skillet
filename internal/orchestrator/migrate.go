package orchestrator

import (
	"github.com/wwwyo/skillet/internal/config"
	"github.com/wwwyo/skillet/internal/fs"
	"github.com/wwwyo/skillet/internal/skill"
)

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
	Scope       skill.Scope
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

// FindSkillsToMigrate finds existing skills in targets that can be migrated.
func (o *Orchestrator) FindSkillsToMigrate(opts MigrateOptions) map[string][]string {
	result := make(map[string][]string)

	for _, t := range o.registry.GetAll() {
		targetSkillsDir, err := t.GetSkillsPath(opts.Scope)
		if err != nil || targetSkillsDir == "" {
			continue
		}

		if !o.fs.Exists(targetSkillsDir) || !o.fs.IsDir(targetSkillsDir) {
			continue
		}

		entries, err := o.fs.ReadDir(targetSkillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// Skip symlinks (already managed by skillet)
			if entry.Type()&fs.ModeSymlink != 0 {
				continue
			}
			if entry.IsDir() {
				skillName := entry.Name()
				if err := skill.ValidateName(skillName); err != nil {
					continue
				}
				skillDir := o.fs.Join(targetSkillsDir, skillName)
				if skill.IsValidSkillDir(o.fs, skillDir) {
					result[t.Name()] = append(result[t.Name()], skillName)
				}
			}
		}
	}

	return result
}

// Migrate moves skills from targets to the agents directory and syncs.
func (o *Orchestrator) Migrate(opts MigrateOptions, existingSkills map[string][]string) (*MigrateResult, error) {
	agentsDir, err := o.cfg.GetAgentsDir(o.fs, opts.ProjectRoot)
	if err != nil {
		return nil, err
	}

	moveResults := o.moveSkillsToAgents(agentsDir, existingSkills, opts)

	// Sync to create links back to targets
	syncResults, err := o.Sync(SyncOptions{Force: true})
	if err != nil {
		return nil, err
	}

	return NewMigrateResult(MigrateResultParams{
		Found:       existingSkills,
		MoveResults: moveResults,
		SyncResults: syncResults,
	}), nil
}

// moveSkillsToAgents moves skills from targets to the agents directory.
func (o *Orchestrator) moveSkillsToAgents(agentsDir string, existingSkills map[string][]string, opts MigrateOptions) []MoveResult {
	skillsDir := o.fs.Join(agentsDir, config.SkillsDir)
	moved := make(map[string]bool)
	var results []MoveResult

	for targetName, skills := range existingSkills {
		t, err := o.registry.Get(targetName)
		if err != nil {
			continue
		}

		targetSkillsDir, err := t.GetSkillsPath(opts.Scope)
		if err != nil || targetSkillsDir == "" {
			continue
		}

		for _, skillName := range skills {
			result := MoveResult{
				SkillName:  skillName,
				FromTarget: targetName,
			}

			srcPath := o.fs.Join(targetSkillsDir, skillName)
			dstPath := o.fs.Join(skillsDir, skillName)

			// Skip if already moved from another target
			if moved[skillName] {
				if err := o.fs.RemoveAll(srcPath); err != nil {
					result.Action = MigrateActionError
					result.Message = "failed to remove duplicate"
					result.Error = err
				} else {
					result.Action = MigrateActionRemoved
					result.Message = "removed duplicate"
				}
				results = append(results, result)
				continue
			}

			// Check if destination already exists
			if o.fs.Exists(dstPath) {
				if err := o.fs.RemoveAll(srcPath); err != nil {
					result.Action = MigrateActionError
					result.Message = "failed to remove after skip"
					result.Error = err
				} else {
					result.Action = MigrateActionSkipped
					result.Message = "already exists in agents"
				}
				results = append(results, result)
				continue
			}

			// Move skill to agents directory
			if err := o.fs.Rename(srcPath, dstPath); err != nil {
				result.Action = MigrateActionError
				result.Message = "failed to move"
				result.Error = err
				results = append(results, result)
				continue
			}

			moved[skillName] = true
			result.Action = MigrateActionMoved
			results = append(results, result)
		}
	}

	return results
}
