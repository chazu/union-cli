# Union MVP — Design

**Status:** Draft for review
**Date:** 2026-04-16
**Module:** `github.com/chazu/union`

## Summary

Union is a CLI for managing reusable snippets of agent context (AGENTS.md /
CLAUDE.md files) across many projects. Snippets — called **clauses** — live
in a central git-backed store and are composed into registered projects —
called **shops** — via marked regions in their AGENTS.md, which union calls
the **contract**.

The design borrows its vocabulary from labor unions: clauses make up the
contract; shops are organized; clauses are ratified or struck from a
contract; clauses are expelled from the store.

## Goals

- Git-versioned snippet store with hierarchical paths (`base/identity`,
  `lang/go`, `servers/foo/user@host` — pass-style).
- Round-trip editing: union owns marked regions inside a contract, but
  never touches user content around them.
- Central registry of "which projects do I manage AGENTS.md for" that
  agents can query.
- No separate "build" step: ratifying/striking a clause mutates the
  contract immediately; edits to the store auto-propagate to ratified
  shops.

## Non-goals (v0.1)

- Profiles / named reusable compositions. A shop's contract *is* the
  composition. Revisit once concrete sharing pressure appears.
- Encryption (pass-specific; not needed here).
- Remote sources (git-registry, HTTP fetch). All clauses are local.
- CUE / conditional inclusion / expression engine.
- Linting (`steward`), lockfile/version pinning, LLM test harness.
- Adopt workflow (ingesting unmarked content from an existing AGENTS.md).
  Deferred to v0.2.

## Concepts

| Concept | Labor word | Meaning |
|---|---|---|
| Snippet in the store | **clause** | A discrete, reusable policy/context chunk |
| Registered project | **shop** | A directory union composes AGENTS.md for |
| Target file in a shop | **contract** | The shop's AGENTS.md (path configurable) |
| Add clause to a contract | **ratify** | Insert marked block into the shop's contract |
| Remove clause from a contract | **strike** | Delete the marked block |
| Remove clause from the store | **expel** | Delete from `$UNION_DIR/clauses/` |
| Register a shop | **organize** | Add to `$UNION_DIR/shops.toml` |
| Unregister a shop | **disband** | Remove from `shops.toml` |

## On-disk layout

```
$UNION_DIR/                          # default: ~/.union, override via UNION_DIR env
  .git/                              # auto-committed on every mutation
  clauses/                           # clause content — hierarchical paths
    base/
      identity.md
      style.md
    lang/
      go.md
  shops.toml                         # registry of organized shops
```

`shops.toml`:

```toml
[shops."/Users/chazu/dev/go/myapp"]
contract = "AGENTS.md"               # filename within shop, default AGENTS.md
# future: profile = "go-dev", tags = [...]
```

## Contract markers

Clauses ratified into a contract are wrapped in HTML comment markers:

```markdown
<!-- BEGIN union:base/identity -->
...clause content (verbatim from the store)...
<!-- END union:base/identity -->
```

- Markers are the sole source of truth for "which clauses are in this
  shop's contract." Union parses the contract to answer `union contract`.
- Content outside marker pairs is preserved untouched. Users can write
  prose before, between, or after ratified blocks.
- On first ratify in an unmarked file: append the block to the end of
  the contract. Subsequent ratifications also append (users move blocks
  manually; union finds them by marker id, not position).
- No content hash in the marker. Git history in the store is the drift
  record.

## Command surface (v0.1)

```
union init                           # create $UNION_DIR, init git, seed shops.toml

# clause ops (store)
union new <path>                     # author new clause: $EDITOR if tty, stdin otherwise
union new <path> -f <file>           # seed from file
union clauses [prefix]               # list clauses (optionally filter by prefix)
union show <path>                    # print clause content
union edit <path>                    # open in $EDITOR, auto-commit on save
union expel <path>                   # delete from store + commit

# shop ops (registry)
union organize [dir]                 # register dir (default .) as a shop
union shops                          # list registered shops
union disband <dir>                  # unregister a shop

# contract ops (operate on current shop — cwd must be an organized shop)
union ratify <path>                  # insert marked block into ./AGENTS.md
union strike <path>                  # remove marked block from ./AGENTS.md
union contract                       # show which clauses are present in ./AGENTS.md
```

### Behaviors worth pinning

- **`union new` input precedence:** `-f <file>` wins if given. Else, if
  stdin is not a tty, read it as the clause body. Else open `$EDITOR`
  on a temp file and use its contents on save.
- **`union ratify` / `union strike` scope:** operate on the current
  working directory only. The cwd must match a registered shop, else
  error with a hint to run `union organize`. No shop-path argument in
  v0.1.
- **Edit propagation (the "no build step" promise):** when a clause is
  edited or expelled, union walks `shops.toml`, finds shops whose
  contract contains a matching `BEGIN union:<path>` marker, and
  rewrites the marked block (or removes it, for expel). Union does
  **not** run `git add`/`git commit` inside shops — the user sees the
  diff and commits. For expel, the entire marker pair is removed.
- **Store git:** every mutation (`new`, `edit`, `expel`) auto-commits in
  `$UNION_DIR` with a generated message (e.g. `new base/identity`,
  `edit lang/go`, `expel tone/snarky`). Authoring environment (user,
  email) inherits from the user's global git config.
- **Missing clause on ratify:** error. No silent creation.
- **Duplicate ratify:** no-op (marker already present). Exit 0.
- **Contract file missing on first ratify:** create it with just the
  marked block. Union never refuses to act because a target file is
  absent — that would make `organize` then `ratify` awkward.

## Architecture

Three internal packages, each with one clear purpose:

### `internal/store`
The clause store. A git-backed directory.

- `Open(dir) (*Store, error)` / `Init(dir) (*Store, error)`
- `Put(path string, body []byte, msg string) error`
- `Get(path string) ([]byte, error)`
- `Delete(path, msg string) error`
- `List(prefix string) ([]string, error)`
- `History(path string) ([]Commit, error)` (reserve for future; not used
  by any v0.1 command, stub if helpful)

Implementation: thin shell-out to `git` (like pudl's `internal/git`).
Every mutating call commits. This is the package that's a candidate for
future extraction to a `gitstore` library (see `TODO-idea.md` in pudl).

### `internal/shop`
The shops registry and contract file manipulation.

- `LoadRegistry(path string) (*Registry, error)`
- `(*Registry) Add(dir, contractName string) error`
- `(*Registry) Remove(dir string) error`
- `(*Registry) List() []Shop`
- `(*Registry) Save() error`

Contract parsing/mutation (pure string ops on markdown):

- `ParseContract(body []byte) ([]MarkedBlock, error)` — returns the list
  of `{path, startLine, endLine, content}` blocks.
- `InsertClause(body []byte, path string, clause []byte) ([]byte, error)`
- `UpdateClause(body []byte, path string, clause []byte) ([]byte, error)`
- `RemoveClause(body []byte, path string) ([]byte, error)`

Keep parsing trivial: line-based scan for `<!-- BEGIN union:X -->` and
matching `<!-- END union:X -->`. No markdown AST. Errors on malformed
pairs (orphan BEGIN/END).

### `internal/cli`
Command wiring uses [spf13/cobra](https://github.com/spf13/cobra). Each
subcommand is its own file with a `cobra.Command` definition. Commands
only orchestrate `store` and `shop`; no business logic in cli.

### `cmd/union/main.go`
Entry point. Parses args, dispatches to `internal/cli`.

## Data flow

**`union new base/identity` (stdin):**
1. cli reads stdin body.
2. `store.Put("base/identity", body, "new base/identity")` writes file
   and commits in the store.
3. No propagation — clause hasn't been ratified anywhere yet.

**`union ratify base/identity` (from cwd `/Users/chazu/dev/go/myapp`):**
1. cli loads `shops.toml`, confirms `/Users/chazu/dev/go/myapp` is
   registered, finds its contract name (`AGENTS.md`).
2. `store.Get("base/identity")` fetches clause body.
3. Reads `AGENTS.md`, calls `shop.InsertClause(body, "base/identity",
   clause)`, writes result back.
4. Does **not** commit in the shop's git repo.

**`union edit base/identity`:**
1. cli opens `$EDITOR` on the clause file.
2. On save, `store.Put(...)` commits.
3. cli walks `shops.toml`, for each shop whose contract contains the
   `BEGIN union:base/identity` marker, calls
   `shop.UpdateClause(...)` and writes the contract back. Uncommitted.
4. Reports which shops were updated.

**`union expel base/identity`:**
1. `store.Delete(...)` removes + commits in store.
2. Walks shops, for each matching shop calls `shop.RemoveClause(...)`
   and writes back. Uncommitted.
3. Reports affected shops.

## Error handling

Errors are returned with enough context to act on:

- Not-in-a-shop: `union ratify` when cwd isn't registered →
  `error: /Users/... is not an organized shop. Run 'union organize' first.`
- Missing clause: `union ratify foo/bar` when clause doesn't exist →
  `error: no such clause: foo/bar. See 'union clauses' for available paths.`
- Malformed markers: `union contract` when BEGIN has no matching END →
  `error: malformed marker in AGENTS.md at line 42 (BEGIN union:foo with
  no END)`
- Store not initialized: any command besides `init` when `$UNION_DIR`
  doesn't exist → `error: no union store at <dir>. Run 'union init'.`

No over-engineered typed errors for v0.1. Plain `fmt.Errorf` with
`%w`-wrapping is enough.

## Testing strategy

- **`internal/store`:** table-driven tests using a temp dir as the store.
  Real `git` binary (skip if `git` not on `$PATH`). Exercises
  Put/Get/Delete/List/history.
- **`internal/shop` parsing:** pure-function tests over canned markdown
  inputs. No filesystem needed. Cover: empty file, single block, multiple
  blocks, blocks with surrounding prose, malformed markers, unicode
  content, trailing-newline preservation.
- **`internal/shop` registry:** temp-dir tests for `shops.toml` load /
  save / add / remove round-trips.
- **CLI end-to-end:** one or two black-box tests per command using a
  temp `UNION_DIR` and a temp "shop" dir. Focus on the propagation flow
  (`new`, `organize`, `ratify`, `edit` → confirm shop contract changed).

Target coverage: 80%+ on `store` and `shop`; cli tested via the e2e
flow rather than per-flag.

## Open questions (for v0.2+)

- Adopt workflow: `union organize --adopt` that scans an existing
  AGENTS.md, proposes chunks, extracts them into clauses, and rewrites
  the file with markers.
- Profiles once a real sharing pattern appears.
- `union repo show <dir>` — operate on a non-cwd shop.
- Lint (`union steward`) — bylaws, token budgets, duplicate headers.
- Should edits propagate into shops' git index? Currently no; reconsider
  if users report surprise.

## Milestones

- **M1:** `init`, `new`, `clauses`, `show`, `edit`, `expel` —
  the store is usable standalone.
- **M2:** `organize`, `shops`, `disband`, `ratify`, `strike`, `contract`
  — shops and contracts end-to-end.
- **M3:** Edit-propagation for `edit` and `expel`. Ship v0.1.
