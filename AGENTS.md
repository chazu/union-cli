# Union — Agent Guide

Union is a composable, versioned snippet manager for agent context files.
Run `union guide overview` for full details.

## Quick Reference

```bash
union guide overview     # what union is
union guide hooks        # cross-harness hook management
union guide clauses      # clause lifecycle
union guide emit         # how emit renders native configs
union guide templates    # template variable reference
```

## Build & Test

```bash
make build    # build with version injection + codesign
make test     # go test ./...
make vet      # go vet ./...
```

## Architecture

```
cmd/union/main.go       → entry point
internal/cli/           → cobra commands
internal/store/         → git-backed clause storage
internal/shop/          → shop registry + contract parsing
internal/harness/       → cross-harness adapters (Claude, OpenCode, Codex, JCode)
internal/qpath/         → qualified path validation
internal/paths/         → filesystem layout
```

## Key Concepts

- **Clause**: versioned markdown snippet in a git-backed store
- **Store**: git repo at `$UNION_DIR/stores/<name>/`
- **Shop**: registered project with a contract file (default: AGENTS.md)
- **Contract**: shop's AGENTS.md with HTML comment markers wrapping clauses
- **Harness**: AI coding tool adapter — manages hooks across Claude Code, OpenCode, Codex, JCode
- **Hook clause**: clause with `type: hook` frontmatter → emits to native harness configs
- **Pointer**: file containing `@AGENTS.md` redirecting harness-specific files to the contract

## Cross-Harness Hooks

Hook clauses use normalized event names that map to each harness's native format.
Unsupported events degrade gracefully (skip, warn, or error per clause config).

```bash
union harness detect     # find harnesses in current shop
union emit               # preview native config changes
union emit --write       # apply changes
union import hooks       # import existing harness hooks
```

## Conventions

- Minimal dependencies: stdlib + cobra + toml + x/term
- Every store mutation auto-commits via git
- Atomic file writes use write-then-rename pattern
- Tests use `t.TempDir()` for isolation
- `$UNION_DIR` env var overrides default `~/.union`

## Adding a New Command

1. Create `internal/cli/<command>.go` with `func new<Command>Cmd() *cobra.Command`
2. Register in `root.go`'s `newRootCmd()` → `root.AddCommand(...)`
3. Add e2e test coverage in `internal/cli/e2e_test.go`

## Adding a New Harness Adapter

1. Create `internal/harness/<name>.go` implementing the `Adapter` interface
2. Register in `harness.go`'s `All()` function
3. Add detection heuristics, event mapping, and Emit/Import methods
4. Add tests in `harness_test.go`
