# AGENTS.md

## Project Overview

Skillet is a Go CLI tool that manages AI agent skills as a Single Source of Truth (SSOT). It synchronizes skills from a central store (`~/.agents/` or `<project>/.agents/`) to various AI client directories (`.claude/`, `.codex/`, etc.).

## Architecture

### Package Structure

```
cmd/skillet/     # Entry point
internal/
├── cli/         # Cobra commands (init, remove, list, sync, status, migrate)
├── config/      # Configuration structs and file I/O
├── skill/       # Skill domain: Skill struct, Store, Loader
├── target/      # AI client adapters (claude, codex) with Target interface
├── sync/        # Synchronization engine
└── fs/          # File system abstraction for testability
```

### Key Concepts

- **cli.App**: Dependency container holding `fs.System` and `config.Config`
- **skill.Store**: Manages skills across scopes with priority-based resolution
- **target.Target**: Interface for AI clients; `BaseTarget` provides shared implementation
- **sync.Engine**: Orchestrates symlink/copy from store to targets
- **fs.System**: Abstracts file operations for testing with `fs.MockSystem`

### Data Flow

```
User Command
    ↓
cli/*.go (Cobra command)
    ↓
skill.Store (get skills by scope/priority)
    ↓
target.Registry (get enabled targets)
    ↓
sync.Engine (symlink/copy skills to targets)
    ↓
Target directories (.claude/skills/, .codex/skills/)
```

## Scope & Priority

Skills are organized by scope with priority resolution:

| Scope | Location | Priority |
|-------|----------|----------|
| Global | `~/.agents/skills/` | 1 (lowest) |
| Project | `<project>/.agents/skills/` | 2 (highest) |

When same-named skills exist in multiple scopes, higher priority wins.

## Coding Rules

### File System Operations

- **Always use `fs.System` interface**, never direct `os` package for file operations
- This enables testing with `fs.MockSystem`
- Exception: `os.Getwd()` is allowed but wrap in testable functions like `FindProjectRootFrom()`

### Error Handling

- Return errors with context using `fmt.Errorf("description: %w", err)`
- Never silently swallow errors - propagate or add to result structs (e.g., `Status.Error`)

### Security

- **Always validate skill names** with `skill.ValidateName()` to prevent path traversal
- Reject names containing `/`, `\`, `..`, or starting with `.`

### Configuration

- Global config: `~/.agents/skillet.yaml`
- Project config: `<project>/.agents/skillet.yaml`
- Use `config.Load()` and `config.LoadProject()` to read

### Adding New Targets

1. Create `internal/target/<name>.go`
2. Embed `BaseTarget` struct
3. Override methods if needed (most use `BaseTarget` defaults)
4. Register in `target.NewRegistry()`

### Adding New Commands

1. Create `internal/cli/<command>.go`
2. Define `NewXxxCmd(app *App) *cobra.Command`
3. Add to `rootCmd.AddCommand()` in `root.go`

## Testing Guidelines

- Use `fs.NewMockSystem()` for file system operations
- Use `config.FindProjectRootFrom(fsys, startDir)` instead of `FindProjectRoot()` for testable project root detection
- Test path traversal attacks in skill name validation

## Dependencies

- **Go**: 1.25.4+


## Common Tasks

### Build
```bash
go build ./cmd/skillet
```

### Run Tests
```bash
go test ./...
```

### Lint
```bash
golangci-lint run
```
