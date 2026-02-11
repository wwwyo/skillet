package skill

import platformfs "github.com/wwwyo/skillet/internal/platform/fs"

// maxValidationDepth is the maximum depth to search for SKILL.md files.
const maxValidationDepth = 5

// isValidSkillDir checks if a directory is a valid skill directory.
// A valid skill directory contains SKILL.md either directly or in a subdirectory.
func isValidSkillDir(fsys platformfs.FileSystem, dir string) bool {
	return isValidSkillDirWithDepth(fsys, dir, 0)
}

func isValidSkillDirWithDepth(fsys platformfs.FileSystem, dir string, depth int) bool {
	if depth > maxValidationDepth {
		return false
	}

	if fsys.Exists(fsys.Join(dir, "SKILL.md")) {
		return true
	}

	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() && isValidSkillDirWithDepth(fsys, fsys.Join(dir, entry.Name()), depth+1) {
			return true
		}
	}

	return false
}
