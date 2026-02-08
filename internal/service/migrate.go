package service

// FindSkillsToMigrate finds existing skills in targets that can be migrated.
func (s *SkillService) FindSkillsToMigrate(opts MigrateOptions) map[string][]string {
	result := make(map[string][]string)

	for _, t := range s.targets.GetAll() {
		targetSkillsDir, err := t.GetSkillsPath(opts.Scope)
		if err != nil || targetSkillsDir == "" {
			continue
		}

		if !s.fs.Exists(targetSkillsDir) || !s.fs.IsDir(targetSkillsDir) {
			continue
		}

		entries, err := s.fs.ReadDir(targetSkillsDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// Skip symlinks (already managed by skillet)
			if entry.Type()&ModeSymlink != 0 {
				continue
			}
			if entry.IsDir() {
				skillName := entry.Name()
				if err := ValidateName(skillName); err != nil {
					continue
				}
				skillDir := s.fs.Join(targetSkillsDir, skillName)
				if IsValidSkillDir(s.fs, skillDir) {
					result[t.Name()] = append(result[t.Name()], skillName)
				}
			}
		}
	}

	return result
}

// Migrate moves skills from targets to the agents directory and syncs.
func (s *SkillService) Migrate(opts MigrateOptions, existingSkills map[string][]string) (*MigrateResult, error) {
	agentsDir, err := s.cfg.GetAgentsDir(s.fs, opts.ProjectRoot)
	if err != nil {
		return nil, err
	}

	moveResults := s.moveSkillsToAgents(agentsDir, existingSkills, opts)

	// Sync to create links back to targets
	syncResults, err := s.Sync(SyncOptions{Force: true})
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
func (s *SkillService) moveSkillsToAgents(agentsDir string, existingSkills map[string][]string, opts MigrateOptions) []MoveResult {
	skillsDir := s.fs.Join(agentsDir, SkillsDirName)
	moved := make(map[string]bool)
	var results []MoveResult

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
			result := MoveResult{
				SkillName:  skillName,
				FromTarget: targetName,
			}

			srcPath := s.fs.Join(targetSkillsDir, skillName)
			dstPath := s.fs.Join(skillsDir, skillName)

			// Skip if already moved from another target
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

			// Check if destination already exists
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

			// Move skill to agents directory
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
