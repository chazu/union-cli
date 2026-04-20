# Multi-Store Design

Status: approved via brainstorming 2026-04-19
Scope: single implementation plan

## Goal

Let a user keep multiple independent clause stores (e.g. `personal`, `work`) under one `$UNION_DIR`, each with its own git repo and remotes, and reference clauses from any of them via a qualified `store:path` syntax. Add a command surface for managing per-store git remotes and sync.

Non-goals: cross-store clause aliasing, per-shop pinning to a subset of stores, migration from the current single-store layout (no existing users).

## On-disk layout

```
$UNION_DIR/
  shops.toml
  stores/
    default/
      .git/
      clauses/...
    personal/
      .git/
      clauses/...
    work/
      .git/
      clauses/...
```

Each `stores/<name>/` is an independent git repo. Remotes, branches, and commit history are per-store. Presence of `stores/<name>/.git` is the source of truth for "is this a store"; no separate manifest.

## Qualified clause path syntax

All user-facing clause references use:

```
<store>:<clause-path>         e.g.  personal:writing/voice
```

- Store names: `[a-z0-9][a-z0-9_-]*`. Validated on `store add`.
- Clause paths: existing rules (no `..`, no spaces, no `//`, no leading `/`).
- Unqualified paths (no colon) are a hard error. There is no default store and no implicit prefix.

Contract markers embed the full qualified path:

```
<!-- BEGIN union:personal:writing/voice -->
...body...
<!-- END union:personal:writing/voice -->
```

Marker regex: `union:([a-z0-9_-]+):([^\s]+)`.

This is a breaking change from the single-store marker format; acceptable because there are no existing users.

## Command surface

### New: `union store ...`

| Command | Behavior |
|---|---|
| `union store add <name>` | Create `$UNION_DIR/stores/<name>/`, `git init`, seed `clauses/.gitkeep`, initial commit |
| `union store list` | List store names by scanning `$UNION_DIR/stores/` for `.git` |
| `union store remove <name>` | Delete the store. Refuses if any shop has ratified a clause from it; lists those shops |
| `union store remote add <store> <name> <url>` | Thin wrapper over `git remote add` |
| `union store remote remove <store> <name>` | `git remote remove` |
| `union store remote list <store>` | Prints `<name>\t<url>` per remote |
| `union store push <store> [remote] [branch]` | `git push` in the store's repo |
| `union store pull <store> [remote] [branch]` | `git pull --rebase` |
| `union store fetch <store> [remote]` | `git fetch` |
| `union store status <store>` | `git status` summary |

No generic `git` escape hatch. No manual `commit` command; clause edits continue to auto-commit.

### Changed: existing commands

- `union init` with no args: creates `$UNION_DIR/` and a `default` store via the same code path as `store add default`.
- `union init <name>`: uses the given name instead of `default`.
- `union new <qpath> [-f FILE]`: writes into the named store.
- `union show|edit|expel|ratify|strike <qpath>`: require qualified paths.
- `union clauses`: lists across all stores as `store:path`, sorted lexically.
- `union clauses <store>:`: filters to one store.
- `union clauses <store>:<prefix>`: prefix filter within a store.
- `union contract`: outputs qualified paths.

Shops may ratify from any store (no pinning).

## Internal architecture

### `internal/paths`

Add:

- `StoresDir() (string, error)` → `$UNION_DIR/stores`
- `StoreDir(name string) (string, error)` → `$UNION_DIR/stores/<name>`, validates the name

Existing `UnionDir` and `ShopsFile` unchanged. `ClausesDir` is removed (no longer a single canonical clauses dir).

### `internal/store`

`Store` remains single-repo. Add package-level helpers:

- `ListStores(unionDir string) ([]string, error)` — scans `stores/` for subdirs containing `.git`, sorted.
- `OpenNamed(unionDir, name string) (*Store, error)` — resolves path and calls `Open`.
- `InitNamed(unionDir, name string) (*Store, error)` — creates `stores/<name>/` and initializes it.

New methods on `*Store`:

- `RemoteAdd(name, url string) error`
- `RemoteRemove(name string) error`
- `Remotes() ([]Remote, error)` with `type Remote struct { Name, URL string }`
- `Push(remote, branch string) error`
- `Pull(remote, branch string) error` (uses `--rebase`)
- `Fetch(remote string) error`
- `Status() (string, error)` (returns `git status --short --branch` output)

Git invocation reuses the existing `(*Store).git` helper.

### `internal/qpath` (new)

```go
type Qualified struct { Store, Path string }
func Parse(s string) (Qualified, error)   // splits on first ':'; validates both halves
func (q Qualified) String() string        // "store:path"
```

Single owner of the `store:path` grammar. Consumed by CLI commands, marker parsing, and contract writing.

### `internal/shop/markers`

- Marker token type becomes `qpath.Qualified`.
- Regex updated as above.
- `Ratify`/`Strike` key blocks by the qualified form, so a contract can hold `personal:x` and `work:x` without collision.

### `internal/cli/propagate.go`

On `edit` or `expel` of a qualified clause, iterate shops, rewrite blocks whose qualified token matches. The propagate loop opens one `*Store` per distinct store encountered during the pass and caches it for the duration.

### Shop registry

`shops.toml` schema is unchanged — it only tracks `dir` → `contract`. Ratified clauses live in the contract file as qualified strings.

## Error handling

- Unqualified path → `clause path must be qualified as <store>:<path>`.
- Unknown store on any operation → `no such store: <name> (run 'union store list')`.
- Invalid store name on `store add` → error citing allowed charset.
- Existing store on `store add` → `store already exists: <name>`.
- `store remove` while ratified clauses exist → refused; error lists the shops.
- Git ops surface git's stderr verbatim and propagate the non-zero exit.
- `pull --rebase` conflicts: user resolves in `$UNION_DIR/stores/<name>/` and re-runs. No union-level rebase helper.
- Legacy unqualified markers encountered during propagation → hard error; re-ratify under new scheme.

## Testing

- `internal/qpath`: unit tests for `Parse` (valid, missing colon, empty halves, bad charset, reused clause-path validation).
- `internal/store`: extend existing tests with `InitNamed`/`OpenNamed`/`ListStores`; add remote add/remove/list and push/pull/fetch tests that use a local bare repo as the remote.
- `internal/shop/markers`: update existing tests to qualified tokens; add cases mixing clauses from two stores in one contract.
- `internal/cli/e2e_test.go`: extend to cover
  - `init` with no args creates `default`
  - `store add personal`, `store list`
  - Full `new`/`ratify`/`edit`/`expel` cycle with qualified paths
  - Two-store shop: ratify `personal:a` and `work:b` into one `AGENTS.md`; edit each; assert propagation touches only the matching block
  - `store remove` refusal when clauses are still ratified
  - `store remote add` + `push` + `pull` against a temp bare repo

## Out of scope

- Migration from the old single-store layout.
- Default-store shorthand (`~/store default personal`).
- Per-shop store allowlists.
- Generic `git` escape hatch under `union store`.
- Manual commit command.
