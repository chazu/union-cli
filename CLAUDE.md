# Union Development Guide

Composable, versioned snippet management for AGENTS.md / CLAUDE.md files.

## Build & Test

```bash
make build          # build binary with version injection + codesign
make install        # install to $GOBIN with version injection
make test           # go test ./...
make vet            # go vet ./...
```

## Architecture

```
cmd/union/main.go       → entry point, delegates to cli.Execute()
internal/cli/           → cobra commands, e2e tests
internal/store/         → git-backed clause storage (auto-commit on mutation)
internal/shop/          → shop registry (TOML) + contract marker parsing
internal/harness/       → cross-harness hook adapters, config, template expansion
internal/qpath/         → qualified path validation (store:clause/path)
internal/paths/         → filesystem layout resolution ($UNION_DIR)
```

## Key Concepts

- **Clause**: versioned markdown snippet stored in a git-backed store
- **Store**: git repo at `$UNION_DIR/stores/<name>/` containing clauses
- **Shop**: registered project directory with a contract file (default: AGENTS.md)
- **Contract**: shop's AGENTS.md with HTML comment markers wrapping ratified clauses
- **Qualified path**: `store:clause/path` — always includes store name
- **Harness**: AI coding tool adapter (Claude Code, OpenCode, Codex, JCode)
- **Hook clause**: clause with YAML frontmatter (type: hook) that emits to native harness configs
- **Pointer**: file containing `@AGENTS.md` redirecting a harness to the canonical contract

## Conventions

- Minimal dependencies: stdlib + cobra + toml + x/term
- Every store mutation auto-commits via git
- Atomic file writes use write-then-rename pattern (see shop/registry.go)
- Tests use `t.TempDir()` for isolation; git-dependent tests call `requireGit()`
- Table-driven tests for validation (see qpath_test.go)
- `$UNION_DIR` env var overrides default `~/.union` (used extensively in tests)

## Adding a New Command

1. Create `internal/cli/<command>.go` with `func new<Command>Cmd() *cobra.Command`
2. Register in `root.go`'s `newRootCmd()` → `root.AddCommand(...)`
3. Add e2e test coverage in `internal/cli/e2e_test.go`
