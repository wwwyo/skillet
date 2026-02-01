# Skillet

AI Agent Skills Manager - Manage your AI agent skills as a Single Source of Truth (SSOT).

## What is Skillet?

Skillet solves two key problems when working with AI coding assistants like Claude Code and Codex CLI:

1. **Skill Fragmentation**: Same skills copied across multiple directories (`.claude/skills/`, `.codex/skills/`) leading to inconsistent updates
2. **Team vs Personal Customization**: Difficulty balancing team-standard skills with personal preferences

Skillet provides:
- A central skill store (`~/.agents/` for global, `.agents/` for project)
- Automatic synchronization to AI client directories
- Priority-based conflict resolution (Project > Global)
- Git-friendly structure for team collaboration

## Installation

```bash
go install github.com/wwwyo/skillet/cmd/skillet@latest
```

## Quick Start

### 1. Initialize Global Store

```bash
skillet init --global
```

This creates `~/.agents/` with the default configuration.

**Tip: Using with Dotfiles**

If you manage your dotfiles in a Git repository, you can specify a custom path:

```bash
skillet init -g --path ~/dotfiles/.agents
```

This allows you to version control your global skills alongside other dotfiles.

### 2. Initialize Project Store

```bash
cd your-project
skillet init --project
```

This creates `.agents/` directory in your project root.

### 3. Sync to AI Clients

```bash
# Dry run to see what would happen
skillet sync --dry-run

# Sync to all targets
skillet sync

# Sync to specific target only
skillet sync --target claude
```

### 4. Check Status

```bash
skillet status
```

## Directory Structure

### Global Configuration (`~/.config/skillet/`)

```
~/.config/skillet/
└── config.yaml           # Global configuration
```

### Global Skills (`~/.agents/` or custom path)

```
~/.agents/
└── skills/
    ├── skill-a/          # Always-active skills
    └── optional/         # Optional skills
```

### Project Store (`<project>/.agents/`)

```
<project>/
├── .agents/
│   ├── skills/
│   │   ├── skill-a/      # Project skills (git tracked)
│   │   └── optional/     # Optional skills (git tracked)
│   └── skillet.yaml      # Project configuration
├── .claude/
│   └── skills/           # Symlinked from .agents (gitignore)
└── .codex/
    └── skills/           # Symlinked from .agents (gitignore)
```

## Commands

| Command | Description |
|---------|-------------|
| `skillet init [--global\|--project]` | Initialize skill store |
| `skillet remove <name> [--scope]` | Remove a skill |
| `skillet list [--scope]` | List skills |
| `skillet sync [--target] [--dry-run] [--force]` | Sync to AI clients |
| `skillet status` | Show sync status |
| `skillet migrate` | Migrate existing skills from targets to agents directory |

## Configuration

### Global Config (`~/.config/skillet/config.yaml`)

```yaml
version: 1
globalPath: ~/.agents     # Path to global skills (customizable for dotfiles)
defaultStrategy: symlink  # symlink or copy

targets:
  claude:
    enabled: true
    globalPath: ~/.claude
  codex:
    enabled: true
    globalPath: ~/.codex
```

### Project Config (`<project>/.agents/skillet.yaml`)

```yaml
version: 1

targets:
  claude:
    enabled: true
  codex:
    enabled: true
```

## Instructions

Detailed instructions for the AI agent...
```

## Priority Resolution

When the same skill name exists in multiple scopes:

```
Project (highest) > Global (lowest)
```

Project-scope skills override global-scope skills.

## Gitignore Setup

Add to your project's `.gitignore`:

```gitignore
# AI client skill directories (managed by skillet)
.claude/skills/
.codex/skills/
```

## Supported Targets

- **Claude Code** (`.claude/`)
- **Codex CLI** (`.codex/`)

## License

MIT
