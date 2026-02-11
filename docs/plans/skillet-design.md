# Skillet - AI Agent Skills Manager Design Document

## Overview

A Go CLI tool that manages, distributes, and composes AI Agent / LLM skills as an SSOT (Single Source of Truth).

## Problems Addressed

### 1. Reproducibility Breakdown Due to Skill Dispersion
- **Current state**: Multiple LLM clients (Claude Code, Codex CLI, etc.) each have their own config directories (`.claude/`, `.codex/`)
- **Pain**: Copying the same skill to multiple places → update gaps → skill behavior drifts across clients
- **Impact**: Unclear "where to put which skill" during onboarding, difficult to reproduce environments

### 2. Difficulty Managing Team Standards
- **Current state**: Putting `.claude/skills/` under git makes skill changes hard to review in PRs
- **Pain**: Skills are added or changed ad hoc, so quality varies across the team
- **Impact**: No mechanism to maintain and manage project-standard skills

## Value Provided

1. **SSOT**: Manage skills in one place and sync them automatically to each LLM client
2. **Diff-friendly review**: Put `.agents/` under git to review skill changes in PRs
3. **Cross-client drift prevention**: Sync the same skill to multiple targets in one go

## Main Use Cases

1. **New member setup**: Run `skillet sync` once to deploy project-standard skills
2. **Enforcing project standards**: Skills in `always/` are applied automatically to everyone
3. **Optional skills**: Choose what you need from `available/`

## Directory Structure

### Global Skill Store (SSOT)
```
~/.agents/
├── skills/
│   ├── always/           # Skills always in use
│   │   ├── design-doc/
│   │   └── exec-plan/
│   └── available/        # Optional skills
│       ├── pdf/
│       └── codex/
└── skillet.yaml          # Global config (target settings)
```

### Project Skill Store (SSOT)
```
<project>/
├── .agents/
│   ├── skills/
│   │   ├── always/       # Project-required skills (git-managed)
│   │   │   └── project-conventions/
│   │   └── available/    # Optional skills (git-managed)
│   │       └── api-guidelines/
│   └── skillet.yaml      # Project config
├── .claude/
│   └── skills/           # ← symlink destination (gitignore)
│       ├── design-doc -> ~/.agents/skills/always/design-doc
│       └── project-conventions -> ../../.agents/skills/always/project-conventions
├── .codex/
│   └── skills/           # ← symlink destination (gitignore)
└── .gitignore            # Add .claude/skills/, .codex/skills/
```

### Key Points
- **SSOT**: `.agents/` is the canonical source for skills (git-managed)
- **LLM skill directories**: `.claude/skills/` etc. are in gitignore
- **Per-user customization**: Each user runs `skillet sync` to choose their preferred skills

## Go Project Structure

```
skillet/
├── cmd/skillet/
│   └── main.go
├── internal/
│   ├── cli/              # Cobra commands
│   │   ├── root.go
│   │   ├── init.go       # skillet init
│   │   ├── add.go        # skillet add
│   │   ├── remove.go     # skillet remove
│   │   ├── sync.go       # skillet sync
│   │   ├── list.go       # skillet list
│   │   └── status.go     # skillet status
│   ├── config/           # Viper config management
│   ├── skill/            # Skill domain logic
│   │   ├── skill.go      # Skill struct
│   │   ├── store.go      # SkillStore
│   │   ├── loader.go     # SKILL.md parser
│   │   └── lock.go       # Lock file management
│   ├── target/           # Target adapters
│   │   ├── target.go     # Target interface
│   │   ├── registry.go   # Target registration
│   │   ├── claude.go     # Claude Code
│   │   ├── codex.go      # Codex CLI
│   │   └── generic.go    # Generic .agent
│   ├── sync/             # Sync engine
│   │   ├── engine.go
│   │   ├── strategy.go   # symlink vs copy
│   │   └── resolver.go   # Conflict resolution
│   └── fs/               # File system abstraction
│       ├── system.go     # System interface
│       ├── real.go       # Implementation
│       └── mock.go       # Mock for tests
├── go.mod
└── go.sum
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `skillet init [--global\|--project]` | Initialize |
| `skillet add <source> [--scope global\|project\|user] [--category always\|available]` | Add skill |
| `skillet remove <name> [--scope global\|project\|user]` | Remove skill |
| `skillet sync [--target claude\|codex] [--dry-run] [--force]` | Sync to targets |
| `skillet list [--scope all\|global\|project\|user]` | List skills |
| `skillet status` | Show sync status |

## Main Data Models

### Skill
```go
type Skill struct {
    Name         string            `yaml:"name"`
    Description  string            `yaml:"description"`
    AllowedTools []string          `yaml:"allowed-tools,omitempty"`
    Path         string            // Path to skill directory
    Scope        Scope             // Global, Project, or User
    Category     Category          // always or available
}

type Scope int
const (
    ScopeGlobal  Scope = iota  // ~/.agents/skills/
    ScopeProject               // <project>/.agents/skills/
)

type Category int
const (
    CategoryAlways    Category = iota  // Always used
    CategoryAvailable                  // Optional
)

// ValidateName validates skill name to prevent path traversal attacks
func ValidateName(name string) error
```

### Target Interface
```go
type Target interface {
    Name() string                              // "claude", "codex"
    GlobalPath() string                        // ~/.claude
    ProjectPath() string                       // .claude
    SkillsDir() string                         // skills
    GetGlobalSkillsPath() (string, error)      // ~/.claude/skills
    GetProjectSkillsPath() string              // .claude/skills
    GetInstalledPathForScope(name string, scope Scope) string
    Install(skill *Skill, opts InstallOptions) error
    Uninstall(skillName string) error
    ListInstalled() ([]string, error)          // Lists all installed skills
}
```

## Sync Strategy

### Default: Symlink
- Space efficient
- Source updates are reflected automatically
- Fall back to copy when crossing file system boundaries

### Priority (Conflict Resolution)
```
Project > Global
```
Project config wins; global is next. Used when skills have the same name.

## Config Files

### ~/.agents/skillet.yaml (Global Config)
```yaml
version: 1
defaultStrategy: symlink  # symlink or copy

# Target settings
targets:
  claude:
    enabled: true
    globalPath: ~/.claude
  codex:
    enabled: true
    globalPath: ~/.codex
```

### <project>/.agents/skillet.yaml (Project Config)
```yaml
version: 1

# Skill selection (from available)
skills:
  always:
    - design-doc
    - project-conventions
  user:
    - codex

# Target override (optional)
targets:
  claude:
    enabled: true
  codex:
    enabled: true
```

### Sync Behavior
1. **always/**: Automatically sync everything (no config needed)
   - `~/.agents/skills/always/*` → Applied to all projects
   - `<project>/.agents/skills/always/*` → Applied to that project
2. **available/**: Sync only what the user selected
   - Skills listed in the `skills` section of `<project>/.agents/skillet.yaml`

## Libraries Used

| Library | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Config management |
| `github.com/adrg/xdg` | XDG path resolution |
| `gopkg.in/yaml.v3` | YAML parsing |

## Skill Format (SKILL.md)

Assumes the SKILL.md format from agentskills.io (https://agentskills.io/home)

```markdown
---
name: my-skill
description: |
  Skill description (1-3 sentences)
...
---

## Usage

Detailed skill description...
```


## Implementation Phases

### Phase 1: Foundation
1. [x] Go module init (`go mod init github.com/wwwyo/skillet`)
2. [x] fs/system.go - File system abstraction
3. [x] skill/skill.go - Skill struct definition
4. [x] skill/loader.go - SKILL.md parser
5. [x] cli/root.go - Cobra root command

### Phase 2: Core Commands
6. [x] config/config.go - Viper config management
7. [x] cli/init.go - `skillet init`
8. [x] skill/store.go - SkillStore implementation
9. [x] cli/add.go - `skillet add`
10. [x] cli/list.go - `skillet list`

### Phase 3: Sync Engine
11. [x] target/target.go - Target interface
12. [x] target/claude.go - Claude Code adapter
13. [x] target/codex.go - Codex adapter
14. [x] sync/engine.go - Sync engine
15. [x] cli/sync.go - `skillet sync`
16. [x] cli/status.go - `skillet status`

### Phase 4: Polish
17. [x] cli/remove.go - `skillet remove`
18. [ ] skill/lock.go - Lock file
19. [ ] Add tests
20. [ ] Improve documentation

## Verification

```bash
# Build
go build -o skillet ./cmd/skillet

# Init test
./skillet init --global
./skillet init --project

# Add skill test
./skillet add ./testdata/skills/sample-skill
./skillet list

# Sync test
./skillet sync --dry-run
./skillet sync --target claude
./skillet status

# Verify
ls -la ~/.claude/skills/
ls -la .claude/skills/
```

## Decisions

- **Directory names**: `~/.agents/` (global), `<project>/.agents/` (project)
- **Skill format**: SKILL.md format (Claude Code compatible)
- **Initial targets**: Claude Code, Codex CLI
- **Sync method**: Symlink (default) / Copy (fallback)
- **gitignore**: `.claude/skills/`, `.codex/skills/`
- **Config files**:
  - Global: `~/.agents/skillet.yaml` (target settings)
  - Project: `<project>/.agents/skillet.yaml` (skill selection)
- **Target OS**: macOS/Linux first (Windows/WSL later)
- **Scope priority**: Project > Global
- **Security**: Path traversal protection for skill names (ValidateName function)
- **Testability**: FindProjectRootFrom function allows specifying start directory
