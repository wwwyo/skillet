package e2e_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSyncGlobalInstallsSkillToAllTargets(t *testing.T) {
	env := newE2EEnv(t, "copy")
	skillName := "sync-e2e-skill"
	createSkill(t, filepath.Join(env.agentsDir, "skills", skillName), skillName)

	out, err := runSkillet(t, env, "sync", "--global")
	if err != nil {
		t.Fatalf("sync failed: %v\noutput:\n%s", err, out)
	}

	for _, target := range []string{".claude", ".codex"} {
		installedSkillFile := filepath.Join(env.root, target, "skills", skillName, "SKILL.md")
		if _, statErr := os.Stat(installedSkillFile); statErr != nil {
			t.Fatalf("expected installed skill file at %s: %v\noutput:\n%s", installedSkillFile, statErr, out)
		}
	}
}

func TestRemoveGlobalRemovesStoreAndTargets(t *testing.T) {
	env := newE2EEnv(t, "symlink")
	skillName := "remove-e2e-skill"
	storeSkillDir := filepath.Join(env.agentsDir, "skills", skillName)
	createSkill(t, storeSkillDir, skillName)

	if out, err := runSkillet(t, env, "sync", "--global"); err != nil {
		t.Fatalf("pre-sync failed: %v\noutput:\n%s", err, out)
	}

	// Ensure the target path is a symlink so this test covers remove order behavior.
	for _, target := range []string{".claude", ".codex"} {
		targetSkillDir := filepath.Join(env.root, target, "skills", skillName)
		fi, err := os.Lstat(targetSkillDir)
		if err != nil {
			t.Fatalf("expected installed target path at %s: %v", targetSkillDir, err)
		}
		if fi.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("expected symlink at %s", targetSkillDir)
		}
	}

	out, err := runSkillet(t, env, "remove", "--global", skillName)
	if err != nil {
		t.Fatalf("remove failed: %v\noutput:\n%s", err, out)
	}

	if _, err := os.Stat(storeSkillDir); !os.IsNotExist(err) {
		t.Fatalf("expected store skill directory to be removed: %s (err=%v)", storeSkillDir, err)
	}

	for _, target := range []string{".claude", ".codex"} {
		targetSkillDir := filepath.Join(env.root, target, "skills", skillName)
		if _, err := os.Lstat(targetSkillDir); !os.IsNotExist(err) {
			t.Fatalf("expected target skill path to be removed: %s (err=%v)", targetSkillDir, err)
		}
	}
}

type e2eEnv struct {
	moduleRoot string
	binaryPath string
	root       string
	agentsDir  string
	configPath string
	homeDir    string
}

func newE2EEnv(t *testing.T, strategy string) *e2eEnv {
	t.Helper()

	moduleRoot := mustModuleRoot(t)
	root := t.TempDir()
	agentsDir := filepath.Join(root, ".agents")
	configPath := filepath.Join(root, "config.yaml")
	homeDir := filepath.Join(root, "home")
	binaryPath := buildSkilletBinary(t, moduleRoot, root)

	for _, dir := range []string{
		homeDir,
		agentsDir,
		filepath.Join(agentsDir, "skills"),
		filepath.Join(agentsDir, "skills", "optional"),
		filepath.Join(root, ".claude"),
		filepath.Join(root, ".codex"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir %s: %v", dir, err)
		}
	}

	cfg := fmt.Sprintf(`version: 1
globalPath: %s
defaultStrategy: %s
targets:
  claude:
    enabled: true
    globalPath: %s
  codex:
    enabled: true
    globalPath: %s
`,
		agentsDir,
		strategy,
		filepath.Join(root, ".claude"),
		filepath.Join(root, ".codex"),
	)
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	return &e2eEnv{
		moduleRoot: moduleRoot,
		binaryPath: binaryPath,
		root:       root,
		agentsDir:  agentsDir,
		configPath: configPath,
		homeDir:    homeDir,
	}
}

func buildSkilletBinary(t *testing.T, moduleRoot, outDir string) string {
	t.Helper()

	binaryPath := filepath.Join(outDir, "skillet-e2e")
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/skillet")
	cmd.Dir = moduleRoot
	cmd.Env = os.Environ()

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build skillet binary: %v\noutput:\n%s", err, out.String())
	}

	return binaryPath
}

func createSkill(t *testing.T, skillDir, skillName string) {
	t.Helper()

	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	content := fmt.Sprintf(`---
name: %s
description: e2e test skill
---

# %s
`, skillName, skillName)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write SKILL.md: %v", err)
	}
}

func runSkillet(t *testing.T, env *e2eEnv, args ...string) (string, error) {
	t.Helper()

	cmdArgs := append([]string{"--config", env.configPath}, args...)
	cmd := exec.Command(env.binaryPath, cmdArgs...)
	cmd.Dir = env.moduleRoot
	cmd.Env = append(os.Environ(), "HOME="+env.homeDir)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	return out.String(), err
}

func mustModuleRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	if !strings.HasSuffix(root, string(filepath.Separator)+"skillet") {
		t.Fatalf("unexpected module root: %s", root)
	}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("go.mod not found under module root %s: %v", root, err)
	}

	return root
}
