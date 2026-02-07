package service

// SkillService coordinates operations between skill stores and targets.
type SkillService struct {
	fs      FileSystem
	store   SkillStore
	targets TargetRegistry
	cfg     *Config
	root    string
}

// NewSkillService creates a new SkillService.
func NewSkillService(fs FileSystem, store SkillStore, targets TargetRegistry, cfg *Config, root string) *SkillService {
	return &SkillService{
		fs:      fs,
		store:   store,
		targets: targets,
		cfg:     cfg,
		root:    root,
	}
}

// filterByScope filters skills by the specified scope.
func filterByScope(skills []*Skill, scope Scope) []*Skill {
	var filtered []*Skill
	for _, s := range skills {
		if s.Scope == scope {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
