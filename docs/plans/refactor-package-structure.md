# Refactor Plan: Package Structure and Responsibilities

## Goal

Reorganize `internal/` around `skill`, `config`, and `usecase`, and keep target awareness in usecase logic without introducing a dedicated `target` package.

This keeps the structure simple while preserving explicit handling for `sync`, `remove`, `status`, and `migrate`.

## Proposed Tree (Post-refactor)

```text
cmd/
└── skillet/
    └── main.go

internal/
├── cli/
│   ├── root.go
│   ├── flags.go
│   ├── init.go
│   ├── list.go
│   ├── sync.go
│   ├── remove.go
│   ├── status.go
│   └── migrate.go
├── config/
│   ├── config.go          # Config model, strategy, target settings
│   ├── store.go           # Load/Save config
│   ├── paths.go           # path expansion and path helpers
│   └── store_test.go
├── skill/
│   ├── model.go           # Skill, Scope, Category, ValidateName
│   ├── store.go           # skill loading and conflict resolution
│   └── store_test.go
├── usecase/
│   ├── setup.go           # SetupService + params/results
│   ├── list.go            # ListService + params/results
│   ├── sync.go            # SyncService + params/results
│   ├── remove.go          # RemoveService + params/results
│   ├── status.go          # StatusService + params/results
│   ├── migrate.go         # MigrateService + params/results
│   └── target_resolver.go # enabled targets + scope-based target paths
└── platform/
    └── fs/
        ├── fs.go
        ├── os_fs.go
        ├── mock_fs.go
        └── fs_test.go
```

## Package Responsibilities

### `internal/cli`
- Cobra command definitions
- Request parsing and output formatting
- Composition root (manual DI wiring)

### `internal/config`
- Configuration schema and defaults
- Config file loading/saving
- General path helpers (`~` expansion, global config path)

### `internal/skill`
- Skill model and validation (`ValidateName`)
- Loading skills from global/project scope
- Conflict resolution (`project > global`)
- No direct `config` import

### `internal/usecase`
- Command-level business logic (`setup`, `list`, `sync`, `remove`, `status`, `migrate`)
- Target resolution for operations (`target_resolver.go`)
- Depends on `skill`, `config`, and `platform/fs`

### `internal/platform/fs`
- Filesystem abstraction and implementations
- Mock filesystem for tests

## DI Structure (Manual DI)

Wiring happens in `internal/cli/root.go`:

1. create filesystem (`platform/fs`)
2. create config store and load config (`config/store`)
3. detect project root
4. create skill store (`skill/store`)
5. create usecase services (inject fs, config, skill store, project root)
6. bind services to CLI commands

## Dependency Rules

Allowed dependency direction:

```text
cli -> usecase/skill/config/platform
usecase -> skill/config/platform
skill -> platform
config -> platform
```

Disallowed:

- `usecase` to `cli`
- circular imports in `usecase`
- filesystem access outside `platform/fs`

## Usecase File Rule

Use a flat package first.

- Define service and related types in the same file (example: `sync.go` contains `SyncService`, `SyncOptions`, `SyncResult`)
- Prefer explicit names (`SyncService`, not generic `Service`)
- Split files only when a single file becomes too large or has mixed responsibilities

## Target Handling Policy

`claude` and `codex` currently differ only by path/config, not behavior.

Still, every relevant usecase must be target-aware:

- `sync`: install/update per target path
- `remove`: uninstall from all enabled targets before removing source skill
- `status`: compare store state vs installed state per target
- `migrate`: inspect target directories and import unmanaged skills

Target handling is implemented as usecase logic with config-driven definitions.
