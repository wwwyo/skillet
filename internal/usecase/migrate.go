package usecase

import (
	"github.com/wwwyo/skillet/internal/config"
	platformfs "github.com/wwwyo/skillet/internal/platform/fs"
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
	MoveResults []MigrateMoveResult
	SyncResults []SyncResult
}

// MigrateMoveResult represents the result of moving a single skill.
type MigrateMoveResult struct {
	SkillName  string
	FromTarget string
	Action     MigrateAction
	Message    string
	Error      error
}

// MigrateService migrates existing target-local skills into the central agents directory.
type MigrateService struct {
	fs      platformfs.FileSystem
	targets *TargetRegistry
	cfg     *config.Config
	syncSvc *SyncService
}

// NewMigrateService creates a new migrate service.
func NewMigrateService(fsys platformfs.FileSystem, cfg *config.Config, root string, syncSvc *SyncService) *MigrateService {
	return &MigrateService{
		fs:      fsys,
		targets: NewTargetRegistry(fsys, root, cfg),
		cfg:     cfg,
		syncSvc: syncSvc,
	}
}

// FindSkillsToMigrate finds existing skills in targets that can be migrated.
func (s *MigrateService) FindSkillsToMigrate(opts MigrateOptions) map[string][]string {
	result := make(map[string][]string)

	for _, t := range s.targets.GetAll() {
		names, err := t.ListMigratable(opts.Scope)
		if err != nil {
			continue
		}
		if len(names) > 0 {
			result[t.Name()] = append(result[t.Name()], names...)
		}
	}

	return result
}

// Migrate moves skills from targets to the agents directory and syncs.
func (s *MigrateService) Migrate(opts MigrateOptions, existingSkills map[string][]string) (*MigrateResult, error) {
	agentsDir, err := s.cfg.GetAgentsDir(s.fs, opts.ProjectRoot)
	if err != nil {
		return nil, err
	}

	moveResults := s.moveSkillsToAgents(agentsDir, existingSkills, opts)

	// Sync to create links back to targets.
	syncResults, err := s.syncSvc.Sync(SyncOptions{Force: true})
	if err != nil {
		return nil, err
	}

	found := existingSkills
	if found == nil {
		found = make(map[string][]string)
	}

	return &MigrateResult{
		Found:       found,
		MoveResults: moveResults,
		SyncResults: syncResults,
	}, nil
}

// HasSkillsToMigrate returns true if there are skills to migrate.
func (r *MigrateResult) HasSkillsToMigrate() bool {
	return len(r.Found) > 0
}

// moveSkillsToAgents moves skills from targets to the agents directory.
func (s *MigrateService) moveSkillsToAgents(agentsDir string, existingSkills map[string][]string, opts MigrateOptions) []MigrateMoveResult {
	skillsDir := s.fs.Join(agentsDir, config.SkillsDirName)
	moved := make(map[string]bool)
	var results []MigrateMoveResult

	for targetName, skills := range existingSkills {
		t, ok := s.targets.Get(targetName)
		if !ok {
			continue
		}

		targetSkillsDir, err := t.GetSkillsPath(opts.Scope)
		if err != nil || targetSkillsDir == "" {
			continue
		}

		for _, skillName := range skills {
			result := MigrateMoveResult{
				SkillName:  skillName,
				FromTarget: targetName,
			}

			srcPath := s.fs.Join(targetSkillsDir, skillName)
			dstPath := s.fs.Join(skillsDir, skillName)

			// Skip if already moved from another target.
			if moved[skillName] {
				if err := s.fs.RemoveAll(srcPath); err != nil {
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

			// Check if destination already exists.
			if s.fs.Exists(dstPath) {
				if err := s.fs.RemoveAll(srcPath); err != nil {
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

			// Move skill to agents directory.
			if err := s.fs.Rename(srcPath, dstPath); err != nil {
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
