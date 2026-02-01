package target

// TargetDef defines default paths for a target.
type TargetDef struct {
	GlobalPath  string
	ProjectPath string
	SkillsDir   string
}

// DefaultTargets contains default definitions for all supported targets.
var DefaultTargets = map[string]TargetDef{
	"claude": {GlobalPath: "~/.claude", ProjectPath: ".claude", SkillsDir: "skills"},
	"codex":  {GlobalPath: "~/.codex", ProjectPath: ".codex", SkillsDir: "skills"},
}
